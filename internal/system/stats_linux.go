//go:build linux

package system

import (
	"bufio"
	"errors"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var cpuTracker = struct {
	sync.Mutex
	sample cpuSample
	usage  float64
	ready  bool
}{}

var networkTracker = struct {
	sync.Mutex
	samples map[string]networkSample
	ready   bool
}{}

func ReadStats() (Stats, error) {
	host, _ := readHost()
	cpu, _ := readCPU()
	mem, _ := readMemory()
	disks, _ := readDisks()
	network, _ := readNetwork()

	return Stats{
		Timestamp: time.Now(),
		Host:      host,
		CPU:       cpu,
		Memory:    mem,
		Disks:     disks,
		Network:   network,
	}, nil
}

func readHost() (HostStats, error) {
	hostname, _ := os.Hostname()
	host := HostStats{Hostname: hostname, OS: "Linux"}

	if data, err := os.ReadFile("/proc/sys/kernel/osrelease"); err == nil {
		host.Kernel = strings.TrimSpace(string(data))
	}
	if data, err := os.ReadFile("/proc/uptime"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) > 0 {
			host.Uptime, _ = strconv.ParseFloat(fields[0], 64)
		}
	}
	return host, nil
}

func readCPU() (CPUStats, error) {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return CPUStats{}, err
	}
	defer file.Close()

	stats := CPUStats{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "model name") && stats.Model == "" {
			stats.Model = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
		}
		if strings.HasPrefix(line, "processor") {
			stats.Cores++
		}
	}
	stats.Usage = readCPUUsage()
	stats.LoadAverage = readLoadAverage()
	return stats, scanner.Err()
}

func readCPUUsage() float64 {
	current, err := readCPUSample()
	if err != nil {
		return 0
	}

	cpuTracker.Lock()
	defer cpuTracker.Unlock()

	if !cpuTracker.ready {
		cpuTracker.sample = current
		cpuTracker.ready = true
		return 0
	}

	total := float64(current.total - cpuTracker.sample.total)
	idle := float64(current.idle - cpuTracker.sample.idle)
	cpuTracker.sample = current

	if total <= 0 {
		return cpuTracker.usage
	}

	cpuTracker.usage = roundPercent((1 - idle/total) * 100)
	return cpuTracker.usage
}

type cpuSample struct {
	total uint64
	idle  uint64
}

func readCPUSample() (cpuSample, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return cpuSample{}, err
	}
	line := strings.SplitN(string(data), "\n", 2)[0]
	fields := strings.Fields(line)
	if len(fields) < 5 || fields[0] != "cpu" {
		return cpuSample{}, errors.New("invalid cpu sample")
	}

	var sample cpuSample
	for i, field := range fields[1:] {
		value, _ := strconv.ParseUint(field, 10, 64)
		sample.total += value
		if i == 3 || i == 4 {
			sample.idle += value
		}
	}
	return sample, nil
}

func readLoadAverage() []float64 {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return nil
	}
	fields := strings.Fields(string(data))
	load := make([]float64, 0, 3)
	for _, field := range fields[:min(3, len(fields))] {
		value, _ := strconv.ParseFloat(field, 64)
		load = append(load, value)
	}
	return load
}

func readMemory() (MemoryStats, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return MemoryStats{}, err
	}
	defer file.Close()

	values := map[string]uint64{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) < 2 {
			continue
		}
		value, _ := strconv.ParseUint(parts[1], 10, 64)
		values[strings.TrimSuffix(parts[0], ":")] = value * 1024
	}

	total := values["MemTotal"]
	available := values["MemAvailable"]
	used := total - available
	return MemoryStats{
		Total:     total,
		Available: available,
		Used:      used,
		Usage:     percent(used, total),
	}, scanner.Err()
}

func readDisks() ([]DiskStats, error) {
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	type diskGroup struct {
		disk        DiskStats
		mountpoints []string
	}

	groups := map[string]*diskGroup{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) < 3 {
			continue
		}
		device, mountpoint, fsType := parts[0], parts[1], parts[2]
		if !strings.HasPrefix(device, "/dev/") {
			continue
		}
		key := device + "|" + fsType
		group, ok := groups[key]
		if !ok {
			group = &diskGroup{
				disk: DiskStats{
					Device:     device,
					Filesystem: fsType,
				},
			}
			groups[key] = group
		}
		group.mountpoints = append(group.mountpoints, mountpoint)

		var stat syscall.Statfs_t
		if err := syscall.Statfs(mountpoint, &stat); err != nil {
			continue
		}
		total := stat.Blocks * uint64(stat.Bsize)
		free := stat.Bavail * uint64(stat.Bsize)
		used := total - free
		inodesTotal := stat.Files
		inodesFree := stat.Ffree
		inodesUsed := inodesTotal - inodesFree
		group.disk.Total = total
		group.disk.Used = used
		group.disk.Free = free
		group.disk.Usage = percent(used, total)
		group.disk.InodesTotal = inodesTotal
		group.disk.InodesUsed = inodesUsed
		group.disk.InodeUsage = percent(inodesUsed, inodesTotal)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	disks := make([]DiskStats, 0, len(groups))
	for _, group := range groups {
		group.mountpoints = uniqueStrings(group.mountpoints)
		sort.Slice(group.mountpoints, func(i, j int) bool {
			if group.mountpoints[i] == "/" {
				return true
			}
			if group.mountpoints[j] == "/" {
				return false
			}
			if len(group.mountpoints[i]) == len(group.mountpoints[j]) {
				return group.mountpoints[i] < group.mountpoints[j]
			}
			return len(group.mountpoints[i]) < len(group.mountpoints[j])
		})
		group.disk.Mountpoint = group.mountpoints[0]
		group.disk.Mountpoints = group.mountpoints
		disks = append(disks, group.disk)
	}

	sort.Slice(disks, func(i, j int) bool {
		if disks[i].Mountpoint == "/" {
			return true
		}
		if disks[j].Mountpoint == "/" {
			return false
		}
		return disks[i].Mountpoint < disks[j].Mountpoint
	})

	return disks, nil
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func readNetwork() ([]NetworkStats, error) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	now := time.Now()
	current := map[string]networkSample{}
	var out []NetworkStats
	scanner := bufio.NewScanner(file)
	for lineNo := 0; scanner.Scan(); lineNo++ {
		if lineNo < 2 {
			continue
		}
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		fields := strings.Fields(parts[1])
		if len(fields) < 16 || name == "lo" {
			continue
		}
		rx, _ := strconv.ParseUint(fields[0], 10, 64)
		tx, _ := strconv.ParseUint(fields[8], 10, 64)
		current[name] = networkSample{rxBytes: rx, txBytes: tx, timestamp: now}
		out = append(out, NetworkStats{Name: name, RxBytes: rx, TxBytes: tx})
	}
	if err := scanner.Err(); err != nil {
		return out, err
	}
	return applyNetworkRates(out, current), nil
}

type networkSample struct {
	rxBytes   uint64
	txBytes   uint64
	timestamp time.Time
}

func applyNetworkRates(stats []NetworkStats, current map[string]networkSample) []NetworkStats {
	networkTracker.Lock()
	defer networkTracker.Unlock()

	if !networkTracker.ready {
		networkTracker.samples = current
		networkTracker.ready = true
		return stats
	}

	for i, adapter := range stats {
		previous, ok := networkTracker.samples[adapter.Name]
		if !ok {
			continue
		}

		elapsed := current[adapter.Name].timestamp.Sub(previous.timestamp).Seconds()
		if elapsed <= 0 {
			continue
		}

		stats[i].RxRate = bytesPerSecond(adapter.RxBytes, previous.rxBytes, elapsed)
		stats[i].TxRate = bytesPerSecond(adapter.TxBytes, previous.txBytes, elapsed)
	}

	networkTracker.samples = current
	return stats
}

func bytesPerSecond(current, previous uint64, elapsed float64) uint64 {
	if current < previous {
		return 0
	}
	return uint64(math.Round(float64(current-previous) / elapsed))
}

func percent(used, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return roundPercent(float64(used) / float64(total) * 100)
}

func roundPercent(value float64) float64 {
	return math.Round(value*10) / 10
}
