package task

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raykavin/helix-acs/packages/datamodel"
)

// Helpers

func makeTask(taskType Type, payload any) *Task {
	raw, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return &Task{
		ID:      "test-task-id",
		Serial:  "SN-TEST",
		Type:    taskType,
		Payload: json.RawMessage(raw),
		Status:  StatusPending,
	}
}

// BuildSetParams: WiFi 2.4 GHz

func TestBuildSetParamsWifi24(t *testing.T) {
	enabled := true
	payload := WiFiPayload{
		Band:     "2.4",
		SSID:     "HomeNet",
		Password: "p@ssw0rd",
		Channel:  6,
		Enabled:  &enabled,
	}

	task := makeTask(TypeWifi, payload)
	executor := NewExecutor()
	mapper := datamodel.NewMapper(datamodel.TR181)

	params, err := executor.BuildSetParams(context.Background(), task, mapper)
	require.NoError(t, err)
	require.NotNil(t, params)

	// Band 0 → index 1 in TR181
	assert.Equal(t, "HomeNet", params["Device.WiFi.SSID.1.SSID"])
	assert.Equal(t, "p@ssw0rd", params["Device.WiFi.AccessPoint.1.Security.KeyPassphrase"])
	assert.Equal(t, "true", params["Device.WiFi.SSID.1.Enable"])
	assert.Equal(t, "6", params["Device.WiFi.Radio.1.Channel"])
}

// BuildSetParams: WiFi 5 GHz

func TestBuildSetParamsWifi5(t *testing.T) {
	enabled := false
	payload := WiFiPayload{
		Band:     "5",
		SSID:     "HomeNet5G",
		Password: "5gpass",
		Channel:  36,
		Enabled:  &enabled,
	}

	task := makeTask(TypeWifi, payload)
	executor := NewExecutor()
	mapper := datamodel.NewMapper(datamodel.TR181)

	params, err := executor.BuildSetParams(context.Background(), task, mapper)
	require.NoError(t, err)
	require.NotNil(t, params)

	// Band 1 → index 2 in TR181
	assert.Equal(t, "HomeNet5G", params["Device.WiFi.SSID.2.SSID"])
	assert.Equal(t, "5gpass", params["Device.WiFi.AccessPoint.2.Security.KeyPassphrase"])
	assert.Equal(t, "false", params["Device.WiFi.SSID.2.Enable"])
	assert.Equal(t, "36", params["Device.WiFi.Radio.2.Channel"])
}

// BuildSetParams: WiFi TR098 mapper

func TestBuildSetParamsWifi24TR098(t *testing.T) {
	payload := WiFiPayload{
		Band:     "2.4",
		SSID:     "LegacyNet",
		Password: "legacypass",
	}

	task := makeTask(TypeWifi, payload)
	executor := NewExecutor()
	mapper := datamodel.NewMapper(datamodel.TR098)

	params, err := executor.BuildSetParams(context.Background(), task, mapper)
	require.NoError(t, err)
	require.NotNil(t, params)

	assert.Equal(t, "LegacyNet", params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"])
	assert.Equal(t, "legacypass", params["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase"])
}

// BuildSetParams: WAN PPPoE

func TestBuildSetParamsWAN(t *testing.T) {
	payload := WANPayload{
		ConnectionType: "pppoe",
		Username:       "ppp_user",
		Password:       "ppp_pass",
	}

	task := makeTask(TypeWAN, payload)
	executor := NewExecutor()
	mapper := datamodel.NewMapper(datamodel.TR181)

	params, err := executor.BuildSetParams(context.Background(), task, mapper)
	require.NoError(t, err)
	require.NotNil(t, params)

	assert.Equal(t, "pppoe", params["Device.IP.Interface.1.IPv4Address.1.AddressingType"])
	assert.Equal(t, "ppp_user", params["Device.PPP.Interface.1.Username"])
	assert.Equal(t, "ppp_pass", params["Device.PPP.Interface.1.Password"])
}

// BuildSetParams: WAN with IP address

func TestBuildSetParamsWANWithIP(t *testing.T) {
	payload := WANPayload{
		ConnectionType: "static",
		IPAddress:      "203.0.113.5",
	}

	task := makeTask(TypeWAN, payload)
	executor := NewExecutor()
	mapper := datamodel.NewMapper(datamodel.TR181)

	params, err := executor.BuildSetParams(context.Background(), task, mapper)
	require.NoError(t, err)
	require.NotNil(t, params)

	assert.Equal(t, "static", params["Device.IP.Interface.1.IPv4Address.1.AddressingType"])
	assert.Equal(t, "203.0.113.5", params["Device.IP.Interface.1.IPv4Address.1.IPAddress"])
}

// BuildSetParams: empty WAN returns error

func TestBuildSetParamsWANEmpty(t *testing.T) {
	payload := WANPayload{}

	task := makeTask(TypeWAN, payload)
	executor := NewExecutor()
	mapper := datamodel.NewMapper(datamodel.TR181)

	_, err := executor.BuildSetParams(context.Background(), task, mapper)
	assert.Error(t, err)
}

// BuildGetParams
func TestBuildGetParams(t *testing.T) {
	payload := GetParamsPayload{
		Parameters: []string{
			"Device.DeviceInfo.Manufacturer",
			"Device.DeviceInfo.SoftwareVersion",
			"Device.DeviceInfo.SerialNumber",
		},
	}

	task := makeTask(TypeGetParams, payload)
	executor := NewExecutor()
	mapper := datamodel.NewMapper(datamodel.TR181)

	names, err := executor.BuildGetParams(context.Background(), task, mapper)
	require.NoError(t, err)
	require.Len(t, names, 3)

	assert.Contains(t, names, "Device.DeviceInfo.Manufacturer")
	assert.Contains(t, names, "Device.DeviceInfo.SoftwareVersion")
	assert.Contains(t, names, "Device.DeviceInfo.SerialNumber")
}

// BuildGetParams: non-GetParams type returns nil

func TestBuildGetParamsNonGetType(t *testing.T) {
	task := makeTask(TypeReboot, struct{}{})
	executor := NewExecutor()
	mapper := datamodel.NewMapper(datamodel.TR181)

	names, err := executor.BuildGetParams(context.Background(), task, mapper)
	assert.NoError(t, err)
	assert.Nil(t, names)
}

// BuildSetParams: TypeSetParams passes through directly

func TestBuildSetParamsDirect(t *testing.T) {
	payload := SetParamsPayload{
		Parameters: map[string]string{
			"Device.ManagementServer.URL":      "http://acs.example.com:7547/",
			"Device.ManagementServer.Username": "admin",
			"Device.ManagementServer.Password": "secret",
		},
	}

	task := makeTask(TypeSetParams, payload)
	executor := NewExecutor()
	mapper := datamodel.NewMapper(datamodel.TR181)

	params, err := executor.BuildSetParams(context.Background(), task, mapper)
	require.NoError(t, err)
	require.NotNil(t, params)

	assert.Equal(t, "http://acs.example.com:7547/", params["Device.ManagementServer.URL"])
	assert.Equal(t, "admin", params["Device.ManagementServer.Username"])
	assert.Equal(t, "secret", params["Device.ManagementServer.Password"])
}

// BuildSetParams: TypeReboot returns nil (no SetParameterValues)

func TestBuildSetParamsRebootReturnsNil(t *testing.T) {
	task := makeTask(TypeReboot, struct{}{})
	executor := NewExecutor()
	mapper := datamodel.NewMapper(datamodel.TR181)

	params, err := executor.BuildSetParams(context.Background(), task, mapper)
	assert.NoError(t, err)
	assert.Nil(t, params)
}

// BuildSetParams: TypeSetParams with empty parameters returns error

func TestBuildSetParamsDirectEmpty(t *testing.T) {
	payload := SetParamsPayload{
		Parameters: map[string]string{},
	}

	task := makeTask(TypeSetParams, payload)
	executor := NewExecutor()
	mapper := datamodel.NewMapper(datamodel.TR181)

	_, err := executor.BuildSetParams(context.Background(), task, mapper)
	assert.Error(t, err)
}

// BuildSetParams: WiFi with no settable fields returns error

func TestBuildSetParamsWifiEmpty(t *testing.T) {
	payload := WiFiPayload{Band: "2.4"}
	// No SSID, Password, Enabled, Channel  all zero values

	task := makeTask(TypeWifi, payload)
	executor := NewExecutor()
	mapper := datamodel.NewMapper(datamodel.TR181)

	_, err := executor.BuildSetParams(context.Background(), task, mapper)
	assert.Error(t, err)
}
