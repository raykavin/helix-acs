package datamodel

// Type constants for CWMP parameter types (xsd types used in TR-069 SOAP).
const (
	TypeString      = "xsd:string"
	TypeBoolean     = "xsd:boolean"
	TypeUnsignedInt = "xsd:unsignedInt"
	TypeInt         = "xsd:int"
	TypeDateTime    = "xsd:dateTime"
	TypeBase64      = "xsd:base64Binary"
)

// ModelType identifies which TR data-model namespace a CPE uses.
type ModelType string

const (
	TR181 ModelType = "tr181"
	TR098 ModelType = "tr098"
)

// UnifiedDevice is a normalised representation of a CPE regardless of whether
// it exposes TR-181 or TR-098 parameters.
type UnifiedDevice struct {
	Manufacturer string
	ModelName    string
	Serial       string
	OUI          string
	ProductClass string
	SWVersion    string
	HWVersion    string
	WANIP        string
	LANIP        string
}

// Mapper translates between raw TR-069 parameter paths/values and the
// normalised UnifiedDevice view. All path methods return the canonical
// parameter path for the requested field under the concrete data-model.
type Mapper interface {
	// DetectModel inspects a flat map of parameter names and returns which
	// data-model the CPE is using.
	DetectModel(params map[string]string) ModelType

	// ExtractDeviceInfo builds a UnifiedDevice from the given parameter map.
	ExtractDeviceInfo(params map[string]string) *UnifiedDevice

	// WiFi (per band)
	// bandIdx: 0 = 2.4 GHz, 1 = 5 GHz.

	WiFiSSIDPath(bandIdx int) string
	WiFiPasswordPath(bandIdx int) string
	WiFiEnabledPath(bandIdx int) string
	WiFiChannelPath(bandIdx int) string
	WiFiBSSIDPath(bandIdx int) string
	WiFiStandardPath(bandIdx int) string
	WiFiSecurityModePath(bandIdx int) string
	WiFiChannelWidthPath(bandIdx int) string
	WiFiTXPowerPath(bandIdx int) string
	WiFiClientCountPath(bandIdx int) string
	WiFiBytesSentPath(bandIdx int) string
	WiFiBytesReceivedPath(bandIdx int) string
	WiFiPacketsSentPath(bandIdx int) string
	WiFiPacketsReceivedPath(bandIdx int) string
	WiFiErrorsSentPath(bandIdx int) string
	WiFiErrorsReceivedPath(bandIdx int) string

	// WAN

	WANConnectionTypePath() string
	WANPPPoEUserPath() string
	WANPPPoEPassPath() string
	WANIPAddressPath() string
	WANGatewayPath() string
	WANDNS1Path() string
	WANDNS2Path() string
	WANMTUPath() string
	WANUptimePath() string
	WANMACPath() string
	WANStatusPath() string
	WANBytesSentPath() string
	WANBytesReceivedPath() string
	WANPacketsSentPath() string
	WANPacketsReceivedPath() string
	WANErrorsSentPath() string
	WANErrorsReceivedPath() string

	// LAN / DHCP

	LANIPAddressPath() string
	LANSubnetMaskPath() string
	DHCPServerEnablePath() string
	DHCPMinAddressPath() string
	DHCPMaxAddressPath() string
	LANDNSPath() string

	// System

	CPEUptimePath() string
	RAMTotalPath() string
	RAMFreePath() string

	// ManagementServer

	ManagementServerURLPath() string
	ManagementServerUserPath() string
	ManagementServerPassPath() string
	ManagementServerInformIntervalPath() string

	// Connected hosts

	HostsBasePath() string
	HostsCountPath() string

	// Diagnostics

	PingDiagBasePath() string
	TracerouteDiagBasePath() string
	DownloadDiagBasePath() string
	UploadDiagBasePath() string

	// Port forwarding

	PortMappingBasePath() string
	PortMappingCountPath() string

	// Web admin interface

	// WebAdminPasswordPath returns the TR-069 parameter path used to set the
	// CPE's local web administration password. Returns "" for data-models that
	// have no standard path (e.g. TR-098, which uses vendor-specific extensions).
	WebAdminPasswordPath() string
}

// NewMapper returns the concrete Mapper for the given ModelType. Unknown types
// fall back to the TR-098 mapper because legacy CPEs are far more common.
func NewMapper(modelType ModelType) Mapper {
	switch modelType {
	case TR181:
		return &TR181Mapper{}
	default:
		return &TR098Mapper{}
	}
}

// DetectFromRootObject inspects the root object string reported by a CPE
// (e.g. from a GetParameterNamesResponse) and returns the corresponding
// ModelType.
func DetectFromRootObject(rootObject string) ModelType {
	if rootObject == "Device." || rootObject == "Device" {
		return TR181
	}
	return TR098
}

// ApplyInstanceMap returns a copy of mapper with its index override fields
// populated from im. Only non-zero fields in im overwrite the mapper's current
// values, so already-configured indices are preserved.
//
// If mapper is neither *TR181Mapper nor *TR098Mapper it is returned unchanged.
func ApplyInstanceMap(mapper Mapper, im InstanceMap) Mapper {
	switch m := mapper.(type) {
	case *TR181Mapper:
		c := *m // shallow copy — safe, all fields are scalars or slices
		if im.WANIPIfaceIdx > 0 {
			c.WANIfaceIdx = im.WANIPIfaceIdx
		}
		if im.LANIPIfaceIdx > 0 {
			c.LANIfaceIdx = im.LANIPIfaceIdx
		}
		if im.PPPIfaceIdx > 0 {
			c.PPPIfaceIdx = im.PPPIfaceIdx
		}
		if len(im.WiFiSSIDIndices) > 0 {
			c.SSIDIndices = im.WiFiSSIDIndices
		}
		if len(im.WiFiRadioIndices) > 0 {
			c.RadioIndices = im.WiFiRadioIndices
		}
		if len(im.WiFiAPIndices) > 0 {
			c.APIndices = im.WiFiAPIndices
		}
		return &c

	case *TR098Mapper:
		c := *m
		if im.WANDeviceIdx > 0 {
			c.WANDeviceIdx = im.WANDeviceIdx
		}
		if im.WANConnDevIdx > 0 {
			c.WANConnDevIdx = im.WANConnDevIdx
		}
		if im.WANIPConnIdx > 0 {
			c.WANIPConnIdx = im.WANIPConnIdx
		}
		if im.WANPPPConnIdx > 0 {
			c.WANPPPConnIdx = im.WANPPPConnIdx
		}
		if im.LANDeviceIdx > 0 {
			c.LANDeviceIdx = im.LANDeviceIdx
		}
		if len(im.WLANIndices) > 0 {
			c.WLANIndices = im.WLANIndices
		}
		return &c
	}

	return mapper
}
