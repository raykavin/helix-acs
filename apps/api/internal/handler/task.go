package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"github.com/raykavin/helix-acs/packages/device"
	"github.com/raykavin/helix-acs/packages/task"
)

// TaskHandler handles all task-related REST endpoints.
type TaskHandler struct {
	taskQueue   task.Queue
	deviceSvc   device.Service
	maxAttempts int
}

// NewTaskHandler creates a TaskHandler.
func NewTaskHandler(taskQueue task.Queue, deviceSvc device.Service, maxAttempts int) *TaskHandler {
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	return &TaskHandler{
		taskQueue:   taskQueue,
		deviceSvc:   deviceSvc,
		maxAttempts: maxAttempts,
	}
}

// taskListResponse is the paginated response for task listings.
type taskListResponse struct {
	Data  any   `json:"data"`
	Total int64 `json:"total"`
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
}

// requireDevice looks up a device by serial and writes a 404 if absent.
// Returns nil when the device was not found (response already written).
func (h *TaskHandler) requireDevice(w http.ResponseWriter, r *http.Request, serial string) *device.Device {
	dev, err := h.deviceSvc.FindBySerial(r.Context(), serial)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch device")
		return nil
	}
	if dev == nil {
		writeError(w, http.StatusNotFound, "device not found")
		return nil
	}
	return dev
}

// enqueueTask marshals payload, builds a Task, enqueues it, and writes 201.
func (h *TaskHandler) enqueueTask(w http.ResponseWriter, r *http.Request, serial string, taskType task.Type, payload any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to marshal payload")
		return
	}

	t := &task.Task{
		ID:          uuid.NewString(),
		Serial:      serial,
		Type:        taskType,
		Payload:     json.RawMessage(raw),
		Status:      task.StatusPending,
		CreatedAt:   time.Now().UTC(),
		MaxAttempts: h.maxAttempts,
	}

	if err := h.taskQueue.Enqueue(r.Context(), t); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to enqueue task")
		return
	}

	writeJSON(w, http.StatusCreated, t)
}

// Task creation endpoints

// CreateWifi handles POST /api/v1/devices/:serial/tasks/wifi
func (h *TaskHandler) CreateWifi(w http.ResponseWriter, r *http.Request) {
	serial := mux.Vars(r)["serial"]
	if h.requireDevice(w, r, serial) == nil {
		return
	}

	var payload task.WiFiPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	h.enqueueTask(w, r, serial, task.TypeWifi, payload)
}

// CreateWAN handles POST /api/v1/devices/:serial/tasks/wan
func (h *TaskHandler) CreateWAN(w http.ResponseWriter, r *http.Request) {
	serial := mux.Vars(r)["serial"]
	if h.requireDevice(w, r, serial) == nil {
		return
	}

	var payload task.WANPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	h.enqueueTask(w, r, serial, task.TypeWAN, payload)
}

// CreateLAN handles POST /api/v1/devices/:serial/tasks/lan
func (h *TaskHandler) CreateLAN(w http.ResponseWriter, r *http.Request) {
	serial := mux.Vars(r)["serial"]
	if h.requireDevice(w, r, serial) == nil {
		return
	}

	var payload task.LANPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	h.enqueueTask(w, r, serial, task.TypeLAN, payload)
}

// CreateReboot handles POST /api/v1/devices/:serial/tasks/reboot
func (h *TaskHandler) CreateReboot(w http.ResponseWriter, r *http.Request) {
	serial := mux.Vars(r)["serial"]
	if h.requireDevice(w, r, serial) == nil {
		return
	}

	// Reboot carries no user-supplied payload.
	h.enqueueTask(w, r, serial, task.TypeReboot, struct{}{})
}

// CreateFactoryReset handles POST /api/v1/devices/:serial/tasks/factory-reset
func (h *TaskHandler) CreateFactoryReset(w http.ResponseWriter, r *http.Request) {
	serial := mux.Vars(r)["serial"]
	if h.requireDevice(w, r, serial) == nil {
		return
	}

	h.enqueueTask(w, r, serial, task.TypeFactoryReset, struct{}{})
}

// CreateSetParams handles POST /api/v1/devices/:serial/tasks/parameters
func (h *TaskHandler) CreateSetParams(w http.ResponseWriter, r *http.Request) {
	serial := mux.Vars(r)["serial"]
	if h.requireDevice(w, r, serial) == nil {
		return
	}

	var payload task.SetParamsPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(payload.Parameters) == 0 {
		writeError(w, http.StatusBadRequest, "parameters map must not be empty")
		return
	}

	h.enqueueTask(w, r, serial, task.TypeSetParams, payload)
}

// CreateFirmware handles POST /api/v1/devices/:serial/tasks/firmware
func (h *TaskHandler) CreateFirmware(w http.ResponseWriter, r *http.Request) {
	serial := mux.Vars(r)["serial"]
	if h.requireDevice(w, r, serial) == nil {
		return
	}

	var payload task.FirmwarePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if payload.URL == "" {
		writeError(w, http.StatusBadRequest, "firmware url is required")
		return
	}

	h.enqueueTask(w, r, serial, task.TypeFirmware, payload)
}

// CreateDiagnostic handles POST /api/v1/devices/:serial/tasks/diagnostic
func (h *TaskHandler) CreateDiagnostic(w http.ResponseWriter, r *http.Request) {
	serial := mux.Vars(r)["serial"]
	if h.requireDevice(w, r, serial) == nil {
		return
	}

	var payload task.DiagnosticPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if payload.Target == "" {
		writeError(w, http.StatusBadRequest, "diagnostic target is required")
		return
	}

	h.enqueueTask(w, r, serial, task.TypeDiagnostic, payload)
}

// CreatePingTest handles POST /api/v1/devices/:serial/tasks/ping
func (h *TaskHandler) CreatePingTest(w http.ResponseWriter, r *http.Request) {
	serial := mux.Vars(r)["serial"]
	if h.requireDevice(w, r, serial) == nil {
		return
	}

	var payload task.PingTestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if payload.Host == "" {
		writeError(w, http.StatusBadRequest, "host is required")
		return
	}

	h.enqueueTask(w, r, serial, task.TypePingTest, payload)
}

// CreateTraceroute handles POST /api/v1/devices/:serial/tasks/traceroute
func (h *TaskHandler) CreateTraceroute(w http.ResponseWriter, r *http.Request) {
	serial := mux.Vars(r)["serial"]
	if h.requireDevice(w, r, serial) == nil {
		return
	}

	var payload task.TraceroutePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if payload.Host == "" {
		writeError(w, http.StatusBadRequest, "host is required")
		return
	}

	h.enqueueTask(w, r, serial, task.TypeTraceroute, payload)
}

// CreateSpeedTest handles POST /api/v1/devices/:serial/tasks/speed-test
func (h *TaskHandler) CreateSpeedTest(w http.ResponseWriter, r *http.Request) {
	serial := mux.Vars(r)["serial"]
	if h.requireDevice(w, r, serial) == nil {
		return
	}

	var payload task.SpeedTestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if payload.DownloadURL == "" {
		writeError(w, http.StatusBadRequest, "download_url is required")
		return
	}

	h.enqueueTask(w, r, serial, task.TypeSpeedTest, payload)
}

// CreateConnectedDevices handles POST /api/v1/devices/:serial/tasks/connected-devices
func (h *TaskHandler) CreateConnectedDevices(w http.ResponseWriter, r *http.Request) {
	serial := mux.Vars(r)["serial"]
	if h.requireDevice(w, r, serial) == nil {
		return
	}

	h.enqueueTask(w, r, serial, task.TypeConnectedDevices, task.ConnectedDevicesPayload{})
}

// CreateCPEStats handles POST /api/v1/devices/:serial/tasks/cpe-stats
func (h *TaskHandler) CreateCPEStats(w http.ResponseWriter, r *http.Request) {
	serial := mux.Vars(r)["serial"]
	if h.requireDevice(w, r, serial) == nil {
		return
	}

	h.enqueueTask(w, r, serial, task.TypeCPEStats, task.CPEStatsPayload{})
}

// CreatePortForwarding handles POST /api/v1/devices/:serial/tasks/port-forwarding
func (h *TaskHandler) CreatePortForwarding(w http.ResponseWriter, r *http.Request) {
	serial := mux.Vars(r)["serial"]
	if h.requireDevice(w, r, serial) == nil {
		return
	}

	var payload task.PortForwardingPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	switch payload.Action {
	case task.PortForwardingAdd:
		if payload.ExternalPort <= 0 || payload.InternalIP == "" || payload.InternalPort <= 0 {
			writeError(w, http.StatusBadRequest, "add requires external_port, internal_ip and internal_port")
			return
		}
	case task.PortForwardingRemove:
		if payload.InstanceNumber <= 0 {
			writeError(w, http.StatusBadRequest, "remove requires instance_number")
			return
		}
	case task.PortForwardingList:
		// no extra validation needed
	default:
		writeError(w, http.StatusBadRequest, "action must be add, remove or list")
		return
	}

	h.enqueueTask(w, r, serial, task.TypePortForwarding, payload)
}

// Task query endpoints

// ListByDevice handles GET /api/v1/devices/:serial/tasks
func (h *TaskHandler) ListByDevice(w http.ResponseWriter, r *http.Request) {
	serial := mux.Vars(r)["serial"]
	if h.requireDevice(w, r, serial) == nil {
		return
	}

	page, limit := paginationParams(r)

	tasks, total, err := h.taskQueue.List(r.Context(), serial, page, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}

	if tasks == nil {
		tasks = []*task.Task{}
	}

	writeJSON(w, http.StatusOK, taskListResponse{
		Data:  tasks,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// Get handles GET /api/v1/tasks/:task_id
func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	taskID := mux.Vars(r)["task_id"]
	if taskID == "" {
		writeError(w, http.StatusBadRequest, "task_id is required")
		return
	}

	t, err := h.taskQueue.GetByID(r.Context(), taskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	writeJSON(w, http.StatusOK, t)
}

// Cancel handles DELETE /api/v1/tasks/:task_id
func (h *TaskHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	taskID := mux.Vars(r)["task_id"]
	if taskID == "" {
		writeError(w, http.StatusBadRequest, "task_id is required")
		return
	}

	// Fetch the task first to obtain its serial (required by Cancel).
	t, err := h.taskQueue.GetByID(r.Context(), taskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	if err := h.taskQueue.Cancel(r.Context(), taskID, t.Serial); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}
