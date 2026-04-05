package api_test

// import (
// 	"bytes"
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	"net/http/httptest"
// 	"testing"
// 	"time"

// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"

// 	"github.com/raykavin/helix-acs/internal/api"
// 	"github.com/raykavin/helix-acs/internal/auth"
// 	"github.com/raykavin/helix-acs/internal/config"
// 	"github.com/raykavin/helix-acs/internal/device"
// 	"github.com/raykavin/helix-acs/internal/task"
// )

// type mockDeviceService struct {
// 	devices map[string]*device.Device
// }

// func newMockDeviceService() *mockDeviceService {
// 	svc := &mockDeviceService{
// 		devices: make(map[string]*device.Device),
// 	}
// 	// Seed one known device for task creation tests
// 	svc.devices["SN-TEST"] = &device.Device{
// 		Serial:       "SN-TEST",
// 		Manufacturer: "Intelbras",
// 		ModelName:    "W5-1200F",
// 		Online:       true,
// 		LastInform:   time.Now(),
// 	}
// 	return svc
// }

// func (m *mockDeviceService) UpsertFromInform(_ context.Context, req *device.UpsertRequest) (*device.Device, error) {
// 	dev := &device.Device{
// 		Serial:       req.Serial,
// 		Manufacturer: req.Manufacturer,
// 		ModelName:    req.ModelName,
// 	}
// 	m.devices[req.Serial] = dev
// 	return dev, nil
// }

// func (m *mockDeviceService) FindBySerial(_ context.Context, serial string) (*device.Device, error) {
// 	dev, ok := m.devices[serial]
// 	if !ok {
// 		return nil, nil
// 	}
// 	return dev, nil
// }

// func (m *mockDeviceService) List(_ context.Context, _ device.DeviceFilter, _, _ int) ([]*device.Device, int64, error) {
// 	out := make([]*device.Device, 0, len(m.devices))
// 	for _, d := range m.devices {
// 		out = append(out, d)
// 	}
// 	return out, int64(len(out)), nil
// }

// func (m *mockDeviceService) UpdateTags(_ context.Context, serial string, tags []string) (*device.Device, error) {
// 	dev, ok := m.devices[serial]
// 	if !ok {
// 		return nil, nil
// 	}
// 	dev.Tags = tags
// 	return dev, nil
// }

// func (m *mockDeviceService) Delete(_ context.Context, serial string) error {
// 	delete(m.devices, serial)
// 	return nil
// }

// func (m *mockDeviceService) SetOnline(_ context.Context, serial string, online bool) error {
// 	if dev, ok := m.devices[serial]; ok {
// 		dev.Online = online
// 	}
// 	return nil
// }

// func (m *mockDeviceService) UpdateInfo(_ context.Context, _ string, _ device.InfoUpdate) error {
// 	return nil
// }

// // Mock task queue
// type mockTaskQueue struct {
// 	tasks map[string]*task.Task
// }

// func newMockTaskQueue() *mockTaskQueue {
// 	return &mockTaskQueue{tasks: make(map[string]*task.Task)}
// }

// func (m *mockTaskQueue) Enqueue(_ context.Context, t *task.Task) error {
// 	m.tasks[t.ID] = t
// 	return nil
// }

// func (m *mockTaskQueue) DequeuePending(_ context.Context, serial string) ([]*task.Task, error) {
// 	var out []*task.Task
// 	for _, t := range m.tasks {
// 		if t.Serial == serial && t.Status == task.StatusPending {
// 			out = append(out, t)
// 		}
// 	}
// 	return out, nil
// }

// func (m *mockTaskQueue) UpdateStatus(_ context.Context, t *task.Task) error {
// 	if existing, ok := m.tasks[t.ID]; ok {
// 		existing.Status = t.Status
// 	}
// 	return nil
// }

// func (m *mockTaskQueue) Cancel(_ context.Context, taskID, _ string) error {
// 	t, ok := m.tasks[taskID]
// 	if !ok {
// 		return fmt.Errorf("task not found: %s", taskID)
// 	}
// 	t.Status = task.StatusCancelled
// 	return nil
// }

// func (m *mockTaskQueue) GetByID(_ context.Context, taskID string) (*task.Task, error) {
// 	t, ok := m.tasks[taskID]
// 	if !ok {
// 		return nil, fmt.Errorf("task not found: %s", taskID)
// 	}
// 	return t, nil
// }

// func (m *mockTaskQueue) List(_ context.Context, serial string, _, _ int) ([]*task.Task, int64, error) {
// 	var out []*task.Task
// 	for _, t := range m.tasks {
// 		if t.Serial == serial {
// 			out = append(out, t)
// 		}
// 	}
// 	return out, int64(len(out)), nil
// }

// func (m *mockTaskQueue) FindExecutingDiagnostics(_ context.Context, serial string) ([]*task.Task, error) {
// 	var out []*task.Task
// 	for _, t := range m.tasks {
// 		if t.Serial == serial && t.Status == task.StatusExecuting && task.IsDiagnosticAsync(t.Type) {
// 			out = append(out, t)
// 		}
// 	}
// 	return out, nil
// }

// // Test server setup
// func newTestServer(t *testing.T) (*httptest.Server, *auth.JWTService) {
// 	t.Helper()

// 	jwtSvc := auth.NewJWTService("test-secret-key", 24*time.Hour, 168*time.Hour)

// 	cfg := &config.Config{
// 		ACS: config.ACSConfig{
// 			Username: "admin",
// 			Password: "admin123",
// 		},
// 		Task: config.TaskConfig{
// 			MaxAttempts: 3,
// 		},
// 		CORS: config.CORSConfig{
// 			AllowedOrigins: []string{"*"},
// 			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
// 		},
// 	}

// 	deviceSvc := newMockDeviceService()
// 	taskQueue := newMockTaskQueue()

// 	router := api.NewRouter(cfg, deviceSvc, taskQueue, jwtSvc, nil, nil)
// 	srv := httptest.NewServer(router)
// 	t.Cleanup(srv.Close)

// 	return srv, jwtSvc
// }

// // getAuthToken calls POST /api/v1/auth/login and returns the access token.
// func getAuthToken(t *testing.T, srv *httptest.Server, username, password string) string {
// 	t.Helper()

// 	body, _ := json.Marshal(map[string]string{
// 		"username": username,
// 		"password": password,
// 	})

// 	resp, err := srv.Client().Post(
// 		srv.URL+"/api/v1/auth/login",
// 		"application/json",
// 		bytes.NewReader(body),
// 	)
// 	require.NoError(t, err)
// 	defer resp.Body.Close()

// 	require.Equal(t, http.StatusOK, resp.StatusCode)

// 	var result map[string]any
// 	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

// 	token, ok := result["token"].(string)
// 	require.True(t, ok, "response should contain a string token field")
// 	require.NotEmpty(t, token)
// 	return token
// }

// // TestLoginSuccess
// func TestLoginSuccess(t *testing.T) {
// 	srv, _ := newTestServer(t)

// 	body, _ := json.Marshal(map[string]string{
// 		"username": "admin",
// 		"password": "admin123",
// 	})

// 	resp, err := srv.Client().Post(
// 		srv.URL+"/api/v1/auth/login",
// 		"application/json",
// 		bytes.NewReader(body),
// 	)
// 	require.NoError(t, err)
// 	defer resp.Body.Close()

// 	assert.Equal(t, http.StatusOK, resp.StatusCode)

// 	var result map[string]any
// 	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

// 	assert.NotEmpty(t, result["token"], "should have an access token")
// 	assert.NotEmpty(t, result["refresh_token"], "should have a refresh token")
// 	assert.NotZero(t, result["expires_in"], "should have expires_in")
// }

// // TestLoginFailure
// func TestLoginFailure(t *testing.T) {
// 	srv, _ := newTestServer(t)

// 	body, _ := json.Marshal(map[string]string{
// 		"username": "admin",
// 		"password": "wrong-password",
// 	})

// 	resp, err := srv.Client().Post(
// 		srv.URL+"/api/v1/auth/login",
// 		"application/json",
// 		bytes.NewReader(body),
// 	)
// 	require.NoError(t, err)
// 	defer resp.Body.Close()

// 	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

// 	var result map[string]any
// 	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
// 	assert.NotEmpty(t, result["error"])
// }

// // TestListDevicesUnauthorized
// func TestListDevicesUnauthorized(t *testing.T) {
// 	srv, _ := newTestServer(t)

// 	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/devices", nil)
// 	resp, err := srv.Client().Do(req)
// 	require.NoError(t, err)
// 	defer resp.Body.Close()

// 	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
// }

// // TestListDevicesAuthorized
// func TestListDevicesAuthorized(t *testing.T) {
// 	srv, _ := newTestServer(t)
// 	token := getAuthToken(t, srv, "admin", "admin123")

// 	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/devices", nil)
// 	req.Header.Set("Authorization", "Bearer "+token)

// 	resp, err := srv.Client().Do(req)
// 	require.NoError(t, err)
// 	defer resp.Body.Close()

// 	assert.Equal(t, http.StatusOK, resp.StatusCode)

// 	var result map[string]any
// 	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

// 	_, hasData := result["data"]
// 	assert.True(t, hasData, "response should have a data field")
// }

// // TestCreateTaskReboot
// func TestCreateTaskReboot(t *testing.T) {
// 	srv, _ := newTestServer(t)
// 	token := getAuthToken(t, srv, "admin", "admin123")

// 	// Device "SN-TEST" is pre-seeded in the mock
// 	req, _ := http.NewRequest(
// 		http.MethodPost,
// 		srv.URL+"/api/v1/devices/SN-TEST/tasks/reboot",
// 		bytes.NewReader([]byte(`{}`)),
// 	)
// 	req.Header.Set("Authorization", "Bearer "+token)
// 	req.Header.Set("Content-Type", "application/json")

// 	resp, err := srv.Client().Do(req)
// 	require.NoError(t, err)
// 	defer resp.Body.Close()

// 	assert.Equal(t, http.StatusCreated, resp.StatusCode)

// 	var result map[string]any
// 	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

// 	taskID, ok := result["id"].(string)
// 	assert.True(t, ok, "response should contain string id field")
// 	assert.NotEmpty(t, taskID)
// 	assert.Equal(t, "reboot", result["type"])
// 	assert.Equal(t, "pending", result["status"])
// }

// // TestCreateTaskRebootDeviceNotFound
// func TestCreateTaskRebootDeviceNotFound(t *testing.T) {
// 	srv, _ := newTestServer(t)
// 	token := getAuthToken(t, srv, "admin", "admin123")

// 	req, _ := http.NewRequest(
// 		http.MethodPost,
// 		srv.URL+"/api/v1/devices/NONEXISTENT/tasks/reboot",
// 		bytes.NewReader([]byte(`{}`)),
// 	)
// 	req.Header.Set("Authorization", "Bearer "+token)
// 	req.Header.Set("Content-Type", "application/json")

// 	resp, err := srv.Client().Do(req)
// 	require.NoError(t, err)
// 	defer resp.Body.Close()

// 	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
// }

// // TestGetTaskNotFound
// func TestGetTaskNotFound(t *testing.T) {
// 	srv, _ := newTestServer(t)
// 	token := getAuthToken(t, srv, "admin", "admin123")

// 	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/tasks/nonexistent-task-id", nil)
// 	req.Header.Set("Authorization", "Bearer "+token)

// 	resp, err := srv.Client().Do(req)
// 	require.NoError(t, err)
// 	defer resp.Body.Close()

// 	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

// 	var result map[string]any
// 	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
// 	assert.NotEmpty(t, result["error"])
// }

// // TestGetTaskFound
// func TestGetTaskFound(t *testing.T) {
// 	srv, _ := newTestServer(t)
// 	token := getAuthToken(t, srv, "admin", "admin123")

// 	// Create a task first
// 	req, _ := http.NewRequest(
// 		http.MethodPost,
// 		srv.URL+"/api/v1/devices/SN-TEST/tasks/reboot",
// 		bytes.NewReader([]byte(`{}`)),
// 	)
// 	req.Header.Set("Authorization", "Bearer "+token)
// 	req.Header.Set("Content-Type", "application/json")

// 	resp, err := srv.Client().Do(req)
// 	require.NoError(t, err)
// 	defer resp.Body.Close()
// 	require.Equal(t, http.StatusCreated, resp.StatusCode)

// 	var created map[string]any
// 	require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))
// 	taskID := created["id"].(string)

// 	// Now fetch it
// 	req2, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/tasks/"+taskID, nil)
// 	req2.Header.Set("Authorization", "Bearer "+token)

// 	resp2, err := srv.Client().Do(req2)
// 	require.NoError(t, err)
// 	defer resp2.Body.Close()

// 	assert.Equal(t, http.StatusOK, resp2.StatusCode)

// 	var result map[string]any
// 	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&result))
// 	assert.Equal(t, taskID, result["id"])
// }

// // TestHealthEndpoint
// func TestHealthEndpoint(t *testing.T) {
// 	srv, _ := newTestServer(t)

// 	// Health endpoint requires no authentication
// 	resp, err := srv.Client().Get(srv.URL + "/health")
// 	require.NoError(t, err)
// 	defer resp.Body.Close()

// 	// Will return 500 because mongo/redis are nil, but the route exists and
// 	// the server is up (not 404 / 401).
// 	assert.NotEqual(t, http.StatusNotFound, resp.StatusCode)
// 	assert.NotEqual(t, http.StatusUnauthorized, resp.StatusCode)
// }
