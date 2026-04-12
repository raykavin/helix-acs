package datamodel

import (
	"fmt"
	"strings"
)

// TR098Mapper implements Mapper for the TR-098 InternetGatewayDevice data-model.
//
// Optional index override fields let callers supply instance numbers discovered
// at runtime (e.g. via DiscoverInstances). A zero value means "use the default
// hardcoded index" so that &TR098Mapper{} continues to behave as before.
type TR098Mapper struct {
	// WANDeviceIdx overrides the WANDevice.{i} index. Default: 1.
	WANDeviceIdx int
	// WANConnDevIdx overrides the WANConnectionDevice.{i} index. Default: 1.
	WANConnDevIdx int
	// WANIPConnIdx overrides the WANIPConnection.{i} index. Default: 1.
	WANIPConnIdx int
	// WANPPPConnIdx overrides the WANPPPConnection.{i} index. Default: 1.
	WANPPPConnIdx int
	// LANDeviceIdx overrides the LANDevice.{i} index. Default: 1.
	LANDeviceIdx int
	// WLANIndices[bandIdx] overrides the WLANConfiguration.{i} instance.
	// Default: bandIdx+1.
	WLANIndices []int
}

// ---- index helpers ----------------------------------------------------------

func (m *TR098Mapper) wanDev() int {
	if m.WANDeviceIdx > 0 {
		return m.WANDeviceIdx
	}
	return 1
}

func (m *TR098Mapper) wanConn() int {
	if m.WANConnDevIdx > 0 {
		return m.WANConnDevIdx
	}
	return 1
}

func (m *TR098Mapper) wanIPConn() int {
	if m.WANIPConnIdx > 0 {
		return m.WANIPConnIdx
	}
	return 1
}

func (m *TR098Mapper) wanPPPConn() int {
	if m.WANPPPConnIdx > 0 {
		return m.WANPPPConnIdx
	}
	return 1
}

func (m *TR098Mapper) lanDev() int {
	if m.LANDeviceIdx > 0 {
		return m.LANDeviceIdx
	}
	return 1
}

func (m *TR098Mapper) wlanIdx(band int) int {
	if band < len(m.WLANIndices) && m.WLANIndices[band] > 0 {
		return m.WLANIndices[band]
	}
	return band + 1
}

// wanBase builds the common WANDevice.N.WANConnectionDevice.M prefix.
func (m *TR098Mapper) wanBase() string {
	return fmt.Sprintf("InternetGatewayDevice.WANDevice.%d.WANConnectionDevice.%d", m.wanDev(), m.wanConn())
}

// ---- Mapper interface -------------------------------------------------------

func (m *TR098Mapper) DetectModel(params map[string]string) ModelType {
	for k := range params {
		if strings.HasPrefix(k, "InternetGatewayDevice.") {
			return TR098
		}
	}
	return TR098
}

func (m *TR098Mapper) ExtractDeviceInfo(params map[string]string) *UnifiedDevice {
	wanIPBase := fmt.Sprintf("%s.WANIPConnection.%d", m.wanBase(), m.wanIPConn())
	lanBase := fmt.Sprintf("InternetGatewayDevice.LANDevice.%d.LANHostConfigManagement.IPInterface.1", m.lanDev())
	return &UnifiedDevice{
		Manufacturer: params["InternetGatewayDevice.DeviceInfo.Manufacturer"],
		ModelName:    params["InternetGatewayDevice.DeviceInfo.ModelName"],
		Serial:       params["InternetGatewayDevice.DeviceInfo.SerialNumber"],
		OUI:          params["InternetGatewayDevice.DeviceInfo.ManufacturerOUI"],
		ProductClass: params["InternetGatewayDevice.DeviceInfo.ProductClass"],
		SWVersion:    params["InternetGatewayDevice.DeviceInfo.SoftwareVersion"],
		HWVersion:    params["InternetGatewayDevice.DeviceInfo.HardwareVersion"],
		WANIP:        params[wanIPBase+".ExternalIPAddress"],
		LANIP:        params[lanBase+".IPInterfaceIPAddress"],
	}
}

// WiFi
// bandIdx 0 → WLANConfiguration.1 (2.4 GHz), bandIdx 1 → WLANConfiguration.2 (5 GHz).

func (m *TR098Mapper) wlanBase(bandIdx int) string {
	return fmt.Sprintf("InternetGatewayDevice.LANDevice.%d.WLANConfiguration.%d", m.lanDev(), m.wlanIdx(bandIdx))
}

func (m *TR098Mapper) WiFiSSIDPath(bandIdx int) string {
	return m.wlanBase(bandIdx) + ".SSID"
}
func (m *TR098Mapper) WiFiPasswordPath(bandIdx int) string {
	return m.wlanBase(bandIdx) + ".PreSharedKey.1.KeyPassphrase"
}
func (m *TR098Mapper) WiFiEnabledPath(bandIdx int) string {
	return m.wlanBase(bandIdx) + ".Enable"
}
func (m *TR098Mapper) WiFiChannelPath(bandIdx int) string {
	return m.wlanBase(bandIdx) + ".Channel"
}
func (m *TR098Mapper) WiFiBSSIDPath(bandIdx int) string {
	return m.wlanBase(bandIdx) + ".BSSID"
}
func (m *TR098Mapper) WiFiStandardPath(bandIdx int) string {
	return m.wlanBase(bandIdx) + ".Standard"
}
func (m *TR098Mapper) WiFiSecurityModePath(bandIdx int) string {
	return m.wlanBase(bandIdx) + ".BeaconType"
}
func (m *TR098Mapper) WiFiChannelWidthPath(bandIdx int) string {
	// Not standardised in base TR-098; most vendors expose it here.
	return m.wlanBase(bandIdx) + ".OperatingChannelBandwidth"
}
func (m *TR098Mapper) WiFiTXPowerPath(bandIdx int) string {
	return m.wlanBase(bandIdx) + ".TransmitPower"
}
func (m *TR098Mapper) WiFiClientCountPath(bandIdx int) string {
	return m.wlanBase(bandIdx) + ".TotalAssociations"
}
func (m *TR098Mapper) WiFiBytesSentPath(bandIdx int) string {
	return m.wlanBase(bandIdx) + ".Stats.BytesSent"
}
func (m *TR098Mapper) WiFiBytesReceivedPath(bandIdx int) string {
	return m.wlanBase(bandIdx) + ".Stats.BytesReceived"
}
func (m *TR098Mapper) WiFiPacketsSentPath(bandIdx int) string {
	return m.wlanBase(bandIdx) + ".Stats.PacketsSent"
}
func (m *TR098Mapper) WiFiPacketsReceivedPath(bandIdx int) string {
	return m.wlanBase(bandIdx) + ".Stats.PacketsReceived"
}
func (m *TR098Mapper) WiFiErrorsSentPath(bandIdx int) string {
	return m.wlanBase(bandIdx) + ".Stats.ErrorsSent"
}
func (m *TR098Mapper) WiFiErrorsReceivedPath(bandIdx int) string {
	return m.wlanBase(bandIdx) + ".Stats.ErrorsReceived"
}

// WAN

func (m *TR098Mapper) WANConnectionTypePath() string {
	return fmt.Sprintf("%s.WANIPConnection.%d.ConnectionType", m.wanBase(), m.wanIPConn())
}
func (m *TR098Mapper) WANPPPoEUserPath() string {
	return fmt.Sprintf("%s.WANPPPConnection.%d.Username", m.wanBase(), m.wanPPPConn())
}
func (m *TR098Mapper) WANPPPoEPassPath() string {
	return fmt.Sprintf("%s.WANPPPConnection.%d.Password", m.wanBase(), m.wanPPPConn())
}
func (m *TR098Mapper) WANIPAddressPath() string {
	return fmt.Sprintf("%s.WANIPConnection.%d.ExternalIPAddress", m.wanBase(), m.wanIPConn())
}
func (m *TR098Mapper) WANGatewayPath() string {
	return fmt.Sprintf("%s.WANIPConnection.%d.DefaultGateway", m.wanBase(), m.wanIPConn())
}
func (m *TR098Mapper) WANDNS1Path() string {
	// TR-098 stores DNS servers as a comma-separated string.
	return fmt.Sprintf("%s.WANIPConnection.%d.DNSServers", m.wanBase(), m.wanIPConn())
}
func (m *TR098Mapper) WANDNS2Path() string {
	// Same field; the caller must split on comma and take index 1.
	return fmt.Sprintf("%s.WANIPConnection.%d.DNSServers", m.wanBase(), m.wanIPConn())
}
func (m *TR098Mapper) WANMTUPath() string {
	return fmt.Sprintf("%s.WANIPConnection.%d.MaxMTUSize", m.wanBase(), m.wanIPConn())
}
func (m *TR098Mapper) WANUptimePath() string {
	return fmt.Sprintf("%s.WANIPConnection.%d.Uptime", m.wanBase(), m.wanIPConn())
}
func (m *TR098Mapper) WANMACPath() string {
	return fmt.Sprintf("InternetGatewayDevice.WANDevice.%d.WANEthernetInterfaceConfig.MACAddress", m.wanDev())
}
func (m *TR098Mapper) WANStatusPath() string {
	return fmt.Sprintf("%s.WANIPConnection.%d.ConnectionStatus", m.wanBase(), m.wanIPConn())
}
func (m *TR098Mapper) WANBytesSentPath() string {
	return fmt.Sprintf("InternetGatewayDevice.WANDevice.%d.WANCommonInterfaceConfig.TotalBytesSent", m.wanDev())
}
func (m *TR098Mapper) WANBytesReceivedPath() string {
	return fmt.Sprintf("InternetGatewayDevice.WANDevice.%d.WANCommonInterfaceConfig.TotalBytesReceived", m.wanDev())
}
func (m *TR098Mapper) WANPacketsSentPath() string {
	return fmt.Sprintf("InternetGatewayDevice.WANDevice.%d.WANCommonInterfaceConfig.TotalPacketsSent", m.wanDev())
}
func (m *TR098Mapper) WANPacketsReceivedPath() string {
	return fmt.Sprintf("InternetGatewayDevice.WANDevice.%d.WANCommonInterfaceConfig.TotalPacketsReceived", m.wanDev())
}
func (m *TR098Mapper) WANErrorsSentPath() string {
	// TR-098 WANCommonInterfaceConfig does not expose error counters.
	// No standard path exists for this metric in the base TR-098 data model.
	return ""
}
func (m *TR098Mapper) WANErrorsReceivedPath() string {
	// TR-098 WANCommonInterfaceConfig does not expose error counters.
	// No standard path exists for this metric in the base TR-098 data model.
	return ""
}

// LAN / DHCP

func (m *TR098Mapper) LANIPAddressPath() string {
	return fmt.Sprintf("InternetGatewayDevice.LANDevice.%d.LANHostConfigManagement.IPInterface.1.IPInterfaceIPAddress", m.lanDev())
}
func (m *TR098Mapper) LANSubnetMaskPath() string {
	return fmt.Sprintf("InternetGatewayDevice.LANDevice.%d.LANHostConfigManagement.IPInterface.1.IPInterfaceSubnetMask", m.lanDev())
}
func (m *TR098Mapper) DHCPServerEnablePath() string {
	return fmt.Sprintf("InternetGatewayDevice.LANDevice.%d.LANHostConfigManagement.DHCPServerEnable", m.lanDev())
}
func (m *TR098Mapper) DHCPMinAddressPath() string {
	return fmt.Sprintf("InternetGatewayDevice.LANDevice.%d.LANHostConfigManagement.MinAddress", m.lanDev())
}
func (m *TR098Mapper) DHCPMaxAddressPath() string {
	return fmt.Sprintf("InternetGatewayDevice.LANDevice.%d.LANHostConfigManagement.MaxAddress", m.lanDev())
}
func (m *TR098Mapper) LANDNSPath() string {
	return fmt.Sprintf("InternetGatewayDevice.LANDevice.%d.LANHostConfigManagement.DNSServers", m.lanDev())
}

// System

func (m *TR098Mapper) CPEUptimePath() string {
	return "InternetGatewayDevice.DeviceInfo.UpTime"
}
func (m *TR098Mapper) RAMTotalPath() string {
	return "InternetGatewayDevice.DeviceInfo.MemoryStatus.Total"
}
func (m *TR098Mapper) RAMFreePath() string {
	return "InternetGatewayDevice.DeviceInfo.MemoryStatus.Free"
}

// ManagementServer

func (m *TR098Mapper) ManagementServerURLPath() string {
	return "InternetGatewayDevice.ManagementServer.URL"
}
func (m *TR098Mapper) ManagementServerUserPath() string {
	return "InternetGatewayDevice.ManagementServer.Username"
}
func (m *TR098Mapper) ManagementServerPassPath() string {
	return "InternetGatewayDevice.ManagementServer.Password"
}
func (m *TR098Mapper) ManagementServerInformIntervalPath() string {
	return "InternetGatewayDevice.ManagementServer.PeriodicInformInterval"
}

// Connected hosts

func (m *TR098Mapper) HostsBasePath() string {
	return fmt.Sprintf("InternetGatewayDevice.LANDevice.%d.Hosts.Host.", m.lanDev())
}
func (m *TR098Mapper) HostsCountPath() string {
	return fmt.Sprintf("InternetGatewayDevice.LANDevice.%d.Hosts.HostNumberOfEntries", m.lanDev())
}

// Diagnostics

func (m *TR098Mapper) PingDiagBasePath() string {
	return "InternetGatewayDevice.IPPingDiagnostics."
}
func (m *TR098Mapper) TracerouteDiagBasePath() string {
	return "InternetGatewayDevice.TraceRouteDiagnostics."
}
func (m *TR098Mapper) DownloadDiagBasePath() string {
	return "InternetGatewayDevice.DownloadDiagnostics."
}
func (m *TR098Mapper) UploadDiagBasePath() string {
	return "InternetGatewayDevice.UploadDiagnostics."
}

// Port forwarding

func (m *TR098Mapper) PortMappingBasePath() string {
	return fmt.Sprintf("%s.WANIPConnection.%d.PortMapping.", m.wanBase(), m.wanIPConn())
}
func (m *TR098Mapper) PortMappingCountPath() string {
	return fmt.Sprintf("%s.WANIPConnection.%d.PortMappingNumberOfEntries", m.wanBase(), m.wanIPConn())
}

// Web admin interface

// WebAdminPasswordPath returns "" for TR-098 because the spec defines no
// standard parameter for the local web admin password  each vendor uses a
// proprietary X_ extension. Callers should fall back to TypeSetParams with
// the vendor-specific path when this returns an empty string.
func (m *TR098Mapper) WebAdminPasswordPath() string { return "" }
