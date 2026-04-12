package task

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strconv"

	"github.com/raykavin/helix-acs/internal/datamodel"
)

// Executor converts Task payloads into the parameter maps / name lists that
// the CWMP session handler needs to build SetParameterValues and
// GetParameterValues requests.
type Executor struct{}

// NewExecutor returns a ready-to-use Executor.
func NewExecutor() *Executor { return &Executor{} }

// BuildSetParams converts a task into a map of TR-069 parameter path → value
// suitable for a SetParameterValues RPC. Returns (nil, nil) for task types
// that do not use SetParameterValues.
func (e *Executor) BuildSetParams(ctx context.Context, t *Task, mapper datamodel.Mapper) (map[string]string, error) {
	_ = ctx
	switch t.Type {

	// WiFi
	case TypeWifi:
		var p WiFiPayload
		if err := json.Unmarshal(t.Payload, &p); err != nil {
			return nil, fmt.Errorf("unmarshal wifi payload: %w", err)
		}
		bandIdx := 0
		if p.Band == "5" {
			bandIdx = 1
		}
		params := make(map[string]string)
		if p.SSID != "" {
			params[mapper.WiFiSSIDPath(bandIdx)] = p.SSID
		}
		if p.Password != "" {
			params[mapper.WiFiPasswordPath(bandIdx)] = p.Password
		}
		if p.Enabled != nil {
			params[mapper.WiFiEnabledPath(bandIdx)] = strconv.FormatBool(*p.Enabled)
		}
		if p.Channel != 0 {
			params[mapper.WiFiChannelPath(bandIdx)] = strconv.Itoa(p.Channel)
		}
		if len(params) == 0 {
			return nil, fmt.Errorf("wifi payload has no settable fields")
		}
		return params, nil

	// WAN
	case TypeWAN:
		var p WANPayload
		if err := json.Unmarshal(t.Payload, &p); err != nil {
			return nil, fmt.Errorf("unmarshal wan payload: %w", err)
		}
		params := make(map[string]string)
		if p.ConnectionType != "" {
			params[mapper.WANConnectionTypePath()] = p.ConnectionType
		}
		if p.Username != "" {
			params[mapper.WANPPPoEUserPath()] = p.Username
		}
		if p.Password != "" {
			params[mapper.WANPPPoEPassPath()] = p.Password
		}
		if p.IPAddress != "" {
			params[mapper.WANIPAddressPath()] = p.IPAddress
		}
		if p.MTU > 0 {
			params[mapper.WANMTUPath()] = strconv.Itoa(p.MTU)
		}
		if len(params) == 0 {
			return nil, fmt.Errorf("wan payload has no settable fields")
		}
		return params, nil

	// LAN
	case TypeLAN:
		var p LANPayload
		if err := json.Unmarshal(t.Payload, &p); err != nil {
			return nil, fmt.Errorf("unmarshal lan payload: %w", err)
		}
		params := make(map[string]string)
		params[mapper.DHCPServerEnablePath()] = strconv.FormatBool(p.DHCPEnabled)
		if p.IPAddress != "" {
			params[mapper.LANIPAddressPath()] = p.IPAddress
		}
		if p.SubnetMask != "" {
			params[mapper.LANSubnetMaskPath()] = p.SubnetMask
		}
		if p.DHCPStart != "" {
			params[mapper.DHCPMinAddressPath()] = p.DHCPStart
		}
		if p.DHCPEnd != "" {
			params[mapper.DHCPMaxAddressPath()] = p.DHCPEnd
		}
		if p.DNSServer != "" {
			params[mapper.LANDNSPath()] = p.DNSServer
		}
		return params, nil

	// SetParams
	case TypeSetParams:
		var p SetParamsPayload
		if err := json.Unmarshal(t.Payload, &p); err != nil {
			return nil, fmt.Errorf("unmarshal set_parameters payload: %w", err)
		}
		if len(p.Parameters) == 0 {
			return nil, fmt.Errorf("set_parameters payload has no parameters")
		}
		out := make(map[string]string, len(p.Parameters))
		maps.Copy(out, p.Parameters)
		return out, nil

	// PingTest
	case TypePingTest:
		var p PingTestPayload
		if err := json.Unmarshal(t.Payload, &p); err != nil {
			return nil, fmt.Errorf("unmarshal ping_test payload: %w", err)
		}
		base := mapper.PingDiagBasePath()
		params := map[string]string{
			base + "Host":             p.Host,
			base + "DiagnosticsState": "Requested",
		}
		if p.Count > 0 {
			params[base+"NumberOfRepetitions"] = strconv.Itoa(p.Count)
		} else {
			params[base+"NumberOfRepetitions"] = "4"
		}
		if p.PacketSize > 0 {
			params[base+"DataBlockSize"] = strconv.Itoa(p.PacketSize)
		}
		if p.Timeout > 0 {
			params[base+"Timeout"] = strconv.Itoa(p.Timeout)
		}
		if p.DSCP > 0 {
			params[base+"DSCP"] = strconv.Itoa(p.DSCP)
		}
		return params, nil

	// Traceroute
	case TypeTraceroute:
		var p TraceroutePayload
		if err := json.Unmarshal(t.Payload, &p); err != nil {
			return nil, fmt.Errorf("unmarshal traceroute payload: %w", err)
		}
		base := mapper.TracerouteDiagBasePath()
		params := map[string]string{
			base + "Host":             p.Host,
			base + "DiagnosticsState": "Requested",
		}
		if p.MaxHops > 0 {
			params[base+"MaxHopCount"] = strconv.Itoa(p.MaxHops)
		} else {
			params[base+"MaxHopCount"] = "30"
		}
		if p.Timeout > 0 {
			params[base+"Timeout"] = strconv.Itoa(p.Timeout)
		}
		if p.PacketSize > 0 {
			params[base+"DataBlockSize"] = strconv.Itoa(p.PacketSize)
		}
		if p.DSCP > 0 {
			params[base+"DSCP"] = strconv.Itoa(p.DSCP)
		}
		return params, nil

	// SpeedTest (download diagnostic)
	case TypeSpeedTest:
		var p SpeedTestPayload
		if err := json.Unmarshal(t.Payload, &p); err != nil {
			return nil, fmt.Errorf("unmarshal speed_test payload: %w", err)
		}
		if p.DownloadURL == "" {
			return nil, fmt.Errorf("speed_test payload requires download_url")
		}
		base := mapper.DownloadDiagBasePath()
		params := map[string]string{
			base + "DownloadURL":      p.DownloadURL,
			base + "DiagnosticsState": "Requested",
		}
		if p.FileSize > 0 {
			params[base+"TestFileLength"] = strconv.Itoa(p.FileSize)
		}
		if p.DSCP > 0 {
			params[base+"DSCP"] = strconv.Itoa(p.DSCP)
		}
		return params, nil

	// WebAdmin  change local web interface password
	case TypeWebAdmin:
		var p WebAdminPayload
		if err := json.Unmarshal(t.Payload, &p); err != nil {
			return nil, fmt.Errorf("unmarshal web_admin payload: %w", err)
		}
		if p.Password == "" {
			return nil, fmt.Errorf("web_admin payload requires password")
		}
		path := mapper.WebAdminPasswordPath()
		if path == "" {
			return nil, fmt.Errorf("web admin password path not supported by this device's data-model (TR-098); use set_parameters with the vendor-specific path")
		}
		return map[string]string{path: p.Password}, nil

	// Legacy Diagnostic
	// Returns nil so session.go falls back to the raw DiagnosticsState path.
	case TypeDiagnostic:
		return nil, nil

	default:
		return nil, nil
	}
}

// BuildGetParams returns the list of TR-069 parameter paths to request in a
// GetParameterValues RPC.
func (e *Executor) BuildGetParams(ctx context.Context, t *Task, mapper datamodel.Mapper) ([]string, error) {
	_ = ctx

	switch t.Type {
	case TypeGetParams:
		var p GetParamsPayload
		if err := json.Unmarshal(t.Payload, &p); err != nil {
			return nil, fmt.Errorf("unmarshal get_parameters payload: %w", err)
		}
		if len(p.Parameters) == 0 {
			return nil, fmt.Errorf("get_parameters payload has no parameter names")
		}
		names := make([]string, len(p.Parameters))
		copy(names, p.Parameters)
		return names, nil

	case TypeConnectedDevices:
		// Fetch the entire Hosts sub-tree; the response will contain all
		// Host.{i}.* parameters which we parse into ConnectedHost structs.
		return []string{mapper.HostsBasePath()}, nil

	case TypeCPEStats:
		return buildCPEStatsPaths(mapper), nil

	case TypePortForwarding:
		var p PortForwardingPayload
		if err := json.Unmarshal(t.Payload, &p); err != nil {
			return nil, fmt.Errorf("unmarshal port_forwarding payload: %w", err)
		}
		if p.Action == PortForwardingList {
			return []string{mapper.PortMappingBasePath()}, nil
		}
		return nil, nil

	default:
		return nil, nil
	}
}

// buildCPEStatsPaths returns all parameter paths needed for a CPE stats snapshot.
func buildCPEStatsPaths(mapper datamodel.Mapper) []string {
	return []string{
		mapper.CPEUptimePath(),
		mapper.RAMTotalPath(),
		mapper.RAMFreePath(),
		mapper.WANBytesSentPath(),
		mapper.WANBytesReceivedPath(),
		mapper.WANPacketsSentPath(),
		mapper.WANPacketsReceivedPath(),
		mapper.WANErrorsSentPath(),
		mapper.WANErrorsReceivedPath(),
		mapper.WANStatusPath(),
		mapper.WANUptimePath(),
	}
}

// BuildDiagResultPaths returns the GetParameterValues paths needed to collect
// the results of an async diagnostic after receiving DIAGNOSTICS COMPLETE.
func BuildDiagResultPaths(taskType Type, mapper datamodel.Mapper) []string {
	switch taskType {
	case TypePingTest, TypeDiagnostic:
		base := mapper.PingDiagBasePath()
		return []string{
			base + "DiagnosticsState",
			base + "SuccessCount",
			base + "FailureCount",
			base + "AverageResponseTime",
			base + "MinimumResponseTime",
			base + "MaximumResponseTime",
		}
	case TypeTraceroute:
		base := mapper.TracerouteDiagBasePath()
		return []string{
			base + "DiagnosticsState",
			base + "ResponseTime",
			base + "NumberOfRouteHops",
			base + "RouteHops.",
		}
	case TypeSpeedTest:
		base := mapper.DownloadDiagBasePath()
		return []string{
			base + "DiagnosticsState",
			base + "BOMTime",
			base + "EOMTime",
			base + "TestBytesReceived",
			base + "TotalBytesReceived",
		}
	}
	return nil
}
