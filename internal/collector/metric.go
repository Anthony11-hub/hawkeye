package collector

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

type SystemMetrics struct {
	Timestamp int64   `json:"timestamp"`
	CPUUsage  float64 `json:"cpu_usage"`
	RAMUsage  float64 `json:"ram_usage"`
	DiskUsage float64 `json:"disk_usage"`
	BytesSent uint64  `json:"bytes_sent"`
	BytesRecv uint64  `json:"bytes_recv"`
}

func FetchMetrics() (SystemMetrics, error) {
	m := SystemMetrics{Timestamp: time.Now().UnixMilli()}

	cpuPercents, err := cpu.Percent(0, false)
	if err != nil {
		return m, fmt.Errorf("cpu: %w", err)
	}
	if len(cpuPercents) > 0 {
		m.CPUUsage = cpuPercents[0]
	}

	vm, err := mem.VirtualMemory()
	if err != nil {
		return m, fmt.Errorf("mem: %w", err)
	}
	m.RAMUsage = vm.UsedPercent

	du, err := disk.Usage("/")
	if err != nil {
		return m, fmt.Errorf("disk: %w", err)
	}
	m.DiskUsage = du.UsedPercent

	io, err := net.IOCounters(false)
	if err != nil {
		return m, fmt.Errorf("net: %w", err)
	}
	if len(io) > 0 {
		m.BytesSent = io[0].BytesSent
		m.BytesRecv = io[0].BytesRecv
	}

	return m, nil
}
