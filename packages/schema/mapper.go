package schema

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/raykavin/helix-acs/packages/datamodel"
)

// SchemaMapper implements [datamodel.Mapper] by resolving TR-069 parameter
// paths from YAML schema files loaded into a Registry.
//
// Path templates may contain {placeholder} tokens that are substituted with
// instance indices discovered at runtime via [datamodel.DiscoverInstances].
// Supported placeholders:
//
//	{ssid_0}, {ssid_1}           – WiFi SSID instance per band
//	{radio_0}, {radio_1}         – WiFi Radio instance per band
//	{ap_0}, {ap_1}               – WiFi AccessPoint instance per band
//	{wan}                        – WAN IP interface (TR-181)
//	{lan}                        – LAN IP interface (TR-181)
//	{ppp}                        – PPP interface (TR-181)
//	{wan_dev}                    – WANDevice index (TR-098)
//	{wan_conn}                   – WANConnectionDevice index (TR-098)
//	{wan_ip_conn}                – WANIPConnection index (TR-098)
//	{wan_ppp_conn}               – WANPPPConnection index (TR-098)
//	{lan_dev}                    – LANDevice index (TR-098)
//	{wlan_0}, {wlan_1}           – WLANConfiguration instance per band (TR-098)
type SchemaMapper struct {
	// params is the merged flat map of parameter-name → path template.
	// Vendor-specific entries override generic model entries when both are present.
	params map[string]string

	// instanceMap carries the runtime-discovered instance indices.
	instanceMap datamodel.InstanceMap

	// modelType is used by DetectModel and ExtractDeviceInfo.
	modelType datamodel.ModelType
}

// NewSchemaMapper builds a SchemaMapper for the given device.
//
// schemaName is the resolved schema identifier (e.g. "tr181",
// "vendor/huawei/tr181"). When a vendor-specific schema is requested the
// generic base schema is loaded first so that operations not overridden by the
// vendor still work.
//
// Returns nil when neither the vendor schema nor its base model schema is
// registered; the caller should fall back to the standard mappers in that case.
func NewSchemaMapper(reg *Registry, schemaName string, im datamodel.InstanceMap) *SchemaMapper {
	baseModel := baseModelName(schemaName) // "tr181" or "tr098"

	if !reg.Has(baseModel) && !reg.Has(schemaName) {
		return nil
	}

	// Start with the generic model parameters, then overlay vendor overrides.
	merged := make(map[string]string)
	for k, v := range reg.ParamMap(baseModel) {
		merged[k] = v
	}
	if schemaName != baseModel {
		for k, v := range reg.ParamMap(schemaName) {
			merged[k] = v
		}
	}

	mt := datamodel.TR181
	if strings.Contains(baseModel, "tr098") {
		mt = datamodel.TR098
	}

	return &SchemaMapper{
		params:      merged,
		instanceMap: im,
		modelType:   mt,
	}
}

// baseModelName extracts the data-model portion from a schema name.
//
//	"tr181"                → "tr181"
//	"tr098"                → "tr098"
//	"vendor/huawei/tr181"  → "tr181"
//	"vendor/zte/tr098"     → "tr098"
func baseModelName(schemaName string) string {
	parts := strings.Split(schemaName, "/")
	last := parts[len(parts)-1]
	return last
}

// path resolution

// resolvePath looks up name in the parameter map and substitutes all
// {placeholder} tokens using the stored instance map.
// Returns "" when the name is not registered or the schema defines an empty path.
func (m *SchemaMapper) resolvePath(name string) string {
	tpl, ok := m.params[name]
	if !ok || tpl == "" {
		return ""
	}
	return m.substitute(tpl)
}

// substitute replaces every known {placeholder} in tpl with its integer value.
func (m *SchemaMapper) substitute(tpl string) string {
	im := m.instanceMap

	r := strings.NewReplacer(
		"{ssid_0}", safeIdx(im.WiFiSSIDIndices, 0, 1),
		"{ssid_1}", safeIdx(im.WiFiSSIDIndices, 1, 2),
		"{radio_0}", safeIdx(im.WiFiRadioIndices, 0, 1),
		"{radio_1}", safeIdx(im.WiFiRadioIndices, 1, 2),
		"{ap_0}", safeIdx(im.WiFiAPIndices, 0, 1),
		"{ap_1}", safeIdx(im.WiFiAPIndices, 1, 2),
		"{wan}", safeInt(im.WANIPIfaceIdx, 1),
		"{lan}", safeInt(im.LANIPIfaceIdx, 2),
		"{ppp}", safeInt(im.PPPIfaceIdx, 1),
		"{wan_dev}", safeInt(im.WANDeviceIdx, 1),
		"{wan_conn}", safeInt(im.WANConnDevIdx, 1),
		"{wan_ip_conn}", safeInt(im.WANIPConnIdx, 1),
		"{wan_ppp_conn}", safeInt(im.WANPPPConnIdx, 1),
		"{lan_dev}", safeInt(im.LANDeviceIdx, 1),
		"{wlan_0}", safeIdx(im.WLANIndices, 0, 1),
		"{wlan_1}", safeIdx(im.WLANIndices, 1, 2),
	)
	return r.Replace(tpl)
}

// safeIdx returns slice[idx] as a string when in bounds, otherwise the
// stringified fallback value.
func safeIdx(slice []int, idx, fallback int) string {
	if idx < len(slice) && slice[idx] > 0 {
		return strconv.Itoa(slice[idx])
	}
	return strconv.Itoa(fallback)
}

// safeInt returns v as a string when v > 0, otherwise the stringified fallback.
func safeInt(v, fallback int) string {
	if v > 0 {
		return strconv.Itoa(v)
	}
	return strconv.Itoa(fallback)
}

// datamodel.Mapper interface

func (m *SchemaMapper) DetectModel(params map[string]string) datamodel.ModelType {
	for k := range params {
		if strings.HasPrefix(k, "Device.") {
			return datamodel.TR181
		}
	}
	return datamodel.TR098
}

func (m *SchemaMapper) ExtractDeviceInfo(params map[string]string) *datamodel.UnifiedDevice {
	// Delegate to the standard mapper for this read-only, informational method.
	return datamodel.NewMapper(m.modelType).ExtractDeviceInfo(params)
}

// WiFi

func (m *SchemaMapper) WiFiSSIDPath(bandIdx int) string {
	return m.resolvePath(fmt.Sprintf("wifi.ssid.%d", bandIdx))
}
func (m *SchemaMapper) WiFiPasswordPath(bandIdx int) string {
	return m.resolvePath(fmt.Sprintf("wifi.password.%d", bandIdx))
}
func (m *SchemaMapper) WiFiEnabledPath(bandIdx int) string {
	return m.resolvePath(fmt.Sprintf("wifi.enabled.%d", bandIdx))
}
func (m *SchemaMapper) WiFiChannelPath(bandIdx int) string {
	return m.resolvePath(fmt.Sprintf("wifi.channel.%d", bandIdx))
}
func (m *SchemaMapper) WiFiBSSIDPath(bandIdx int) string {
	return m.resolvePath(fmt.Sprintf("wifi.bssid.%d", bandIdx))
}
func (m *SchemaMapper) WiFiStandardPath(bandIdx int) string {
	return m.resolvePath(fmt.Sprintf("wifi.standard.%d", bandIdx))
}
func (m *SchemaMapper) WiFiSecurityModePath(bandIdx int) string {
	return m.resolvePath(fmt.Sprintf("wifi.security_mode.%d", bandIdx))
}
func (m *SchemaMapper) WiFiChannelWidthPath(bandIdx int) string {
	return m.resolvePath(fmt.Sprintf("wifi.channel_width.%d", bandIdx))
}
func (m *SchemaMapper) WiFiTXPowerPath(bandIdx int) string {
	return m.resolvePath(fmt.Sprintf("wifi.tx_power.%d", bandIdx))
}
func (m *SchemaMapper) WiFiClientCountPath(bandIdx int) string {
	return m.resolvePath(fmt.Sprintf("wifi.client_count.%d", bandIdx))
}
func (m *SchemaMapper) WiFiBytesSentPath(bandIdx int) string {
	return m.resolvePath(fmt.Sprintf("wifi.stats.bytes_sent.%d", bandIdx))
}
func (m *SchemaMapper) WiFiBytesReceivedPath(bandIdx int) string {
	return m.resolvePath(fmt.Sprintf("wifi.stats.bytes_received.%d", bandIdx))
}
func (m *SchemaMapper) WiFiPacketsSentPath(bandIdx int) string {
	return m.resolvePath(fmt.Sprintf("wifi.stats.packets_sent.%d", bandIdx))
}
func (m *SchemaMapper) WiFiPacketsReceivedPath(bandIdx int) string {
	return m.resolvePath(fmt.Sprintf("wifi.stats.packets_received.%d", bandIdx))
}
func (m *SchemaMapper) WiFiErrorsSentPath(bandIdx int) string {
	return m.resolvePath(fmt.Sprintf("wifi.stats.errors_sent.%d", bandIdx))
}
func (m *SchemaMapper) WiFiErrorsReceivedPath(bandIdx int) string {
	return m.resolvePath(fmt.Sprintf("wifi.stats.errors_received.%d", bandIdx))
}

// WAN

func (m *SchemaMapper) WANConnectionTypePath() string { return m.resolvePath("wan.connection_type") }
func (m *SchemaMapper) WANPPPoEUserPath() string      { return m.resolvePath("wan.pppoe_user") }
func (m *SchemaMapper) WANPPPoEPassPath() string      { return m.resolvePath("wan.pppoe_pass") }
func (m *SchemaMapper) WANIPAddressPath() string      { return m.resolvePath("wan.ip_address") }
func (m *SchemaMapper) WANGatewayPath() string        { return m.resolvePath("wan.gateway") }
func (m *SchemaMapper) WANDNS1Path() string           { return m.resolvePath("wan.dns1") }
func (m *SchemaMapper) WANDNS2Path() string           { return m.resolvePath("wan.dns2") }
func (m *SchemaMapper) WANMTUPath() string            { return m.resolvePath("wan.mtu") }
func (m *SchemaMapper) WANUptimePath() string         { return m.resolvePath("wan.uptime") }
func (m *SchemaMapper) WANMACPath() string            { return m.resolvePath("wan.mac") }
func (m *SchemaMapper) WANStatusPath() string         { return m.resolvePath("wan.status") }
func (m *SchemaMapper) WANBytesSentPath() string      { return m.resolvePath("wan.stats.bytes_sent") }
func (m *SchemaMapper) WANBytesReceivedPath() string {
	return m.resolvePath("wan.stats.bytes_received")
}
func (m *SchemaMapper) WANPacketsSentPath() string { return m.resolvePath("wan.stats.packets_sent") }
func (m *SchemaMapper) WANPacketsReceivedPath() string {
	return m.resolvePath("wan.stats.packets_received")
}
func (m *SchemaMapper) WANErrorsSentPath() string { return m.resolvePath("wan.stats.errors_sent") }
func (m *SchemaMapper) WANErrorsReceivedPath() string {
	return m.resolvePath("wan.stats.errors_received")
}

// LAN / DHCP

func (m *SchemaMapper) LANIPAddressPath() string     { return m.resolvePath("lan.ip_address") }
func (m *SchemaMapper) LANSubnetMaskPath() string    { return m.resolvePath("lan.subnet_mask") }
func (m *SchemaMapper) DHCPServerEnablePath() string { return m.resolvePath("dhcp.server_enable") }
func (m *SchemaMapper) DHCPMinAddressPath() string   { return m.resolvePath("dhcp.min_address") }
func (m *SchemaMapper) DHCPMaxAddressPath() string   { return m.resolvePath("dhcp.max_address") }
func (m *SchemaMapper) LANDNSPath() string           { return m.resolvePath("lan.dns") }

// System

func (m *SchemaMapper) CPEUptimePath() string { return m.resolvePath("system.uptime") }
func (m *SchemaMapper) RAMTotalPath() string  { return m.resolvePath("system.ram_total") }
func (m *SchemaMapper) RAMFreePath() string   { return m.resolvePath("system.ram_free") }

// Management server

func (m *SchemaMapper) ManagementServerURLPath() string {
	return m.resolvePath("mgmt.url")
}
func (m *SchemaMapper) ManagementServerUserPath() string {
	return m.resolvePath("mgmt.user")
}
func (m *SchemaMapper) ManagementServerPassPath() string {
	return m.resolvePath("mgmt.pass")
}
func (m *SchemaMapper) ManagementServerInformIntervalPath() string {
	return m.resolvePath("mgmt.interval")
}

// Connected hosts

func (m *SchemaMapper) HostsBasePath() string  { return m.resolvePath("hosts.base") }
func (m *SchemaMapper) HostsCountPath() string { return m.resolvePath("hosts.count") }

// Diagnostics

func (m *SchemaMapper) PingDiagBasePath() string       { return m.resolvePath("diag.ping.base") }
func (m *SchemaMapper) TracerouteDiagBasePath() string { return m.resolvePath("diag.traceroute.base") }
func (m *SchemaMapper) DownloadDiagBasePath() string   { return m.resolvePath("diag.download.base") }
func (m *SchemaMapper) UploadDiagBasePath() string     { return m.resolvePath("diag.upload.base") }

// Port forwarding

func (m *SchemaMapper) PortMappingBasePath() string  { return m.resolvePath("portmapping.base") }
func (m *SchemaMapper) PortMappingCountPath() string { return m.resolvePath("portmapping.count") }

// Web admin

func (m *SchemaMapper) WebAdminPasswordPath() string { return m.resolvePath("admin.password") }
