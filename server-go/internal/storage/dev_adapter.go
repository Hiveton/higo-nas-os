package storage

import (
	"context"
	"time"
)

type Adapter interface {
	Pools(context.Context) ([]StoragePool, error)
	Disks(context.Context) ([]Disk, error)
	SmartReports(context.Context) ([]SmartReport, error)
}

type DevAdapter struct {
	pools        []StoragePool
	disks        []Disk
	smartReports []SmartReport
}

func NewDevAdapter() *DevAdapter {
	updatedAt := time.Date(2026, 5, 6, 9, 30, 0, 0, time.UTC)
	disks := []Disk{
		{Slot: "1", Size: "245 GB", State: DiskStateHealthy, Temperature: "N/A", Serial: "/dev/disk-root", Health: HealthHealthy, Role: "volume", PoolID: "host-dev-root", Model: "开发主机根卷", Interface: "mount"},
		{Slot: "2", Size: "245 GB", State: DiskStateHealthy, Temperature: "N/A", Serial: "/dev/disk-data", Health: HealthHealthy, Role: "volume", PoolID: "host-dev-data", Model: "开发主机数据卷", Interface: "mount"},
		{Slot: "3", Size: "215 MB", State: DiskStateHealthy, Temperature: "N/A", Serial: "/dev/disk-external", Health: HealthWarning, Role: "volume", PoolID: "host-dev-external", Model: "开发主机外接卷", Interface: "mount"},
	}
	return &DevAdapter{
		pools: []StoragePool{
			{ID: "host-dev-root", Name: "开发主机根卷", Type: "主机卷", UsedPercent: 10, Total: "245 GB", Health: HealthHealthy, Temperature: "N/A"},
			{ID: "host-dev-data", Name: "开发主机数据卷", Type: "主机卷", UsedPercent: 41, Total: "245 GB", Health: HealthHealthy, Temperature: "N/A"},
			{ID: "host-dev-external", Name: "开发主机外接卷", Type: "主机卷", UsedPercent: 93, Total: "215 MB", Health: HealthWarning, Temperature: "N/A"},
		},
		disks:        disks,
		smartReports: smartReportsFromDisks(disks, updatedAt),
	}
}

func (a *DevAdapter) Pools(ctx context.Context) ([]StoragePool, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return clonePools(a.pools), nil
}

func (a *DevAdapter) Disks(ctx context.Context) ([]Disk, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return cloneDisks(a.disks), nil
}

func (a *DevAdapter) SmartReports(ctx context.Context) ([]SmartReport, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return cloneSmartReports(a.smartReports), nil
}

func smartReportsFromDisks(disks []Disk, updatedAt time.Time) []SmartReport {
	reports := make([]SmartReport, 0, len(disks))
	for _, disk := range disks {
		reports = append(reports, SmartReport{
			DiskSlot:    disk.Slot,
			Serial:      disk.Serial,
			Health:      disk.Health,
			Temperature: disk.Temperature,
			UpdatedAt:   updatedAt,
			Attributes: []SmartAttribute{
				{Name: "Reallocated_Sector_Ct", Value: 100, Threshold: 10, Status: "ok"},
				{Name: "Power_On_Hours", Value: 92, Threshold: 0, Status: "ok"},
				{Name: "Temperature_Celsius", Value: temperatureValue(disk.Temperature), Threshold: 55, Status: "ok"},
			},
		})
	}
	return reports
}

func temperatureValue(value string) int {
	switch value {
	case "35°C":
		return 35
	case "36°C":
		return 36
	case "37°C":
		return 37
	case "38°C":
		return 38
	case "42°C":
		return 42
	default:
		return 0
	}
}
