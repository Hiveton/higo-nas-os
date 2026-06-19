package hardware

import "time"

// Inventory is the consolidated static hardware descriptor for the host. Unlike
// the monitoring domain (live percentages/trends), this captures the relatively
// stable identity of the machine: host, board, CPU, memory, NICs and sensors.
type Inventory struct {
	Host        HostInfo           `json:"host"`
	Board       BoardInfo          `json:"board"`
	CPU         CPUInfo            `json:"cpu"`
	Memory      MemoryInfo         `json:"memory"`
	Network     []NetworkInterface `json:"network"`
	Sensors     SensorInfo         `json:"sensors"`
	Adapter     string             `json:"adapter"` // "linux" | "dev"
	CollectedAt time.Time          `json:"collectedAt"`
}

type HostInfo struct {
	Hostname string    `json:"hostname"`
	OSName   string    `json:"osName"`
	Kernel   string    `json:"kernel"`
	Arch     string    `json:"arch"`
	Uptime   string    `json:"uptime"`
	BootedAt time.Time `json:"bootedAt"`
}

type BoardInfo struct {
	Vendor  string `json:"vendor"`
	Product string `json:"product"`
	Serial  string `json:"serial"`
	BIOS    string `json:"bios"`
}

type CPUInfo struct {
	Model         string  `json:"model"`
	PhysicalCores int     `json:"physicalCores"`
	LogicalCores  int     `json:"logicalCores"`
	MHz           float64 `json:"mhz"`
}

type MemoryInfo struct {
	TotalBytes  uint64  `json:"totalBytes"`
	UsedBytes   uint64  `json:"usedBytes"`
	UsedPercent float64 `json:"usedPercent"`
}

type NetworkInterface struct {
	Name     string   `json:"name"`
	MAC      string   `json:"mac"`
	IPv4     []string `json:"ipv4"`
	SpeedMbps int     `json:"speedMbps"`
	State    string   `json:"state"`
}

type SensorInfo struct {
	Temperatures []TempReading `json:"temperatures"`
	Fans         []FanReading  `json:"fans"`
}

type TempReading struct {
	Label   string  `json:"label"`
	Celsius float64 `json:"celsius"`
}

type FanReading struct {
	Label string  `json:"label"`
	RPM   float64 `json:"rpm"`
}
