package task

import (
	"context"
	"encoding/json"
	"time"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusExecuting Status = "executing"
	StatusDone      Status = "done"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

type Type string

const (
	// Configuration tasks
	TypeWifi         Type = "wifi"
	TypeWAN          Type = "wan"
	TypeLAN          Type = "lan"
	TypeReboot       Type = "reboot"
	TypeFactoryReset Type = "factory_reset"
	TypeSetParams    Type = "set_parameters"
	TypeGetParams    Type = "get_parameters"
	TypeFirmware     Type = "firmware"

	// Diagnostic tasks
	TypeDiagnostic Type = "diagnostic"
	TypePingTest   Type = "ping_test"
	TypeTraceroute Type = "traceroute"
	TypeSpeedTest  Type = "speed_test"

	// Informational tasks
	TypeConnectedDevices Type = "connected_devices"
	TypeCPEStats         Type = "cpe_stats"

	// Port-forwarding management
	TypePortForwarding Type = "port_forwarding"

	// Web admin interface password change
	TypeWebAdmin Type = "web_admin"

	// Internal synthetic type NEVER stored in Redis.
	// Created in-memory to collect async diagnostic results.
	TypeGetDiagResult Type = "_get_diag_result"
)

// IsDiagnosticAsync returns true for tasks that complete asynchronously via the
// "8 DIAGNOSTICS COMPLETE" Inform event rather than immediately.
func IsDiagnosticAsync(t Type) bool {
	switch t {
	case TypePingTest, TypeTraceroute, TypeSpeedTest, TypeDiagnostic:
		return true
	}
	return false
}

type Task struct {
	ID          string          `json:"id"`
	Serial      string          `json:"serial"`
	Type        Type            `json:"type"`
	Payload     json.RawMessage `json:"payload"`
	Status      Status          `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
	ExecutedAt  *time.Time      `json:"executed_at,omitempty"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	Result      any             `json:"result,omitempty"`
	Error       string          `json:"error,omitempty"`
	Attempts    int             `json:"attempts"`
	MaxAttempts int             `json:"max_attempts"`
}

// Payload types

type WiFiPayload struct {
	Band     string `json:"band"` // "2.4" or "5"
	SSID     string `json:"ssid"`
	Password string `json:"password"`
	Channel  int    `json:"channel,omitempty"`
	Enabled  *bool  `json:"enabled,omitempty"`
}

type WANPayload struct {
	ConnectionType string `json:"connection_type"` // "pppoe", "dhcp", "static"
	Username       string `json:"username,omitempty"`
	Password       string `json:"password,omitempty"`
	IPAddress      string `json:"ip_address,omitempty"`
	SubnetMask     string `json:"subnet_mask,omitempty"`
	Gateway        string `json:"gateway,omitempty"`
	DNS1           string `json:"dns1,omitempty"`
	DNS2           string `json:"dns2,omitempty"`
	VLAN           int    `json:"vlan,omitempty"`
	MTU            int    `json:"mtu,omitempty"`
}

type LANPayload struct {
	DHCPEnabled bool   `json:"dhcp_enabled"`
	IPAddress   string `json:"ip_address,omitempty"`
	SubnetMask  string `json:"subnet_mask,omitempty"`
	DHCPStart   string `json:"dhcp_start,omitempty"`
	DHCPEnd     string `json:"dhcp_end,omitempty"`
	DNSServer   string `json:"dns_server,omitempty"`
	LeaseTime   int    `json:"lease_time,omitempty"`
}

type SetParamsPayload struct {
	Parameters map[string]string `json:"parameters"`
}

type GetParamsPayload struct {
	Parameters []string `json:"parameters"`
}

type FirmwarePayload struct {
	URL      string `json:"url"`
	Version  string `json:"version,omitempty"`
	FileType string `json:"file_type,omitempty"` // "1 Firmware Upgrade Image"
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// DiagnosticPayload is the legacy payload
type DiagnosticPayload struct {
	DiagType string `json:"diag_type"` // "ping" or "traceroute"
	Target   string `json:"target"`
	Count    int    `json:"count,omitempty"`
}

// PingTestPayload configures an IPPingDiagnostics test.
type PingTestPayload struct {
	Host       string `json:"host"`
	Count      int    `json:"count,omitempty"`       // default 4
	PacketSize int    `json:"packet_size,omitempty"` // bytes, default 64
	Timeout    int    `json:"timeout,omitempty"`     // ms per ping, default 1000
	DSCP       int    `json:"dscp,omitempty"`
}

// TraceroutePayload configures a TraceRouteDiagnostics test.
type TraceroutePayload struct {
	Host       string `json:"host"`
	MaxHops    int    `json:"max_hops,omitempty"`    // default 30
	Timeout    int    `json:"timeout,omitempty"`     // ms per hop, default 5000
	PacketSize int    `json:"packet_size,omitempty"` // default 38
	DSCP       int    `json:"dscp,omitempty"`
}

// SpeedTestPayload configures DownloadDiagnostics
type SpeedTestPayload struct {
	DownloadURL string `json:"download_url"`
	UploadURL   string `json:"upload_url,omitempty"`

	// FileSize instructs the CPE on the test object size in bytes (download).
	// Leave 0 to let the CPE choose.
	FileSize int `json:"file_size,omitempty"`

	// EthernetPriority / DSCP for test traffic.
	DSCP int `json:"dscp,omitempty"`
}

// ConnectedDevicesPayload has no configuration  fetches all hosts as-is.
type ConnectedDevicesPayload struct{}

// CPEStatsPayload has no configuration  fetches all WAN/LAN/WiFi counters.
type CPEStatsPayload struct{}

// PortForwardingAction discriminates the three port-forwarding operations.
type PortForwardingAction string

const (
	PortForwardingAdd    PortForwardingAction = "add"
	PortForwardingRemove PortForwardingAction = "remove"
	PortForwardingList   PortForwardingAction = "list"
)

// PortForwardingPayload carries the parameters for port-forwarding tasks.
type PortForwardingPayload struct {
	Action         PortForwardingAction `json:"action"`
	Protocol       string               `json:"protocol,omitempty"` // "TCP", "UDP", "TCP/UDP"
	ExternalPort   int                  `json:"external_port,omitempty"`
	InternalIP     string               `json:"internal_ip,omitempty"`
	InternalPort   int                  `json:"internal_port,omitempty"`
	Description    string               `json:"description,omitempty"`
	Enabled        *bool                `json:"enabled,omitempty"`
	InstanceNumber int                  `json:"instance_number,omitempty"` // for remove
}

// GetDiagResultPayload is the payload for the synthetic TypeGetDiagResult task
// used to collect async diagnostic results in a follow-up CWMP round-trip.
type GetDiagResultPayload struct {
	OriginalTaskID   string   `json:"original_task_id"`
	OriginalTaskType Type     `json:"original_task_type"`
	Paths            []string `json:"paths"`
}

// Result types

// PingResult is stored in Task.Result for TypePingTest.
type PingResult struct {
	Host            string  `json:"host"`
	PacketsSent     int     `json:"packets_sent"`
	PacketsReceived int     `json:"packets_received"`
	PacketLossPct   float64 `json:"packet_loss_pct"`
	MinRTTMs        int     `json:"min_rtt_ms"`
	AvgRTTMs        int     `json:"avg_rtt_ms"`
	MaxRTTMs        int     `json:"max_rtt_ms"`
}

// TracerouteHop is one hop in a traceroute result.
type TracerouteHop struct {
	HopNumber int    `json:"hop"`
	Host      string `json:"host"`
	RTTMs     int    `json:"rtt_ms"`
}

// TracerouteResult is stored in Task.Result for TypeTraceroute.
type TracerouteResult struct {
	Host     string          `json:"host"`
	MaxHops  int             `json:"max_hops"`
	HopCount int             `json:"hop_count"`
	Hops     []TracerouteHop `json:"hops"`
}

// SpeedTestResult is stored in Task.Result for TypeSpeedTest.
type SpeedTestResult struct {
	DownloadURL        string  `json:"download_url"`
	DownloadSpeedMbps  float64 `json:"download_speed_mbps"`
	DownloadDurationMs int     `json:"download_duration_ms"`
	DownloadBytesTotal int64   `json:"download_bytes_total"`
	UploadURL          string  `json:"upload_url,omitempty"`
	UploadSpeedMbps    float64 `json:"upload_speed_mbps,omitempty"`
	UploadDurationMs   int     `json:"upload_duration_ms,omitempty"`
	UploadBytesTotal   int64   `json:"upload_bytes_total,omitempty"`
}

// CPEStatsResult is stored in Task.Result for TypeCPEStats.
type CPEStatsResult struct {
	UptimeSeconds int64 `json:"uptime_seconds"`
	RAMTotalKB    int64 `json:"ram_total_kb"`
	RAMFreeKB     int64 `json:"ram_free_kb"`
	WANBytesSent  int64 `json:"wan_bytes_sent"`
	WANBytesRecv  int64 `json:"wan_bytes_recv"`
	WANPktsSent   int64 `json:"wan_pkts_sent"`
	WANPktsRecv   int64 `json:"wan_pkts_recv"`
	WANErrsSent   int64 `json:"wan_errs_sent"`
	WANErrsRecv   int64 `json:"wan_errs_recv"`
}

// WebAdminPayload carries the new password for the CPE web admin interface.
type WebAdminPayload struct {
	Password string `json:"password"`
}

// PortForwardingRule is one entry in the CPE port-mapping table.
type PortForwardingRule struct {
	InstanceNumber int    `json:"instance"`
	Enabled        bool   `json:"enabled"`
	Protocol       string `json:"protocol"`
	ExternalPort   int    `json:"external_port"`
	InternalIP     string `json:"internal_ip"`
	InternalPort   int    `json:"internal_port"`
	Description    string `json:"description"`
}

// Queue interface

type Queue interface {
	Enqueue(ctx context.Context, task *Task) error
	DequeuePending(ctx context.Context, serial string) ([]*Task, error)
	UpdateStatus(ctx context.Context, task *Task) error
	Cancel(ctx context.Context, taskID, serial string) error
	GetByID(ctx context.Context, taskID string) (*Task, error)
	List(ctx context.Context, serial string, page, limit int) ([]*Task, int64, error)
	// FindExecutingDiagnostics returns all executing async-diagnostic tasks for
	// the given device (PingTest, Traceroute, SpeedTest, Diagnostic).
	FindExecutingDiagnostics(ctx context.Context, serial string) ([]*Task, error)
}
