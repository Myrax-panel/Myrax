//go:build linux

package system

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func ReadLogs(source, level, query string, limit int) ([]LogEntry, error) {
	limit = clampLimit(limit, 20, 300, 120)
	args := []string{"-n", strconv.Itoa(limit), "--no-pager", "-o", "json"}
	if source == "myrax" || source == "" {
		args = append([]string{"-u", "myrax.service"}, args...)
	}

	output, err := exec.Command("journalctl", args...).Output()
	if err != nil {
		return nil, err
	}

	level = strings.ToLower(strings.TrimSpace(level))
	query = strings.ToLower(strings.TrimSpace(query))
	entries := []LogEntry{}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		entry := parseJournalEntry(scanner.Text())
		if entry.Message == "" {
			continue
		}
		if level != "" && level != "all" && entry.Level != level {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(entry.Message+" "+entry.Unit), query) {
			continue
		}
		entries = append(entries, entry)
	}
	return entries, scanner.Err()
}

func ReadProcesses(limit int) ([]ProcessInfo, error) {
	limit = clampLimit(limit, 10, 300, 80)
	output, err := exec.Command("ps", "-eo", "pid,ppid,comm,%cpu,%mem,rss,state", "--sort=-%cpu", "--no-headers").Output()
	if err != nil {
		return nil, err
	}

	processes := []ProcessInfo{}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 7 {
			continue
		}
		pid, _ := strconv.Atoi(fields[0])
		ppid, _ := strconv.Atoi(fields[1])
		cpu, _ := strconv.ParseFloat(fields[3], 64)
		mem, _ := strconv.ParseFloat(fields[4], 64)
		rssKB, _ := strconv.ParseUint(fields[5], 10, 64)
		processes = append(processes, ProcessInfo{
			PID:     pid,
			PPID:    ppid,
			Name:    fields[2],
			CPU:     roundPercent(cpu),
			Memory:  roundPercent(mem),
			RSS:     rssKB * 1024,
			State:   fields[6],
			Service: processService(pid),
		})
		if len(processes) >= limit {
			break
		}
	}
	return processes, scanner.Err()
}

func KillProcess(pid int) error {
	if pid <= 1 {
		return errors.New("invalid pid")
	}
	return syscall.Kill(pid, syscall.SIGTERM)
}

func ReadServices(limit int) ([]ServiceInfo, error) {
	limit = clampLimit(limit, 20, 500, 120)
	output, err := exec.Command("systemctl", "list-units", "--type=service", "--all", "--no-legend", "--no-pager").Output()
	if err != nil {
		return nil, err
	}

	services := []ServiceInfo{}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}
		services = append(services, ServiceInfo{
			Name:        fields[0],
			Load:        fields[1],
			Active:      fields[2],
			Sub:         fields[3],
			Description: strings.Join(fields[4:], " "),
		})
		if len(services) >= limit {
			break
		}
	}
	return services, scanner.Err()
}

func RunServiceAction(name, action string) error {
	if !validServiceName(name) {
		return errors.New("invalid service name")
	}
	switch action {
	case "start", "stop", "restart":
	default:
		return errors.New("invalid service action")
	}
	return exec.Command("systemctl", action, name).Run()
}

func ReadNetworkDetails() (NetworkDetails, error) {
	stats, _ := readNetwork()
	details := NetworkDetails{
		Gateway:     defaultGateway(),
		DNS:         dnsServers(),
		ListenPorts: listenPorts(),
	}
	for _, adapter := range stats {
		details.Interfaces = append(details.Interfaces, InterfaceDetails{
			Name:      adapter.Name,
			State:     readSysString(adapter.Name, "operstate"),
			MTU:       readSysUint(adapter.Name, "mtu"),
			MAC:       readSysString(adapter.Name, "address"),
			Addresses: interfaceAddresses(adapter.Name),
			RxBytes:   adapter.RxBytes,
			TxBytes:   adapter.TxBytes,
			RxRate:    adapter.RxRate,
			TxRate:    adapter.TxRate,
		})
	}
	return details, nil
}

func ReadDiskDetails() ([]DiskStats, error) {
	return readDisks()
}

func parseJournalEntry(line string) LogEntry {
	raw := map[string]any{}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return LogEntry{Level: "info", Message: line}
	}
	unit, _ := raw["_SYSTEMD_UNIT"].(string)
	message, _ := raw["MESSAGE"].(string)
	timestamp, _ := raw["__REALTIME_TIMESTAMP"].(string)
	priority, _ := raw["PRIORITY"].(string)
	return LogEntry{
		Timestamp: formatJournalTimestamp(timestamp),
		Level:     priorityLevel(priority),
		Unit:      unit,
		Message:   message,
	}
}

func formatJournalTimestamp(raw string) string {
	micros, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || micros <= 0 {
		return raw
	}
	return time.UnixMicro(micros).Format(time.RFC3339)
}

func priorityLevel(priority string) string {
	switch priority {
	case "0", "1", "2", "3":
		return "error"
	case "4":
		return "warning"
	case "5":
		return "notice"
	default:
		return "info"
	}
}

func processService(pid int) string {
	data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "cgroup"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		for _, part := range strings.Split(line, "/") {
			if strings.HasSuffix(part, ".service") {
				return part
			}
		}
	}
	return ""
}

func defaultGateway() string {
	output, err := exec.Command("ip", "route", "show", "default").Output()
	if err != nil {
		return ""
	}
	fields := strings.Fields(string(output))
	for i, field := range fields {
		if field == "via" && i+1 < len(fields) {
			return fields[i+1]
		}
	}
	return ""
}

func dnsServers() []string {
	data, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		return nil
	}
	servers := []string{}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "nameserver" {
			servers = append(servers, fields[1])
		}
	}
	return servers
}

func interfaceAddresses(name string) []string {
	output, err := exec.Command("ip", "-o", "addr", "show", "dev", name).Output()
	if err != nil {
		return nil
	}
	addresses := []string{}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		fields := strings.Fields(line)
		for i, field := range fields {
			if (field == "inet" || field == "inet6") && i+1 < len(fields) {
				addresses = append(addresses, fields[i+1])
			}
		}
	}
	return addresses
}

func listenPorts() []ListenPort {
	output, err := exec.Command("ss", "-H", "-lntu").Output()
	if err != nil {
		return nil
	}
	ports := []ListenPort{}
	seen := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		proto := strings.ToLower(fields[0])
		address, port := splitListenAddress(fields[4])
		key := proto + address + port
		if seen[key] {
			continue
		}
		seen[key] = true
		ports = append(ports, ListenPort{Protocol: proto, Address: address, Port: port})
	}
	sort.Slice(ports, func(i, j int) bool {
		if ports[i].Protocol == ports[j].Protocol {
			return ports[i].Port < ports[j].Port
		}
		return ports[i].Protocol < ports[j].Protocol
	})
	return ports
}

func splitListenAddress(value string) (string, string) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "[") {
		end := strings.LastIndex(value, "]")
		if end > 0 {
			address := value[1:end]
			port := strings.TrimPrefix(value[end+1:], ":")
			return address, port
		}
	}
	index := strings.LastIndex(value, ":")
	if index < 0 {
		return value, ""
	}
	return strings.Trim(value[:index], "[]"), value[index+1:]
}

func readSysString(name, field string) string {
	data, err := os.ReadFile(filepath.Join("/sys/class/net", name, field))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func readSysUint(name, field string) uint64 {
	value, _ := strconv.ParseUint(readSysString(name, field), 10, 64)
	return value
}

func validServiceName(name string) bool {
	if !strings.HasSuffix(name, ".service") || strings.Contains(name, "/") {
		return false
	}
	for _, char := range name {
		if char == '.' || char == '-' || char == '_' || (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') {
			continue
		}
		return false
	}
	return true
}

func clampLimit(value, minValue, maxValue, defaultValue int) int {
	if value == 0 {
		return defaultValue
	}
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
