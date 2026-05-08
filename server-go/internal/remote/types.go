package remote

import "time"

type TunnelMode string

const (
	TunnelModeRelay    TunnelMode = "智能中继"
	TunnelModeDirect   TunnelMode = "直连优先"
	TunnelModeDisabled TunnelMode = "disabled"
)

type ShareScanState string

const (
	ShareScanIdle ShareScanState = "idle"
	ShareScanSafe ShareScanState = "safe"
	ShareScanRisk ShareScanState = "risk"
)

type RemoteStatus struct {
	ChannelEnabled   bool           `json:"channelEnabled"`
	ChannelState     string         `json:"channelState"`
	Domain           string         `json:"domain"`
	TunnelMode       TunnelMode     `json:"tunnelMode"`
	TunnelState      string         `json:"tunnelState"`
	MFAEnabled       bool           `json:"mfaEnabled"`
	Token            DomainToken    `json:"token"`
	TokenState       string         `json:"tokenState"`
	BoundDeviceCount int            `json:"boundDeviceCount"`
	DeviceCount      int            `json:"deviceCount"`
	ActivePolicy     AccessPolicy   `json:"activePolicy"`
	Policies         []AccessPolicy `json:"policies"`
	Feedback         string         `json:"feedback"`
}

type BoundDevice struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	Location string `json:"location"`
	Bound    bool   `json:"bound"`
	LastSeen string `json:"lastSeen"`
}

type LoginAlert struct {
	ID       string `json:"id"`
	Location string `json:"location"`
	Device   string `json:"device"`
	Action   string `json:"action"`
	State    string `json:"state"`
}

type AccessPolicy struct {
	Key   string `json:"key"`
	Name  string `json:"name"`
	Scope string `json:"scope"`
	Risk  string `json:"risk"`
}

type DomainToken struct {
	Version   int       `json:"version"`
	Domain    string    `json:"domain"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type ShareScanResult struct {
	State   ShareScanState `json:"state"`
	Message string         `json:"message"`
	Checks  []string       `json:"checks"`
}
