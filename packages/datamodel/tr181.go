package datamodel

import (
	"fmt"
	"strings"
)

// TR181Mapper implements Mapper for the TR-181 Device: data-model (Device.).
//
// Optional index override fields let callers supply instance numbers discovered
// at runtime (e.g. via DiscoverInstances). A zero value means "use the default
// hardcoded index" so that &TR181Mapper{} continues to behave as before.
type TR181Mapper struct {
	// WANIfaceIdx overrides the Device.IP.Interface.{i} index for WAN paths.
	// Default: 1.
	WANIfaceIdx int
	// LANIfaceIdx overrides the Device.IP.Interface.{i} index for LAN paths.
	// Default: 2.
	LANIfaceIdx int
	// PPPIfaceIdx overrides the Device.PPP.Interface.{i} index.
	// Default: 1.
	PPPIfaceIdx int
	// SSIDIndices[bandIdx] overrides the Device.WiFi.SSID.{i} instance.
	// Default: bandIdx+1.
	SSIDIndices []int
	// RadioIndices[bandIdx] overrides the Device.WiFi.Radio.{i} instance.
	// Default: bandIdx+1.
	RadioIndices []int
	// APIndices[bandIdx] overrides the Device.WiFi.AccessPoint.{i} instance.
	// Default: bandIdx+1.
	APIndices []int
}

// ---- index helpers ----------------------------------------------------------

func (m *TR181Mapper) wanIface() int {
	if m.WANIfaceIdx > 0 {
		return m.WANIfaceIdx
	}
	return 1
}

func (m *TR181Mapper) lanIface() int {
	if m.LANIfaceIdx > 0 {
		return m.LANIfaceIdx
	}
	return 2
}

func (m *TR181Mapper) pppIface() int {
	if m.PPPIfaceIdx > 0 {
		return m.PPPIfaceIdx
	}
	return 1
}

func (m *TR181Mapper) ssidIdx(band int) int {
	if band < len(m.SSIDIndices) && m.SSIDIndices[band] > 0 {
		return m.SSIDIndices[band]
	}
	return band + 1
}

func (m *TR181Mapper) radioIdx(band int) int {
	if band < len(m.RadioIndices) && m.RadioIndices[band] > 0 {
		return m.RadioIndices[band]
	}
	return band + 1
}

func (m *TR181Mapper) apIdx(band int) int {
	if band < len(m.APIndices) && m.APIndices[band] > 0 {
		return m.APIndices[band]
	}
	return band + 1
}

// ---- Mapper interface -------------------------------------------------------

func (m *TR181Mapper) DetectModel(params map[string]string) ModelType {
	for k := range params {
		if strings.HasPrefix(k, "Device.") {
			return TR181
		}
	}
	return TR098
}

func (m *TR181Mapper) ExtractDeviceInfo(params map[string]string) *UnifiedDevice {
	return &UnifiedDevice{
		Manufacturer: params["Device.DeviceInfo.Manufacturer"],
		ModelName:    params["Device.DeviceInfo.ModelName"],
		Serial:       params["Device.DeviceInfo.SerialNumber"],
		OUI:          params["Device.DeviceInfo.ManufacturerOUI"],
		ProductClass: params["Device.DeviceInfo.ProductClass"],
		SWVersion:    params["Device.DeviceInfo.SoftwareVersion"],
		HWVersion:    params["Device.DeviceInfo.HardwareVersion"],
		WANIP:        params[fmt.Sprintf("Device.IP.Interface.%d.IPv4Address.1.IPAddress", m.wanIface())],
		LANIP:        params[fmt.Sprintf("Device.IP.Interface.%d.IPv4Address.1.IPAddress", m.lanIface())],
	}
}

// WiFi
// bandIdx 0 → 2.4 GHz radio / SSID, bandIdx 1 → 5 GHz.

func (m *TR181Mapper) WiFiSSIDPath(bandIdx int) string {
	return fmt.Sprintf("Device.WiFi.SSID.%d.SSID", m.ssidIdx(bandIdx))
}
func (m *TR181Mapper) WiFiPasswordPath(bandIdx int) string {
	return fmt.Sprintf("Device.WiFi.AccessPoint.%d.Security.KeyPassphrase", m.apIdx(bandIdx))
}
func (m *TR181Mapper) WiFiEnabledPath(bandIdx int) string {
	return fmt.Sprintf("Device.WiFi.SSID.%d.Enable", m.ssidIdx(bandIdx))
}
func (m *TR181Mapper) WiFiChannelPath(bandIdx int) string {
	return fmt.Sprintf("Device.WiFi.Radio.%d.Channel", m.radioIdx(bandIdx))
}
func (m *TR181Mapper) WiFiBSSIDPath(bandIdx int) string {
	return fmt.Sprintf("Device.WiFi.SSID.%d.BSSID", m.ssidIdx(bandIdx))
}
func (m *TR181Mapper) WiFiStandardPath(bandIdx int) string {
	return fmt.Sprintf("Device.WiFi.Radio.%d.OperatingStandards", m.radioIdx(bandIdx))
}
func (m *TR181Mapper) WiFiSecurityModePath(bandIdx int) string {
	return fmt.Sprintf("Device.WiFi.AccessPoint.%d.Security.ModeEnabled", m.apIdx(bandIdx))
}
func (m *TR181Mapper) WiFiChannelWidthPath(bandIdx int) string {
	return fmt.Sprintf("Device.WiFi.Radio.%d.OperatingChannelBandwidth", m.radioIdx(bandIdx))
}
func (m *TR181Mapper) WiFiTXPowerPath(bandIdx int) string {
	return fmt.Sprintf("Device.WiFi.Radio.%d.TransmitPower", m.radioIdx(bandIdx))
}
func (m *TR181Mapper) WiFiClientCountPath(bandIdx int) string {
	return fmt.Sprintf("Device.WiFi.AccessPoint.%d.AssociatedDeviceNumberOfEntries", m.apIdx(bandIdx))
}
func (m *TR181Mapper) WiFiBytesSentPath(bandIdx int) string {
	return fmt.Sprintf("Device.WiFi.SSID.%d.Stats.BytesSent", m.ssidIdx(bandIdx))
}
func (m *TR181Mapper) WiFiBytesReceivedPath(bandIdx int) string {
	return fmt.Sprintf("Device.WiFi.SSID.%d.Stats.BytesReceived", m.ssidIdx(bandIdx))
}
func (m *TR181Mapper) WiFiPacketsSentPath(bandIdx int) string {
	return fmt.Sprintf("Device.WiFi.SSID.%d.Stats.PacketsSent", m.ssidIdx(bandIdx))
}
func (m *TR181Mapper) WiFiPacketsReceivedPath(bandIdx int) string {
	return fmt.Sprintf("Device.WiFi.SSID.%d.Stats.PacketsReceived", m.ssidIdx(bandIdx))
}
func (m *TR181Mapper) WiFiErrorsSentPath(bandIdx int) string {
	return fmt.Sprintf("Device.WiFi.SSID.%d.Stats.ErrorsSent", m.ssidIdx(bandIdx))
}
func (m *TR181Mapper) WiFiErrorsReceivedPath(bandIdx int) string {
	return fmt.Sprintf("Device.WiFi.SSID.%d.Stats.ErrorsReceived", m.ssidIdx(bandIdx))
}

// WAN

func (m *TR181Mapper) WANConnectionTypePath() string {
	return fmt.Sprintf("Device.IP.Interface.%d.IPv4Address.1.AddressingType", m.wanIface())
}
func (m *TR181Mapper) WANPPPoEUserPath() string {
	return fmt.Sprintf("Device.PPP.Interface.%d.Username", m.pppIface())
}
func (m *TR181Mapper) WANPPPoEPassPath() string {
	return fmt.Sprintf("Device.PPP.Interface.%d.Password", m.pppIface())
}
func (m *TR181Mapper) WANIPAddressPath() string {
	return fmt.Sprintf("Device.IP.Interface.%d.IPv4Address.1.IPAddress", m.wanIface())
}
func (m *TR181Mapper) WANGatewayPath() string {
	return "Device.Routing.Router.1.IPv4Forwarding.1.GatewayIPAddress"
}
func (m *TR181Mapper) WANDNS1Path() string {
	return "Device.DNS.Client.Server.1.DNSServer"
}
func (m *TR181Mapper) WANDNS2Path() string {
	return "Device.DNS.Client.Server.2.DNSServer"
}
func (m *TR181Mapper) WANMTUPath() string {
	return fmt.Sprintf("Device.IP.Interface.%d.MaxMTUSize", m.wanIface())
}
func (m *TR181Mapper) WANUptimePath() string {
	// Seconds since last state change on the WAN IP interface.
	return fmt.Sprintf("Device.IP.Interface.%d.LastChange", m.wanIface())
}
func (m *TR181Mapper) WANMACPath() string {
	return "Device.Ethernet.Interface.1.MACAddress"
}
func (m *TR181Mapper) WANStatusPath() string {
	return fmt.Sprintf("Device.IP.Interface.%d.Status", m.wanIface())
}
func (m *TR181Mapper) WANBytesSentPath() string {
	return fmt.Sprintf("Device.IP.Interface.%d.Stats.BytesSent", m.wanIface())
}
func (m *TR181Mapper) WANBytesReceivedPath() string {
	return fmt.Sprintf("Device.IP.Interface.%d.Stats.BytesReceived", m.wanIface())
}
func (m *TR181Mapper) WANPacketsSentPath() string {
	return fmt.Sprintf("Device.IP.Interface.%d.Stats.PacketsSent", m.wanIface())
}
func (m *TR181Mapper) WANPacketsReceivedPath() string {
	return fmt.Sprintf("Device.IP.Interface.%d.Stats.PacketsReceived", m.wanIface())
}
func (m *TR181Mapper) WANErrorsSentPath() string {
	return fmt.Sprintf("Device.IP.Interface.%d.Stats.ErrorsSent", m.wanIface())
}
func (m *TR181Mapper) WANErrorsReceivedPath() string {
	return fmt.Sprintf("Device.IP.Interface.%d.Stats.ErrorsReceived", m.wanIface())
}

// LAN / DHCP

func (m *TR181Mapper) LANIPAddressPath() string {
	return fmt.Sprintf("Device.IP.Interface.%d.IPv4Address.1.IPAddress", m.lanIface())
}
func (m *TR181Mapper) LANSubnetMaskPath() string {
	return fmt.Sprintf("Device.IP.Interface.%d.IPv4Address.1.SubnetMask", m.lanIface())
}
func (m *TR181Mapper) DHCPServerEnablePath() string {
	return "Device.DHCPv4.Server.Enable"
}
func (m *TR181Mapper) DHCPMinAddressPath() string {
	return "Device.DHCPv4.Server.Pool.1.MinAddress"
}
func (m *TR181Mapper) DHCPMaxAddressPath() string {
	return "Device.DHCPv4.Server.Pool.1.MaxAddress"
}
func (m *TR181Mapper) LANDNSPath() string {
	return "Device.DHCPv4.Server.Pool.1.DNSServers"
}

// System

func (m *TR181Mapper) CPEUptimePath() string {
	return "Device.DeviceInfo.UpTime"
}
func (m *TR181Mapper) RAMTotalPath() string {
	return "Device.DeviceInfo.MemoryStatus.Total"
}
func (m *TR181Mapper) RAMFreePath() string {
	return "Device.DeviceInfo.MemoryStatus.Free"
}

// ManagementServer

func (m *TR181Mapper) ManagementServerURLPath() string {
	return "Device.ManagementServer.URL"
}
func (m *TR181Mapper) ManagementServerUserPath() string {
	return "Device.ManagementServer.Username"
}
func (m *TR181Mapper) ManagementServerPassPath() string {
	return "Device.ManagementServer.Password"
}
func (m *TR181Mapper) ManagementServerInformIntervalPath() string {
	return "Device.ManagementServer.PeriodicInformInterval"
}

// Connected hosts

func (m *TR181Mapper) HostsBasePath() string {
	return "Device.Hosts.Host."
}
func (m *TR181Mapper) HostsCountPath() string {
	return "Device.Hosts.HostNumberOfEntries"
}

// Diagnostics

func (m *TR181Mapper) PingDiagBasePath() string {
	return "Device.IP.Diagnostics.IPPing."
}
func (m *TR181Mapper) TracerouteDiagBasePath() string {
	return "Device.IP.Diagnostics.TraceRoute."
}
func (m *TR181Mapper) DownloadDiagBasePath() string {
	return "Device.IP.Diagnostics.DownloadDiagnostics."
}
func (m *TR181Mapper) UploadDiagBasePath() string {
	return "Device.IP.Diagnostics.UploadDiagnostics."
}

// Port forwarding

func (m *TR181Mapper) PortMappingBasePath() string {
	return "Device.NAT.PortMapping."
}
func (m *TR181Mapper) PortMappingCountPath() string {
	return "Device.NAT.PortMappingNumberOfEntries"
}

// Web admin interface

// WebAdminPasswordPath returns the TR-181 standard path for the local web
// administration password (Users table, first user  typically the admin).
func (m *TR181Mapper) WebAdminPasswordPath() string {
	return "Device.Users.User.1.Password"
}
