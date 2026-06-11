<p align="center">
  <img src="docs/banner.png" alt="Myrax — lightweight Linux server control panel" />
</p>

# Myrax

Lightweight control panel for Linux servers. One Go binary, no database, no docker —
realtime metrics, processes, services, network, disk and a plugin system, all in a fast web UI.

<p>
  <a href="https://myrax.app"><img src="https://img.shields.io/badge/website-myrax.app-bae7ff?style=for-the-badge&labelColor=101010" alt="Website" /></a>
  <a href="https://docs.myrax.app"><img src="https://img.shields.io/badge/docs-docs.myrax.app-c4f7d1?style=for-the-badge&labelColor=101010" alt="Docs" /></a>
  <a href="https://github.com/Myrax-panel/Myrax/releases/latest"><img src="https://img.shields.io/github/v/release/Myrax-panel/Myrax?style=for-the-badge&labelColor=101010&color=fff6bd&label=release" alt="Latest release" /></a>
</p>

## Install

On a fresh server (root required):

```sh
curl -fsSL https://raw.githubusercontent.com/Myrax-panel/Myrax/main/scripts/install.sh | sudo sh
```
or
```sh
curl -fsSL https://sh.myrax.app/install.sh | sudo sh
```

The installer downloads the binary, asks for a port, panel path and admin credentials,
then registers a service (systemd or runit). Open `http://ip:port` and log in.

Supported: Linux amd64 / arm64 · debian, ubuntu, arch, void and anything else with systemd or runit.

To update later — press **update** in the panel header, or run:

```sh
myrax update
```

## What's inside

- **Overview** — CPU / RAM / disk gauges with live sparklines, network throughput, top processes, host info. Cards turn yellow/red when a metric crosses its threshold.
- **Processes** — sort, inspect, kill.
- **Services** — start / stop / restart systemd units.
- **Network & Disk** — per-interface rates, mounted volumes, usage.
- **Logs** — journal tail in the browser.
- **Control** — reboot, shutdown, self-update.
- **Plugins** — trusted add-ons with their own pages and backend runtimes.
- Dark and light theme, mobile layout.

## Plugins

Built-in store entries install by name:

```sh
myrax plugin install marzban    # manage a Marzban panel from Myrax
myrax plugin install terminal   # full PTY shell in the browser
```

Any other plugin installs from a GitHub URL or a local path:

```sh
myrax plugin install https://github.com/Myrax/name-plugin
```

Useful commands:

| Command | What it does |
|---|---|
| `myrax plugin store` | show the built-in catalog |
| `myrax plugin list` | list installed plugins (interactive enable/disable) |
| `myrax plugin update <id>` | update a plugin from its source |
| `myrax plugin remove` | remove a plugin |
| `myrax add-ons enable` | turn the plugin system on |

The same store is available in the panel: **Plugins → store**.

## CLI

| Command | What it does |
|---|---|
| `myrax serve` | run the panel (what the service unit runs) |
| `myrax configure` | set bind, port, panel path and credentials |
| `myrax update` | self-update from the latest GitHub release |
| `myrax version` | print version |
| `myrax uninstall` | remove binary, service and data |

Config lives in `/etc/myrax/config.toml`, plugin data in `/var/lib/myrax`.

## Building from source

The web UI is built first and embedded into the Go binary, so the order matters:
**web → go**. Requirements: Go 1.26+, Node 20+.

The binary always targets Linux — Windows and macOS work fine as build hosts
via cross-compilation.

### On Linux / macOS

```sh
git clone https://github.com/Myrax-panel/Myrax
cd Myrax

# 1. build the web UI (output goes to internal/webassets/dist)
cd web
npm install
npm run build
cd ..

# 2. build the binary
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o bin/myrax-linux-amd64 ./cmd/myrax

# arm64 variant
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o bin/myrax-linux-arm64 ./cmd/myrax
```

### On Windows (PowerShell)

```powershell
git clone https://github.com/Myrax-panel/Myrax
cd Myrax

# 1. build the web UI
cd web
npm install
npm run build
cd ..

# 2. cross-compile for Linux
$env:CGO_ENABLED = "0"; $env:GOOS = "linux"; $env:GOARCH = "amd64"
go build -trimpath -ldflags="-s -w" -o bin/myrax-linux-amd64 ./cmd/myrax

# arm64 variant
$env:GOARCH = "arm64"
go build -trimpath -ldflags="-s -w" -o bin/myrax-linux-arm64 ./cmd/myrax
```

Copy the binary to a server and run it, or install it over an existing setup:

```sh
sudo install -m 0755 myrax-linux-amd64 /usr/local/bin/myrax
sudo systemctl restart myrax
```

### Dev loop

```sh
cd web && npm run dev        # vite dev server for the UI
go run ./cmd/myrax serve     # backend (run on a Linux machine / WSL)
```

## Project layout

```
cmd/myrax/          entry point
internal/server/    HTTP API + SSE
internal/system/    metrics collectors (/proc, /sys)
internal/plugins/   plugin manager, store catalog, supervisor
internal/webassets/ embedded web build
web/                Svelte UI
scripts/install.sh  installer
```

## Screens

<p>
  <img src="docs/overview.png" alt="Overview" />
  <img src="docs/processes.png" alt="Processes" />
  <img src="docs/control.png" alt="Control" />
  <img src="docs/settings.png" alt="Settings" />
</p>
