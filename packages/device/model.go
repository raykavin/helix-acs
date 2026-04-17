package device

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// WANInfo holds the current state of the WAN interface.
type WANInfo struct {
	ConnectionType string `bson:"connection_type" json:"connection_type"`
	IPAddress      string `bson:"ip_address"      json:"ip_address"`
	SubnetMask     string `bson:"subnet_mask"     json:"subnet_mask"`
	Gateway        string `bson:"gateway"         json:"gateway"`
	DNS1           string `bson:"dns1"            json:"dns1"`
	DNS2           string `bson:"dns2"            json:"dns2"`
	MACAddress     string `bson:"mac_address"     json:"mac_address"`
	PPPoEUsername  string `bson:"pppoe_username"  json:"pppoe_username"`
	MTU            int    `bson:"mtu"             json:"mtu"`
	LinkStatus     string `bson:"link_status"     json:"link_status"`
	UptimeSeconds  int64  `bson:"uptime_seconds"  json:"uptime_seconds"`

	// Traffic counters
	BytesSent       int64 `bson:"bytes_sent"        json:"bytes_sent"`
	BytesReceived   int64 `bson:"bytes_received"    json:"bytes_received"`
	PacketsSent     int64 `bson:"packets_sent"      json:"packets_sent"`
	PacketsReceived int64 `bson:"packets_received"  json:"packets_received"`
	ErrorsSent      int64 `bson:"errors_sent"       json:"errors_sent"`
	ErrorsReceived  int64 `bson:"errors_received"   json:"errors_received"`
}

// DHCPLease represents a single active DHCP lease.
type DHCPLease struct {
	MACAddress string    `bson:"mac"        json:"mac"`
	IPAddress  string    `bson:"ip"         json:"ip"`
	Hostname   string    `bson:"hostname"   json:"hostname"`
	ExpireTime time.Time `bson:"expire_time" json:"expire_time,omitzero"`
}

// LANInfo holds LAN/DHCP configuration and active leases.
type LANInfo struct {
	IPAddress    string      `bson:"ip_address"    json:"ip_address"`
	SubnetMask   string      `bson:"subnet_mask"   json:"subnet_mask"`
	DHCPEnabled  bool        `bson:"dhcp_enabled"  json:"dhcp_enabled"`
	DHCPStart    string      `bson:"dhcp_start"    json:"dhcp_start"`
	DHCPEnd      string      `bson:"dhcp_end"      json:"dhcp_end"`
	DNSServers   string      `bson:"dns_servers"   json:"dns_servers"`
	ActiveLeases int         `bson:"active_leases" json:"active_leases"`
	Leases       []DHCPLease `bson:"leases"        json:"leases,omitempty"`
}

// WiFiInfo holds the current state of a single wireless radio / SSID.
type WiFiInfo struct {
	Band             string `bson:"band"              json:"band"` // "2.4GHz" or "5GHz"
	SSID             string `bson:"ssid"              json:"ssid"`
	Enabled          bool   `bson:"enabled"           json:"enabled"`
	BSSID            string `bson:"bssid"             json:"bssid"`
	Channel          int    `bson:"channel"           json:"channel"`
	ChannelWidth     string `bson:"channel_width"     json:"channel_width"`
	Standard         string `bson:"standard"          json:"standard"`
	SecurityMode     string `bson:"security_mode"     json:"security_mode"`
	TXPower          int    `bson:"tx_power"          json:"tx_power"`
	ConnectedClients int    `bson:"connected_clients" json:"connected_clients"`

	// Traffic counters
	BytesSent       int64 `bson:"bytes_sent"        json:"bytes_sent"`
	BytesReceived   int64 `bson:"bytes_received"    json:"bytes_received"`
	PacketsSent     int64 `bson:"packets_sent"      json:"packets_sent"`
	PacketsReceived int64 `bson:"packets_received"  json:"packets_received"`
	ErrorsSent      int64 `bson:"errors_sent"       json:"errors_sent"`
	ErrorsReceived  int64 `bson:"errors_received"   json:"errors_received"`
}

// ConnectedHost represents a host currently connected to the CPE.
type ConnectedHost struct {
	MACAddress string `bson:"mac"       json:"mac"`
	IPAddress  string `bson:"ip"        json:"ip"`
	Hostname   string `bson:"hostname"  json:"hostname"`
	Interface  string `bson:"interface" json:"interface"` // "LAN", "2.4GHz", "5GHz"
	Active     bool   `bson:"active"    json:"active"`
	LeaseTime  int    `bson:"lease_time" json:"lease_time"` // remaining seconds
}

// Device

type Device struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Serial       string             `bson:"serial"        json:"serial"`
	OUI          string             `bson:"oui"           json:"oui"`
	Manufacturer string             `bson:"manufacturer"  json:"manufacturer"`
	ModelName    string             `bson:"model_name"    json:"model_name"`
	ProductClass string             `bson:"product_class" json:"product_class"`
	DataModel    string             `bson:"data_model"    json:"data_model"` // "tr181" or "tr098"
	Schema       string             `bson:"schema"        json:"schema"`      // resolved schema name, e.g. "tr181" or "vendor/huawei/tr181"
	Online       bool               `bson:"online"        json:"online"`
	LastInform   time.Time          `bson:"last_inform"   json:"last_inform"`
	IPAddress    string             `bson:"ip_address"    json:"ip_address"`
	WANIP        string             `bson:"wan_ip"        json:"wan_ip"`
	SWVersion    string             `bson:"sw_version"    json:"sw_version"`
	HWVersion    string             `bson:"hw_version"    json:"hw_version"`
	BLVersion    string             `bson:"bl_version"    json:"bl_version"`

	// System info
	UptimeSeconds int64  `bson:"uptime_seconds" json:"uptime_seconds,omitempty"`
	RAMTotal      int64  `bson:"ram_total"      json:"ram_total,omitempty"`
	RAMFree       int64  `bson:"ram_free"       json:"ram_free,omitempty"`
	ACSURL        string `bson:"acs_url"        json:"acs_url,omitempty"`

	// Rich sub-documents
	WAN            *WANInfo        `bson:"wan"             json:"wan,omitempty"`
	LAN            *LANInfo        `bson:"lan"             json:"lan,omitempty"`
	WiFi24         *WiFiInfo       `bson:"wifi_24"         json:"wifi_24,omitempty"`
	WiFi5          *WiFiInfo       `bson:"wifi_5"          json:"wifi_5,omitempty"`
	ConnectedHosts []ConnectedHost `bson:"connected_hosts" json:"connected_hosts,omitempty"`

	// Raw parameter map and metadata
	Parameters map[string]string `bson:"parameters" json:"parameters"`
	Tags       []string          `bson:"tags"       json:"tags"`
	CreatedAt  time.Time         `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time         `bson:"updated_at" json:"updated_at"`
}

// Filter

type DeviceFilter struct {
	Online       *bool
	Manufacturer string
	ModelName    string
	Tag          string
	WANIP        string
	Serial       string
}

type UpsertRequest struct {
	Serial        string
	OUI           string
	Manufacturer  string
	ModelName     string
	ProductClass  string
	DataModel     string
	Schema        string // resolved schema name, e.g. "tr181" or "vendor/huawei/tr181"
	IPAddress     string
	WANIP         string
	SWVersion     string
	HWVersion     string
	BLVersion     string
	UptimeSeconds int64
	RAMTotal      int64
	RAMFree       int64
	ACSURL        string
	Parameters    map[string]string
}

type UpdateRequest struct {
	Tags []string       `json:"tags"`
	Meta map[string]any `json:"meta,omitempty"`
}

// InfoUpdate carries optional rich sub-documents to be merged into a device.
type InfoUpdate struct {
	WAN            *WANInfo
	LAN            *LANInfo
	WiFi24         *WiFiInfo
	WiFi5          *WiFiInfo
	ConnectedHosts []ConnectedHost
}

// Interfaces

type Repository interface {
	Upsert(ctx context.Context, req *UpsertRequest) (*Device, error)
	FindBySerial(ctx context.Context, serial string) (*Device, error)
	Find(ctx context.Context, filter DeviceFilter, skip, limit int64) ([]*Device, int64, error)
	UpdateTags(ctx context.Context, serial string, tags []string) error
	Delete(ctx context.Context, serial string) error
	SetOnline(ctx context.Context, serial string, online bool) error
	UpdateParameters(ctx context.Context, serial string, params map[string]string) error
	UpdateInfo(ctx context.Context, serial string, upd InfoUpdate) error
}

type Service interface {
	UpsertFromInform(ctx context.Context, req *UpsertRequest) (*Device, error)
	FindBySerial(ctx context.Context, serial string) (*Device, error)
	List(ctx context.Context, filter DeviceFilter, page, limit int) ([]*Device, int64, error)
	UpdateTags(ctx context.Context, serial string, tags []string) (*Device, error)
	Delete(ctx context.Context, serial string) error
	SetOnline(ctx context.Context, serial string, online bool) error
	UpdateInfo(ctx context.Context, serial string, upd InfoUpdate) error
}
