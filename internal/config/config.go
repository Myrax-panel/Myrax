package config

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Host             string `json:"host"`
	Port             int    `json:"port"`
	PanelPath        string `json:"panelPath"`
	AddOnsEnabled    bool   `json:"addOnsEnabled"`
	AuthUsername     string `json:"authUsername"`
	AuthPasswordHash string `json:"-"`
	SessionSecret    string `json:"-"`
	ConfigPath       string `json:"configPath"`
}

func Default() Config {
	return Config{
		Host:          "127.0.0.1",
		Port:          1487,
		PanelPath:     "/",
		AddOnsEnabled: false,
		ConfigPath:    defaultConfigPath(),
	}
}

func LoadFile(path string, cfg *Config) error {
	if path == "" {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), `"`)
		switch key {
		case "host":
			cfg.Host = value
		case "port":
			port, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("invalid port in config: %w", err)
			}
			cfg.Port = port
		case "panel_path":
			cfg.PanelPath = NormalizePanelPath(value)
		case "add_ons":
			enabled, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("invalid add_ons in config: %w", err)
			}
			cfg.AddOnsEnabled = enabled
		case "auth_user":
			cfg.AuthUsername = value
		case "auth_password_hash":
			cfg.AuthPasswordHash = value
		case "session_secret":
			cfg.SessionSecret = value
		}
	}
	cfg.PanelPath = NormalizePanelPath(cfg.PanelPath)
	return scanner.Err()
}

func SaveFile(path string, cfg Config) error {
	if path == "" {
		return nil
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return fmt.Errorf("invalid port: %d", cfg.Port)
	}
	if strings.TrimSpace(cfg.Host) == "" || strings.ContainsAny(cfg.Host, " \t\r\n") {
		return fmt.Errorf("invalid host: %q", cfg.Host)
	}
	cfg.PanelPath = NormalizePanelPath(cfg.PanelPath)
	if strings.ContainsAny(cfg.PanelPath, " \t\r\n?#") {
		return fmt.Errorf("invalid panel path: %q", cfg.PanelPath)
	}
	if reservedPanelPath(cfg.PanelPath) {
		return fmt.Errorf("reserved panel path: %q", cfg.PanelPath)
	}
	if cfg.AuthUsername != "" && (len(cfg.AuthUsername) < 3 || len(cfg.AuthUsername) > 64 || strings.ContainsAny(cfg.AuthUsername, " \t\r\n")) {
		return fmt.Errorf("invalid auth user")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return err
	}

	body := fmt.Sprintf("host = %q\nport = %d\npanel_path = %q\nadd_ons = %t\n", cfg.Host, cfg.Port, cfg.PanelPath, cfg.AddOnsEnabled)
	if cfg.AuthUsername != "" {
		body += fmt.Sprintf("auth_user = %q\n", cfg.AuthUsername)
	}
	if cfg.AuthPasswordHash != "" {
		body += fmt.Sprintf("auth_password_hash = %q\n", cfg.AuthPasswordHash)
	}
	if cfg.SessionSecret != "" {
		body += fmt.Sprintf("session_secret = %q\n", cfg.SessionSecret)
	}
	return os.WriteFile(path, []byte(body), 0600)
}

func NormalizePanelPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "/" {
		return "/"
	}
	value = "/" + strings.Trim(value, "/")
	cleaned := path.Clean(value)
	if cleaned == "." || cleaned == "/" {
		return "/"
	}
	return cleaned
}

func (cfg Config) AuthConfigured() bool {
	return strings.TrimSpace(cfg.AuthUsername) != "" && strings.TrimSpace(cfg.AuthPasswordHash) != "" && strings.TrimSpace(cfg.SessionSecret) != ""
}

func reservedPanelPath(value string) bool {
	if value == "/" {
		return false
	}
	for _, prefix := range []string{"/api", "/assets", "/addons"} {
		if value == prefix || strings.HasPrefix(value, prefix+"/") {
			return true
		}
	}
	return false
}
