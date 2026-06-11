<script>
  import { Button } from 'bits-ui';
  import { Download, FileText, Plug, Power, RefreshCw, Trash2 } from '@lucide/svelte';

  export let payload = { addOnsEnabled: false, plugins: [] };
  export let store = [];
  export let onInstall = async () => {};
  export let onSetEnabled = async () => {};
  export let onRemove = async () => {};
  export let onRestart = async () => {};
  export let onUpdate = async () => {};
  export let onReadLogs = async () => [];
  export let updates = [];
  export let onError = () => {};

  let source = '';
  let pending = '';
  let error = '';
  let logPlugin = '';
  let logEntries = [];
  let view = 'installed';

  $: plugins = payload?.plugins || [];
  $: runtimes = Object.fromEntries((payload?.runtimes || []).map((item) => [item.id, item]));
  $: updatesById = Object.fromEntries((updates || []).map((item) => [item.id, item]));

  async function install() {
    error = '';
    const nextSource = source.trim();
    if (!nextSource) {
      error = 'Source is required';
      onError(error);
      return;
    }
    pending = 'install';
    try {
      await onInstall(nextSource);
      source = '';
    } catch (requestError) {
      error = requestError.message;
      onError(error);
    } finally {
      pending = '';
    }
  }

  async function togglePlugin(plugin) {
    pending = `toggle:${plugin.id}`;
    error = '';
    try {
      await onSetEnabled(plugin.id, !plugin.enabled);
    } catch (requestError) {
      error = requestError.message;
      onError(error);
    } finally {
      pending = '';
    }
  }

  async function removePlugin(plugin) {
    pending = `remove:${plugin.id}`;
    error = '';
    try {
      await onRemove(plugin.id);
    } catch (requestError) {
      error = requestError.message;
      onError(error);
    } finally {
      pending = '';
    }
  }

  async function restartPlugin(plugin) {
    pending = `restart:${plugin.id}`;
    error = '';
    try {
      await onRestart(plugin.id);
    } catch (requestError) {
      error = requestError.message;
      onError(error);
    } finally {
      pending = '';
    }
  }

  async function updatePlugin(plugin) {
    pending = `update:${plugin.id}`;
    error = '';
    try {
      await onUpdate(plugin.id);
    } catch (requestError) {
      error = requestError.message;
      onError(error);
    } finally {
      pending = '';
    }
  }

  async function installFromStore(entry) {
    pending = `store:${entry.id}`;
    error = '';
    try {
      await onInstall(entry.id);
    } catch (requestError) {
      error = requestError.message;
      onError(error);
    } finally {
      pending = '';
    }
  }

  async function readLogs(plugin) {
    if (logPlugin === plugin.id) {
      logPlugin = '';
      logEntries = [];
      return;
    }
    pending = `logs:${plugin.id}`;
    error = '';
    try {
      logEntries = await onReadLogs(plugin.id);
      logPlugin = plugin.id;
    } catch (requestError) {
      error = requestError.message;
      onError(error);
    } finally {
      pending = '';
    }
  }
</script>

<section class="plugins-grid">
  <div class="view-tabs">
    <button class="view-tab" class:active={view === 'installed'} type="button" onclick={() => (view = 'installed')}>
      installed · {plugins.length}
    </button>
    <button class="view-tab" class:active={view === 'store'} type="button" onclick={() => (view = 'store')}>
      store · {store.length}
    </button>
  </div>

  {#if view === 'store'}
    <article class="settings-panel wide">
      <div class="panel-title">
        <h2>Plugin store</h2>
        <span>built-in catalog</span>
      </div>
      <div class="plugin-list">
        {#each store as entry}
          <div class="plugin-row">
            <div class="plugin-card-top">
              <strong>{entry.name}</strong>
              <Button.Root
                class="store-install-button"
                disabled={entry.installed || pending === `store:${entry.id}`}
                onclick={() => installFromStore(entry)}
              >
                <Download size="14" />
                {entry.installed ? 'installed' : pending === `store:${entry.id}` ? 'installing' : 'install'}
              </Button.Root>
            </div>
            <div class="plugin-main">
              {#if entry.description}
                <p>{entry.description}</p>
              {/if}
              <span>myrax plugin install {entry.id}</span>
              <div class="plugin-meta">
                {#each entry.tags || [] as tag}
                  <small>{tag}</small>
                {/each}
                {#if entry.installed}
                  <small class="running">installed {entry.installedVersion}</small>
                {/if}
              </div>
            </div>
          </div>
        {/each}
        {#if store.length === 0}
          <div class="empty-state">
            <Plug size="22" />
            <strong>store unavailable</strong>
          </div>
        {/if}
      </div>
    </article>
  {:else}
  <article class="settings-panel install-panel">
    <div class="panel-title">
      <h2>Install plugin</h2>
      <span>GitHub URL, server path or store name</span>
    </div>
    <div class="install-row">
      <input class:error-input={error} bind:value={source} placeholder="marzban / https://github.com/you/your-plugin" oninput={() => (error = '')} />
      <Button.Root class="install-button" disabled={pending === 'install'} onclick={install}>
        <Download size="16" />
        {pending === 'install' ? 'installing' : 'install'}
      </Button.Root>
    </div>
  </article>

  <article class="settings-panel wide">
    <div class="panel-title">
      <h2>Installed</h2>
      <span>{plugins.length} plugins</span>
    </div>

    {#if plugins.length === 0}
      <div class="empty-state">
        <Plug size="22" />
        <strong>no plugins installed</strong>
      </div>
    {:else}
      <div class="plugin-list">
        {#each plugins as plugin}
          {@const update = updatesById[plugin.id]}
          <div class="plugin-row" class:update-ready={update?.updateAvailable}>
            <div class="plugin-card-top">
              <strong>{plugin.name || plugin.id}</strong>
              <div class="plugin-actions">
                {#if update?.updateAvailable}
                  <Button.Root class="plugin-update-button" title={`Update ${plugin.id}`} aria-label={`Update ${plugin.id}`} disabled={pending === `update:${plugin.id}`} onclick={() => updatePlugin(plugin)}>
                    <Download size="14" />
                    <span>{pending === `update:${plugin.id}` ? 'updating' : 'update'}</span>
                  </Button.Root>
                {/if}
                <Button.Root class="mini-button" title={`Restart ${plugin.id}`} aria-label={`Restart ${plugin.id}`} disabled={pending === `restart:${plugin.id}` || !plugin.runtime?.command} onclick={() => restartPlugin(plugin)}>
                  <RefreshCw size="14" />
                </Button.Root>
                <Button.Root class="mini-button" title={`Open ${plugin.id} logs`} aria-label={`Open ${plugin.id} logs`} disabled={pending === `logs:${plugin.id}`} onclick={() => readLogs(plugin)}>
                  <FileText size="14" />
                </Button.Root>
                <Button.Root class="mini-button" title={`${plugin.enabled ? 'Disable' : 'Enable'} ${plugin.id}`} aria-label={`${plugin.enabled ? 'Disable' : 'Enable'} ${plugin.id}`} disabled={pending === `toggle:${plugin.id}`} onclick={() => togglePlugin(plugin)}>
                  <Power size="14" />
                </Button.Root>
                <Button.Root class="danger-button" title={`Remove ${plugin.id}`} aria-label={`Remove ${plugin.id}`} disabled={pending === `remove:${plugin.id}`} onclick={() => removePlugin(plugin)}>
                  <Trash2 size="15" />
                </Button.Root>
              </div>
            </div>
            <div class="plugin-main">
              <div class="plugin-name-row">
                {#if update?.updateAvailable}
                  <small class="update-badge">update {update.latestVersion}</small>
                {/if}
              </div>
              {#if plugin.description}
                <p>{plugin.description}</p>
              {/if}
              <span>{plugin.id} / {plugin.version || '0.0.0'} / {plugin.ui?.mode || 'native'}</span>
              <div class="plugin-meta">
                <small class:running={runtimes[plugin.id]?.running} class:failed={runtimes[plugin.id]?.status === 'failed'}>
                  runtime {runtimes[plugin.id]?.status || (plugin.runtime?.command ? 'stopped' : 'none')}
                </small>
                {#if plugin.source}
                  <small class="source-badge">source saved</small>
                {/if}
              </div>
              {#if plugin.error}<em>{plugin.error}</em>{/if}
              {#if update?.error}<em>{update.error}</em>{/if}
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </article>
  {/if}

  {#if logPlugin}
    <article class="settings-panel wide logs-panel">
      <div class="panel-title">
        <h2>{logPlugin} logs</h2>
        <span>{logEntries.length} lines</span>
      </div>
      <pre>{logEntries.map((entry) => entry.line).join('\n') || 'no logs'}</pre>
    </article>
  {/if}
</section>

<style>
  .plugins-grid {
    display: grid;
    gap: 16px;
  }

  .view-tabs {
    display: flex;
    gap: 8px;
  }

  .view-tab {
    min-height: 38px;
    padding: 0 18px;
    border: 1px solid var(--line);
    border-radius: 999px;
    color: var(--muted);
    background: var(--panel);
    font-size: 0.78rem;
    font-weight: 800;
    transition: color 160ms ease, background 160ms ease, border-color 160ms ease;
  }

  .view-tab:hover:not(.active) {
    color: var(--text);
    background: var(--panel-2);
  }

  .view-tab.active {
    color: #101010;
    background: #f5f5f5;
    border-color: #f5f5f5;
  }

  :global(.store-install-button) {
    min-height: 38px;
    padding: 0 16px;
    display: inline-flex;
    align-items: center;
    gap: 8px;
    border: 1px solid var(--line);
    border-radius: 999px;
    color: #101010;
    background: #f5f5f5;
    font-size: 0.74rem;
    font-weight: 900;
    white-space: nowrap;
    transition: color 160ms ease, background 160ms ease, border-color 160ms ease;
  }

  :global(.store-install-button:hover:not(:disabled)) {
    color: var(--text);
    background: #2a2b2f;
    border-color: var(--line);
  }

  :global(.store-install-button:disabled) {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .panel-title span,
  .plugin-row span {
    color: var(--muted);
    font-size: 0.72rem;
  }

  .plugin-row small {
    width: fit-content;
    padding: 4px 8px;
    border: 1px solid var(--line);
    border-radius: 999px;
    color: var(--muted);
    font-size: 0.66rem;
    font-weight: 800;
  }

  .plugin-row small.running {
    color: #101010;
    background: var(--ok);
    border-color: var(--ok);
  }

  .plugin-row small.failed {
    color: #101010;
    background: #ffd8d6;
    border-color: #ffd8d6;
  }

  .plugin-row small.update-badge {
    color: #101010;
    background: var(--accent-2);
    border-color: var(--accent-2);
  }

  .plugin-row small.source-badge {
    color: var(--muted);
    background: transparent;
  }

  .panel-title {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 14px;
    margin-bottom: 14px;
  }

  .install-panel,
  .settings-panel.wide {
    padding: 18px;
  }

  .install-row {
    display: grid;
    grid-template-columns: 1fr auto;
    align-items: stretch;
    gap: 10px;
  }

  input {
    width: 100%;
    height: 54px;
    min-height: 42px;
    padding: 0 13px;
    border: 1px solid var(--line);
    border-radius: 14px;
    color: var(--text);
    background: var(--panel-2);
    outline: none;
  }

  input:focus {
    border-color: var(--text);
  }

  input.error-input {
    border-color: #ff8f8a;
    background: color-mix(in srgb, #ffd8d6 14%, var(--panel-2));
    box-shadow: 0 0 0 3px rgb(255 143 138 / 12%);
  }

  :global(.install-button) {
    height: 54px;
    min-height: 54px;
    margin-top: 0;
    padding: 0 18px;
    border: 1px solid var(--line);
    border-radius: 14px;
    color: #101010;
    background: #f5f5f5;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    font-size: 0.78rem;
    font-weight: 900;
    line-height: 1;
    white-space: nowrap;
    transition:
      transform 160ms ease,
      color 160ms ease,
      background 160ms ease,
      border-color 160ms ease;
  }

  :global(.install-button:hover:not(:disabled)) {
    color: var(--text);
    background: #2a2b2f;
    border-color: var(--line);
  }

  :global(.install-button:disabled) {
    opacity: 0.45;
    cursor: not-allowed;
  }

  .plugin-list {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 14px;
  }

  .plugin-row {
    min-height: 126px;
    display: grid;
    grid-template-rows: auto 1fr;
    gap: 12px;
    padding: 16px;
    border: 1px solid var(--line-soft);
    border-radius: 18px;
    background: color-mix(in srgb, var(--panel-2) 44%, transparent);
    transition:
      border-color 160ms ease,
      background 160ms ease,
      transform 160ms ease;
  }

  .plugin-row.update-ready {
    border-color: color-mix(in srgb, var(--accent-2) 48%, var(--line));
    background: color-mix(in srgb, var(--accent-2) 8%, var(--panel-2));
  }

  .plugin-row:hover {
    border-color: var(--line);
    background: var(--panel-2);
  }

  .plugin-main {
    min-width: 0;
    display: grid;
    align-content: start;
    gap: 8px;
  }

  .plugin-card-top {
    min-width: 0;
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    align-items: start;
    gap: 12px;
  }

  .plugin-name-row,
  .plugin-meta {
    min-width: 0;
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 8px;
  }

  .plugin-row strong {
    color: var(--text);
    font-size: 0.95rem;
    line-height: 1.25;
    overflow-wrap: anywhere;
  }

  .plugin-row p {
    margin: 0;
    color: var(--text);
    opacity: 0.82;
    font-size: 0.78rem;
    line-height: 1.45;
  }

  .plugin-row em {
    color: #ffd8d6;
    font-size: 0.7rem;
    font-style: normal;
  }

  .plugin-actions {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    justify-content: flex-end;
    gap: 6px;
    max-width: 192px;
  }

  :global(.plugin-update-button) {
    width: 38px;
    min-width: 38px;
    height: 38px;
    padding: 0;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 0;
    border: 1px solid var(--accent-2);
    border-radius: 14px;
    color: #101010;
    background: var(--accent-2);
    font-size: 0.7rem;
    font-weight: 900;
    transition:
      color 160ms ease,
      background 160ms ease,
      border-color 160ms ease;
  }

  :global(.plugin-update-button span) {
    display: none;
  }

  :global(.plugin-update-button:hover:not(:disabled)) {
    color: var(--text);
    background: #2a2b2f;
    border-color: var(--line);
  }

  :global(.plugin-update-button:disabled) {
    opacity: 0.5;
    cursor: not-allowed;
  }

  :global(.plugin-actions .mini-button) {
    width: 38px;
    height: 38px;
    flex: 0 0 38px;
    padding: 0;
    justify-content: center;
    display: inline-flex;
    align-items: center;
    border-radius: 14px;
    overflow: hidden;
    font-size: 0;
    line-height: 0;
  }

  :global(.plugin-actions .mini-button svg),
  :global(.plugin-actions .danger-button svg) {
    flex: 0 0 auto;
  }

  :global(.danger-button) {
    width: 38px;
    height: 38px;
    flex: 0 0 38px;
    padding: 0;
    display: grid;
    place-items: center;
    border: 1px solid var(--line);
    border-radius: 14px;
    color: #101010;
    background: #ffd8d6;
    overflow: hidden;
    font-size: 0;
    line-height: 0;
  }

  .empty-state {
    min-height: 160px;
    display: grid;
    place-items: center;
    gap: 10px;
    color: var(--muted);
    text-align: center;
  }

  .logs-panel {
    padding: 18px;
  }

  .logs-panel pre {
    max-height: 280px;
    overflow: auto;
    margin: 0;
    padding: 14px;
    border: 1px solid var(--line-soft);
    border-radius: 16px;
    color: var(--text);
    background: var(--panel-2);
    font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
    font-size: 0.72rem;
    line-height: 1.55;
    white-space: pre-wrap;
  }

  @media (max-width: 980px) {
    .plugins-grid,
    .install-row {
      grid-template-columns: 1fr;
    }

    .plugin-list {
      grid-template-columns: 1fr;
    }

    .settings-panel.wide {
      grid-column: auto;
    }

    .plugin-actions {
      justify-content: flex-start;
    }
  }
</style>
