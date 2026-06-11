package plugins

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	ManifestFilePrimary   = "myrax.plugin.toml"
	ManifestFileSecondary = "plugin.toml"
)

type Manifest struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	Version string          `json:"version"`
	Myrax   string          `json:"myrax"`
	UI      UIManifest      `json:"ui"`
	Runtime RuntimeManifest `json:"runtime"`
	Access  AccessManifest  `json:"access"`
}

type UIManifest struct {
	Mode   string   `json:"mode"`
	Entry  string   `json:"entry"`
	Styles []string `json:"styles"`
}

type RuntimeManifest struct {
	Enabled   bool     `json:"enabled"`
	Command   string   `json:"command"`
	Args      []string `json:"args"`
	Port      int      `json:"port"`
	Transport string   `json:"transport"`
}

type AccessManifest struct {
	Level string `json:"level"`
}

func ReadManifest(dir string) (Manifest, error) {
	path := join(dir, ManifestFilePrimary)
	if !exists(path) {
		path = join(dir, ManifestFileSecondary)
	}
	if !exists(path) {
		return Manifest{}, fmt.Errorf("missing %s or %s", ManifestFilePrimary, ManifestFileSecondary)
	}

	file, err := os.Open(path)
	if err != nil {
		return Manifest{}, err
	}
	defer file.Close()

	var manifest Manifest
	section := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if commentAt := strings.Index(value, " #"); commentAt >= 0 {
			value = strings.TrimSpace(value[:commentAt])
		}

		switch section {
		case "":
			switch key {
			case "id":
				manifest.ID = stringValue(value)
			case "name":
				manifest.Name = stringValue(value)
			case "version":
				manifest.Version = stringValue(value)
			case "myrax":
				manifest.Myrax = stringValue(value)
			}
		case "ui":
			switch key {
			case "mode":
				manifest.UI.Mode = stringValue(value)
			case "entry":
				manifest.UI.Entry = stringValue(value)
			case "styles":
				manifest.UI.Styles = arrayValue(value)
			}
		case "runtime":
			switch key {
			case "enabled":
				manifest.Runtime.Enabled, _ = strconv.ParseBool(stringValue(value))
			case "command":
				manifest.Runtime.Command = stringValue(value)
			case "args":
				manifest.Runtime.Args = arrayValue(value)
			case "port":
				manifest.Runtime.Port, _ = strconv.Atoi(stringValue(value))
			case "transport":
				manifest.Runtime.Transport = stringValue(value)
			}
		case "access":
			if key == "level" {
				manifest.Access.Level = stringValue(value)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return Manifest{}, err
	}
	if manifest.ID == "" {
		return Manifest{}, fmt.Errorf("plugin id is required")
	}
	if !validID(manifest.ID) {
		return Manifest{}, fmt.Errorf("invalid plugin id: %s", manifest.ID)
	}
	if manifest.Name == "" {
		manifest.Name = manifest.ID
	}
	if manifest.Version == "" {
		manifest.Version = "0.0.0"
	}
	if manifest.UI.Mode == "" && manifest.UI.Entry != "" {
		manifest.UI.Mode = "native"
	}
	if manifest.Runtime.Transport == "" {
		manifest.Runtime.Transport = "tcp"
	}
	if manifest.Access.Level == "" {
		manifest.Access.Level = "trusted"
	}
	return manifest, nil
}

func stringValue(value string) string {
	return strings.Trim(strings.TrimSpace(value), `"`)
}

func arrayValue(value string) []string {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "[") || !strings.HasSuffix(value, "]") {
		item := stringValue(value)
		if item == "" {
			return nil
		}
		return []string{item}
	}
	inner := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "["), "]"))
	if inner == "" {
		return nil
	}
	parts := strings.Split(inner, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := stringValue(part)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

func validID(id string) bool {
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return id != "." && id != ".."
}
