#!/bin/sh
set -eu

APP_NAME="myrax"
SERVICE_NAME="myrax"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/myrax"
DATA_DIR="/var/lib/myrax"
CONFIG_FILE="$CONFIG_DIR/config.toml"
DEFAULT_PORT="${MYRAX_PORT:-1487}"
DEFAULT_HOST="${MYRAX_HOST:-0.0.0.0}"
DEFAULT_PANEL_PATH="${MYRAX_PANEL_PATH:-/}"
VERSION="${MYRAX_VERSION:-latest}"
BASE_URL="${MYRAX_BASE_URL:-https://github.com/Myrax-panel/Myrax/releases}"

log() {
  printf '%s\n' "myrax: $*"
}

need_root() {
  if [ "$(id -u)" -ne 0 ]; then
    log "run this installer as root, for example:"
    log "curl -fsSL https://raw.githubusercontent.com/Myrax-panel/Myrax/main/scripts/install.sh | sudo sh"
    exit 1
  fi
}

detect_access_host() {
  case "$DEFAULT_HOST" in
    ""|0.0.0.0|::|\*) ;;
    *) printf '%s' "$DEFAULT_HOST"; return ;;
  esac

  if command -v ip >/dev/null 2>&1; then
    route_host="$(ip route get 1.1.1.1 2>/dev/null | awk '{
      for (i = 1; i <= NF; i++) {
        if ($i == "src") {
          print $(i + 1)
          exit
        }
      }
    }')"
    if [ -n "$route_host" ]; then
      printf '%s' "$route_host"
      return
    fi
  fi

  if command -v hostname >/dev/null 2>&1; then
    for host in $(hostname -I 2>/dev/null); do
      case "$host" in
        127.*|::1|169.254.*|fe80:*) continue ;;
      esac
      printf '%s' "$host"
      return
    done
  fi

  printf 'SERVER_IP'
}

format_url_host() {
  case "$1" in
    *:*) printf '[%s]' "$1" ;;
    *) printf '%s' "$1" ;;
  esac
}

normalize_panel_path() {
  value="${1:-/}"
  case "$value" in
    http://*|https://*) value="$(printf '%s' "$value" | sed 's#^[a-zA-Z][a-zA-Z0-9+.-]*://[^/]*##')" ;;
  esac
  value="/$(printf '%s' "$value" | sed 's#^/*##; s#/*$##')"
  case "$value" in
    "/"|"") printf '/' ;;
    *) printf '%s' "$value" ;;
  esac
}

tty_available() {
  [ -r /dev/tty ] && [ -w /dev/tty ]
}

menu_line() {
  printf '%s\n' "$*" >/dev/tty
}

menu_read() {
  prompt="$1"
  default="$2"
  menu_line ""
  printf '  %s' "$prompt" >/dev/tty
  if [ -n "$default" ]; then
    printf ' [%s]' "$default" >/dev/tty
  fi
  printf ': ' >/dev/tty
  IFS= read -r value </dev/tty || value=""
  if [ -z "$value" ]; then
    value="$default"
  fi
  printf '%s' "$value"
}

menu_read_secret() {
  prompt="$1"
  menu_line ""
  printf '  %s: ' "$prompt" >/dev/tty
  old_state="$(stty -g </dev/tty 2>/dev/null || true)"
  stty -echo </dev/tty 2>/dev/null || true
  IFS= read -r value </dev/tty || value=""
  if [ -n "$old_state" ]; then
    stty "$old_state" </dev/tty 2>/dev/null || true
  else
    stty echo </dev/tty 2>/dev/null || true
  fi
  menu_line ""
  printf '%s' "$value"
}

draw_install_menu() {
  if command -v clear >/dev/null 2>&1; then
    clear >/dev/tty 2>/dev/null || true
  fi
  menu_line "+------------------------------------------------------------+"
  menu_line "| myrax install                                              |"
  menu_line "+------------------------------------------------------------+"
  menu_line "| Configure panel access.                                   |"
  menu_line "| Default panel path is /. Login and password are required.  |"
  menu_line "+------------------------------------------------------------+"
}

config_value() {
  key="$1"
  [ -f "$CONFIG_FILE" ] || return 0
  awk -F= -v key="$key" '
    $1 ~ "^[[:space:]]*" key "[[:space:]]*$" {
      value = $2
      sub(/^[[:space:]]*/, "", value)
      sub(/[[:space:]]*$/, "", value)
      gsub(/^"|"$/, "", value)
      print value
      exit
    }
  ' "$CONFIG_FILE" 2>/dev/null || true
}

read_required() {
  prompt="$1"
  default="$2"
  while :; do
    value="$(menu_read "$prompt" "$default")"
    if [ -n "$value" ]; then
      printf '%s' "$value"
      return
    fi
    menu_line "  value is required"
  done
}

detect_arch() {
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64) printf 'amd64' ;;
    aarch64|arm64) printf 'arm64' ;;
    *) log "unsupported architecture: $arch"; exit 1 ;;
  esac
}

detect_init() {
  if command -v systemctl >/dev/null 2>&1 && [ -d /run/systemd/system ]; then
    printf 'systemd'
    return
  fi
  if command -v sv >/dev/null 2>&1 || [ -d /etc/sv ]; then
    printf 'runit'
    return
  fi
  log "unsupported init system; expected systemd or runit"
  exit 1
}

detect_pkg_manager() {
  if command -v apt-get >/dev/null 2>&1; then
    printf 'apt'
    return
  fi
  if command -v pacman >/dev/null 2>&1; then
    printf 'pacman'
    return
  fi
  if command -v xbps-install >/dev/null 2>&1; then
    printf 'xbps'
    return
  fi
  printf 'none'
}

install_dependencies() {
  pkg_manager="$(detect_pkg_manager)"
  case "$pkg_manager" in
    apt)
      apt-get update
      apt-get install -y ca-certificates curl
      ;;
    pacman)
      pacman -Sy --needed --noconfirm ca-certificates curl
      ;;
    xbps)
      xbps-install -Sy ca-certificates curl
      ;;
    none)
      command -v curl >/dev/null 2>&1 || {
        log "curl is required"
        exit 1
      }
      ;;
  esac
}

download_binary() {
  arch="$(detect_arch)"
  tmp_file="$(mktemp)"

  if [ -n "${MYRAX_BINARY_URL:-}" ]; then
    url="$MYRAX_BINARY_URL"
  elif [ "$VERSION" = "latest" ]; then
    url="$BASE_URL/latest/download/myrax-linux-$arch"
  else
    url="$BASE_URL/download/$VERSION/myrax-linux-$arch"
  fi

  log "downloading $url"
  curl -fsSL --retry 5 --retry-delay 2 "$url" -o "$tmp_file"
  chmod 0755 "$tmp_file"
  install -m 0755 "$tmp_file" "$INSTALL_DIR/$APP_NAME"
  rm -f "$tmp_file"
}

configure_panel() {
  mkdir -p "$CONFIG_DIR" "$DATA_DIR"
  chmod 0750 "$CONFIG_DIR" "$DATA_DIR"

  existing_auth=false
  if [ -f "$CONFIG_FILE" ] && grep -q '^auth_password_hash' "$CONFIG_FILE" 2>/dev/null; then
    existing_auth=true
  fi

  existing_host="$(config_value host)"
  existing_port="$(config_value port)"
  existing_panel_path="$(config_value panel_path)"
  existing_username="$(config_value auth_user)"

  host="${MYRAX_HOST:-${existing_host:-$DEFAULT_HOST}}"
  port="${MYRAX_PORT:-${existing_port:-$DEFAULT_PORT}}"
  keep_panel_path="$(normalize_panel_path "${existing_panel_path:-$DEFAULT_PANEL_PATH}")"
  panel_path="$(normalize_panel_path "${MYRAX_PANEL_PATH:-$DEFAULT_PANEL_PATH}")"
  username="${MYRAX_ADMIN_USER:-${existing_username:-admin}}"
  password="${MYRAX_ADMIN_PASSWORD:-}"

  if tty_available; then
    draw_install_menu
    if [ -f "$CONFIG_FILE" ] && [ "$existing_auth" = true ]; then
      menu_line ""
      menu_line "  Existing protected config found: $CONFIG_FILE"
      reconfigure="$(menu_read "Reconfigure panel access? yes/no" "yes")"
      case "$reconfigure" in
        y|Y|yes|YES|Yes) ;;
        *)
          DEFAULT_PORT="$port"
          DEFAULT_PANEL_PATH="$keep_panel_path"
          log "keeping existing $CONFIG_FILE"
          return
          ;;
      esac
    fi
    panel_path="$(normalize_panel_path "$(menu_read "Panel path or full URL" "$panel_path")")"
    port="$(read_required "Port" "$port")"
    username="$(read_required "Login" "$username")"
    while :; do
      password="$(menu_read_secret "Password")"
      confirm="$(menu_read_secret "Repeat password")"
      if [ -z "$password" ]; then
        menu_line "  password is required"
        continue
      fi
      if [ "${#password}" -lt 12 ]; then
        menu_line "  password must be at least 12 characters"
        continue
      fi
      if [ "$password" = "$confirm" ]; then
        break
      fi
      menu_line "  passwords do not match"
    done
  else
    if [ "$existing_auth" = true ] && [ -z "${MYRAX_FORCE_CONFIGURE:-}" ] && [ -z "$password" ]; then
      DEFAULT_PORT="$port"
      DEFAULT_PANEL_PATH="$keep_panel_path"
      log "keeping existing $CONFIG_FILE"
      return
    fi
    if [ -z "$username" ] || [ -z "$password" ]; then
      log "interactive terminal is required, or set MYRAX_ADMIN_USER and MYRAX_ADMIN_PASSWORD"
      exit 1
    fi
  fi

  case "$port" in
    ''|*[!0-9]*) log "invalid port: $port"; exit 1 ;;
  esac
  if [ "$port" -lt 1 ] || [ "$port" -gt 65535 ]; then
    log "invalid port: $port"
    exit 1
  fi
  if [ "${#password}" -lt 12 ]; then
    log "password must be at least 12 characters"
    exit 1
  fi

  printf '%s' "$password" | "$INSTALL_DIR/$APP_NAME" configure \
    --config "$CONFIG_FILE" \
    --host "$host" \
    --port "$port" \
    --panel-path "$panel_path" \
    --username "$username" \
    --password-stdin >/dev/null
  chmod 0600 "$CONFIG_FILE"
  DEFAULT_PORT="$port"
  DEFAULT_PANEL_PATH="$panel_path"
  log "configured $CONFIG_FILE"
}

install_systemd() {
  cat >/etc/systemd/system/$SERVICE_NAME.service <<EOF
[Unit]
Description=Myrax server control panel
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=$INSTALL_DIR/$APP_NAME serve --config $CONFIG_FILE
Restart=on-failure
RestartSec=3
User=root
Group=root
NoNewPrivileges=true
ProtectSystem=full
ReadWritePaths=$CONFIG_DIR $DATA_DIR
ProtectHome=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

  systemctl daemon-reload
  systemctl enable --now "$SERVICE_NAME.service"
}

install_runit() {
  mkdir -p /etc/sv/$SERVICE_NAME
  cat >/etc/sv/$SERVICE_NAME/run <<EOF
#!/bin/sh
exec $INSTALL_DIR/$APP_NAME serve --config $CONFIG_FILE
EOF
  chmod 0755 /etc/sv/$SERVICE_NAME/run
  ln -sfn /etc/sv/$SERVICE_NAME /var/service/$SERVICE_NAME
}

main() {
  need_root
  init_system="$(detect_init)"
  install_dependencies
  download_binary
  configure_panel

  case "$init_system" in
    systemd) install_systemd ;;
    runit) install_runit ;;
  esac

  log "installed on port $DEFAULT_PORT"
  access_host="$(format_url_host "$(detect_access_host)")"
  log "open http://$access_host:$DEFAULT_PORT$DEFAULT_PANEL_PATH"
}

main "$@"
