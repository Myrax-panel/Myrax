package server

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"myrax/internal/actions"
	"myrax/internal/auth"
	"myrax/internal/config"
	"myrax/internal/plugins"
	"myrax/internal/system"
	"myrax/internal/version"
	"myrax/internal/webassets"
)

const sessionCookieName = "myrax_session"

func Serve(cfg config.Config) error {
	mux := http.NewServeMux()
	pluginManager := plugins.NewManager()
	sessionManager, err := newSessionManager(cfg)
	if err != nil {
		return err
	}
	api := &apiServer{
		cfg:          cfg,
		plugins:      pluginManager,
		supervisor:   plugins.NewSupervisor(pluginManager, localCoreURL(cfg.Port)),
		sessions:     sessionManager,
		loginLimiter: auth.NewLoginLimiter(5, 5*time.Minute),
	}
	if cfg.AddOnsEnabled {
		if err := api.supervisor.StartEnabled(); err != nil {
			slog.Warn("plugin supervisor start failed", "error", err)
		}
	}

	mux.HandleFunc("GET /api/health", api.health)
	mux.HandleFunc("GET /api/session", api.session)
	mux.HandleFunc("POST /api/login", api.login)
	mux.HandleFunc("POST /api/logout", api.requireAuth(api.logout))
	mux.HandleFunc("GET /api/config", api.requireAuth(api.config))
	mux.HandleFunc("PUT /api/config", api.requireAuth(api.updateConfig))
	mux.HandleFunc("GET /api/stats", api.requireAuth(api.stats))
	mux.HandleFunc("GET /api/events/stats", api.requireAuth(api.statsEvents))
	mux.HandleFunc("GET /api/logs", api.requireAuth(api.logs))
	mux.HandleFunc("GET /api/events/logs", api.requireAuth(api.logsEvents))
	mux.HandleFunc("GET /api/processes", api.requireAuth(api.processes))
	mux.HandleFunc("POST /api/processes/kill", api.requireAuth(api.killProcess))
	mux.HandleFunc("GET /api/services", api.requireAuth(api.services))
	mux.HandleFunc("POST /api/services/action", api.requireAuth(api.serviceAction))
	mux.HandleFunc("GET /api/network", api.requireAuth(api.network))
	mux.HandleFunc("GET /api/disks", api.requireAuth(api.disks))
	mux.HandleFunc("GET /api/plugins", api.requireAuth(api.pluginsList))
	mux.HandleFunc("GET /api/updates", api.requireAuth(api.updates))
	mux.HandleFunc("GET /api/plugins/store", api.requireAuth(api.pluginsStore))
	mux.HandleFunc("POST /api/plugins/install", api.requireAuth(api.pluginsInstall))
	mux.HandleFunc("POST /api/plugins/enable", api.requireAuth(api.pluginsEnable))
	mux.HandleFunc("POST /api/plugins/disable", api.requireAuth(api.pluginsDisable))
	mux.HandleFunc("POST /api/plugins/remove", api.requireAuth(api.pluginsRemove))
	mux.HandleFunc("GET /api/plugins/{id}/logs", api.requireAuth(api.pluginLogs))
	mux.HandleFunc("POST /api/plugins/{id}/restart", api.requireAuth(api.pluginRestart))
	mux.HandleFunc("POST /api/plugins/{id}/update", api.requireAuth(api.pluginUpdate))
	mux.HandleFunc("/api/plugins/{id}/proxy/{path...}", api.requireAuth(api.pluginProxy))
	mux.HandleFunc("/api/plugins/{id}/ws/{path...}", api.requireAuth(api.pluginWebSocketProxy))
	mux.HandleFunc("POST /api/actions/reboot", api.requireAuth(api.reboot))
	mux.HandleFunc("POST /api/actions/shutdown", api.requireAuth(api.shutdown))
	mux.HandleFunc("POST /api/actions/reload", api.requireAuth(api.reload))
	mux.HandleFunc("POST /api/actions/update", api.requireAuth(api.updateCore))
	mux.HandleFunc("GET /addons/", api.requireAuth(api.pluginAssets))
	mux.Handle("GET /assets/", staticHandler())
	mux.HandleFunc("/", api.panel)

	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	srv := http.Server{
		Addr:              addr,
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	slog.Info("myrax listening", "addr", addr)
	return srv.ListenAndServe()
}

type apiServer struct {
	mu           sync.Mutex
	cfg          config.Config
	plugins      plugins.Manager
	supervisor   *plugins.Supervisor
	sessions     *auth.SessionManager
	loginLimiter *auth.LoginLimiter
}

func (a *apiServer) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"service": "myrax",
	})
}

func (a *apiServer) session(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	cfg := a.cfg
	a.mu.Unlock()
	if !cfg.AuthConfigured() || a.sessions == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"authenticated": false,
			"configured":    false,
		})
		return
	}
	session, ok := a.sessionFromRequest(r)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"authenticated": false,
			"configured":    true,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"configured":    true,
		"username":      session.Username,
	})
}

func (a *apiServer) login(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	cfg := a.cfg
	a.mu.Unlock()
	if !cfg.AuthConfigured() || a.sessions == nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("authentication is not configured"))
		return
	}
	key := remoteIP(r)
	if !a.loginLimiter.Allow(key) {
		writeError(w, http.StatusTooManyRequests, fmt.Errorf("too many login attempts"))
		return
	}

	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	validPassword, err := auth.VerifyPassword(payload.Password, cfg.AuthPasswordHash)
	valid := err == nil && subtleStringEqual(payload.Username, cfg.AuthUsername) && validPassword
	if !valid {
		a.loginLimiter.RecordFailure(key)
		time.Sleep(350 * time.Millisecond)
		writeError(w, http.StatusUnauthorized, fmt.Errorf("invalid username or password"))
		return
	}

	token, session, err := a.sessions.Create(cfg.AuthUsername)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	a.loginLimiter.Clear(key)
	setSessionCookie(w, r, token, session.ExpiresAt)
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"username":      session.Username,
		"expiresAt":     session.ExpiresAt.Format(time.RFC3339),
	})
}

func (a *apiServer) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil && a.sessions != nil {
		a.sessions.Delete(cookie.Value)
	}
	clearSessionCookie(w, r)
	writeJSON(w, http.StatusOK, map[string]bool{"authenticated": false})
}

func (a *apiServer) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a.mu.Lock()
		authConfigured := a.cfg.AuthConfigured()
		a.mu.Unlock()
		if !authConfigured || a.sessions == nil {
			writeError(w, http.StatusServiceUnavailable, fmt.Errorf("authentication is not configured"))
			return
		}
		if _, ok := a.sessionFromRequest(r); !ok {
			writeError(w, http.StatusUnauthorized, fmt.Errorf("authentication required"))
			return
		}
		next(w, r)
	}
}

func (a *apiServer) sessionFromRequest(r *http.Request) (auth.Session, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || a.sessions == nil {
		return auth.Session{}, false
	}
	return a.sessions.Get(cookie.Value)
}

func (a *apiServer) config(w http.ResponseWriter, _ *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"host":          a.cfg.Host,
		"port":          a.cfg.Port,
		"panelPath":     a.cfg.PanelPath,
		"addOnsEnabled": a.cfg.AddOnsEnabled,
		"authUsername":  a.cfg.AuthUsername,
		"configPath":    a.cfg.ConfigPath,
	})
}

func (a *apiServer) updateConfig(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Host          *string `json:"host"`
		Port          *int    `json:"port"`
		PanelPath     *string `json:"panelPath"`
		AddOnsEnabled *bool   `json:"addOnsEnabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	a.mu.Lock()
	next := a.cfg
	a.mu.Unlock()
	if payload.Host != nil {
		next.Host = strings.TrimSpace(*payload.Host)
	}
	if payload.Port != nil {
		next.Port = *payload.Port
	}
	if payload.PanelPath != nil {
		next.PanelPath = config.NormalizePanelPath(*payload.PanelPath)
	}
	if payload.AddOnsEnabled != nil {
		next.AddOnsEnabled = *payload.AddOnsEnabled
	}

	if err := config.SaveFile(next.ConfigPath, next); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	a.mu.Lock()
	previous := a.cfg
	a.cfg = next
	a.mu.Unlock()

	if payload.AddOnsEnabled != nil && previous.AddOnsEnabled != next.AddOnsEnabled {
		if next.AddOnsEnabled {
			if err := a.supervisor.StartEnabled(); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
		} else {
			a.supervisor.StopAll()
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"host":          next.Host,
		"port":          next.Port,
		"panelPath":     next.PanelPath,
		"addOnsEnabled": next.AddOnsEnabled,
		"authUsername":  next.AuthUsername,
		"configPath":    next.ConfigPath,
		"reload":        true,
	})
}

func (a *apiServer) stats(w http.ResponseWriter, _ *http.Request) {
	stats, err := system.ReadStats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (a *apiServer) statsEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, errors.New("streaming unsupported"))
		return
	}

	ticker := time.NewTicker(eventInterval(r, 2*time.Second))
	defer ticker.Stop()

	for {
		stats, err := system.ReadStats()
		if err == nil {
			payload, _ := json.Marshal(stats)
			_, _ = fmt.Fprintf(w, "event: stats\ndata: %s\n\n", payload)
			flusher.Flush()
		}

		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

func eventInterval(r *http.Request, fallback time.Duration) time.Duration {
	interval, err := time.ParseDuration(r.URL.Query().Get("interval"))
	if err != nil {
		return fallback
	}
	if interval < time.Second {
		return time.Second
	}
	if interval > 30*time.Second {
		return 30 * time.Second
	}
	return interval
}

func (a *apiServer) logs(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	level := r.URL.Query().Get("level")
	query := r.URL.Query().Get("query")
	limit := queryInt(r, "limit", 120)

	entries, err := system.ReadLogs(source, level, query, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (a *apiServer) logsEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, errors.New("streaming unsupported"))
		return
	}

	ticker := time.NewTicker(eventInterval(r, 2*time.Second))
	defer ticker.Stop()

	for {
		entries, err := system.ReadLogs(r.URL.Query().Get("source"), r.URL.Query().Get("level"), r.URL.Query().Get("query"), queryInt(r, "limit", 120))
		if err == nil {
			payload, _ := json.Marshal(entries)
			_, _ = fmt.Fprintf(w, "event: logs\ndata: %s\n\n", payload)
			flusher.Flush()
		}

		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

func (a *apiServer) processes(w http.ResponseWriter, r *http.Request) {
	processes, err := system.ReadProcesses(queryInt(r, "limit", 80))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, processes)
}

func (a *apiServer) killProcess(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PID int `json:"pid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := system.KillProcess(payload.PID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]bool{"accepted": true})
}

func (a *apiServer) services(w http.ResponseWriter, r *http.Request) {
	services, err := system.ReadServices(queryInt(r, "limit", 120))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, services)
}

func (a *apiServer) serviceAction(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Name   string `json:"name"`
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := system.RunServiceAction(payload.Name, payload.Action); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]bool{"accepted": true})
}

func (a *apiServer) network(w http.ResponseWriter, _ *http.Request) {
	network, err := system.ReadNetworkDetails()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, network)
}

func (a *apiServer) disks(w http.ResponseWriter, _ *http.Request) {
	disks, err := system.ReadDiskDetails()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, disks)
}

func (a *apiServer) pluginsStore(w http.ResponseWriter, _ *http.Request) {
	installed := map[string]string{}
	if items, err := a.plugins.List(); err == nil {
		for _, item := range items {
			installed[item.ID] = item.Version
		}
	}
	type storeItem struct {
		plugins.StoreEntry
		Installed        bool   `json:"installed"`
		InstalledVersion string `json:"installedVersion,omitempty"`
	}
	entries := make([]storeItem, 0, len(plugins.Store))
	for _, entry := range plugins.Store {
		version, ok := installed[entry.ID]
		entries = append(entries, storeItem{StoreEntry: entry, Installed: ok, InstalledVersion: version})
	}
	writeJSON(w, http.StatusOK, map[string]any{"plugins": entries})
}

func (a *apiServer) pluginsList(w http.ResponseWriter, _ *http.Request) {
	items, err := a.plugins.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	a.mu.Lock()
	addOnsEnabled := a.cfg.AddOnsEnabled
	a.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"addOnsEnabled": addOnsEnabled,
		"plugins":       items,
		"runtimes":      a.supervisor.Statuses(),
	})
}

func (a *apiServer) updates(w http.ResponseWriter, _ *http.Request) {
	core := coreUpdateInfo()
	pluginUpdates, err := a.plugins.CheckUpdates()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"core":    core,
		"plugins": pluginUpdates,
	})
}

func (a *apiServer) pluginsInstall(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Source string `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := a.plugins.Install(payload.Source)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	a.mu.Lock()
	addOnsEnabled := a.cfg.AddOnsEnabled
	a.mu.Unlock()
	if addOnsEnabled && item.Runtime.Enabled && item.Runtime.Command != "" {
		_ = a.supervisor.Stop(item.ID)
		if err := a.supervisor.Start(item.ID); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *apiServer) pluginsEnable(w http.ResponseWriter, r *http.Request) {
	a.pluginsSetEnabled(w, r, true)
}

func (a *apiServer) pluginsDisable(w http.ResponseWriter, r *http.Request) {
	a.pluginsSetEnabled(w, r, false)
}

func (a *apiServer) pluginsSetEnabled(w http.ResponseWriter, r *http.Request, enabled bool) {
	var payload struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := a.plugins.SetEnabled(payload.ID, enabled); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	a.mu.Lock()
	addOnsEnabled := a.cfg.AddOnsEnabled
	a.mu.Unlock()
	if enabled && addOnsEnabled {
		if err := a.supervisor.Start(payload.ID); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	if !enabled {
		if err := a.supervisor.Stop(payload.ID); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	writeJSON(w, http.StatusAccepted, map[string]bool{"accepted": true})
}

func (a *apiServer) pluginsRemove(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := a.supervisor.Stop(payload.ID); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := a.plugins.Remove(payload.ID); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	a.supervisor.Forget(payload.ID)
	writeJSON(w, http.StatusAccepted, map[string]bool{"accepted": true})
}

func (a *apiServer) pluginLogs(w http.ResponseWriter, r *http.Request) {
	entries, err := a.plugins.ReadLogs(r.PathValue("id"), queryInt(r, "limit", 200))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (a *apiServer) pluginRestart(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	a.mu.Lock()
	addOnsEnabled := a.cfg.AddOnsEnabled
	a.mu.Unlock()
	if !addOnsEnabled {
		writeError(w, http.StatusBadRequest, fmt.Errorf("add-ons are disabled"))
		return
	}
	if err := a.supervisor.Restart(id); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusAccepted, a.supervisor.Status(id))
}

func (a *apiServer) pluginUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_ = a.supervisor.Stop(id)
	item, err := a.plugins.Update(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	a.supervisor.Forget(id)
	a.mu.Lock()
	addOnsEnabled := a.cfg.AddOnsEnabled
	a.mu.Unlock()
	if addOnsEnabled && item.Enabled && item.Runtime.Enabled && item.Runtime.Command != "" {
		if err := a.supervisor.Start(item.ID); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	writeJSON(w, http.StatusAccepted, item)
}

func (a *apiServer) pluginProxy(w http.ResponseWriter, r *http.Request) {
	target, err := a.supervisor.ProxyTarget(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	proxy := httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = target.Address
			req.URL.Path = "/" + r.PathValue("path")
			req.Host = target.Address
		},
	}
	if target.Transport == "unix" {
		proxy.Transport = &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				dialer := net.Dialer{}
				return dialer.DialContext(ctx, "unix", target.Address)
			},
		}
	}
	proxy.ServeHTTP(w, r)
}

func (a *apiServer) pluginWebSocketProxy(w http.ResponseWriter, r *http.Request) {
	target, err := a.supervisor.ProxyTarget(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	targetURL := "ws://" + target.Address + "/" + r.PathValue("path")
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}
	dialer := *websocket.DefaultDialer
	if target.Transport == "unix" {
		dialer.NetDialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
			netDialer := net.Dialer{}
			return netDialer.DialContext(ctx, "unix", target.Address)
		}
		targetURL = "ws://unix/" + r.PathValue("path")
		if r.URL.RawQuery != "" {
			targetURL += "?" + r.URL.RawQuery
		}
	}
	upstreamConn, _, err := dialer.Dial(targetURL, nil)
	if err != nil {
		_ = clientConn.Close()
		return
	}
	plugins.PipeWebSockets(r.Context(), clientConn, upstreamConn)
}

func (a *apiServer) pluginAssets(w http.ResponseWriter, r *http.Request) {
	relative := strings.TrimPrefix(r.URL.Path, "/addons/")
	parts := strings.SplitN(relative, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		http.NotFound(w, r)
		return
	}
	root := filepath.Join(a.plugins.RootDir(), "installed", parts[0])
	target := filepath.Join(root, filepath.FromSlash(parts[1]))
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	cleanTarget, err := filepath.Abs(target)
	if err != nil || !strings.HasPrefix(cleanTarget, cleanRoot+string(os.PathSeparator)) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid addon path"))
		return
	}
	http.ServeFile(w, r, cleanTarget)
}

func queryInt(r *http.Request, key string, fallback int) int {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return n
}

func (a *apiServer) reboot(w http.ResponseWriter, r *http.Request) {
	if err := actions.Reboot(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]bool{"accepted": true})
}

func (a *apiServer) shutdown(w http.ResponseWriter, r *http.Request) {
	if err := actions.Shutdown(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]bool{"accepted": true})
}

func (a *apiServer) reload(w http.ResponseWriter, r *http.Request) {
	if err := actions.Reload(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]bool{"accepted": true})
}

func (a *apiServer) updateCore(w http.ResponseWriter, r *http.Request) {
	if err := actions.UpdateLatestDetached(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]bool{"accepted": true})
}

func coreUpdateInfo() map[string]any {
	latest, err := latestReleaseVersion()
	info := map[string]any{
		"currentVersion":  version.Version,
		"latestVersion":   latest,
		"updateAvailable": latest != "" && normalizeReleaseVersion(latest) != normalizeReleaseVersion(version.Version),
		"downloadUrl":     "",
		"releasePage":     version.GitHubReleasesPage,
		"error":           "",
	}
	if latest != "" {
		info["downloadUrl"] = version.GitHubLatestDownloadURL(runtime.GOARCH)
	}
	if err != nil {
		info["error"] = err.Error()
		info["updateAvailable"] = false
	}
	return info
}

func latestReleaseVersion() (string, error) {
	request, err := http.NewRequest(http.MethodGet, version.GitHubReleasesAPI, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "myrax-panel")
	client := http.Client{Timeout: 12 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("release check failed: %s", response.Status)
	}
	var releases []struct {
		TagName string `json:"tag_name"`
		Draft   bool   `json:"draft"`
	}
	if err := json.NewDecoder(response.Body).Decode(&releases); err != nil {
		return "", err
	}
	for _, release := range releases {
		if release.Draft {
			continue
		}
		if strings.TrimSpace(release.TagName) != "" {
			return release.TagName, nil
		}
	}
	return "", fmt.Errorf("latest release tag is empty")
}

func normalizeReleaseVersion(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.TrimPrefix(value, "v")
	return value
}

func staticHandler() http.Handler {
	dist, err := fs.Sub(webassets.Dist, "dist")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(dist))
}

func (a *apiServer) panel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writePanelNotFound(w, r)
		return
	}
	a.mu.Lock()
	panelPath := config.NormalizePanelPath(a.cfg.PanelPath)
	a.mu.Unlock()
	if panelPath == "/" {
		if r.URL.Path != "/" {
			writePanelNotFound(w, r)
			return
		}
	} else if r.URL.Path != panelPath && r.URL.Path != panelPath+"/" {
		writePanelNotFound(w, r)
		return
	}
	index, err := webassets.Dist.ReadFile("dist/index.html")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(index)
}

func writePanelNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = fmt.Fprintf(w, `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>404</title>
  <link rel="preconnect" href="https://fonts.googleapis.com" />
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
  <link href="https://fonts.googleapis.com/css2?family=Unbounded:wght@400;600;800&display=swap" rel="stylesheet" />
  <style>
    :root {
      --bg: #101010;
      --panel: #161616;
      --panel-2: #202124;
      --line: #34363b;
      --line-soft: #25262a;
      --text: #f4f4f5;
      --muted: #8d929c;
      --accent: #bae7ff;
      --warn: #fff6bd;
      --danger: #ffd8d6;
      color-scheme: dark;
      font-family: "Unbounded", ui-sans-serif, system-ui, sans-serif;
    }
    * { box-sizing: border-box; }
    body {
      min-width: 320px;
      min-height: 100vh;
      margin: 0;
      display: grid;
      place-items: center;
      padding: 24px;
      color: var(--text);
      background: var(--bg);
      text-rendering: geometricPrecision;
    }
    main {
      width: min(760px, 100%%);
      display: grid;
      gap: 16px;
    }
    .code {
      width: fit-content;
      padding: 6px 10px;
      border: 1px solid var(--line);
      border-radius: 999px;
      color: var(--muted);
      background: var(--panel);
      font-size: 0.72rem;
      font-weight: 800;
    }
    .card {
      position: relative;
      overflow: hidden;
      padding: clamp(22px, 5vw, 44px);
      border: 1px solid var(--line);
      border-radius: 32px;
      background: var(--panel);
      box-shadow: 0 28px 100px rgb(0 0 0 / 38%%);
    }
    .card::before,
    .card::after {
      content: "";
      position: absolute;
      border: 1px solid var(--line-soft);
      border-radius: 999px;
      pointer-events: none;
    }
    .card::before {
      width: 220px;
      height: 220px;
      right: -82px;
      top: -86px;
    }
    .card::after {
      width: 120px;
      height: 120px;
      right: 78px;
      bottom: -62px;
    }
    .digits {
      position: relative;
      z-index: 1;
      display: grid;
      grid-template-columns: repeat(3, minmax(74px, 1fr));
      gap: 10px;
      margin-bottom: 26px;
    }
    .digits span {
      min-height: clamp(94px, 17vw, 148px);
      display: grid;
      place-items: center;
      border: 1px solid var(--line);
      border-radius: 24px;
      color: #101010;
      background: var(--text);
      font-size: clamp(3.2rem, 13vw, 7.2rem);
      font-weight: 800;
      line-height: 1;
    }
    .digits span:nth-child(2) {
      color: #101010;
      background: var(--accent);
      transform: translateY(14px) rotate(-2deg);
    }
    .digits span:nth-child(3) {
      background: var(--warn);
      transform: translateY(-8px) rotate(2deg);
    }
    h1 {
      position: relative;
      z-index: 1;
      max-width: 620px;
      margin: 0;
      color: var(--text);
      font-size: clamp(2rem, 7vw, 4.7rem);
      line-height: 0.96;
      letter-spacing: 0;
    }
    p {
      position: relative;
      z-index: 1;
      max-width: 560px;
      margin: 14px 0 0;
      color: var(--muted);
      font-size: 0.82rem;
      font-weight: 600;
      line-height: 1.65;
    }
    .path {
      position: relative;
      z-index: 1;
      width: fit-content;
      max-width: 100%%;
      margin-top: 18px;
      overflow: hidden;
      padding: 10px 12px;
      border: 1px solid var(--line-soft);
      border-radius: 14px;
      color: var(--danger);
      background: var(--panel-2);
      font-size: 0.72rem;
      font-weight: 700;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
    @media (max-width: 560px) {
      body { padding: 14px; }
      .card { border-radius: 24px; }
      .digits { grid-template-columns: 1fr; }
      .digits span:nth-child(2),
      .digits span:nth-child(3) {
        transform: none;
      }
    }
  </style>
</head>
<body>
  <main>
    <div class="code">404 / route missing</div>
    <section class="card" aria-labelledby="title">
      <div class="digits" aria-hidden="true"><span>4</span><span>0</span><span>4</span></div>
      <h1 id="title">Panel is not here</h1>
      <p>The requested address is not the configured panel route. Use the install URL from this server.</p>
      <div class="path">%s</div>
    </section>
  </main>
</body>
</html>`, html.EscapeString(r.URL.Path))
}

func newSessionManager(cfg config.Config) (*auth.SessionManager, error) {
	if !cfg.AuthConfigured() {
		return nil, nil
	}
	return auth.NewSessionManager(cfg.SessionSecret, 12*time.Hour)
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   requestIsHTTPS(r),
	})
}

func clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   requestIsHTTPS(r),
	})
}

func requestIsHTTPS(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}

func subtleStringEqual(left, right string) bool {
	if len(left) != len(right) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func localCoreURL(port int) string {
	return fmt.Sprintf("http://127.0.0.1:%d", port)
}
