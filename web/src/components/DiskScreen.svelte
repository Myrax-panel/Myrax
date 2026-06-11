<script>
  import { onMount } from 'svelte';
  import { Progress, ScrollArea } from 'bits-ui';
  import { bytes, intervalMs, percent } from '../lib/format.js';

  export let refreshRate = '2s';

  let disks = [];
  let error = '';
  let timer = null;
  let mounted = false;

  $: mountpointCount = disks.reduce((total, disk) => total + (disk.mountpoints?.length || 1), 0);

  $: if (mounted) {
    refreshRate;
    restartTimer();
  }

  async function loadDisks() {
    try {
      const response = await fetch('/api/disks');
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      disks = await response.json();
      error = '';
    } catch (requestError) {
      error = requestError.message;
    }
  }

  function restartTimer() {
    clearInterval(timer);
    timer = setInterval(loadDisks, intervalMs(refreshRate));
  }

  onMount(() => {
    mounted = true;
    loadDisks();
    return () => {
      mounted = false;
      clearInterval(timer);
    };
  });
</script>

<section class="panel disk-panel">
  <div class="panel-head">
    <h2>Disk</h2>
    <span>{disks.length} devices / {mountpointCount} mountpoints</span>
  </div>

  {#if error}
    <div class="inline-error">{error}</div>
  {/if}

  <ScrollArea.Root class="disk-scroll">
    <ScrollArea.Viewport class="disk-viewport">
      <div class="disk-list">
        {#each disks as disk}
          <article class="disk-row">
            <div class="disk-title">
              <strong>{disk.device || disk.mountpoint}</strong>
              <span>{disk.filesystem}</span>
              <span class="mounts">{(disk.mountpoints || [disk.mountpoint]).join(', ')}</span>
            </div>
            <div class="usage-block">
              <div class="usage-line">
                <span>space</span>
                <strong>{percent(disk.usage)} - {bytes(disk.used)} used - {bytes(disk.free)} free</strong>
              </div>
              <Progress.Root class="usage-bar" value={disk.usage} max={100}>
                <span style={`width:${Math.min(disk.usage, 100)}%`}></span>
              </Progress.Root>
            </div>
            <div class="usage-block">
              <div class="usage-line">
                <span>inodes</span>
                <strong>{percent(disk.inodeUsage)} - {disk.inodesUsed || 0} / {disk.inodesTotal || 0}</strong>
              </div>
              <Progress.Root class="usage-bar muted" value={disk.inodeUsage} max={100}>
                <span style={`width:${Math.min(disk.inodeUsage, 100)}%`}></span>
              </Progress.Root>
            </div>
            <div class="disk-total">
              <span>total</span>
              <strong>{bytes(disk.total)}</strong>
            </div>
          </article>
        {/each}
      </div>
    </ScrollArea.Viewport>
    <ScrollArea.Scrollbar class="scrollbar" orientation="vertical">
      <ScrollArea.Thumb class="thumb" />
    </ScrollArea.Scrollbar>
  </ScrollArea.Root>
</section>

<style>
  .disk-panel {
    padding: 18px;
  }

  .panel-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 14px;
  }

  .panel-head span {
    color: var(--muted);
    font-size: 0.72rem;
  }

  .inline-error {
    margin-bottom: 12px;
    color: #ffd8d6;
    font-size: 0.76rem;
  }

  :global(.disk-scroll) {
    height: min(640px, calc(100vh - 188px));
    overflow: hidden;
  }

  :global(.disk-viewport) {
    height: 100%;
    padding-right: 12px;
  }

  .disk-list {
    display: grid;
    gap: 12px;
  }

  .disk-row {
    display: grid;
    grid-template-columns: 1fr 1.3fr 1.3fr 120px;
    gap: 14px;
    align-items: center;
    padding: 14px;
    border: 1px solid var(--line-soft);
    border-radius: 18px;
    background: #111111;
  }

  .disk-title {
    display: grid;
    gap: 6px;
  }

  .mounts {
    overflow: hidden;
    color: var(--muted);
    font-size: 0.7rem;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .disk-title strong,
  .usage-line strong,
  .disk-total strong {
    color: var(--text);
  }

  .disk-title span,
  .usage-line span,
  .disk-total span {
    color: var(--muted);
    font-size: 0.7rem;
  }

  .usage-block {
    display: grid;
    gap: 8px;
  }

  .usage-line,
  .disk-total {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    font-size: 0.7rem;
  }

  :global(.usage-bar) {
    height: 7px;
    overflow: hidden;
    border-radius: 999px;
    background: #25262a;
  }

  :global(.usage-bar span) {
    display: block;
    height: 100%;
    border-radius: inherit;
    background: #d8f0ff;
  }

  :global(.usage-bar.muted span) {
    background: #fff8c8;
  }

  @media (max-width: 980px) {
    .disk-row {
      grid-template-columns: 1fr;
    }
  }
</style>
