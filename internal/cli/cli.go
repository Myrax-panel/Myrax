package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"myrax/internal/actions"
	"myrax/internal/auth"
	"myrax/internal/config"
	"myrax/internal/pluginruntime"
	"myrax/internal/plugins"
	"myrax/internal/server"
	"myrax/internal/uninstall"
	"myrax/internal/version"

	"golang.org/x/term"
)

func Run(args []string) int {
	if len(args) == 0 {
		return runServe(nil)
	}

	switch args[0] {
	case "--help", "-h", "help":
		printHelp()
		return 0
	case "--version", "-v", "version":
		fmt.Println(version.Version)
		return 0
	case "--uninstall", "uninstall":
		if err := uninstall.Run(false); err != nil {
			fmt.Fprintf(os.Stderr, "uninstall failed: %v\n", err)
			return 1
		}
		return 0
	case "update":
		if err := actions.UpdateLatest(); err != nil {
			fmt.Fprintf(os.Stderr, "update failed: %v\n", err)
			return 1
		}
		fmt.Println("myrax update started")
		return 0
	case "add-ons", "addons":
		return runAddOns(args[1:])
	case "configure":
		return runConfigure(args[1:])
	case "plugin", "plugins":
		return runPlugin(args[1:])
	case "plugin-runtime":
		return runPluginRuntime(args[1:])
	case "serve":
		return runServe(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", args[0])
		printHelp()
		return 2
	}
}

func runServe(args []string) int {
	cfg := config.Default()
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--host":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--host requires a value")
				return 2
			}
			i++
			cfg.Host = args[i]
		case "--port":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--port requires a value")
				return 2
			}
			i++
			port, err := strconv.Atoi(args[i])
			if err != nil || port < 1 || port > 65535 {
				fmt.Fprintf(os.Stderr, "invalid port: %s\n", args[i])
				return 2
			}
			cfg.Port = port
		case "--config":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--config requires a value")
				return 2
			}
			i++
			cfg.ConfigPath = args[i]
		default:
			fmt.Fprintf(os.Stderr, "unknown serve option: %s\n", args[i])
			return 2
		}
	}

	if err := config.LoadFile(cfg.ConfigPath, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "config warning: %v\n", err)
	}
	if err := server.Serve(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "server failed: %v\n", err)
		return 1
	}
	return 0
}

func runConfigure(args []string) int {
	configPath := config.Default().ConfigPath
	for i := 0; i < len(args); i++ {
		if args[i] == "--config" && i+1 < len(args) {
			configPath = args[i+1]
			break
		}
	}

	cfg := config.Default()
	cfg.ConfigPath = configPath
	if err := config.LoadFile(cfg.ConfigPath, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "config warning: %v\n", err)
	}

	readPassword := false
	usernameSet := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--config":
			i++
		case "--host":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--host requires a value")
				return 2
			}
			i++
			cfg.Host = args[i]
		case "--port":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--port requires a value")
				return 2
			}
			i++
			port, err := strconv.Atoi(args[i])
			if err != nil || port < 1 || port > 65535 {
				fmt.Fprintf(os.Stderr, "invalid port: %s\n", args[i])
				return 2
			}
			cfg.Port = port
		case "--panel-path":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--panel-path requires a value")
				return 2
			}
			i++
			cfg.PanelPath = config.NormalizePanelPath(args[i])
		case "--username":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--username requires a value")
				return 2
			}
			i++
			cfg.AuthUsername = strings.TrimSpace(args[i])
			usernameSet = true
		case "--password-stdin":
			readPassword = true
		default:
			fmt.Fprintf(os.Stderr, "unknown configure option: %s\n", args[i])
			return 2
		}
	}

	if !usernameSet && cfg.AuthUsername == "" {
		cfg.AuthUsername = "admin"
	}
	if !validAuthUsername(cfg.AuthUsername) {
		fmt.Fprintln(os.Stderr, "invalid username: use 3-64 characters: letters, numbers, dot, underscore, dash")
		return 2
	}
	if readPassword {
		data, err := io.ReadAll(io.LimitReader(os.Stdin, 4096))
		if err != nil {
			fmt.Fprintf(os.Stderr, "password read failed: %v\n", err)
			return 1
		}
		password := strings.TrimRight(string(data), "\r\n")
		if len(password) < 12 || len(password) > 256 {
			fmt.Fprintln(os.Stderr, "password must be 12-256 characters")
			return 2
		}
		hash, err := auth.HashPassword(password)
		if err != nil {
			fmt.Fprintf(os.Stderr, "password hash failed: %v\n", err)
			return 1
		}
		cfg.AuthPasswordHash = hash
	}
	if cfg.AuthPasswordHash == "" {
		fmt.Fprintln(os.Stderr, "password is required; pass --password-stdin")
		return 2
	}
	if cfg.SessionSecret == "" {
		secret, err := auth.GenerateSecret()
		if err != nil {
			fmt.Fprintf(os.Stderr, "session secret failed: %v\n", err)
			return 1
		}
		cfg.SessionSecret = secret
	}
	if err := config.SaveFile(cfg.ConfigPath, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "configure failed: %v\n", err)
		return 1
	}
	fmt.Printf("configured: %s:%d%s\n", cfg.Host, cfg.Port, cfg.PanelPath)
	return 0
}

func validAuthUsername(username string) bool {
	if len(username) < 3 || len(username) > 64 {
		return false
	}
	for _, r := range username {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func runPluginRuntime(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "plugin-runtime requires a runtime name")
		return 2
	}
	switch args[0] {
	case "terminal":
		if err := pluginruntime.RunTerminal(); err != nil {
			fmt.Fprintf(os.Stderr, "terminal runtime failed: %v\n", err)
			return 1
		}
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown plugin runtime: %s\n", args[0])
		return 2
	}
}

func runAddOns(args []string) int {
	if len(args) == 0 {
		args = []string{"status"}
	}

	command := args[0]
	if command == "--true" || command == "true" {
		command = "enable"
	}
	if command == "--false" || command == "false" {
		command = "disable"
	}

	switch command {
	case "status":
		cfg := loadConfigForCLI()
		if cfg.AddOnsEnabled {
			fmt.Println("add-ons: enabled")
		} else {
			fmt.Println("add-ons: disabled")
		}
		return 0
	case "enable":
		if err := saveAddOns(true); err != nil {
			fmt.Fprintf(os.Stderr, "add-ons failed: %v\n", err)
			return 1
		}
		fmt.Println("add-ons: enabled")
		return 0
	case "disable":
		if err := saveAddOns(false); err != nil {
			fmt.Fprintf(os.Stderr, "add-ons failed: %v\n", err)
			return 1
		}
		fmt.Println("add-ons: disabled")
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown add-ons command: %s\n", args[0])
		return 2
	}
}

func runPlugin(args []string) int {
	if len(args) == 0 {
		args = []string{"list"}
	}

	manager := plugins.NewManager()
	switch args[0] {
	case "install":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "plugin install requires a GitHub URL, local path or store name (see: myrax plugin store)")
			return 2
		}
		if err := postLocalAPI("/api/plugins/install", map[string]string{"source": args[1]}); err == nil {
			fmt.Println("plugin installed")
			return 0
		}
		plugin, err := manager.Install(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "plugin install failed: %v\n", err)
			return 1
		}
		_ = postLocalAPI(fmt.Sprintf("/api/plugins/%s/restart", url.PathEscape(plugin.ID)), nil)
		fmt.Printf("plugin installed: %s %s (enabled)\n", plugin.ID, plugin.Version)
		return 0
	case "list":
		return runPluginList(manager)
	case "store":
		return runPluginStore(manager)
	case "remove":
		return runPluginRemove(manager)
	case "enable", "disable":
		return runPluginList(manager)
	case "logs":
		return runPluginLogs(manager, args[1:])
	case "restart":
		return runPluginRestart(args[1:])
	case "update":
		return runPluginUpdate(manager, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown plugin command: %s\n", args[0])
		return 2
	}
}

func loadConfigForCLI() config.Config {
	cfg := config.Default()
	if err := config.LoadFile(cfg.ConfigPath, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "config warning: %v\n", err)
	}
	return cfg
}

func saveAddOns(enabled bool) error {
	cfg := loadConfigForCLI()
	cfg.AddOnsEnabled = enabled
	return config.SaveFile(cfg.ConfigPath, cfg)
}

func runPluginList(manager plugins.Manager) int {
	items, err := manager.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "plugin list failed: %v\n", err)
		return 1
	}
	if len(items) == 0 {
		fmt.Println("plugins: none installed")
		printStore(nil)
		return 0
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		printPlugins(items)
		return 0
	}

	index := 0
	for {
		renderPluginList(items, index)
		key, err := readKey()
		if err != nil {
			fmt.Fprintf(os.Stderr, "plugin list failed: %v\n", err)
			return 1
		}
		switch key {
		case "up":
			if index > 0 {
				index--
			}
		case "down":
			if index < len(items)-1 {
				index++
			}
		case "enter":
			next := !items[index].Enabled
			if err := setPluginEnabled(manager, items[index].ID, next); err != nil {
				fmt.Fprintf(os.Stderr, "\nplugin toggle failed: %v\n", err)
				return 1
			}
			items[index].Enabled = next
			if next {
				items[index].Status = "enabled"
			} else {
				items[index].Status = "installed"
			}
		case "q", "esc", "ctrl-c":
			fmt.Print("\033[2J\033[H")
			return 0
		}
	}
}

func runPluginRemove(manager plugins.Manager) int {
	items, err := manager.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "plugin remove failed: %v\n", err)
		return 1
	}
	if len(items) == 0 {
		fmt.Println("plugins: none installed")
		return 0
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprintln(os.Stderr, "plugin remove requires an interactive terminal")
		return 2
	}

	index := 0
	for {
		renderPluginRemove(items, index)
		key, err := readKey()
		if err != nil {
			fmt.Fprintf(os.Stderr, "plugin remove failed: %v\n", err)
			return 1
		}
		switch key {
		case "up":
			if index > 0 {
				index--
			}
		case "down":
			if index < len(items)-1 {
				index++
			}
		case "enter":
			if err := removePlugin(manager, items[index].ID); err != nil {
				fmt.Fprintf(os.Stderr, "\nplugin remove failed: %v\n", err)
				return 1
			}
			fmt.Print("\033[2J\033[H")
			fmt.Printf("plugin removed: %s\n", items[index].ID)
			return 0
		case "q", "esc", "ctrl-c":
			fmt.Print("\033[2J\033[H")
			return 0
		}
	}
}

func setPluginEnabled(manager plugins.Manager, id string, enabled bool) error {
	endpoint := "/api/plugins/disable"
	if enabled {
		endpoint = "/api/plugins/enable"
	}
	if err := postLocalAPI(endpoint, map[string]string{"id": id}); err == nil {
		return nil
	}
	return manager.SetEnabled(id, enabled)
}

func removePlugin(manager plugins.Manager, id string) error {
	if err := postLocalAPI("/api/plugins/remove", map[string]string{"id": id}); err == nil {
		return nil
	}
	return manager.Remove(id)
}

func updatePlugin(manager plugins.Manager, id string) error {
	if err := postLocalAPI(fmt.Sprintf("/api/plugins/%s/update", url.PathEscape(id)), nil); err == nil {
		return nil
	}
	_, err := manager.Update(id)
	return err
}

func postLocalAPI(path string, payload any) error {
	cfg := loadConfigForCLI()
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	client := http.Client{Timeout: 15 * time.Second}
	response, err := client.Post(fmt.Sprintf("http://127.0.0.1:%d%s", cfg.Port, path), "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func runPluginLogs(manager plugins.Manager, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "plugin logs requires a plugin id")
		return 2
	}
	limit := 200
	if len(args) >= 3 && args[1] == "--limit" {
		value, err := strconv.Atoi(args[2])
		if err == nil {
			limit = value
		}
	}
	entries, err := manager.ReadLogs(args[0], limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "plugin logs failed: %v\n", err)
		return 1
	}
	for _, entry := range entries {
		fmt.Println(entry.Line)
	}
	return 0
}

func runPluginRestart(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "plugin restart requires a plugin id")
		return 2
	}
	cfg := loadConfigForCLI()
	client := http.Client{Timeout: 10 * time.Second}
	endpoint := fmt.Sprintf("http://127.0.0.1:%d/api/plugins/%s/restart", cfg.Port, url.PathEscape(args[0]))
	response, err := client.Post(endpoint, "application/json", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "plugin restart failed: %v\n", err)
		return 1
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		fmt.Fprintf(os.Stderr, "plugin restart failed: HTTP %d\n", response.StatusCode)
		return 1
	}
	fmt.Printf("plugin restarted: %s\n", args[0])
	return 0
}

func runPluginUpdate(manager plugins.Manager, args []string) int {
	if len(args) > 0 {
		if err := updatePlugin(manager, args[0]); err != nil {
			fmt.Fprintf(os.Stderr, "plugin update failed: %v\n", err)
			return 1
		}
		fmt.Printf("plugin updated: %s\n", args[0])
		return 0
	}
	items, err := manager.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "plugin update failed: %v\n", err)
		return 1
	}
	if len(items) == 0 {
		fmt.Println("plugins: none installed")
		return 0
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprintln(os.Stderr, "plugin update requires a plugin id or an interactive terminal")
		return 2
	}
	index := 0
	for {
		renderPluginUpdate(items, index)
		key, err := readKey()
		if err != nil {
			fmt.Fprintf(os.Stderr, "plugin update failed: %v\n", err)
			return 1
		}
		switch key {
		case "up":
			if index > 0 {
				index--
			}
		case "down":
			if index < len(items)-1 {
				index++
			}
		case "enter":
			if err := updatePlugin(manager, items[index].ID); err != nil {
				fmt.Fprintf(os.Stderr, "\nplugin update failed: %v\n", err)
				return 1
			}
			fmt.Print("\033[2J\033[H")
			fmt.Printf("plugin updated: %s\n", items[index].ID)
			return 0
		case "q", "esc", "ctrl-c":
			fmt.Print("\033[2J\033[H")
			return 0
		}
	}
}

func printPlugins(items []plugins.Plugin) {
	for _, item := range items {
		state := "disabled"
		if item.Enabled {
			state = "enabled"
		}
		fmt.Printf("%-18s %-9s %s\n", item.ID, state, item.Version)
	}
}

func renderPluginList(items []plugins.Plugin, index int) {
	fmt.Print("\033[2J\033[H")
	fmt.Println("Plugins")
	fmt.Println()
	for i, item := range items {
		cursor := " "
		if i == index {
			cursor = ">"
		}
		state := "disabled"
		if item.Enabled {
			state = "enabled"
		}
		fmt.Printf("%s %-18s %-9s %s\n", cursor, item.ID, state, item.Version)
	}
	fmt.Println()
	fmt.Println("Enter toggle   q exit")
}

func renderPluginRemove(items []plugins.Plugin, index int) {
	fmt.Print("\033[2J\033[H")
	fmt.Println("Remove plugin")
	fmt.Println()
	for i, item := range items {
		cursor := " "
		if i == index {
			cursor = ">"
		}
		fmt.Printf("%s %-18s %s\n", cursor, item.ID, item.Version)
	}
	fmt.Println()
	fmt.Println("Enter remove   q cancel")
}

func renderPluginUpdate(items []plugins.Plugin, index int) {
	fmt.Print("\033[2J\033[H")
	fmt.Println("Update plugin")
	fmt.Println()
	for i, item := range items {
		cursor := " "
		if i == index {
			cursor = ">"
		}
		fmt.Printf("%s %-18s %s\n", cursor, item.ID, item.Version)
	}
	fmt.Println()
	fmt.Println("Enter update   q cancel")
}

func readKey() (string, error) {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	buffer := make([]byte, 8)
	n, err := os.Stdin.Read(buffer)
	if err != nil {
		return "", err
	}
	input := string(buffer[:n])
	switch {
	case input == "\r" || input == "\n":
		return "enter", nil
	case input == "q" || input == "Q":
		return "q", nil
	case input == "\x03":
		return "ctrl-c", nil
	case input == "\x1b":
		return "esc", nil
	case strings.Contains(input, "[A"):
		return "up", nil
	case strings.Contains(input, "[B"):
		return "down", nil
	default:
		return "", nil
	}
}

func printHelp() {
	fmt.Print(`Myrax server control panel

Usage:
  myrax [serve] [--host HOST] [--port PORT] [--config PATH]
  myrax configure --username USER --password-stdin [--panel-path PATH] [--port PORT]
  myrax add-ons enable|disable|status
  myrax plugin install <github-url|path>
  myrax plugin list
  myrax plugin remove
  myrax plugin enable
  myrax plugin disable
  myrax plugin logs <id>
  myrax plugin restart <id>
  myrax plugin update [id]
  myrax update
  myrax --version
  myrax --uninstall
  myrax --help

Commands:
  serve        Start the web panel and API server. This is the default command.
  configure    Write host, port, panel path, and admin credentials.
  add-ons      Enable or disable trusted add-ons globally.
  plugin       Install, list, remove, enable, and disable add-ons.
  update       Download the latest release, keep config/plugins, and restart service.

Options:
  --host       Bind host. Default: 127.0.0.1
  --port       Bind port. Default: 1487
  --config     Config path. Default: /etc/myrax/config.toml
`)
}

func runPluginStore(manager plugins.Manager) int {
	installed := map[string]string{}
	if items, err := manager.List(); err == nil {
		for _, item := range items {
			installed[item.ID] = item.Version
		}
	}
	printStore(installed)
	return 0
}

func printStore(installed map[string]string) {
	fmt.Println("\nplugin store:")
	for _, entry := range plugins.Store {
		status := ""
		if version, ok := installed[entry.ID]; ok {
			status = " [installed " + version + "]"
		}
		fmt.Printf("  %-10s %s%s\n", entry.ID, entry.Description, status)
	}
	fmt.Println("\ninstall with: myrax plugin install <name>")
}
