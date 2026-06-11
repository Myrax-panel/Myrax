package plugins

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Manager struct {
	rootDir string
}

type Plugin struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Version     string          `json:"version"`
	Latest      string          `json:"latest,omitempty"`
	HasUpdate   bool            `json:"hasUpdate"`
	Enabled     bool            `json:"enabled"`
	Status      string          `json:"status"`
	Source      string          `json:"source"`
	Path        string          `json:"path"`
	InstalledAt string          `json:"installedAt"`
	UpdatedAt   string          `json:"updatedAt"`
	Error       string          `json:"error,omitempty"`
	UI          UIManifest      `json:"ui"`
	Runtime     RuntimeManifest `json:"runtime"`
	Access      AccessManifest  `json:"access"`
}

type LogEntry struct {
	Line string `json:"line"`
}

type State struct {
	Plugins map[string]PluginState `json:"plugins"`
}

type PluginState struct {
	Enabled     bool   `json:"enabled"`
	Source      string `json:"source"`
	InstalledAt string `json:"installedAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type UpdateInfo struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	CurrentVersion  string `json:"currentVersion"`
	LatestVersion   string `json:"latestVersion"`
	UpdateAvailable bool   `json:"updateAvailable"`
	Source          string `json:"source"`
	Error           string `json:"error,omitempty"`
}

func NewManager() Manager {
	return Manager{rootDir: defaultRootDir()}
}

func (m Manager) RootDir() string {
	return m.rootDir
}

func (m Manager) PluginDir(id string) string {
	return filepath.Join(m.installedDir(), id)
}

func (m Manager) DataDir(id string) string {
	return filepath.Join(m.rootDir, "data", id)
}

func (m Manager) LogsDir() string {
	return filepath.Join(m.rootDir, "logs")
}

func (m Manager) LogPath(id string) string {
	return filepath.Join(m.LogsDir(), id+".log")
}

func (m Manager) List() ([]Plugin, error) {
	state, err := m.readState()
	if err != nil {
		return nil, err
	}

	dir := m.installedDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Plugin{}, nil
		}
		return nil, err
	}

	plugins := make([]Plugin, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pluginDir := filepath.Join(dir, entry.Name())
		manifest, manifestErr := ReadManifest(pluginDir)
		itemState := state.Plugins[entry.Name()]
		plugin := Plugin{
			ID:          entry.Name(),
			Name:        entry.Name(),
			Enabled:     itemState.Enabled,
			Status:      "installed",
			Source:      itemState.Source,
			Path:        pluginDir,
			InstalledAt: itemState.InstalledAt,
			UpdatedAt:   itemState.UpdatedAt,
		}
		if manifestErr != nil {
			plugin.Status = "invalid"
			plugin.Error = manifestErr.Error()
		} else {
			plugin.ID = manifest.ID
			plugin.Name = manifest.Name
			plugin.Version = manifest.Version
			plugin.UI = manifest.UI
			plugin.Runtime = manifest.Runtime
			plugin.Access = manifest.Access
			if plugin.Enabled {
				plugin.Status = "enabled"
			}
		}
		plugins = append(plugins, plugin)
	}
	sort.Slice(plugins, func(i, j int) bool {
		return strings.ToLower(plugins[i].ID) < strings.ToLower(plugins[j].ID)
	})
	return plugins, nil
}

func (m Manager) Get(id string) (Plugin, error) {
	items, err := m.List()
	if err != nil {
		return Plugin{}, err
	}
	for _, item := range items {
		if item.ID == id {
			return item, nil
		}
	}
	return Plugin{}, fmt.Errorf("plugin not installed: %s", id)
}

func (m Manager) ReadLogs(id string, limit int) ([]LogEntry, error) {
	if strings.TrimSpace(id) == "" || !validID(id) {
		return nil, fmt.Errorf("invalid plugin id: %s", id)
	}
	if limit < 1 {
		limit = 200
	}
	if limit > 2000 {
		limit = 2000
	}
	data, err := os.ReadFile(m.LogPath(id))
	if err != nil {
		if os.IsNotExist(err) {
			return []LogEntry{}, nil
		}
		return nil, err
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	entries := make([]LogEntry, 0, limit)
	start := len(lines) - limit
	if start < 0 {
		start = 0
	}
	for _, line := range lines[start:] {
		if line != "" {
			entries = append(entries, LogEntry{Line: line})
		}
	}
	return entries, nil
}

func (m Manager) Install(source string) (Plugin, error) {
	return m.install(ResolveSource(source), nil)
}

func (m Manager) Update(id string) (Plugin, error) {
	id = strings.TrimSpace(id)
	if id == "" || !validID(id) {
		return Plugin{}, fmt.Errorf("invalid plugin id: %s", id)
	}
	state, err := m.readState()
	if err != nil {
		return Plugin{}, err
	}
	current, ok := state.Plugins[id]
	if !ok || strings.TrimSpace(current.Source) == "" {
		return Plugin{}, fmt.Errorf("plugin source is not available: %s", id)
	}
	manifest, err := m.inspectSource(current.Source)
	if err != nil {
		return Plugin{}, err
	}
	if manifest.ID != id {
		return Plugin{}, fmt.Errorf("plugin id changed from %s to %s", id, manifest.ID)
	}
	enabled := current.Enabled
	return m.install(current.Source, &enabled)
}

func (m Manager) CheckUpdates() ([]UpdateInfo, error) {
	items, err := m.List()
	if err != nil {
		return nil, err
	}
	updates := make([]UpdateInfo, 0, len(items))
	for _, item := range items {
		info := UpdateInfo{
			ID:             item.ID,
			Name:           item.Name,
			CurrentVersion: item.Version,
			LatestVersion:  item.Version,
			Source:         item.Source,
		}
		if strings.TrimSpace(item.Source) == "" {
			info.Error = "source is not available"
			updates = append(updates, info)
			continue
		}
		manifest, err := m.inspectSource(item.Source)
		if err != nil {
			info.Error = err.Error()
			updates = append(updates, info)
			continue
		}
		info.Name = manifest.Name
		info.LatestVersion = manifest.Version
		info.UpdateAvailable = versionChanged(item.Version, manifest.Version)
		updates = append(updates, info)
	}
	return updates, nil
}

func (m Manager) install(source string, forceEnabled *bool) (Plugin, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return Plugin{}, fmt.Errorf("plugin source is required")
	}

	tempDir, err := os.MkdirTemp("", "myrax-plugin-*")
	if err != nil {
		return Plugin{}, err
	}
	defer os.RemoveAll(tempDir)

	pluginDir, err := m.resolveSource(source, tempDir)
	if err != nil {
		return Plugin{}, err
	}
	manifest, err := ReadManifest(pluginDir)
	if err != nil {
		return Plugin{}, err
	}

	target := filepath.Join(m.installedDir(), manifest.ID)
	if err := os.MkdirAll(m.installedDir(), 0750); err != nil {
		return Plugin{}, err
	}
	if err := os.RemoveAll(target); err != nil {
		return Plugin{}, err
	}
	if err := copyDir(pluginDir, target); err != nil {
		return Plugin{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	state, err := m.readState()
	if err != nil {
		return Plugin{}, err
	}
	existing := state.Plugins[manifest.ID]
	if existing.InstalledAt == "" {
		existing.InstalledAt = now
	}
	if forceEnabled == nil {
		existing.Enabled = true
	} else {
		existing.Enabled = *forceEnabled
	}
	existing.Source = source
	existing.UpdatedAt = now
	state.Plugins[manifest.ID] = existing
	if err := m.writeState(state); err != nil {
		return Plugin{}, err
	}

	return Plugin{
		ID:          manifest.ID,
		Name:        manifest.Name,
		Version:     manifest.Version,
		Enabled:     existing.Enabled,
		Status:      statusForEnabled(existing.Enabled),
		Source:      source,
		Path:        target,
		InstalledAt: existing.InstalledAt,
		UpdatedAt:   existing.UpdatedAt,
		UI:          manifest.UI,
		Runtime:     manifest.Runtime,
		Access:      manifest.Access,
	}, nil
}

func (m Manager) inspectSource(source string) (Manifest, error) {
	tempDir, err := os.MkdirTemp("", "myrax-plugin-check-*")
	if err != nil {
		return Manifest{}, err
	}
	defer os.RemoveAll(tempDir)

	pluginDir, err := m.resolveSource(source, tempDir)
	if err != nil {
		return Manifest{}, err
	}
	return ReadManifest(pluginDir)
}

func (m Manager) SetEnabled(id string, enabled bool) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("plugin id is required")
	}
	if !exists(filepath.Join(m.installedDir(), id)) {
		return fmt.Errorf("plugin not installed: %s", id)
	}
	state, err := m.readState()
	if err != nil {
		return err
	}
	item := state.Plugins[id]
	item.Enabled = enabled
	item.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if item.InstalledAt == "" {
		item.InstalledAt = item.UpdatedAt
	}
	state.Plugins[id] = item
	return m.writeState(state)
}

func (m Manager) Remove(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("plugin id is required")
	}
	if !validID(id) {
		return fmt.Errorf("invalid plugin id: %s", id)
	}
	if err := os.RemoveAll(filepath.Join(m.installedDir(), id)); err != nil {
		return err
	}
	state, err := m.readState()
	if err != nil {
		return err
	}
	delete(state.Plugins, id)
	return m.writeState(state)
}

func (m Manager) readState() (State, error) {
	state := State{Plugins: map[string]PluginState{}}
	data, err := os.ReadFile(m.statePath())
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return state, err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return state, err
	}
	if state.Plugins == nil {
		state.Plugins = map[string]PluginState{}
	}
	return state, nil
}

func (m Manager) writeState(state State) error {
	if err := os.MkdirAll(m.rootDir, 0750); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.statePath(), append(data, '\n'), 0644)
}

func (m Manager) installedDir() string {
	return filepath.Join(m.rootDir, "installed")
}

func (m Manager) statePath() string {
	return filepath.Join(m.rootDir, "state.json")
}

func (m Manager) resolveSource(source string, tempDir string) (string, error) {
	if looksLikeGitHub(source) {
		return downloadGitHub(source, tempDir)
	}
	path := source
	if strings.HasPrefix(path, "file://") {
		path = strings.TrimPrefix(path, "file://")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("plugin path must be a directory: %s", source)
	}
	return filepath.Abs(path)
}

func copyDir(source string, target string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(target, 0750)
		}
		if entry.IsDir() && entry.Name() == ".git" {
			return filepath.SkipDir
		}
		dest := filepath.Join(target, rel)
		if entry.IsDir() {
			return os.MkdirAll(dest, 0750)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		return copyFile(path, dest, info.Mode())
	})
}

func copyFile(source string, target string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0750); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func statusForEnabled(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "installed"
}

func versionChanged(current string, latest string) bool {
	current = normalizeVersion(current)
	latest = normalizeVersion(latest)
	return current != "" && latest != "" && current != latest
}

func normalizeVersion(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.TrimPrefix(value, "v")
	return value
}

func join(parts ...string) string {
	return filepath.Join(parts...)
}
