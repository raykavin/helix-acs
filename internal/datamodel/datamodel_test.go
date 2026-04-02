package datamodel

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// DetectFromRootObject

func TestDetectFromRootObject(t *testing.T) {
	assert.Equal(t, TR181, DetectFromRootObject("Device."))
	assert.Equal(t, TR181, DetectFromRootObject("Device"))
	assert.Equal(t, TR098, DetectFromRootObject("InternetGatewayDevice."))
	assert.Equal(t, TR098, DetectFromRootObject("InternetGatewayDevice"))
	assert.Equal(t, TR098, DetectFromRootObject(""))
	assert.Equal(t, TR098, DetectFromRootObject("unknown"))
}

// TR181Mapper path methods

func TestTR181Mapper(t *testing.T) {
	m := &TR181Mapper{}

	// WiFi band 0 (2.4 GHz)
	assert.Equal(t, "Device.WiFi.SSID.1.SSID", m.WiFiSSIDPath(0))
	assert.Equal(t, "Device.WiFi.AccessPoint.1.Security.KeyPassphrase", m.WiFiPasswordPath(0))
	assert.Equal(t, "Device.WiFi.SSID.1.Enable", m.WiFiEnabledPath(0))
	assert.Equal(t, "Device.WiFi.Radio.1.Channel", m.WiFiChannelPath(0))

	// WiFi band 1 (5 GHz)
	assert.Equal(t, "Device.WiFi.SSID.2.SSID", m.WiFiSSIDPath(1))
	assert.Equal(t, "Device.WiFi.AccessPoint.2.Security.KeyPassphrase", m.WiFiPasswordPath(1))
	assert.Equal(t, "Device.WiFi.SSID.2.Enable", m.WiFiEnabledPath(1))
	assert.Equal(t, "Device.WiFi.Radio.2.Channel", m.WiFiChannelPath(1))

	// WAN
	assert.Equal(t, "Device.IP.Interface.1.IPv4Address.1.AddressingType", m.WANConnectionTypePath())
	assert.Equal(t, "Device.PPP.Interface.1.Username", m.WANPPPoEUserPath())
	assert.Equal(t, "Device.PPP.Interface.1.Password", m.WANPPPoEPassPath())
	assert.Equal(t, "Device.IP.Interface.1.IPv4Address.1.IPAddress", m.WANIPAddressPath())

	// Management server
	assert.Equal(t, "Device.ManagementServer.URL", m.ManagementServerURLPath())
	assert.Equal(t, "Device.ManagementServer.Username", m.ManagementServerUserPath())
	assert.Equal(t, "Device.ManagementServer.Password", m.ManagementServerPassPath())
	assert.Equal(t, "Device.ManagementServer.PeriodicInformInterval", m.ManagementServerInformIntervalPath())

	// LAN / DHCP
	assert.Equal(t, "Device.IP.Interface.2.IPv4Address.1.IPAddress", m.LANIPAddressPath())
	assert.Equal(t, "Device.IP.Interface.2.IPv4Address.1.SubnetMask", m.LANSubnetMaskPath())
	assert.Equal(t, "Device.DHCPv4.Server.Enable", m.DHCPServerEnablePath())
	assert.Equal(t, "Device.DHCPv4.Server.Pool.1.MinAddress", m.DHCPMinAddressPath())
	assert.Equal(t, "Device.DHCPv4.Server.Pool.1.MaxAddress", m.DHCPMaxAddressPath())
}

// TR098Mapper path methods

func TestTR098Mapper(t *testing.T) {
	m := &TR098Mapper{}

	// WiFi band 0 (2.4 GHz)
	assert.Equal(t, "InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID", m.WiFiSSIDPath(0))
	assert.Equal(t, "InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.PreSharedKey.1.KeyPassphrase", m.WiFiPasswordPath(0))
	assert.Equal(t, "InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.Enable", m.WiFiEnabledPath(0))
	assert.Equal(t, "InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.Channel", m.WiFiChannelPath(0))

	// WiFi band 1 (5 GHz)
	assert.Equal(t, "InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.SSID", m.WiFiSSIDPath(1))
	assert.Equal(t, "InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.PreSharedKey.1.KeyPassphrase", m.WiFiPasswordPath(1))
	assert.Equal(t, "InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.Enable", m.WiFiEnabledPath(1))
	assert.Equal(t, "InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.Channel", m.WiFiChannelPath(1))

	// WAN
	assert.Equal(t, "InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANIPConnection.1.ConnectionType", m.WANConnectionTypePath())
	assert.Equal(t, "InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1.Username", m.WANPPPoEUserPath())
	assert.Equal(t, "InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1.Password", m.WANPPPoEPassPath())
	assert.Equal(t, "InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANIPConnection.1.ExternalIPAddress", m.WANIPAddressPath())

	// Management server
	assert.Equal(t, "InternetGatewayDevice.ManagementServer.URL", m.ManagementServerURLPath())
	assert.Equal(t, "InternetGatewayDevice.ManagementServer.Username", m.ManagementServerUserPath())
	assert.Equal(t, "InternetGatewayDevice.ManagementServer.Password", m.ManagementServerPassPath())
	assert.Equal(t, "InternetGatewayDevice.ManagementServer.PeriodicInformInterval", m.ManagementServerInformIntervalPath())

	// LAN / DHCP
	assert.Equal(t, "InternetGatewayDevice.LANDevice.1.LANHostConfigManagement.IPInterface.1.IPInterfaceIPAddress", m.LANIPAddressPath())
	assert.Equal(t, "InternetGatewayDevice.LANDevice.1.LANHostConfigManagement.IPInterface.1.IPInterfaceSubnetMask", m.LANSubnetMaskPath())
	assert.Equal(t, "InternetGatewayDevice.LANDevice.1.LANHostConfigManagement.DHCPServerEnable", m.DHCPServerEnablePath())
	assert.Equal(t, "InternetGatewayDevice.LANDevice.1.LANHostConfigManagement.MinAddress", m.DHCPMinAddressPath())
	assert.Equal(t, "InternetGatewayDevice.LANDevice.1.LANHostConfigManagement.MaxAddress", m.DHCPMaxAddressPath())
}

// DetectModel via TR181Mapper

func TestTR181DetectModel(t *testing.T) {
	m := &TR181Mapper{}

	// At least one key starting with "Device." → TR181
	params := map[string]string{
		"Device.DeviceInfo.Manufacturer": "Intelbras",
		"Device.DeviceInfo.SerialNumber": "SN001",
	}
	assert.Equal(t, TR181, m.DetectModel(params))
}

func TestTR181DetectModelFallback(t *testing.T) {
	m := &TR181Mapper{}

	// No "Device." prefix → fallback TR098
	params := map[string]string{
		"InternetGatewayDevice.DeviceInfo.Manufacturer": "TP-Link",
	}
	assert.Equal(t, TR098, m.DetectModel(params))
}

// DetectModel via TR098Mapper

func TestTR098DetectModel(t *testing.T) {
	m := &TR098Mapper{}

	params := map[string]string{
		"InternetGatewayDevice.DeviceInfo.Manufacturer": "TP-Link",
		"InternetGatewayDevice.DeviceInfo.SerialNumber": "SN002",
	}
	assert.Equal(t, TR098, m.DetectModel(params))
}

func TestTR098DetectModelAlwaysTR098(t *testing.T) {
	m := &TR098Mapper{}

	// TR098Mapper always returns TR098 regardless
	params := map[string]string{
		"Device.DeviceInfo.Manufacturer": "Intelbras",
	}
	assert.Equal(t, TR098, m.DetectModel(params))
}

// NewMapper

func TestNewMapper(t *testing.T) {
	m181 := NewMapper(TR181)
	require.NotNil(t, m181)
	_, ok := m181.(*TR181Mapper)
	assert.True(t, ok, "TR181 should return *TR181Mapper")

	m098 := NewMapper(TR098)
	require.NotNil(t, m098)
	_, ok2 := m098.(*TR098Mapper)
	assert.True(t, ok2, "TR098 should return *TR098Mapper")

	// Unknown falls back to TR098
	mUnknown := NewMapper(ModelType("unknown"))
	require.NotNil(t, mUnknown)
	_, ok3 := mUnknown.(*TR098Mapper)
	assert.True(t, ok3, "unknown model type should fall back to *TR098Mapper")
}

// ExtractDeviceInfo – TR181

func TestExtractDeviceInfoTR181(t *testing.T) {
	m := &TR181Mapper{}

	params := map[string]string{
		"Device.DeviceInfo.Manufacturer":                "Intelbras",
		"Device.DeviceInfo.ModelName":                   "W5-1200F",
		"Device.DeviceInfo.SerialNumber":                "SN111",
		"Device.DeviceInfo.ManufacturerOUI":             "001122",
		"Device.DeviceInfo.ProductClass":                "WiFiRouter",
		"Device.DeviceInfo.SoftwareVersion":             "1.2.3",
		"Device.DeviceInfo.HardwareVersion":             "v2",
		"Device.IP.Interface.1.IPv4Address.1.IPAddress": "203.0.113.10",
		"Device.IP.Interface.2.IPv4Address.1.IPAddress": "192.168.1.1",
	}

	dev := m.ExtractDeviceInfo(params)
	require.NotNil(t, dev)
	assert.Equal(t, "Intelbras", dev.Manufacturer)
	assert.Equal(t, "W5-1200F", dev.ModelName)
	assert.Equal(t, "SN111", dev.Serial)
	assert.Equal(t, "001122", dev.OUI)
	assert.Equal(t, "WiFiRouter", dev.ProductClass)
	assert.Equal(t, "1.2.3", dev.SWVersion)
	assert.Equal(t, "v2", dev.HWVersion)
	assert.Equal(t, "203.0.113.10", dev.WANIP)
	assert.Equal(t, "192.168.1.1", dev.LANIP)
}

// ExtractDeviceInfo – TR098

func TestExtractDeviceInfoTR098(t *testing.T) {
	m := &TR098Mapper{}

	params := map[string]string{
		"InternetGatewayDevice.DeviceInfo.Manufacturer":                                                "TP-Link",
		"InternetGatewayDevice.DeviceInfo.ModelName":                                                   "TL-WR841N",
		"InternetGatewayDevice.DeviceInfo.SerialNumber":                                                "SN222",
		"InternetGatewayDevice.DeviceInfo.ManufacturerOUI":                                             "AABBCC",
		"InternetGatewayDevice.DeviceInfo.ProductClass":                                                "Router",
		"InternetGatewayDevice.DeviceInfo.SoftwareVersion":                                             "4.0.0",
		"InternetGatewayDevice.DeviceInfo.HardwareVersion":                                             "v1",
		"InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANIPConnection.1.ExternalIPAddress":  "198.51.100.5",
		"InternetGatewayDevice.LANDevice.1.LANHostConfigManagement.IPInterface.1.IPInterfaceIPAddress": "192.168.0.1",
	}

	dev := m.ExtractDeviceInfo(params)
	require.NotNil(t, dev)
	assert.Equal(t, "TP-Link", dev.Manufacturer)
	assert.Equal(t, "TL-WR841N", dev.ModelName)
	assert.Equal(t, "SN222", dev.Serial)
	assert.Equal(t, "AABBCC", dev.OUI)
	assert.Equal(t, "Router", dev.ProductClass)
	assert.Equal(t, "4.0.0", dev.SWVersion)
	assert.Equal(t, "v1", dev.HWVersion)
	assert.Equal(t, "198.51.100.5", dev.WANIP)
	assert.Equal(t, "192.168.0.1", dev.LANIP)
}

// ValidateType

func TestValidateTypeString(t *testing.T) {
	assert.NoError(t, ValidateType(TypeString, ""))
	assert.NoError(t, ValidateType(TypeString, "any arbitrary value 123!@#"))
}

func TestValidateTypeBoolean(t *testing.T) {
	assert.NoError(t, ValidateType(TypeBoolean, "0"))
	assert.NoError(t, ValidateType(TypeBoolean, "1"))
	assert.NoError(t, ValidateType(TypeBoolean, "true"))
	assert.NoError(t, ValidateType(TypeBoolean, "false"))

	assert.Error(t, ValidateType(TypeBoolean, "True"))
	assert.Error(t, ValidateType(TypeBoolean, "False"))
	assert.Error(t, ValidateType(TypeBoolean, "yes"))
	assert.Error(t, ValidateType(TypeBoolean, "no"))
	assert.Error(t, ValidateType(TypeBoolean, ""))
}

func TestValidateTypeUnsignedInt(t *testing.T) {
	assert.NoError(t, ValidateType(TypeUnsignedInt, "0"))
	assert.NoError(t, ValidateType(TypeUnsignedInt, "3600"))
	assert.NoError(t, ValidateType(TypeUnsignedInt, "18446744073709551615")) // max uint64

	assert.Error(t, ValidateType(TypeUnsignedInt, "-1"))
	assert.Error(t, ValidateType(TypeUnsignedInt, "abc"))
	assert.Error(t, ValidateType(TypeUnsignedInt, "3.14"))
}

func TestValidateTypeInt(t *testing.T) {
	assert.NoError(t, ValidateType(TypeInt, "0"))
	assert.NoError(t, ValidateType(TypeInt, "-100"))
	assert.NoError(t, ValidateType(TypeInt, "9223372036854775807")) // max int64

	assert.Error(t, ValidateType(TypeInt, "abc"))
	assert.Error(t, ValidateType(TypeInt, "1.5"))
}

func TestValidateTypeDateTime(t *testing.T) {
	assert.NoError(t, ValidateType(TypeDateTime, "2024-01-01T00:00:00Z"))
	assert.NoError(t, ValidateType(TypeDateTime, "2024-06-15T12:30:00+05:00"))
	assert.NoError(t, ValidateType(TypeDateTime, "2024-06-15T12:30:00"))
	assert.NoError(t, ValidateType(TypeDateTime, "2024-06-15"))
	// TR-069 unknown time sentinel
	assert.NoError(t, ValidateType(TypeDateTime, "0001-01-01T00:00:00Z"))
	assert.NoError(t, ValidateType(TypeDateTime, "0001-01-01T00:00:00"))

	assert.Error(t, ValidateType(TypeDateTime, "not-a-date"))
	assert.Error(t, ValidateType(TypeDateTime, "01/01/2024"))
}

func TestValidateTypeBase64(t *testing.T) {
	// Base64 is accepted without structural validation
	assert.NoError(t, ValidateType(TypeBase64, "SGVsbG8gV29ybGQ="))
	assert.NoError(t, ValidateType(TypeBase64, ""))
}

func TestValidateTypeUnknown(t *testing.T) {
	// Unknown types are silently accepted
	assert.NoError(t, ValidateType("xsd:anyType", "whatever"))
	assert.NoError(t, ValidateType("vendor:custom", "value"))
}
