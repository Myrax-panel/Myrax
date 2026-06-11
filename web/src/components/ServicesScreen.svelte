<script>
  import { onMount } from 'svelte';
  import { AlertDialog, Button, ScrollArea, Select } from 'bits-ui';
  import { ChevronDown, Play, RotateCw, Square } from '@lucide/svelte';
  import { intervalMs } from '../lib/format.js';

  export let refreshRate = '2s';

  let services = [];
  let error = '';
  let timer = null;
  let mounted = false;
  let statusFilter = 'all';
  let query = '';
  let sortBy = 'name';
  let sortDirection = 'asc';
  let confirmOpen = false;
  let selectedAction = null;
  let pending = '';

  const statusOptions = [
    { value: 'all', label: 'all' },
    { value: 'active', label: 'active' },
    { value: 'inactive', label: 'inactive' },
    { value: 'failed', label: 'failed' }
  ];

  const columns = [
    { value: 'name', label: 'unit' },
    { value: 'active', label: 'status' },
    { value: 'sub', label: 'sub' },
    { value: 'description', label: 'description' }
  ];

  $: statusLabel = statusOptions.find((item) => item.value === statusFilter)?.label || 'all';
  $: filteredServices = services.filter((service) => {
    const haystack = `${service.name} ${service.active} ${service.sub} ${service.description}`.toLowerCase();
    return (statusFilter === 'all' || service.active === statusFilter) && haystack.includes(query.toLowerCase());
  }).sort(compareServices);
  $: if (mounted) {
    refreshRate;
    restartTimer();
  }

  function setSort(field) {
    if (sortBy === field) {
      sortDirection = sortDirection === 'asc' ? 'desc' : 'asc';
      return;
    }

    sortBy = field;
    sortDirection = 'asc';
  }

  function compareServices(a, b) {
    const left = String(a[sortBy] || '').toLowerCase();
    const right = String(b[sortBy] || '').toLowerCase();
    const result = left.localeCompare(right);
    return sortDirection === 'asc' ? result : -result;
  }

  async function loadServices() {
    try {
      const response = await fetch('/api/services?limit=180');
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      services = await response.json();
      error = '';
    } catch (requestError) {
      error = requestError.message;
    }
  }

  function restartTimer() {
    clearInterval(timer);
    timer = setInterval(loadServices, intervalMs(refreshRate));
  }

  function ask(action, service) {
    selectedAction = { action, service };
    confirmOpen = true;
  }

  async function runSelectedAction() {
    if (!selectedAction) return;
    const { action, service } = selectedAction;
    pending = `${action}:${service.name}`;

    try {
      const response = await fetch('/api/services/action', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: service.name, action })
      });
      if (!response.ok) {
        const body = await response.json().catch(() => ({}));
        throw new Error(body.error || `HTTP ${response.status}`);
      }
      await loadServices();
      error = '';
    } catch (requestError) {
      error = requestError.message;
    } finally {
      pending = '';
      selectedAction = null;
    }
  }

  onMount(() => {
    mounted = true;
    loadServices();
    return () => {
      mounted = false;
      clearInterval(timer);
    };
  });
</script>

<section class="panel services-panel">
  <div class="toolbar">
    <div class="search-box">
      <input bind:value={query} placeholder="search service" />
    </div>
    <Select.Root type="single" bind:value={statusFilter} items={statusOptions}>
      <Select.Trigger class="select-trigger">
        <Select.Value placeholder={statusLabel} />
        <ChevronDown size="15" />
      </Select.Trigger>
      <Select.Portal>
        <Select.Content class="select-content" sideOffset={8}>
          <Select.Viewport>
            {#each statusOptions as item}
              <Select.Item class="select-item" value={item.value} label={item.label}>{item.label}</Select.Item>
            {/each}
          </Select.Viewport>
        </Select.Content>
      </Select.Portal>
    </Select.Root>
  </div>

  {#if error}
    <div class="inline-error">{error}</div>
  {/if}

  <ScrollArea.Root class="services-scroll">
    <ScrollArea.Viewport class="services-viewport">
      <div class="services-table">
        <div class="head">
          {#each columns as column}
            <button
              type="button"
              class:active={sortBy === column.value}
              class="head-button"
              onclick={() => setSort(column.value)}
            >
              {column.label}
            </button>
          {/each}
          <span></span>
        </div>
        {#each filteredServices as service}
          <div class="service-row">
            <strong>{service.name}</strong>
            <span class="status {service.active}">{service.active}</span>
            <span>{service.sub}</span>
            <span class="description">{service.description}</span>
            <span class="row-actions">
              {#if service.active === 'active'}
                <Button.Root class="mini-button" disabled={pending === `restart:${service.name}`} onclick={() => ask('restart', service)}>
                  <RotateCw size="14" />
                </Button.Root>
                <Button.Root class="mini-button danger" disabled={pending === `stop:${service.name}`} onclick={() => ask('stop', service)}>
                  <Square size="14" />
                </Button.Root>
              {:else}
                <Button.Root class="mini-button" disabled={pending === `start:${service.name}`} onclick={() => ask('start', service)}>
                  <Play size="14" />
                </Button.Root>
              {/if}
            </span>
          </div>
        {/each}
      </div>
    </ScrollArea.Viewport>
    <ScrollArea.Scrollbar class="scrollbar" orientation="vertical">
      <ScrollArea.Thumb class="thumb" />
    </ScrollArea.Scrollbar>
  </ScrollArea.Root>

  <AlertDialog.Root bind:open={confirmOpen}>
    <AlertDialog.Portal>
      <AlertDialog.Overlay class="overlay" />
      <AlertDialog.Content class="dialog">
        <AlertDialog.Title>{selectedAction?.action} service?</AlertDialog.Title>
        <AlertDialog.Description>
          Run systemctl {selectedAction?.action} {selectedAction?.service?.name}.
        </AlertDialog.Description>
        <div class="dialog-row">
          <AlertDialog.Cancel class="dialog-button">cancel</AlertDialog.Cancel>
          <AlertDialog.Action class="dialog-button danger-fill" onclick={runSelectedAction}>
            confirm
          </AlertDialog.Action>
        </div>
      </AlertDialog.Content>
    </AlertDialog.Portal>
  </AlertDialog.Root>
</section>

<style>
  .services-panel {
    padding: 18px;
  }

  .toolbar {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 14px;
  }

  .search-box {
    min-height: 38px;
    flex: 1;
    display: flex;
    align-items: center;
    padding: 0 12px;
    border: 1px solid var(--line);
    border-radius: 999px;
    background: #202124;
  }

  input {
    width: 100%;
    border: 0;
    outline: 0;
    color: var(--text);
    background: transparent;
    font-size: 0.78rem;
  }

  .inline-error {
    margin-bottom: 12px;
    color: #ffd8d6;
    font-size: 0.76rem;
  }

  :global(.services-scroll) {
    height: min(640px, calc(100vh - 188px));
    overflow: hidden;
  }

  :global(.services-viewport) {
    height: 100%;
    padding-right: 12px;
  }

  .services-table {
    min-width: 880px;
  }

  .head,
  .service-row {
    display: grid;
    grid-template-columns: 1.3fr 100px 110px 1.4fr 112px;
    gap: 12px;
    align-items: center;
    min-height: 42px;
    border-bottom: 1px solid var(--line-soft);
    color: var(--muted);
    font-size: 0.72rem;
  }

  .head {
    font-weight: 800;
  }

  .head-button {
    width: fit-content;
    padding: 0;
    border: 0;
    color: #777c86;
    background: transparent;
    font-size: 0.72rem;
    font-weight: 700;
    text-align: left;
  }

  .head-button.active {
    color: var(--text);
    font-weight: 900;
  }

  .service-row strong,
  .description {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .service-row strong {
    color: var(--text);
  }

  .status {
    width: fit-content;
    padding: 3px 8px;
    border-radius: 999px;
    color: #101010;
    background: #d8f0ff;
    font-weight: 800;
  }

  .status.inactive {
    color: var(--muted);
    background: #202124;
  }

  .status.failed {
    background: #ffd8d6;
  }

  .row-actions {
    display: inline-flex;
    justify-content: flex-end;
    gap: 6px;
  }

  :global(.mini-button) {
    width: 32px;
    height: 32px;
    min-width: 32px;
    min-height: 32px;
    padding: 0;
    display: grid;
    place-items: center;
    border: 1px solid var(--line);
    border-radius: 12px;
    color: var(--text);
    background: #202124;
  }

  :global(.mini-button.danger) {
    color: #101010;
    background: #ffd8d6;
  }

  @media (max-width: 760px) {
    .toolbar {
      align-items: stretch;
      flex-direction: column;
    }
  }
</style>
