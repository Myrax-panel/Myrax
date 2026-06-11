//go:build linux

package pluginruntime

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

func RunTerminal() error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"runtime": "terminal",
			"plugin":  os.Getenv("MYRAX_PLUGIN_ID"),
		})
	})
	mux.HandleFunc("GET /terminal", terminalSocket)

	server := http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	transport := os.Getenv("MYRAX_PLUGIN_TRANSPORT")
	if transport == "unix" {
		socketPath := os.Getenv("MYRAX_PLUGIN_SOCKET")
		if socketPath == "" {
			return fmt.Errorf("MYRAX_PLUGIN_SOCKET is required for unix transport")
		}
		_ = os.Remove(socketPath)
		listener, err := net.Listen("unix", socketPath)
		if err != nil {
			return err
		}
		defer listener.Close()
		return server.Serve(listener)
	}

	port := os.Getenv("MYRAX_PLUGIN_PORT")
	if port == "" || port == "0" {
		return fmt.Errorf("MYRAX_PLUGIN_PORT is required")
	}
	server.Addr = "127.0.0.1:" + port
	return server.ListenAndServe()
}

func terminalSocket(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	shell := os.Getenv("SHELL")
	if shell == "" || !fileExists(shell) {
		shell = "/bin/bash"
	}
	if !fileExists(shell) {
		shell = "/bin/sh"
	}

	command := exec.Command(shell)
	command.Dir = terminalWorkingDir()
	command.Env = append(os.Environ(), "TERM=xterm-256color")
	terminal, err := pty.StartWithSize(command, &pty.Winsize{Rows: 28, Cols: 96})
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		return
	}
	defer terminal.Close()
	defer command.Process.Kill()

	done := make(chan struct{})
	go func() {
		buffer := make([]byte, 4096)
		for {
			n, err := terminal.Read(buffer)
			if n > 0 {
				if writeErr := conn.WriteMessage(websocket.BinaryMessage, buffer[:n]); writeErr != nil {
					break
				}
			}
			if err != nil {
				break
			}
		}
		close(done)
	}()

	for {
		select {
		case <-done:
			return
		default:
		}
		_, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if handleTerminalControl(terminal, payload) {
			continue
		}
		if _, err := terminal.Write(payload); err != nil {
			return
		}
	}
}

func handleTerminalControl(terminal *os.File, payload []byte) bool {
	text := strings.TrimSpace(string(payload))
	if !strings.HasPrefix(text, `{"type":"resize"`) {
		return false
	}
	var message struct {
		Type string `json:"type"`
		Cols uint16 `json:"cols"`
		Rows uint16 `json:"rows"`
	}
	if err := json.Unmarshal(payload, &message); err != nil {
		return false
	}
	if message.Type != "resize" || message.Cols == 0 || message.Rows == 0 {
		return true
	}
	_ = pty.Setsize(terminal, &pty.Winsize{Cols: message.Cols, Rows: message.Rows})
	return true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func terminalWorkingDir() string {
	home, err := os.UserHomeDir()
	if err == nil && home != "" && fileExists(home) {
		return home
	}
	if home := os.Getenv("HOME"); home != "" && fileExists(home) {
		return home
	}
	return "/"
}

