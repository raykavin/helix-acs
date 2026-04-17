package cwmp

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/raykavin/helix-acs/packages/datamodel"
	"github.com/raykavin/helix-acs/packages/device"
	"github.com/raykavin/helix-acs/packages/logger"
	"github.com/raykavin/helix-acs/packages/schema"
	"github.com/raykavin/helix-acs/packages/task"
)

type SessionState int

const (
	StateNew        SessionState = iota // freshly created, no Inform yet
	StateInform                         // Inform received and processed
	StateProcessing                     // executing pending tasks
	StateDone                           // session complete
)

// Session tracks per-connection CWMP state.
type Session struct {
	ID           string
	DeviceSerial string
	State        SessionState
	CreatedAt    time.Time
	mu           sync.Mutex
	pendingTasks []*task.Task     // tasks waiting to be dispatched to the CPE
	currentTask  *task.Task       // task currently awaiting a CPE response
	mapper       datamodel.Mapper // data-model mapper resolved during Inform

	// Port-forwarding AddObject follow-up: after receiving AddObjectResponse
	// this function is called with the new instance number to build the
	// subsequent SetParameterValues request.
	addObjFollowUp func(instanceNum int) ([]byte, error)
}

func (s *Session) setState(st SessionState) {
	s.mu.Lock()
	s.State = st
	s.mu.Unlock()
}

// SessionManager manages active CWMP sessions, keyed by session ID.
type SessionManager struct {
	sessions sync.Map
	mu       sync.RWMutex
}

// NewSessionManager creates a new SessionManager.
func NewSessionManager() *SessionManager { return &SessionManager{} }

// GetOrCreate returns the Session for the given sessionID, creating a new one if needed.
func (sm *SessionManager) GetOrCreate(sessionID string) *Session {
	s := &Session{
		ID:        sessionID,
		State:     StateNew,
		CreatedAt: time.Now().UTC(),
	}
	actual, _ := sm.sessions.LoadOrStore(sessionID, s)
	return actual.(*Session)
}

// Delete removes the Session for the given sessionID.
func (sm *SessionManager) Delete(sessionID string) { sm.sessions.Delete(sessionID) }

// Cleanup removes sessions older than 30 minutes.
func (sm *SessionManager) Cleanup() {
	cutoff := time.Now().UTC().Add(-30 * time.Minute)
	sm.sessions.Range(func(key, value any) bool {
		s := value.(*Session)
		if s.CreatedAt.Before(cutoff) {
			sm.sessions.Delete(key)
		}
		return true
	})
}

// Handler implements http.Handler and manages CWMP sessions and message handling.
type Handler struct {
	deviceSvc      device.Service
	taskQueue      task.Queue
	sessionMgr     *SessionManager
	log            logger.Logger
	acsUsername    string
	acsPassword    string
	acsURL         string
	informInterval time.Duration
	schemaRegistry *schema.Registry
	schemaResolver *schema.Resolver
}

// NewHandler creates a new CWMP Handler with the given dependencies and configuration.
func NewHandler(
	deviceSvc device.Service,
	taskQueue task.Queue,
	log logger.Logger,
	username, password, acsURL string,
	informInterval time.Duration,
	schemaReg *schema.Registry,
) *Handler {
	return &Handler{
		deviceSvc:      deviceSvc,
		taskQueue:      taskQueue,
		sessionMgr:     NewSessionManager(),
		log:            log,
		acsUsername:    username,
		acsPassword:    password,
		acsURL:         acsURL,
		informInterval: informInterval,
		schemaRegistry: schemaReg,
		schemaResolver: schema.NewResolver(schemaReg),
	}
}

// ServeHTTP implements the full CWMP session lifecycle per TR-069.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sessionID := h.getSessionID(r)
	session := h.sessionMgr.GetOrCreate(sessionID)

	http.SetCookie(w, &http.Cookie{
		Name:     "cwmp-session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
	})

	ctx := r.Context()

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		h.log.WithError(err).
			WithField("session", sessionID).Error("CWMP: Failed to read request body")

		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Empty body - CPE acknowledges the last ACS response.
	// Dispatch next pending task, or close session.
	if len(bytes.TrimSpace(body)) == 0 {
		session.mu.Lock()
		var next *task.Task
		if len(session.pendingTasks) > 0 {
			next = session.pendingTasks[0]
			session.pendingTasks = session.pendingTasks[1:]
		}
		session.mu.Unlock()

		if next != nil {
			h.dispatchTask(ctx, w, session, next)
			return
		}
		session.setState(StateDone)
		h.sessionMgr.Delete(sessionID)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	env, err := ParseEnvelope(body)
	if err != nil {
		h.log.WithError(err).WithField("session", sessionID).Error("CWMP: Failed to parse SOAP envelope")
		h.writeSoapFault(w, sessionID, "Client", "9003", "Invalid arguments")
		return
	}

	switch {
	case env.Body.Inform != nil:
		h.handleInform(ctx, w, r, env, session)

	case env.Body.GetRPCMethods != nil:
		respXML, ferr := h.handleGetRPCMethods(ctx, env)
		if ferr != nil {
			h.log.WithError(ferr).Error("CWMP: handleGetRPCMethods")
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		writeXML(w, respXML)

	case env.Body.TransferComplete != nil:
		respXML, ferr := h.handleTransferComplete(ctx, env)
		if ferr != nil {
			h.log.WithError(ferr).Error("CWMP: handleTransferComplete")
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		writeXML(w, respXML)

	// CPE responses to ACS-initiated RPC requests
	case env.Body.SetParameterValuesResponse != nil:
		h.handleSetParamValuesResponse(ctx, w, session)

	case env.Body.GetParameterValuesResponse != nil:
		h.handleGetParamValuesResponse(ctx, w, session, env.Body.GetParameterValuesResponse)

	case env.Body.RebootResponse != nil:
		h.handleTaskResponse(ctx, w, session, nil, "")

	case env.Body.FactoryResetResponse != nil:
		h.handleTaskResponse(ctx, w, session, nil, "")

	case env.Body.DownloadResponse != nil:
		h.handleDownloadResponse(ctx, w, session, env.Body.DownloadResponse)

	case env.Body.AddObjectResponse != nil:
		h.handleAddObjectResponse(ctx, w, session, env.Body.AddObjectResponse)

	case env.Body.DeleteObjectResponse != nil:
		h.handleTaskResponse(ctx, w, session, nil, "")

	case env.Body.Fault != nil:
		faultMsg := fmt.Sprintf("fault %s: %s", env.Body.Fault.FaultCode, env.Body.Fault.FaultString)
		h.handleTaskResponse(ctx, w, session, nil, faultMsg)

	default:
		h.log.WithField("session", sessionID).
			Warn("CWMP: Unrecognised message body; closing session")
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleInform processes the Inform message, upserts the device, loads pending tasks,
func (h *Handler) handleInform(ctx context.Context, w http.ResponseWriter, _ *http.Request, env *Envelope, session *Session) {
	id := headerID(env)
	upsertReq := h.extractInformParams(env)

	session.mu.Lock()
	session.DeviceSerial = upsertReq.Serial
	session.mu.Unlock()

	h.log.
		WithField("session", session.ID).
		WithField("serial", upsertReq.Serial).
		WithField("manufacturer", upsertReq.Manufacturer).
		Info("CWMP: Inform received from CPE")

	dev, err := h.deviceSvc.UpsertFromInform(ctx, upsertReq)
	if err != nil {
		h.log.WithError(err).WithField("serial", upsertReq.Serial).Error("CWMP: Upsert device failed")
		h.writeSoapFault(w, id, "Server", "9002", "Internal error")
		return
	}

	h.log.WithField("serial", dev.Serial).
		WithField("id", dev.ID.Hex()).Debug("CWMP: Device upserted")

	modelType := datamodel.DetectFromRootObject(firstRootObject(upsertReq.Parameters))
	instanceMap := datamodel.DiscoverInstances(upsertReq.Parameters)

	// Resolve the schema name for this device (e.g. "tr181", "vendor/huawei/tr181").
	schemaName := h.schemaResolver.Resolve(upsertReq.Manufacturer, upsertReq.ProductClass, upsertReq.DataModel)
	upsertReq.Schema = schemaName

	h.log.
		WithField("serial", upsertReq.Serial).
		WithField("schema", schemaName).
		Debug("CWMP: resolved device schema")

	// Build a schema-driven mapper; fall back to the standard mapper when
	// the registry is empty or the schema is not found.
	var mapper datamodel.Mapper
	if sm := schema.NewSchemaMapper(h.schemaRegistry, schemaName, instanceMap); sm != nil {
		mapper = sm
	} else {
		mapper = datamodel.ApplyInstanceMap(datamodel.NewMapper(modelType), instanceMap)
	}

	// Fetch real pending tasks.
	pendingTasks, err := h.taskQueue.DequeuePending(ctx, upsertReq.Serial)
	if err != nil {
		h.log.WithError(err).WithField("serial", upsertReq.Serial).Error("CWMP: Dequeue pending tasks failed")
	}

	// On "8 DIAGNOSTICS COMPLETE", prepend synthetic result-collection tasks
	// so results are fetched before executing new configuration tasks.
	var synthTasks []*task.Task
	if hasDiagnosticsCompleteEvent(env.Body.Inform) {
		diagTasks, _ := h.taskQueue.FindExecutingDiagnostics(ctx, upsertReq.Serial)
		for _, dt := range diagTasks {
			paths := task.BuildDiagResultPaths(dt.Type, mapper)
			if len(paths) == 0 {
				continue
			}
			payload, _ := json.Marshal(task.GetDiagResultPayload{
				OriginalTaskID:   dt.ID,
				OriginalTaskType: dt.Type,
				Paths:            paths,
			})
			synthTasks = append(synthTasks, &task.Task{
				ID:      dt.ID + "_collect",
				Serial:  upsertReq.Serial,
				Type:    task.TypeGetDiagResult,
				Payload: json.RawMessage(payload),
				Status:  task.StatusPending,
			})
		}
	}

	allTasks := append(synthTasks, pendingTasks...)

	session.mu.Lock()
	session.pendingTasks = allTasks
	session.mapper = mapper
	session.State = StateProcessing
	session.mu.Unlock()

	respXML, err := BuildInformResponse(id)
	if err != nil {
		h.log.WithError(err).Error("CWMP: Build InformResponse failed")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeXML(w, respXML)
}

// handleTransferComplete processes the TransferComplete message, marking the related task done or failed.
func (h *Handler) handleTransferComplete(ctx context.Context, env *Envelope) ([]byte, error) {
	tc := env.Body.TransferComplete
	id := headerID(env)

	h.log.
		WithField("command_key", tc.CommandKey).
		WithField("fault_code", tc.FaultStruct.FaultCode).
		Info("CWMP: Transfer complete received")

	if tc.CommandKey != "" {
		t, err := h.taskQueue.GetByID(ctx, tc.CommandKey)
		if err == nil && t != nil {
			now := time.Now().UTC()
			t.CompletedAt = &now
			if tc.FaultStruct.FaultCode == 0 {
				t.Status = task.StatusDone
			} else {
				t.Status = task.StatusFailed
				t.Error = fmt.Sprintf("fault %d: %s", tc.FaultStruct.FaultCode, tc.FaultStruct.FaultString)
			}
			if uerr := h.taskQueue.UpdateStatus(ctx, t); uerr != nil {
				h.log.WithError(uerr).WithField("task_id", t.ID).Error("CWMP: Failed to update task after TransferComplete")
			}
		}
	}

	return BuildEnvelope(id, Body{TransferComplete: nil})
}

// handleGetRPCMethods returns the list of supported RPC methods.
func (h *Handler) handleGetRPCMethods(_ context.Context, env *Envelope) ([]byte, error) {
	id := headerID(env)
	methods := []string{
		"GetRPCMethods", "SetParameterValues", "GetParameterValues",
		"GetParameterNames", "AddObject", "DeleteObject",
		"Download", "Reboot", "FactoryReset",
	}
	return BuildEnvelope(id, Body{
		GetRPCMethodsResponse: &GetRPCMethodsResponse{
			MethodList: MethodList{Methods: methods},
		},
	})
}

// handleSetParamValuesResponse handles SetParameterValuesResponse.
// For async diagnostic tasks the response only means "diagnostic started"
// we keep them executing until DIAGNOSTICS COMPLETE arrives in a later Inform.
func (h *Handler) handleSetParamValuesResponse(ctx context.Context, w http.ResponseWriter, session *Session) {
	session.mu.Lock()
	t := session.currentTask
	session.mu.Unlock()

	if t != nil && task.IsDiagnosticAsync(t.Type) {
		// Diagnostic is running asynchronously on the CPE.
		// Clear currentTask so the slot is available, but leave the task's
		// Redis status as StatusExecuting.
		session.mu.Lock()
		session.currentTask = nil
		session.mu.Unlock()

		h.log.
			WithField("task_id", t.ID).
			WithField("type", string(t.Type)).
			Info("CWMP: Async diagnostic started on CPE")

		h.dispatchNextOrClose(ctx, w, session)
		return
	}

	h.handleTaskResponse(ctx, w, session, nil, "")
}

// handleGetParamValuesResponse routes the parsed parameter map to the correct
// result handler depending on the current task type.
func (h *Handler) handleGetParamValuesResponse(
	ctx context.Context,
	w http.ResponseWriter,
	session *Session,
	resp *GetParameterValuesResponse,
) {
	params := buildGetParamResult(resp)

	session.mu.Lock()
	t := session.currentTask
	session.currentTask = nil
	mapper := session.mapper
	session.mu.Unlock()

	if t == nil {
		h.dispatchNextOrClose(ctx, w, session)
		return
	}

	switch t.Type {

	case task.TypeGetDiagResult:
		// Synthetic collect task resolve original diagnostic task.
		var p task.GetDiagResultPayload
		if err := json.Unmarshal(t.Payload, &p); err != nil {
			h.log.WithError(err).WithField("task_id", t.ID).Error("CWMP: Unmarshal GetDiagResultPayload")
			break
		}
		origTask, err := h.taskQueue.GetByID(ctx, p.OriginalTaskID)
		if err != nil || origTask == nil {
			h.log.WithError(err).WithField("orig_id", p.OriginalTaskID).Error("CWMP: Original diag task not found")
			break
		}
		result := parseDiagResult(p.OriginalTaskType, params, mapper, origTask)
		now := time.Now().UTC()
		origTask.CompletedAt = &now
		origTask.Status = task.StatusDone
		origTask.Result = result
		if uerr := h.taskQueue.UpdateStatus(ctx, origTask); uerr != nil {
			h.log.WithError(uerr).WithField("task_id", origTask.ID).Error("CWMP: Update diag task status")
		}
		// Persist result into device info where applicable.
		h.persistDiagToDevice(ctx, origTask.Serial, p.OriginalTaskType, result, params, mapper)

	case task.TypeConnectedDevices:
		hosts := parseConnectedHosts(params, mapper)
		now := time.Now().UTC()
		t.CompletedAt = &now
		t.Status = task.StatusDone
		t.Result = hosts
		_ = h.taskQueue.UpdateStatus(ctx, t)
		_ = h.deviceSvc.UpdateInfo(ctx, t.Serial, device.InfoUpdate{ConnectedHosts: hosts})

	case task.TypeCPEStats:
		statsResult, wanPartial := parseCPEStats(params, mapper)
		now := time.Now().UTC()
		t.CompletedAt = &now
		t.Status = task.StatusDone
		t.Result = statsResult
		_ = h.taskQueue.UpdateStatus(ctx, t)
		_ = h.deviceSvc.UpdateInfo(ctx, t.Serial, device.InfoUpdate{WAN: &wanPartial})

	case task.TypePortForwarding:
		rules := parsePortMappingRules(params, mapper)
		now := time.Now().UTC()
		t.CompletedAt = &now
		t.Status = task.StatusDone
		t.Result = rules
		_ = h.taskQueue.UpdateStatus(ctx, t)

	default:
		// Generic GetParams  store raw map.
		h.completeTask(ctx, t, params, "")
	}

	h.dispatchNextOrClose(ctx, w, session)
}

// handleDownloadResponse handles the CPE's immediate reply to a Download RPC.
func (h *Handler) handleDownloadResponse(ctx context.Context, w http.ResponseWriter, session *Session, resp *DownloadResponse) {
	if resp.Status == 0 {
		h.handleTaskResponse(ctx, w, session, nil, "")
		return
	}
	// Status=1: async download; keep currentTask alive for TransferComplete.
	session.mu.Lock()
	var next *task.Task
	if len(session.pendingTasks) > 0 {
		next = session.pendingTasks[0]
		session.pendingTasks = session.pendingTasks[1:]
	}
	session.mu.Unlock()

	if next != nil {
		h.dispatchTask(ctx, w, session, next)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleAddObjectResponse receives the new instance number and sends the
// follow-up SetParameterValues to configure the newly created object.
func (h *Handler) handleAddObjectResponse(ctx context.Context, w http.ResponseWriter, session *Session, resp *AddObjectResponse) {
	session.mu.Lock()
	fn := session.addObjFollowUp
	session.addObjFollowUp = nil
	session.mu.Unlock()

	if fn == nil {
		// No follow-up registered; treat as done.
		h.handleTaskResponse(ctx, w, session, nil, "")
		return
	}

	taskXML, err := fn(resp.InstanceNumber)
	if err != nil {
		h.log.WithError(err).WithField("instance", resp.InstanceNumber).Error("CWMP: build port-mapping SetParameterValues")
		h.handleTaskResponse(ctx, w, session, nil, err.Error())
		return
	}
	// currentTask remains set  SetParameterValuesResponse will close it.
	writeXML(w, taskXML)
}

// handleTaskResponse marks the current in-flight task done or failed and
// dispatches the next pending task.
func (h *Handler) handleTaskResponse(ctx context.Context, w http.ResponseWriter, session *Session, result any, errMsg string) {
	session.mu.Lock()
	t := session.currentTask
	session.currentTask = nil
	session.mu.Unlock()

	if t != nil {
		h.completeTask(ctx, t, result, errMsg)
	}

	h.dispatchNextOrClose(ctx, w, session)
}

// dispatchTask marks t as executing, builds its CWMP XML, and writes it to w.
func (h *Handler) dispatchTask(ctx context.Context, w http.ResponseWriter, session *Session, t *task.Task) {
	// Synthetic tasks (TypeGetDiagResult) are never written to Redis.
	if t.Type != task.TypeGetDiagResult {
		now := time.Now().UTC()
		t.Status = task.StatusExecuting
		t.ExecutedAt = &now
		t.Attempts++
		if err := h.taskQueue.UpdateStatus(ctx, t); err != nil {
			h.log.WithError(err).WithField("task_id", t.ID).Error("CWMP: Mark task executing failed")
			h.dispatchNextOrClose(ctx, w, session)
			return
		}
	}

	session.mu.Lock()
	mapper := session.mapper
	session.mu.Unlock()

	taskXML, buildErr := h.executeTask(ctx, t, mapper, session, w)
	if buildErr != nil {
		now2 := time.Now().UTC()
		if t.Type != task.TypeGetDiagResult {
			t.Status = task.StatusFailed
			t.CompletedAt = &now2
			t.Error = buildErr.Error()
			if uerr := h.taskQueue.UpdateStatus(ctx, t); uerr != nil {
				h.log.WithError(uerr).WithField("task_id", t.ID).Error("CWMP: Update failed task status")
			}
		}
		h.log.WithError(buildErr).WithField("task_id", t.ID).Error("CWMP: execute task failed")
		h.dispatchNextOrClose(ctx, w, session)
		return
	}

	if taskXML == nil {
		// executeTask handled the response itself (e.g. AddObject sets up follow-up).
		return
	}

	session.mu.Lock()
	session.currentTask = t
	session.mu.Unlock()

	h.log.
		WithField("task_id", t.ID).
		WithField("type", string(t.Type)).
		Info("CWMP: dispatching task to CPE")

	writeXML(w, taskXML)
}

// executeTask converts a Task into CWMP XML bytes.
// For port-forwarding add it also configures session.addObjFollowUp and returns nil.
func (h *Handler) executeTask(ctx context.Context, t *task.Task, mapper datamodel.Mapper, session *Session, w http.ResponseWriter) ([]byte, error) {
	exe := task.NewExecutor()

	switch t.Type {

	// SetParameterValues-based tasks
	case task.TypeWifi, task.TypeWAN, task.TypeLAN, task.TypeSetParams,
		task.TypePingTest, task.TypeTraceroute, task.TypeSpeedTest:
		params, err := exe.BuildSetParams(ctx, t, mapper)
		if err != nil {
			return nil, err
		}
		if len(params) == 0 {
			return nil, fmt.Errorf("task %s produced no parameters", t.ID)
		}
		return BuildSetParameterValues(t.ID, params)

	// Legacy diagnostic
	case task.TypeDiagnostic:
		var p task.DiagnosticPayload
		if err := json.Unmarshal(t.Payload, &p); err != nil {
			return nil, fmt.Errorf("unmarshal diagnostic payload: %w", err)
		}
		params, _ := exe.BuildSetParams(ctx, t, mapper)
		if len(params) == 0 {
			params = map[string]string{diagnosticStatePath(p.DiagType, mapper): "Requested"}
		}
		return BuildSetParameterValues(t.ID, params)

	// GetParameterValues-based tasks
	case task.TypeGetParams, task.TypeConnectedDevices, task.TypeCPEStats:
		names, err := exe.BuildGetParams(ctx, t, mapper)
		if err != nil {
			return nil, err
		}
		if len(names) == 0 {
			return nil, fmt.Errorf("task %s has no parameter names", t.ID)
		}
		return BuildGetParameterValues(t.ID, names)

	// Synthetic diagnostic result collection
	case task.TypeGetDiagResult:
		var p task.GetDiagResultPayload
		if err := json.Unmarshal(t.Payload, &p); err != nil {
			return nil, fmt.Errorf("unmarshal GetDiagResultPayload: %w", err)
		}
		if len(p.Paths) == 0 {
			return nil, fmt.Errorf("GetDiagResult task %s has no paths", t.ID)
		}
		return BuildGetParameterValues(t.ID, p.Paths)

	// Port forwarding
	case task.TypePortForwarding:
		var p task.PortForwardingPayload
		if err := json.Unmarshal(t.Payload, &p); err != nil {
			return nil, fmt.Errorf("unmarshal port_forwarding payload: %w", err)
		}
		switch p.Action {
		case task.PortForwardingAdd:
			// Step 1: AddObject to create the new instance.
			// Step 2 (on AddObjectResponse): SetParameterValues to configure it.
			base := mapper.PortMappingBasePath()
			xml, err := BuildAddObject(t.ID, base)
			if err != nil {
				return nil, err
			}
			pCopy := p
			session.mu.Lock()
			session.currentTask = t
			session.addObjFollowUp = func(instanceNum int) ([]byte, error) {
				params := buildPortMappingParams(base, instanceNum, pCopy)
				return BuildSetParameterValues(t.ID, params)
			}
			session.mu.Unlock()
			writeXML(w, xml)
			return nil, nil // signal: already wrote response

		case task.PortForwardingRemove:
			if p.InstanceNumber <= 0 {
				return nil, fmt.Errorf("port_forwarding remove requires instance_number")
			}
			objPath := fmt.Sprintf("%s%d.", mapper.PortMappingBasePath(), p.InstanceNumber)
			return BuildDeleteObject(t.ID, objPath)

		case task.PortForwardingList:
			names, _ := exe.BuildGetParams(ctx, t, mapper)
			if len(names) == 0 {
				return nil, fmt.Errorf("port_forwarding list produced no paths")
			}
			return BuildGetParameterValues(t.ID, names)

		default:
			return nil, fmt.Errorf("unknown port_forwarding action: %s", p.Action)
		}

	// Simple one-shot tasks
	case task.TypeReboot:
		return BuildReboot(t.ID, t.ID)

	case task.TypeFactoryReset:
		return BuildFactoryReset(t.ID)

	case task.TypeFirmware:
		var p task.FirmwarePayload
		if err := json.Unmarshal(t.Payload, &p); err != nil {
			return nil, err
		}
		ft := p.FileType
		if ft == "" {
			ft = "1 Firmware Upgrade Image"
		}
		return BuildDownload(t.ID, &Download{
			CommandKey: t.ID,
			FileType:   ft,
			URL:        p.URL,
			Username:   p.Username,
			Password:   p.Password,
		})

	default:
		return nil, fmt.Errorf("unknown task type: %s", t.Type)
	}
}

// dispatchNextOrClose pops the next pending task and dispatches it, or closes
// the session with HTTP 204 if the queue is empty.
func (h *Handler) dispatchNextOrClose(ctx context.Context, w http.ResponseWriter, session *Session) {
	session.mu.Lock()
	var next *task.Task
	if len(session.pendingTasks) > 0 {
		next = session.pendingTasks[0]
		session.pendingTasks = session.pendingTasks[1:]
	}
	session.mu.Unlock()

	if next != nil {
		h.dispatchTask(ctx, w, session, next)
		return
	}
	session.setState(StateDone)
	w.WriteHeader(http.StatusNoContent)
}

// completeTask marks a task done or failed and persists the update to Redis.
func (h *Handler) completeTask(ctx context.Context, t *task.Task, result any, errMsg string) {
	now := time.Now().UTC()
	t.CompletedAt = &now
	if errMsg != "" {
		t.Status = task.StatusFailed
		t.Error = errMsg
	} else {
		t.Status = task.StatusDone
		if result != nil {
			t.Result = result
		}
	}
	if uerr := h.taskQueue.UpdateStatus(ctx, t); uerr != nil {
		h.log.WithError(uerr).WithField("task_id", t.ID).Error("CWMP: Update completed task status")
		return
	}

	h.log.
		WithField("task_id", t.ID).
		WithField("status", string(t.Status)).
		Info("CWMP: task completed")
}

// parseDiagResult dispatches to the correct result parser based on task type.
func parseDiagResult(taskType task.Type, params map[string]string, mapper datamodel.Mapper, origTask *task.Task) any {
	switch taskType {
	case task.TypePingTest, task.TypeDiagnostic:
		return parsePingResult(params, mapper)
	case task.TypeTraceroute:
		return parseTracerouteResult(params, mapper)
	case task.TypeSpeedTest:
		var p task.SpeedTestPayload
		_ = json.Unmarshal(origTask.Payload, &p)
		return parseSpeedTestResult(params, mapper, p)
	}
	return params
}

// persistDiagToDevice stores diagnostic results back into the device document
// where applicable (e.g. connected hosts → device.connected_hosts).
func (h *Handler) persistDiagToDevice(
	ctx context.Context,
	serial string,
	taskType task.Type,
	result any,
	params map[string]string,
	mapper datamodel.Mapper,
) {
	switch taskType {
	case task.TypeConnectedDevices:
		if hosts, ok := result.([]device.ConnectedHost); ok {
			_ = h.deviceSvc.UpdateInfo(ctx, serial, device.InfoUpdate{ConnectedHosts: hosts})
		}
	case task.TypeCPEStats:
		_, wan := parseCPEStats(params, mapper)
		_ = h.deviceSvc.UpdateInfo(ctx, serial, device.InfoUpdate{WAN: &wan})
	}
}

func (h *Handler) extractInformParams(env *Envelope) *device.UpsertRequest {
	inf := env.Body.Inform
	req := &device.UpsertRequest{
		Serial:       inf.DeviceId.SerialNumber,
		OUI:          inf.DeviceId.OUI,
		Manufacturer: inf.DeviceId.Manufacturer,
		ProductClass: inf.DeviceId.ProductClass,
		Parameters:   make(map[string]string, len(inf.ParameterList.ParameterValueStructs)),
	}

	for _, pv := range inf.ParameterList.ParameterValueStructs {
		name := pv.Name
		val := pv.Value.Data
		req.Parameters[name] = val

		lower := strings.ToLower(name)
		switch {
		case strings.HasSuffix(lower, "modelname"):
			req.ModelName = val
		case strings.HasSuffix(lower, "softwareversion"):
			req.SWVersion = val
		case strings.HasSuffix(lower, "hardwareversion"):
			req.HWVersion = val
		case strings.HasSuffix(lower, "bootloaderversion"):
			req.BLVersion = val
		case strings.HasSuffix(lower, "externalipaddress") || strings.HasSuffix(lower, "wanipaddress"):
			req.WANIP = val
		case strings.HasSuffix(lower, "ipaddress") && req.IPAddress == "":
			req.IPAddress = val
		case strings.HasSuffix(lower, "uptime"):
			if v, err := parseInt64(val); err == nil {
				req.UptimeSeconds = v
			}
		case strings.HasSuffix(lower, "memorystatus.total"):
			if v, err := parseInt64(val); err == nil {
				req.RAMTotal = v
			}
		case strings.HasSuffix(lower, "memorystatus.free"):
			if v, err := parseInt64(val); err == nil {
				req.RAMFree = v
			}
		case strings.HasSuffix(lower, "managementserver.url"):
			req.ACSURL = val
		}
	}

	for name := range req.Parameters {
		if strings.HasPrefix(name, "Device.") {
			req.DataModel = "tr181"
			break
		} else if strings.HasPrefix(name, "InternetGatewayDevice.") {
			req.DataModel = "tr098"
			break
		}
	}

	return req
}

// hasDiagnosticsCompleteEvent returns true when the Inform carries the
// "8 DIAGNOSTICS COMPLETE" event code.
func hasDiagnosticsCompleteEvent(inf *InformRequest) bool {
	if inf == nil {
		return false
	}
	for _, ev := range inf.Event.Events {
		if strings.Contains(ev.EventCode, "DIAGNOSTICS COMPLETE") ||
			ev.EventCode == "8 DIAGNOSTICS COMPLETE" {
			return true
		}
	}
	return false
}

func (h *Handler) getSessionID(r *http.Request) string {
	if c, err := r.Cookie("cwmp-session"); err == nil && c.Value != "" {
		return c.Value
	}
	if body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)); err == nil && len(body) > 0 {
		r.Body = io.NopCloser(bytes.NewReader(body))
		var env Envelope
		if xmlErr := xml.NewDecoder(bytes.NewReader(body)).Decode(&env); xmlErr == nil {
			if id := headerID(&env); id != "" {
				return id
			}
		}
	}
	return uuid.NewString()
}

func (h *Handler) writeSoapFault(w http.ResponseWriter, id, faultCode, cwmpCode, cwmpMsg string) {
	body := Body{
		Fault: &SOAPFault{
			FaultCode:   faultCode,
			FaultString: cwmpMsg,
			Detail: FaultDetail{
				CWMPFault: CWMPFault{
					FaultCode:   cwmpCode,
					FaultString: cwmpMsg,
				},
			},
		},
	}
	respXML, err := BuildEnvelope(id, body)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeXML(w, respXML)
}

func writeXML(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func headerID(env *Envelope) string {
	if env != nil && env.Header.ID != nil {
		return env.Header.ID.Value
	}
	return uuid.NewString()
}

func firstRootObject(params map[string]string) string {
	for k := range params {
		if strings.HasPrefix(k, "Device.") {
			return "Device."
		}
		if strings.HasPrefix(k, "InternetGatewayDevice.") {
			return "InternetGatewayDevice."
		}
	}
	return ""
}

func diagnosticStatePath(diagType string, mapper datamodel.Mapper) string {
	lower := strings.ToLower(diagType)
	if lower == "traceroute" {
		return mapper.TracerouteDiagBasePath() + "DiagnosticsState"
	}
	return mapper.PingDiagBasePath() + "DiagnosticsState"
}

func unmarshalPayload(t *task.Task, dst any) error {
	if err := json.Unmarshal(t.Payload, dst); err != nil {
		return fmt.Errorf("unmarshal payload for task %s: %w", t.ID, err)
	}
	return nil
}

// buildGetParamResult converts a GetParameterValuesResponse into a flat
// map[string]string suitable for storage in Task.Result.
func buildGetParamResult(resp *GetParameterValuesResponse) map[string]string {
	out := make(map[string]string, len(resp.ParameterList.ParameterValueStructs))
	for _, pv := range resp.ParameterList.ParameterValueStructs {
		out[pv.Name] = pv.Value.Data
	}
	return out
}

func parseInt64(s string) (int64, error) {
	var v int64
	_, err := fmt.Sscanf(s, "%d", &v)
	return v, err
}
