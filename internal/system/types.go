package system

import "time"

type Stats struct {
	Timestamp time.Time      `json:"timestamp"`
	Host      HostStats      `json:"host"`
	CPU       CPUStats       `json:"cpu"`
	Memory    MemoryStats    `json:"memory"`
	Disks     []DiskStats    `json:"disks"`
	Network   []NetworkStats `json:"network"`
}

type HostStats struct {
	Hostname string  `json:"hostname"`
	OS       string  `json:"os"`
	Kernel   string  `json:"kernel"`
	Uptime   float64 `json:"uptime"`
}

type CPUStats struct {
	Model       string    `json:"model"`
	Cores       int       `json:"cores"`
	Usage       float64   `json:"usage"`
	LoadAverage []float64 `json:"loadAverage"`
}

type MemoryStats struct {
	Total     uint64  `json:"total"`
	Available uint64  `json:"available"`
	Used      uint64  `json:"used"`
	Usage     float64 `json:"usage"`
}

type DiskStats struct {
	Device      string   `json:"device"`
	Mountpoint  string   `json:"mountpoint"`
	Mountpoints []string `json:"mountpoints"`
	Filesystem  string   `json:"filesystem"`
	Total       uint64   `json:"total"`
	Used        uint64   `json:"used"`
	Free        uint64   `json:"free"`
	Usage       float64  `json:"usage"`
	InodesTotal uint64   `json:"inodesTotal"`
	InodesUsed  uint64   `json:"inodesUsed"`
	InodeUsage  float64  `json:"inodeUsage"`
}

type NetworkStats struct {
	Name    string `json:"name"`
	RxBytes uint64 `json:"rxBytes"`
	TxBytes uint64 `json:"txBytes"`
	RxRate  uint64 `json:"rxRate"`
	TxRate  uint64 `json:"txRate"`
}

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Unit      string `json:"unit"`
	Message   string `json:"message"`
}

type ProcessInfo struct {
	PID     int     `json:"pid"`
	PPID    int     `json:"ppid"`
	Name    string  `json:"name"`
	State   string  `json:"state"`
	CPU     float64 `json:"cpu"`
	Memory  float64 `json:"memory"`
	RSS     uint64  `json:"rss"`
	Service string  `json:"service"`
}

type ServiceInfo struct {
	Name        string `json:"name"`
	Load        string `json:"load"`
	Active      string `json:"active"`
	Sub         string `json:"sub"`
	Description string `json:"description"`
}

type NetworkDetails struct {
	Gateway     string             `json:"gateway"`
	DNS         []string           `json:"dns"`
	Interfaces  []InterfaceDetails `json:"interfaces"`
	ListenPorts []ListenPort       `json:"listenPorts"`
}

type InterfaceDetails struct {
	Name      string   `json:"name"`
	State     string   `json:"state"`
	MTU       uint64   `json:"mtu"`
	MAC       string   `json:"mac"`
	Addresses []string `json:"addresses"`
	RxBytes   uint64   `json:"rxBytes"`
	TxBytes   uint64   `json:"txBytes"`
	RxRate    uint64   `json:"rxRate"`
	TxRate    uint64   `json:"txRate"`
}

type ListenPort struct {
	Protocol string `json:"protocol"`
	Address  string `json:"address"`
	Port     string `json:"port"`
}
