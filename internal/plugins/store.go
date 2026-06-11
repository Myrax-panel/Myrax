package plugins

import "strings"

type StoreEntry struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Source      string   `json:"source"`
	Tags        []string `json:"tags"`
}

var Store = []StoreEntry{
	{
		ID:          "marzban",
		Name:        "Marzban",
		Description: "Manage users, nodes, hosts, admins and the Xray core of an existing Marzban panel.",
		Source:      "https://github.com/Myrax-panel/Marzban",
		Tags:        []string{"vpn", "xray", "management"},
	},
	{
		ID:          "terminal",
		Name:        "Terminal",
		Description: "Full PTY shell in the browser. xterm.js frontend over a websocket transport.",
		Source:      "https://github.com/Myrax-panel/terminal",
		Tags:        []string{"shell", "pty", "xterm"},
	},
}

func StoreLookup(name string) (StoreEntry, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, entry := range Store {
		if entry.ID == name {
			return entry, true
		}
	}
	return StoreEntry{}, false
}

func ResolveSource(source string) string {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" || strings.ContainsAny(trimmed, "/\\:.") {
		return source
	}
	if entry, ok := StoreLookup(trimmed); ok {
		return entry.Source
	}
	return source
}
