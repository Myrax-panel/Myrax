<script>
  import { onMount } from 'svelte';
  import { Button, ScrollArea, Select } from 'bits-ui';
  import { ChevronDown, Search } from '@lucide/svelte';

  export let refreshRate = '2s';

  let entries = [];
  let error = '';
  let stream = null;
  let mounted = false;
  let source = 'myrax';
  let level = 'all';
  let draftQuery = '';
  let query = '';

  const sourceOptions = [
    { value: 'myrax', label: 'myrax' },
    { value: 'system', label: 'system' }
  ];

  const levelOptions = [
    { value: 'all', label: 'all' },
    { value: 'error', label: 'error' },
    { value: 'warning', label: 'warning' },
    { value: 'notice', label: 'notice' },
    { value: 'info', label: 'info' }
  ];

  $: sourceLabel = sourceOptions.find((item) => item.value === source)?.label || 'myrax';
  $: levelLabel = levelOptions.find((item) => item.value === level)?.label || 'all';
  $: if (mounted) {
    refreshRate;
    source;
    level;
    query;
    connectLogs();
  }

  function connectLogs() {
    stream?.close();
    const params = new URLSearchParams({
      source,
      level,
      query,
      interval: refreshRate,
      limit: '160'
    });
    stream = new EventSource(`/api/events/logs?${params.toString()}`);
    stream.addEventListener('logs', (event) => {
      entries = JSON.parse(event.data);
      error = '';
    });
    stream.onerror = () => {
      error = 'log stream unavailable';
    };
  }

  function applyQuery() {
    query = draftQuery.trim();
  }

  function shortTime(value) {
    if (!value) return '';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return value;
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  }

  onMount(() => {
    mounted = true;
    return () => stream?.close();
  });
</script>

<section class="panel logs-panel">
  <div class="toolbar">
    <Select.Root type="single" bind:value={source} items={sourceOptions}>
      <Select.Trigger class="select-trigger">
        <Select.Value placeholder={sourceLabel} />
        <ChevronDown size="15" />
      </Select.Trigger>
      <Select.Portal>
        <Select.Content class="select-content" sideOffset={8}>
          <Select.Viewport>
            {#each sourceOptions as item}
              <Select.Item class="select-item" value={item.value} label={item.label}>{item.label}</Select.Item>
            {/each}
          </Select.Viewport>
        </Select.Content>
      </Select.Portal>
    </Select.Root>

    <Select.Root type="single" bind:value={level} items={levelOptions}>
      <Select.Trigger class="select-trigger">
        <Select.Value placeholder={levelLabel} />
        <ChevronDown size="15" />
      </Select.Trigger>
      <Select.Portal>
        <Select.Content class="select-content" sideOffset={8}>
          <Select.Viewport>
            {#each levelOptions as item}
              <Select.Item class="select-item" value={item.value} label={item.label}>{item.label}</Select.Item>
            {/each}
          </Select.Viewport>
        </Select.Content>
      </Select.Portal>
    </Select.Root>

    <div class="search-box">
      <Search size="15" />
      <input
        bind:value={draftQuery}
        placeholder="search"
        onkeydown={(event) => event.key === 'Enter' && applyQuery()}
      />
    </div>

    <Button.Root class="panel-button" onclick={applyQuery}>apply</Button.Root>
  </div>

  {#if error}
    <div class="inline-error">{error}</div>
  {/if}

  <ScrollArea.Root class="log-scroll">
    <ScrollArea.Viewport class="log-viewport">
      {#if entries.length}
        {#each entries as entry}
          <article class="log-row">
            <span class="time">{shortTime(entry.timestamp)}</span>
            <span class="level {entry.level}">{entry.level}</span>
            <span class="unit">{entry.unit || source}</span>
            <p>{entry.message}</p>
          </article>
        {/each}
      {:else}
        <div class="empty">no logs</div>
      {/if}
    </ScrollArea.Viewport>
    <ScrollArea.Scrollbar class="scrollbar" orientation="vertical">
      <ScrollArea.Thumb class="thumb" />
    </ScrollArea.Scrollbar>
  </ScrollArea.Root>
</section>

<style>
  .logs-panel {
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
    gap: 8px;
    padding: 0 12px;
    border: 1px solid var(--line);
    border-radius: 999px;
    color: var(--muted);
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

  :global(.panel-button) {
    min-height: 38px;
    padding: 0 13px;
    border: 1px solid var(--line);
    border-radius: 999px;
    color: #101010;
    background: #f5f5f5;
    font-size: 0.76rem;
    font-weight: 800;
  }

  .inline-error {
    margin-bottom: 12px;
    color: #ffd8d6;
    font-size: 0.76rem;
  }

  :global(.log-scroll) {
    height: min(620px, calc(100vh - 190px));
    overflow: hidden;
  }

  :global(.log-viewport) {
    height: 100%;
    padding-right: 12px;
  }

  .log-row {
    display: grid;
    grid-template-columns: 76px 78px 160px minmax(0, 1fr);
    gap: 10px;
    align-items: start;
    min-height: 38px;
    padding: 9px 0;
    border-bottom: 1px solid var(--line-soft);
    color: var(--muted);
    font-size: 0.72rem;
  }

  .log-row p {
    color: var(--text);
    line-height: 1.55;
  }

  .level {
    width: fit-content;
    padding: 3px 8px;
    border-radius: 999px;
    color: #101010;
    background: #d8f0ff;
    font-weight: 800;
  }

  .level.error {
    background: #ffd8d6;
  }

  .level.warning {
    background: #fff8c8;
  }

  .unit {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .empty {
    display: grid;
    min-height: 280px;
    place-items: center;
    border: 1px dashed var(--line);
    border-radius: 18px;
    color: var(--muted);
    font-size: 0.78rem;
  }

  @media (max-width: 760px) {
    .toolbar,
    .log-row {
      grid-template-columns: 1fr;
    }

    .toolbar {
      display: grid;
    }

    .log-row {
      display: grid;
      gap: 6px;
    }
  }
</style>
