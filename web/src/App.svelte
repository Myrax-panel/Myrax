<script>
  import { onMount } from 'svelte';
  import { Collapsible, Tabs } from 'bits-ui';
  import { Activity, Boxes, FileText, HardDrive, ListTree, Network, Plug, Power, RefreshCw, Settings } from '@lucide/svelte';
  import AuthScreen from './components/AuthScreen.svelte';
  import BootLoader from './components/BootLoader.svelte';
  import BottomNav from './components/BottomNav.svelte';
  import ControlScreen from './components/ControlScreen.svelte';
  import DiskScreen from './components/DiskScreen.svelte';
  import LogsScreen from './components/LogsScreen.svelte';
  import NetworkScreen from './components/NetworkScreen.svelte';
  import OverviewScreen from './components/OverviewScreen.svelte';
  import PluginPageHost from './components/PluginPageHost.svelte';
  import PluginsScreen from './components/PluginsScreen.svelte';
  import ProcessesScreen from './components/ProcessesScreen.svelte';
  import SettingsScreen from './components/SettingsScreen.svelte';
  import Sidebar from './components/Sidebar.svelte';
  import ServicesScreen from './components/ServicesScreen.svelte';
  import Topbar from './components/Topbar.svelte';
  import ToastHost from './components/ToastHost.svelte';
  import { showToast } from './lib/toast.js';
  import { pushMetrics } from './lib/history.js';

  let stats = null;
  let config = null;
  let error = '';
  let currentTab = 'overview';
  let sidebarOpen = true;
  let pendingAction = '';
  let pluginsPayload = { addOnsEnabled: false, plugins: [] };
  let storePlugins = [];
  let updatesPayload = { core: null, plugins: [] };
  let coreUpdatePending = false;
  let pluginPages = [];
  let pluginNavItems = [];
  let removedNavItems = [];
  let loadedPluginKeys = new Set();
  let refreshRate = '2s';
  let themeMode = 'system';
  let accessUrl = '';
  let reloadUrl = '';
  let events = null;
  let eventsStarted = false;
  let systemThemeQuery = null;
  let systemThemeHandler = null;
  let authChecked = false;
  let authenticated = false;
  let authConfigured = true;
  let authPending = false;
  let authError = '';

  const navItems = [
    { value: 'overview', label: 'Overview', icon: Activity },
    { value: 'logs', label: 'Logs', icon: FileText },
    { value: 'processes', label: 'Processes', icon: ListTree },
    { value: 'services', label: 'Services', icon: Boxes },
    { value: 'network', label: 'Network', icon: Network },
    { value: 'disk', label: 'Disk', icon: HardDrive },
    { value: 'plugins', label: 'Plugins', icon: Plug },
    { value: 'control', label: 'Control', icon: Power },
    { value: 'settings', label: 'Settings', icon: Settings }
  ];

  const refreshOptions = [
    { value: '1s', label: '1s' },
    { value: '2s', label: '2s' },
    { value: '5s', label: '5s' }
  ];

  const themeOptions = [
    { value: 'system', label: 'system' },
    { value: 'dark', label: 'dark' },
    { value: 'light', label: 'light' }
  ];

  const actions = [
    { id: 'reboot', label: 'Reboot', icon: RefreshCw },
    { id: 'shutdown', label: 'Shutdown', icon: Power }
  ];

  async function loadInitial() {
    try {
      const [statsResponse, configResponse, pluginsResponse] = await Promise.all([
        fetch('/api/stats'),
        fetch('/api/config'),
        fetch('/api/plugins')
      ]);
      for (const response of [statsResponse, configResponse, pluginsResponse]) {
        if (!response.ok) {
          const payload = await response.json().catch(() => ({}));
          const responseError = new Error(payload.error || `HTTP ${response.status}`);
          responseError.status = response.status;
          throw responseError;
        }
      }
      stats = await statsResponse.json();
      sampleStats(stats);
      config = await configResponse.json();
      pluginsPayload = await pluginsResponse.json();
      error = '';
      loadStore();
      loadNativePlugins();
      checkUpdates();
    } catch (requestError) {
      if (requestError.status === 401) {
        authenticated = false;
        events?.close();
        eventsStarted = false;
      } else {
        error = requestError.message || '';
      }
    }
  }

  async function checkSession() {
    const response = await fetch('/api/session');
    const payload = await response.json();
    if (!response.ok) {
      throw new Error(payload.error || `HTTP ${response.status}`);
    }
    authConfigured = payload.configured !== false;
    authenticated = Boolean(payload.authenticated);
    return authenticated;
  }

  async function login(username, password) {
    authPending = true;
    authError = '';
    try {
      const response = await fetch('/api/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password })
      });
      const payload = await response.json();
      if (!response.ok) {
        throw new Error(payload.error || `HTTP ${response.status}`);
      }
      authenticated = true;
      authConfigured = true;
      await loadInitial();
      eventsStarted = true;
      connectEvents(refreshRate);
    } catch (requestError) {
      authError = requestError.message;
    } finally {
      authPending = false;
    }
  }

  async function logout() {
    await fetch('/api/logout', { method: 'POST' }).catch(() => {});
    events?.close();
    eventsStarted = false;
    authenticated = false;
    stats = null;
    config = null;
    pluginsPayload = { addOnsEnabled: false, plugins: [] };
    updatesPayload = { core: null, plugins: [] };
    pluginPages = [];
    pluginNavItems = [];
    removedNavItems = [];
    loadedPluginKeys = new Set();
    currentTab = 'overview';
  }

  function sampleStats(next) {
    const adapters = next?.network || [];
    pushMetrics({
      cpu: next?.cpu?.usage || 0,
      ram: next?.memory?.usage || 0,
      disk: next?.disks?.[0]?.usage || 0,
      netRx: adapters.reduce((sum, item) => sum + (item.rxRate || 0), 0),
      netTx: adapters.reduce((sum, item) => sum + (item.txRate || 0), 0)
    });
  }

  function connectEvents(interval) {
    events?.close();
    events = new EventSource(`/api/events/stats?interval=${encodeURIComponent(interval)}`);
    events.addEventListener('stats', (event) => {
      stats = JSON.parse(event.data);
      sampleStats(stats);
    });
  }

  async function runAction(name) {
    pendingAction = name;
    error = '';

    try {
      const response = await fetch(`/api/actions/${name}`, { method: 'POST' });
      if (!response.ok) {
        const payload = await response.json();
        throw new Error(payload.error || `HTTP ${response.status}`);
      }
    } catch (requestError) {
      error = requestError.message;
    } finally {
      pendingAction = '';
    }
  }

  async function saveConfig(nextConfig) {
    const previous = config;
    const response = await fetch('/api/config', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(nextConfig)
    });
    const payload = await response.json();
    if (!response.ok) {
      throw new Error(payload.error || `HTTP ${response.status}`);
    }
    config = payload;
    reloadUrl = buildAccessUrl(payload);
    pluginsPayload = { ...pluginsPayload, addOnsEnabled: Boolean(payload.addOnsEnabled) };
    // Host and port changes require a server reload.
    if ((previous?.host || '') !== payload.host || Number(previous?.port || 0) !== Number(payload.port || 0)) {
      showReloadToast();
    } else {
      await loadPlugins();
      showToast({
        title: payload.addOnsEnabled ? 'Add-ons enabled' : 'Add-ons disabled',
        message: payload.addOnsEnabled ? 'Trusted plugins can now extend the panel.' : 'Installed plugins are paused.',
        timeout: 2600
      });
    }
  }

  async function loadPlugins() {
    const response = await fetch('/api/plugins');
    const payload = await response.json();
    if (!response.ok) {
      throw new Error(payload.error || `HTTP ${response.status}`);
    }
    pluginsPayload = payload;
    loadStore();
    loadNativePlugins();
  }

  async function loadStore() {
    try {
      const response = await fetch('/api/plugins/store');
      if (!response.ok) {
        return;
      }
      const payload = await response.json();
      storePlugins = payload.plugins || [];
    } catch {
      // The store list is decorative; the next plugins refresh retries.
    }
  }

  async function checkUpdates() {
    try {
      const response = await fetch('/api/updates');
      const payload = await response.json();
      if (!response.ok) {
        throw new Error(payload.error || `HTTP ${response.status}`);
      }
      updatesPayload = payload;
    } catch (requestError) {
      showToast({ title: 'Update check failed', message: requestError.message, tone: 'error', timeout: 4200 });
    }
  }

  async function installPlugin(source) {
    const response = await fetch('/api/plugins/install', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ source })
    });
    const payload = await response.json();
    if (!response.ok) {
      throw new Error(payload.error || `HTTP ${response.status}`);
    }
    await loadPlugins();
    checkUpdates();
    showToast({ title: 'Plugin installed', message: `${payload.name || payload.id} enabled.`, timeout: 2600 });
  }

  async function updatePlugin(id) {
    const response = await fetch(`/api/plugins/${encodeURIComponent(id)}/update`, { method: 'POST' });
    const payload = await response.json();
    if (!response.ok) {
      throw new Error(payload.error || `HTTP ${response.status}`);
    }
    await loadPlugins();
    checkUpdates();
    showToast({ title: 'Plugin updated', message: `${payload.name || payload.id} is now ${payload.version}.`, timeout: 2600 });
    window.setTimeout(() => {
      window.location.reload();
    }, 700);
  }

  async function setPluginEnabled(id, enabled) {
    const response = await fetch(`/api/plugins/${enabled ? 'enable' : 'disable'}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id })
    });
    const payload = await response.json();
    if (!response.ok) {
      throw new Error(payload.error || `HTTP ${response.status}`);
    }
    await loadPlugins();
  }

  async function removePlugin(id) {
    const response = await fetch('/api/plugins/remove', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id })
    });
    const payload = await response.json();
    if (!response.ok) {
      throw new Error(payload.error || `HTTP ${response.status}`);
    }
    pluginPages = pluginPages.filter((item) => item.pluginId !== id);
    pluginNavItems = pluginNavItems.filter((item) => item.pluginId !== id);
    await loadPlugins();
  }

  async function restartPlugin(id) {
    const response = await fetch(`/api/plugins/${encodeURIComponent(id)}/restart`, { method: 'POST' });
    const payload = await response.json();
    if (!response.ok) {
      throw new Error(payload.error || `HTTP ${response.status}`);
    }
    await loadPlugins();
  }

  async function readPluginLogs(id) {
    const response = await fetch(`/api/plugins/${encodeURIComponent(id)}/logs?limit=160`);
    const payload = await response.json();
    if (!response.ok) {
      throw new Error(payload.error || `HTTP ${response.status}`);
    }
    return payload;
  }

  async function reloadCore() {
    const target = reloadUrl || buildAccessUrl(config);
    await fetch('/api/actions/reload', { method: 'POST' });
    window.setTimeout(() => {
      window.location.href = target;
    }, 1400);
  }

  async function readResponseError(response) {
    const text = await response.text().catch(() => '');
    if (!text) {
      return `HTTP ${response.status}`;
    }

    try {
      return JSON.parse(text).error || text;
    } catch {
      return text;
    }
  }

  function delay(ms) {
    return new Promise((resolve) => window.setTimeout(resolve, ms));
  }

  function isTransientUpdateResponse(response) {
    return response.status === 502 || response.status === 503 || response.status === 504;
  }

  function isTransientUpdateError(error) {
    const message = String(error?.message || error || '');
    return /Failed to fetch|NetworkError|Load failed|Network request failed|fetch|HTTP 502|HTTP 503|HTTP 504/i.test(message);
  }

  async function waitForUpdatedCore(previousVersion, timeoutMs = 15000) {
    const startedAt = Date.now();

    while (Date.now() - startedAt < timeoutMs) {
      try {
        const response = await fetch('/api/updates', { cache: 'no-store' });
        if (response.ok) {
          const payload = await response.json();
          const currentVersion = payload?.core?.currentVersion;
          if (!previousVersion || (currentVersion && currentVersion !== previousVersion)) {
            return true;
          }
        }
      } catch {
        // Expected while systemd restarts Myrax.
      }

      await delay(1200);
    }

    return false;
  }

  async function updateCore() {
    const previousVersion = updatesPayload?.core?.currentVersion;
    coreUpdatePending = true;

    // A dropped request usually means systemd has started replacing the binary.
    try {
      const response = await fetch('/api/actions/update', { method: 'POST' });
      if (!response.ok) {
        if (isTransientUpdateResponse(response)) {
          throw new Error(`HTTP ${response.status}`);
        }
        throw new Error(await readResponseError(response));
      }
    } catch (requestError) {
      if (!isTransientUpdateError(requestError)) {
        showToast({ title: 'Update failed', message: requestError.message, tone: 'error', timeout: 5000 });
        coreUpdatePending = false;
        return;
      }
    }

    try {
      showToast({ title: 'Myrax update started', message: 'Core binary is being replaced and service will restart.', timeout: 3600 });
      const isUpdated = await waitForUpdatedCore(previousVersion);
      if (!isUpdated) {
        window.location.reload();
        return;
      }
      window.location.reload();
    } finally {
      coreUpdatePending = false;
    }
  }

  function showReloadToast() {
    showToast({
      title: 'Требуется перезагрузка ядра',
      message: 'Bind и port применятся после reload.',
      actionLabel: 'Reload',
      timeout: 0,
      onAction: reloadCore
    });
  }

  // Plugin-owned DOM is tagged for cleanup on unload or update.
  function createPluginAPI(plugin) {
    const pluginPrefix = `plugin:${plugin.id}:`;
    const api = {
      id: plugin.id,
      plugin,
      sidebar: {
        add(item) {
          addPluginPage(plugin, item);
        },
        remove(value) {
          removedNavItems = [...new Set([...removedNavItems, value])];
        }
      },
      routes: {
        add(route) {
          addPluginPage(plugin, route);
        }
      },
      styles: {
        inject(css) {
          const style = document.createElement('style');
          style.dataset.myraxPlugin = plugin.id;
          style.textContent = css;
          document.head.appendChild(style);
        },
        load(href) {
          const link = document.createElement('link');
          link.rel = 'stylesheet';
          link.dataset.myraxPlugin = plugin.id;
          if (href.startsWith('http')) {
            link.href = href;
          } else {
            // Keep plugin CSS in sync with the versioned JS entry.
            const cacheKey = encodeURIComponent(plugin.updatedAt || plugin.version || '');
            const path = `/addons/${plugin.id}/${href.replace(/^\/+/, '')}`;
            link.href = cacheKey ? `${path}?v=${cacheKey}` : path;
          }
          document.head.appendChild(link);
        }
      },
      layout: {
        hide(selector) {
          api.styles.inject(`${selector}{display:none!important}`);
        },
        show(selector) {
          api.styles.inject(`${selector}{display:revert!important}`);
        }
      },
      toast: showToast,
      refresh: loadInitial,
      navigate(value) {
        currentTab = value.startsWith(pluginPrefix) ? value : `${pluginPrefix}${value}`;
      }
    };
    return api;
  }

  function resolvePluginIcon(plugin, icon) {
    if (!icon || typeof icon !== 'string') {
      return icon || Plug;
    }
    const trimmed = icon.trim();
    if (!trimmed) {
      return Plug;
    }
    if (/^(<svg|<\?xml)/i.test(trimmed) || /^(https?:|data:|\/)/i.test(trimmed)) {
      return trimmed;
    }
    return `/addons/${plugin.id}/${trimmed.replace(/^\/+/, '')}`;
  }

  function addPluginPage(plugin, page) {
    const pageValue = `plugin:${plugin.id}:${page.value || page.path || page.label || 'page'}`;
    const icon = resolvePluginIcon(plugin, page.icon);
    const entry = {
      value: pageValue,
      pluginId: plugin.id,
      label: page.label || plugin.name || plugin.id,
      icon,
      render: page.render,
      html: page.html,
      mount: page.mount
    };
    pluginPages = [...pluginPages.filter((item) => item.value !== pageValue), entry];
    pluginNavItems = [
      ...pluginNavItems.filter((item) => item.value !== pageValue),
      { value: pageValue, label: entry.label, icon, pluginId: plugin.id }
    ];
  }

  // Synchronize mounted plugins with the server state.
  async function loadNativePlugins() {
    if (!pluginsPayload?.addOnsEnabled) {
      pluginPages = [];
      pluginNavItems = [];
      removedNavItems = [];
      loadedPluginKeys = new Set();
      document.querySelectorAll('[data-myrax-plugin]').forEach((node) => node.remove());
      return;
    }
    const enabledPlugins = pluginsPayload.plugins?.filter((plugin) => plugin.enabled && plugin.ui?.entry) || [];
    const enabledIds = new Set(enabledPlugins.map((plugin) => plugin.id));
    pluginPages = pluginPages.filter((item) => enabledIds.has(item.pluginId));
    pluginNavItems = pluginNavItems.filter((item) => enabledIds.has(item.pluginId));
    document.querySelectorAll('[data-myrax-plugin]').forEach((node) => {
      if (!enabledIds.has(node.dataset.myraxPlugin)) {
        node.remove();
      }
    });
    loadedPluginKeys = new Set([...loadedPluginKeys].filter((key) => enabledIds.has(key.split(':')[0])));
    window.myrax = window.myrax || {};
    for (const plugin of enabledPlugins) {
      const key = `${plugin.id}:${plugin.version}:${plugin.updatedAt || ''}`;
      if (loadedPluginKeys.has(key)) {
        continue;
      }
      document.querySelectorAll(`[data-myrax-plugin="${plugin.id}"]`).forEach((node) => node.remove());
      loadedPluginKeys = new Set([...loadedPluginKeys].filter((item) => item.split(':')[0] !== plugin.id));
      loadedPluginKeys.add(key);
      for (const style of plugin.ui?.styles || []) {
        createPluginAPI(plugin).styles.load(style);
      }
      try {
        const module = await import(/* @vite-ignore */ `/addons/${plugin.id}/${plugin.ui.entry}?v=${encodeURIComponent(plugin.updatedAt || plugin.version)}`);
        const api = createPluginAPI(plugin);
        window.myrax[plugin.id] = api;
        if (typeof module.register === 'function') {
          await module.register(api);
        }
      } catch (pluginError) {
        showToast({ title: `Plugin failed: ${plugin.id}`, message: pluginError.message, timeout: 5000 });
      }
    }
  }

  function applyTheme(mode) {
    const resolved = mode === 'system' ? (systemThemeQuery?.matches ? 'light' : 'dark') : mode;
    document.documentElement.dataset.theme = resolved;
    document.documentElement.style.colorScheme = resolved;
  }

  function buildAccessUrl(nextConfig) {
    const url = new URL(window.location.href);
    if (nextConfig?.port) {
      url.port = String(nextConfig.port);
    }
    url.pathname = nextConfig?.panelPath || '/';
    url.search = '';
    url.hash = '';
    return url.toString().replace(/\/$/, nextConfig?.panelPath && nextConfig.panelPath !== '/' ? '' : '/');
  }

  $: refreshLabel = refreshOptions.find((item) => item.value === refreshRate)?.label || '2s';
  $: themeLabel = themeOptions.find((item) => item.value === themeMode)?.label || 'system';
  $: allNavItems = [...navItems, ...pluginNavItems].filter((item) => !removedNavItems.includes(item.value));
  // Start live side effects only after authentication succeeds.
  $: if (eventsStarted) {
    connectEvents(refreshRate);
  }
  $: if (eventsStarted) {
    localStorage.setItem('myrax-theme', themeMode);
    applyTheme(themeMode);
  }
  $: if (eventsStarted) {
    accessUrl = buildAccessUrl(config);
  }

  onMount(() => {
    themeMode = localStorage.getItem('myrax-theme') || 'system';
    systemThemeQuery = window.matchMedia('(prefers-color-scheme: light)');
    systemThemeHandler = () => applyTheme(themeMode);
    systemThemeQuery.addEventListener('change', systemThemeHandler);
    applyTheme(themeMode);
    accessUrl = buildAccessUrl(config);
    checkSession()
      .then(async (ok) => {
        authChecked = true;
        if (ok) {
          await loadInitial();
          eventsStarted = true;
        }
      })
      .catch((requestError) => {
        authChecked = true;
        authError = requestError.message;
      });

    return () => {
      events?.close();
      systemThemeQuery?.removeEventListener('change', systemThemeHandler);
    };
  });
</script>

<svelte:head>
  <title>Myrax Panel</title>
  <link rel="preconnect" href="https://fonts.googleapis.com" />
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin="anonymous" />
  <link
    href="https://fonts.googleapis.com/css2?family=Unbounded:wght@400;500;600;700;800&display=swap"
    rel="stylesheet"
  />
</svelte:head>

{#if !authChecked}
  <BootLoader />
{:else if !authenticated}
  <AuthScreen configured={authConfigured} error={authError} pending={authPending} onLogin={login} />
{:else}
  <Tabs.Root bind:value={currentTab}>
    <Collapsible.Root bind:open={sidebarOpen}>
      <div class="shell" class:sidebar-open={sidebarOpen}>
        <Sidebar navItems={allNavItems} bind:sidebarOpen />

        <main class="main">
          <Topbar
            hostname={stats?.host?.hostname || 'Linux Node'}
            coreUpdate={updatesPayload?.core}
            updatePending={coreUpdatePending}
            onUpdate={updateCore}
            onRefresh={loadInitial}
            onLogout={logout}
          />

        {#if error}
          <div class="error-line">{error}</div>
        {/if}

        <Tabs.Content value="overview" class="screen">
          <OverviewScreen {stats} {refreshRate} />
        </Tabs.Content>

        <Tabs.Content value="logs" class="screen">
          <LogsScreen {refreshRate} />
        </Tabs.Content>

        <Tabs.Content value="processes" class="screen">
          <ProcessesScreen {refreshRate} />
        </Tabs.Content>

        <Tabs.Content value="services" class="screen">
          <ServicesScreen {refreshRate} />
        </Tabs.Content>

        <Tabs.Content value="network" class="screen">
          <NetworkScreen {refreshRate} />
        </Tabs.Content>

        <Tabs.Content value="disk" class="screen">
          <DiskScreen {refreshRate} />
        </Tabs.Content>

        <Tabs.Content value="plugins" class="screen">
          <PluginsScreen
            payload={pluginsPayload}
            store={storePlugins}
            onInstall={installPlugin}
            onSetEnabled={setPluginEnabled}
            onRemove={removePlugin}
            onRestart={restartPlugin}
            onUpdate={updatePlugin}
            onReadLogs={readPluginLogs}
            updates={updatesPayload?.plugins || []}
            onError={(message) => showToast({ title: 'Plugin error', message, tone: 'danger' })}
          />
        </Tabs.Content>

        <Tabs.Content value="control" class="screen">
          <ControlScreen {actions} {pendingAction} onRunAction={runAction} />
        </Tabs.Content>

        <Tabs.Content value="settings" class="screen">
          <SettingsScreen
            {config}
            bind:refreshRate
            bind:themeMode
            {refreshLabel}
            {refreshOptions}
            {themeLabel}
            {themeOptions}
            {accessUrl}
            addOnsEnabled={Boolean(config?.addOnsEnabled)}
            onSaveConfig={saveConfig}
            onSetAddOns={(enabled) => saveConfig({ ...config, addOnsEnabled: enabled })}
          />
        </Tabs.Content>

        {#each pluginPages as page}
          <Tabs.Content value={page.value} class="screen">
            <PluginPageHost {page} />
          </Tabs.Content>
        {/each}
        </main>

        <BottomNav navItems={allNavItems} />
        <ToastHost />
      </div>
    </Collapsible.Root>
  </Tabs.Root>
{/if}
