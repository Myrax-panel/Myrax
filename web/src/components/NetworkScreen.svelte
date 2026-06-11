<script>
  import { onMount } from 'svelte';
  import { ScrollArea } from 'bits-ui';
  import { byteRate, bytes, intervalMs } from '../lib/format.js';

  export let refreshRate = '2s';

  let network = null;
  let error = '';
  let timer = null;
  let mounted = false;

  $: if (mounted) {
    refreshRate;
    restartTimer();
  }

  async function loadNetwork() {
    try {
      const response = await fetch('/api/network');
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      network = await response.json();
      error = '';
    } catch (requestError) {
      error = requestError.message;
    }
  }

  function restartTimer() {
    clearInterval(timer);
    timer = setInterval(loadNetwork, intervalMs(refreshRate));
  }

  onMount(() => {
    mounted = true;
    loadNetwork();
    return () => {
      mounted = false;
      clearInterval(timer);
    };
  });
</script>

<section class="network-grid">
  <article class="panel summary-panel">
    <h2>Network</h2>
    {#if error}
      <div class="inline-error">{error}</div>
    {/if}
    <div class="table">
      <div><span>gateway</span><strong>{network?.gateway || '-'}</strong></div>
      <div><span>dns</span><strong>{network?.dns?.join(', ') || '-'}</strong></div>
    </div>
  </article>

  <article class="panel interfaces-panel">
    <h2>Interfaces</h2>
    <div class="interface-list">
      {#each network?.interfaces || [] as adapter}
        <div class="interface-card">
          <div class="interface-head">
            <strong>{adapter.name}</strong>
            <span class="state {adapter.state}">{adapter.state || '-'}</span>
          </div>
          <div class="traffic">
            <span>rx {byteRate(adapter.rxRate)}</span>
            <span>tx {byteRate(adapter.txRate)}</span>
          </div>
          <div class="mini-table">
            <div><span>addresses</span><strong>{adapter.addresses?.join(', ') || '-'}</strong></div>
            <div><span>mac</span><strong>{adapter.mac || '-'}</strong></div>
            <div><span>mtu</span><strong>{adapter.mtu || '-'}</strong></div>
            <div><span>total</span><strong>{bytes(adapter.rxBytes)} / {bytes(adapter.txBytes)}</strong></div>
          </div>
        </div>
      {/each}
    </div>
  </article>

  <article class="panel ports-panel">
    <h2>Listen ports</h2>
    <ScrollArea.Root class="ports-scroll">
      <ScrollArea.Viewport class="ports-viewport">
        <div class="ports-table">
          <div class="head">
            <span>proto</span>
            <span>address</span>
            <span>port</span>
          </div>
          {#each network?.listenPorts || [] as port}
            <div class="port-row">
              <span>{port.protocol}</span>
              <strong>{port.address || '*'}</strong>
              <span>{port.port}</span>
            </div>
          {/each}
        </div>
      </ScrollArea.Viewport>
      <ScrollArea.Scrollbar class="scrollbar" orientation="vertical">
        <ScrollArea.Thumb class="thumb" />
      </ScrollArea.Scrollbar>
    </ScrollArea.Root>
  </article>
</section>

<style>
  .network-grid {
    display: grid;
    grid-template-columns: 0.85fr 1.15fr;
    gap: 16px;
  }

  .panel {
    padding: 18px;
  }

  h2 {
    margin-bottom: 14px;
  }

  .interfaces-panel,
  .ports-panel {
    min-height: 260px;
  }

  .interfaces-panel {
    grid-row: span 2;
  }

  .interface-list {
    display: grid;
    gap: 12px;
  }

  .interface-card {
    padding: 14px;
    border: 1px solid var(--line-soft);
    border-radius: 18px;
    background: #111111;
  }

  .interface-head,
  .traffic {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .interface-head strong {
    color: var(--text);
  }

  .state {
    padding: 3px 8px;
    border-radius: 999px;
    color: var(--muted);
    background: #202124;
    font-size: 0.68rem;
    font-weight: 800;
  }

  .state.up {
    color: #101010;
    background: #c4f7d1;
  }

  .traffic {
    margin: 12px 0;
    color: var(--text);
    font-size: 0.78rem;
    font-weight: 800;
  }

  .mini-table div {
    min-height: 34px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    border-bottom: 1px solid var(--line-soft);
    color: var(--muted);
    font-size: 0.7rem;
  }

  .mini-table strong {
    max-width: 62%;
    overflow: hidden;
    color: var(--text);
    text-align: right;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  :global(.ports-scroll) {
    height: 280px;
    overflow: hidden;
  }

  :global(.ports-viewport) {
    height: 100%;
    padding-right: 12px;
  }

  .head,
  .port-row {
    display: grid;
    grid-template-columns: 80px 1fr 80px;
    gap: 12px;
    align-items: center;
    min-height: 36px;
    border-bottom: 1px solid var(--line-soft);
    color: var(--muted);
    font-size: 0.72rem;
  }

  .head {
    color: var(--text);
    font-weight: 800;
  }

  .port-row strong {
    color: var(--text);
  }

  .inline-error {
    margin-bottom: 12px;
    color: #ffd8d6;
    font-size: 0.76rem;
  }

  @media (max-width: 980px) {
    .network-grid {
      grid-template-columns: 1fr;
    }

    .interfaces-panel {
      grid-row: auto;
    }
  }
</style>
