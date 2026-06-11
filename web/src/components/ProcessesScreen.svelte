<script>
  import { onMount } from 'svelte';
  import { AlertDialog, Button, ScrollArea } from 'bits-ui';
  import { CircleX, RotateCw } from '@lucide/svelte';
  import { bytes, intervalMs, percent } from '../lib/format.js';

  export let refreshRate = '2s';

  let processes = [];
  let error = '';
  let timer = null;
  let mounted = false;
  let sortBy = 'cpu';
  let sortDirection = 'desc';
  let pending = null;
  let confirmOpen = false;
  let selectedAction = null;

  const columns = [
    { value: 'pid', label: 'pid' },
    { value: 'name', label: 'name' },
    { value: 'cpu', label: 'cpu' },
    { value: 'memory', label: 'ram' },
    { value: 'rss', label: 'rss' },
    { value: 'service', label: 'service' }
  ];

  $: sortedProcesses = [...processes].sort(compareProcesses);
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
    sortDirection = ['name', 'service'].includes(field) ? 'asc' : 'desc';
  }

  function compareProcesses(a, b) {
    const left = processSortValue(a, sortBy);
    const right = processSortValue(b, sortBy);
    const result = typeof left === 'string' ? left.localeCompare(right) : left - right;
    return sortDirection === 'asc' ? result : -result;
  }

  function processSortValue(process, field) {
    if (field === 'name' || field === 'service') {
      return String(process[field] || '').toLowerCase();
    }
    return Number(process[field] || 0);
  }

  async function loadProcesses() {
    try {
      const response = await fetch('/api/processes?limit=120');
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      processes = await response.json();
      error = '';
    } catch (requestError) {
      error = requestError.message;
    }
  }

  function restartTimer() {
    clearInterval(timer);
    timer = setInterval(loadProcesses, intervalMs(refreshRate));
  }

  function ask(action, process) {
    selectedAction = { action, process };
    confirmOpen = true;
  }

  async function runSelectedAction() {
    if (!selectedAction) return;
    const { action, process } = selectedAction;
    pending = `${action}:${process.pid}`;

    try {
      if (action === 'kill') {
        await postJSON('/api/processes/kill', { pid: process.pid });
      } else if (action === 'restart') {
        await postJSON('/api/services/action', { name: process.service, action: 'restart' });
      }
      await loadProcesses();
      error = '';
    } catch (requestError) {
      error = requestError.message;
    } finally {
      pending = null;
      selectedAction = null;
    }
  }

  async function postJSON(url, payload) {
    const response = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    if (!response.ok) {
      const body = await response.json().catch(() => ({}));
      throw new Error(body.error || `HTTP ${response.status}`);
    }
  }

  onMount(() => {
    mounted = true;
    loadProcesses();
    return () => {
      mounted = false;
      clearInterval(timer);
    };
  });
</script>

<section class="panel process-panel">
  <div class="toolbar">
    <div>
      <h2>Processes</h2>
      <p>{processes.length} running entries</p>
    </div>
  </div>

  {#if error}
    <div class="inline-error">{error}</div>
  {/if}

  <ScrollArea.Root class="process-scroll">
    <ScrollArea.Viewport class="process-viewport">
      <div class="process-table">
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
        {#each sortedProcesses as process}
          <div class="process-row">
            <span>{process.pid}</span>
            <strong>{process.name}</strong>
            <span>{percent(process.cpu)}</span>
            <span>{percent(process.memory)}</span>
            <span>{bytes(process.rss)}</span>
            <span class="service">{process.service || '-'}</span>
            <span class="row-actions">
              {#if process.service}
                <Button.Root
                  class="mini-button"
                  disabled={pending === `restart:${process.pid}`}
                  onclick={() => ask('restart', process)}
                  aria-label={`Restart ${process.service}`}
                >
                  <RotateCw size="14" />
                </Button.Root>
              {/if}
              <Button.Root
                class="mini-button danger"
                disabled={pending === `kill:${process.pid}`}
                onclick={() => ask('kill', process)}
                aria-label={`Terminate ${process.name}`}
              >
                <CircleX size="15" />
              </Button.Root>
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
        <AlertDialog.Title>
          {selectedAction?.action === 'restart' ? 'Restart service?' : 'Kill process?'}
        </AlertDialog.Title>
        <AlertDialog.Description>
          {#if selectedAction?.action === 'restart'}
            Restart {selectedAction?.process?.service}.
          {:else}
            Send SIGTERM to PID {selectedAction?.process?.pid}.
          {/if}
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
  .process-panel {
    padding: 18px;
  }

  .toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 14px;
  }

  .toolbar p {
    margin-top: 6px;
    color: var(--muted);
    font-size: 0.72rem;
  }

  .inline-error {
    margin-bottom: 12px;
    color: #ffd8d6;
    font-size: 0.76rem;
  }

  :global(.process-scroll) {
    height: min(640px, calc(100vh - 188px));
    overflow: hidden;
  }

  :global(.process-viewport) {
    height: 100%;
    padding-right: 12px;
  }

  .process-table {
    min-width: 860px;
  }

  .head,
  .process-row {
    display: grid;
    grid-template-columns: 70px 1.2fr 80px 80px 100px 1.2fr 104px;
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

  .process-row strong {
    color: var(--text);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .service {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
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
