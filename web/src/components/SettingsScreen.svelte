<script>
  import { Button, Label, Select, Separator } from 'bits-ui';
  import { ChevronDown } from '@lucide/svelte';

  export let refreshRate = '2s';
  export let refreshLabel = '2s';
  export let refreshOptions = [];
  export let themeMode = 'system';
  export let themeLabel = 'system';
  export let themeOptions = [];
  export let config = null;
  export let accessUrl = '';
  export let addOnsEnabled = false;
  export let onSaveConfig = async () => {};
  export let onSetAddOns = async () => {};

  let hostValue = '0.0.0.0';
  let portValue = 1487;
  let configKey = '';
  let pending = false;
  let error = '';

  $: nextConfigKey = `${config?.host || '0.0.0.0'}:${config?.port || 1487}`;
  $: if (nextConfigKey !== configKey) {
    hostValue = config?.host || '0.0.0.0';
    portValue = config?.port || 1487;
    configKey = nextConfigKey;
  }
  $: dirty = hostValue !== (config?.host || '0.0.0.0') || Number(portValue) !== Number(config?.port || 1487);

  async function saveConfig() {
    error = '';
    const port = Number(portValue);
    if (!hostValue.trim()) {
      error = 'Bind is required';
      return;
    }
    if (!Number.isInteger(port) || port < 1 || port > 65535) {
      error = 'Port must be 1-65535';
      return;
    }

    pending = true;
    try {
      await onSaveConfig({ host: hostValue.trim(), port, addOnsEnabled });
    } catch (requestError) {
      error = requestError.message;
    } finally {
      pending = false;
    }
  }
</script>

<section class="settings-grid">
  <article class="settings-panel">
    <div class="setting-row">
      <Label.Root>refresh</Label.Root>
      <Select.Root type="single" bind:value={refreshRate} items={refreshOptions}>
        <Select.Trigger class="select-trigger">
          <Select.Value placeholder={refreshLabel} />
          <ChevronDown size="15" />
        </Select.Trigger>
        <Select.Portal>
          <Select.Content class="select-content" sideOffset={8}>
            <Select.Viewport>
              {#each refreshOptions as item}
                <Select.Item class="select-item" value={item.value} label={item.label}>
                  {item.label}
                </Select.Item>
              {/each}
            </Select.Viewport>
          </Select.Content>
        </Select.Portal>
      </Select.Root>
    </div>

    <Separator.Root class="separator" />

    <div class="setting-row">
      <Label.Root>Add-ons</Label.Root>
      <button
        class="addons-toggle"
        class:on={addOnsEnabled}
        type="button"
        aria-pressed={addOnsEnabled}
        onclick={() => onSetAddOns(!addOnsEnabled)}
      >
        <span class="addons-knob"></span>
        <span class="label-off">off</span>
        <span class="label-on">on</span>
      </button>
    </div>

    <Separator.Root class="separator" />

    <div class="setting-row">
      <Label.Root>theme</Label.Root>
      <Select.Root type="single" bind:value={themeMode} items={themeOptions}>
        <Select.Trigger class="select-trigger">
          <Select.Value placeholder={themeLabel} />
          <ChevronDown size="15" />
        </Select.Trigger>
        <Select.Portal>
          <Select.Content class="select-content" sideOffset={8}>
            <Select.Viewport>
              {#each themeOptions as item}
                <Select.Item class="select-item" value={item.value} label={item.label}>
                  {item.label}
                </Select.Item>
              {/each}
            </Select.Viewport>
          </Select.Content>
        </Select.Portal>
      </Select.Root>
    </div>
  </article>

  <article class="settings-panel">
    <div class="panel-title">
      <h2>Bind editor</h2>
      <span>{config?.configPath || '/etc/myrax/config.toml'}</span>
    </div>

    <div class="form-grid">
      <label class="field">
        <span>bind</span>
        <input bind:value={hostValue} placeholder="0.0.0.0" />
      </label>
      <label class="field">
        <span>port</span>
        <input bind:value={portValue} inputmode="numeric" type="number" min="1" max="65535" />
      </label>
    </div>

    {#if error}
      <div class="inline-error">{error}</div>
    {/if}

    <Button.Root class="save-button" disabled={!dirty || pending} onclick={saveConfig}>
      {pending ? 'saving' : 'save'}
    </Button.Root>
  </article>

  <article class="settings-panel wide">
    <div class="panel-title">
      <h2>Access info</h2>
    </div>
    <div class="table">
      <div><span>url</span><strong>{accessUrl || '-'}</strong></div>
      <div><span>current bind</span><strong>{config?.host || '0.0.0.0'}:{config?.port || 1487}</strong></div>
    </div>
  </article>
</section>

<style>
  .settings-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(280px, 1fr));
    gap: 16px;
  }

  .settings-panel {
    padding: 18px;
  }

  .settings-panel.wide {
    grid-column: 1 / -1;
  }

  .setting-row {
    min-height: 54px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 14px;
  }

  .panel-title {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 14px;
    margin-bottom: 14px;
  }

  .panel-title span {
    max-width: 58%;
    overflow: hidden;
    color: var(--muted);
    font-size: 0.68rem;
    text-align: right;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .form-grid {
    display: grid;
    grid-template-columns: 1fr 140px;
    gap: 10px;
  }

  .field {
    display: grid;
    gap: 8px;
  }

  .field span {
    color: var(--muted);
    font-size: 0.7rem;
  }

  input {
    width: 100%;
    min-height: 40px;
    padding: 0 12px;
    border: 1px solid var(--line);
    border-radius: 14px;
    color: var(--text);
    background: var(--panel-2);
    outline: none;
  }

  input:focus {
    border-color: var(--text);
  }

  .inline-error {
    margin-top: 10px;
    color: #ffd8d6;
    font-size: 0.76rem;
  }

  :global(.save-button) {
    min-height: 40px;
    margin-top: 14px;
    padding: 0 15px;
    border: 1px solid var(--line);
    border-radius: 999px;
    color: #101010;
    background: #f5f5f5;
    font-size: 0.78rem;
    font-weight: 900;
  }

  :global(.save-button:disabled) {
    opacity: 0.45;
    cursor: not-allowed;
  }

  .addons-toggle {
    position: relative;
    width: 116px;
    min-height: 38px;
    display: grid;
    grid-template-columns: 1fr 1fr;
    align-items: center;
    padding: 4px;
    border: 1px solid var(--line);
    border-radius: 999px;
    background: var(--panel-2);
    font-size: 0.78rem;
    font-weight: 800;
    cursor: pointer;
    transition: border-color 160ms ease;
  }

  .addons-toggle.on {
    border-color: var(--text);
  }

  .addons-knob {
    position: absolute;
    top: 4px;
    bottom: 4px;
    left: 4px;
    width: calc(50% - 4px);
    border-radius: 999px;
    background: color-mix(in srgb, var(--muted) 28%, transparent);
    transition: left 200ms ease, background 180ms ease;
  }

  .addons-toggle.on .addons-knob {
    left: 50%;
    background: var(--text);
  }

  .label-off,
  .label-on {
    position: relative;
    z-index: 1;
    text-align: center;
    transition: color 160ms ease;
  }

  .label-off {
    color: var(--text);
  }

  .addons-toggle.on .label-off {
    color: var(--muted);
  }

  .label-on {
    color: var(--muted);
  }

  .addons-toggle.on .label-on {
    color: var(--bg);
  }

  @media (max-width: 980px) {
    .settings-grid,
    .form-grid {
      grid-template-columns: 1fr;
    }

    .settings-panel.wide {
      grid-column: auto;
    }
  }
</style>
