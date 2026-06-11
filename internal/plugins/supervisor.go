package plugins

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type Supervisor struct {
	mu       sync.Mutex
	manager  Manager
	coreURL  string
	runtimes map[string]*pluginRuntime
}

type RuntimeStatus struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Running   bool   `json:"running"`
	PID       int    `json:"pid"`
	Port      int    `json:"port"`
	Transport string `json:"transport"`
	Socket    string `json:"socket,omitempty"`
	StartedAt string `json:"startedAt,omitempty"`
	StoppedAt string `json:"stoppedAt,omitempty"`
	Error     string `json:"error,omitempty"`
	LogPath   string `json:"logPath"`
}

type ProxyTarget struct {
	Transport string
	Address   string
}

type pluginRuntime struct {
	plugin    Plugin
	command   *exec.Cmd
	logFile   *os.File
	status    RuntimeStatus
	done      chan struct{}
	stopAsked bool
}

func NewSupervisor(manager Manager, coreURL string) *Supervisor {
	return &Supervisor{
		manager:  manager,
		coreURL:  coreURL,
		runtimes: map[string]*pluginRuntime{},
	}
}

func (s *Supervisor) StartEnabled() error {
	items, err := s.manager.List()
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.Enabled && item.Runtime.Enabled && item.Runtime.Command != "" {
			if err := s.Start(item.ID); err != nil {
				s.setFailed(item, err)
			}
		}
	}
	return nil
}

func (s *Supervisor) StopAll() {
	items := s.Statuses()
	for _, item := range items {
		_ = s.Stop(item.ID)
	}
}

func (s *Supervisor) Forget(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.runtimes, id)
}

func (s *Supervisor) Start(id string) error {
	item, err := s.manager.Get(id)
	if err != nil {
		return err
	}
	if !item.Runtime.Enabled || item.Runtime.Command == "" {
		return nil
	}

	s.mu.Lock()
	if runtime, ok := s.runtimes[item.ID]; ok && runtime.status.Running {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	transport := item.Runtime.Transport
	if transport == "" {
		transport = "tcp"
	}
	if transport != "tcp" && transport != "unix" {
		return fmt.Errorf("unsupported plugin transport: %s", transport)
	}
	if err := os.MkdirAll(s.manager.DataDir(item.ID), 0750); err != nil {
		return err
	}
	if err := os.MkdirAll(s.manager.LogsDir(), 0750); err != nil {
		return err
	}

	port := item.Runtime.Port
	socketPath := filepath.Join(s.manager.DataDir(item.ID), "runtime.sock")
	if transport == "tcp" {
		if port == 0 {
			port, err = freePort()
			if err != nil {
				return err
			}
		}
		socketPath = ""
	} else {
		port = 0
		_ = os.Remove(socketPath)
	}

	logPath := s.manager.LogPath(item.ID)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(logFile, "\n[%s] starting %s\n", time.Now().UTC().Format(time.RFC3339), item.ID)

	commandPath := item.Runtime.Command
	if !filepath.IsAbs(commandPath) {
		commandPath = filepath.Join(item.Path, commandPath)
	}
	cmd := exec.Command(commandPath, item.Runtime.Args...)
	cmd.Dir = item.Path
	cmd.Env = append(os.Environ(),
		"MYRAX_PLUGIN_ID="+item.ID,
		"MYRAX_PLUGIN_DIR="+item.Path,
		"MYRAX_PLUGIN_DATA="+s.manager.DataDir(item.ID),
		"MYRAX_PLUGIN_PORT="+strconv.Itoa(port),
		"MYRAX_PLUGIN_TRANSPORT="+transport,
		"MYRAX_PLUGIN_SOCKET="+socketPath,
		"MYRAX_CORE_URL="+s.coreURL,
		"MYRAX_PLUGIN_LOG="+logPath,
	)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	runtime := &pluginRuntime{
		plugin:  item,
		command: cmd,
		logFile: logFile,
		done:    make(chan struct{}),
		status: RuntimeStatus{
			ID:        item.ID,
			Status:    "running",
			Running:   true,
			PID:       cmd.Process.Pid,
			Port:      port,
			Transport: transport,
			Socket:    socketPath,
			StartedAt: now,
			LogPath:   logPath,
		},
	}

	s.mu.Lock()
	s.runtimes[item.ID] = runtime
	s.mu.Unlock()

	go s.wait(runtime)
	return nil
}

func (s *Supervisor) Stop(id string) error {
	s.mu.Lock()
	runtime, ok := s.runtimes[id]
	if !ok {
		s.mu.Unlock()
		return nil
	}
	if !runtime.status.Running {
		s.mu.Unlock()
		return nil
	}
	runtime.stopAsked = true
	var process *os.Process
	if runtime.command != nil {
		process = runtime.command.Process
	}
	done := runtime.done
	s.mu.Unlock()

	if process == nil {
		return nil
	}
	_ = process.Signal(os.Interrupt)

	if waitForRuntime(done, 3*time.Second) {
		return nil
	}
	_ = process.Kill()
	if waitForRuntime(done, 2*time.Second) {
		return nil
	}
	return nil
}

func (s *Supervisor) Restart(id string) error {
	if err := s.Stop(id); err != nil {
		return err
	}
	time.Sleep(150 * time.Millisecond)
	return s.Start(id)
}

func (s *Supervisor) Status(id string) RuntimeStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	if runtime, ok := s.runtimes[id]; ok {
		return runtime.status
	}
	return RuntimeStatus{
		ID:      id,
		Status:  "stopped",
		Running: false,
		LogPath: s.manager.LogPath(id),
	}
}

func (s *Supervisor) Statuses() []RuntimeStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	statuses := make([]RuntimeStatus, 0, len(s.runtimes))
	for _, runtime := range s.runtimes {
		statuses = append(statuses, runtime.status)
	}
	return statuses
}

func (s *Supervisor) ProxyTarget(id string) (ProxyTarget, error) {
	status := s.Status(id)
	if !status.Running {
		return ProxyTarget{}, fmt.Errorf("plugin runtime is not running: %s", id)
	}
	if status.Transport == "unix" {
		if status.Socket == "" {
			return ProxyTarget{}, fmt.Errorf("plugin unix socket is not available: %s", id)
		}
		return ProxyTarget{Transport: "unix", Address: status.Socket}, nil
	}
	if status.Port == 0 {
		return ProxyTarget{}, fmt.Errorf("plugin runtime port is not available: %s", id)
	}
	return ProxyTarget{Transport: "tcp", Address: "127.0.0.1:" + strconv.Itoa(status.Port)}, nil
}

func (s *Supervisor) wait(runtime *pluginRuntime) {
	err := runtime.command.Wait()
	now := time.Now().UTC().Format(time.RFC3339)
	defer close(runtime.done)

	s.mu.Lock()
	defer s.mu.Unlock()

	status := runtime.status
	status.Running = false
	status.PID = 0
	status.StoppedAt = now
	if runtime.stopAsked {
		status.Status = "stopped"
	} else if err != nil {
		status.Status = "failed"
		status.Error = err.Error()
	} else {
		status.Status = "stopped"
	}
	runtime.status = status
	_, _ = fmt.Fprintf(runtime.logFile, "[%s] %s\n", now, status.Status)
	_ = runtime.logFile.Close()
}

func (s *Supervisor) setFailed(item Plugin, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runtimes[item.ID] = &pluginRuntime{
		plugin: item,
		done:   closedRuntimeDone(),
		status: RuntimeStatus{
			ID:      item.ID,
			Status:  "failed",
			Running: false,
			Error:   err.Error(),
			LogPath: s.manager.LogPath(item.ID),
		},
	}
}

func waitForRuntime(done <-chan struct{}, timeout time.Duration) bool {
	if done == nil {
		return true
	}
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

func closedRuntimeDone() chan struct{} {
	done := make(chan struct{})
	close(done)
	return done
}

func freePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func PipeWebSockets(ctx context.Context, left WebSocketConn, right WebSocketConn) {
	var once sync.Once
	done := make(chan struct{})
	closeBoth := func() {
		once.Do(func() {
			_ = left.Close()
			_ = right.Close()
			close(done)
		})
	}
	go pipeWS(ctx, left, right, closeBoth)
	go pipeWS(ctx, right, left, closeBoth)
	select {
	case <-ctx.Done():
		closeBoth()
	case <-done:
	}
}

type WebSocketConn interface {
	NextReader() (int, io.Reader, error)
	NextWriter(int) (io.WriteCloser, error)
	Close() error
}

func pipeWS(ctx context.Context, source WebSocketConn, target WebSocketConn, closeBoth func()) {
	defer closeBoth()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		messageType, reader, err := source.NextReader()
		if err != nil {
			return
		}
		writer, err := target.NextWriter(messageType)
		if err != nil {
			return
		}
		if _, err := io.Copy(writer, reader); err != nil {
			_ = writer.Close()
			return
		}
		if err := writer.Close(); err != nil {
			return
		}
	}
}
