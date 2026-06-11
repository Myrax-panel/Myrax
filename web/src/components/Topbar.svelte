<script>
  import { Button } from 'bits-ui';
  import { Download, LogOut, RefreshCw } from '@lucide/svelte';

  export let hostname = 'Linux Node';
  export let coreUpdate = null;
  export let updatePending = false;
  export let onUpdate = async () => {};
  export let onRefresh = () => {};
  export let onLogout = () => {};
</script>

<header class="topbar">
  <div>
    <div class="eyebrow">server</div>
    <h1>{hostname}</h1>
  </div>

  <div class="top-actions">
    {#if coreUpdate?.updateAvailable}
      <Button.Root class="pill ghost update-button" disabled={updatePending} title={`Update Myrax to ${coreUpdate.latestVersion}`} onclick={onUpdate}>
        <Download size="16" />
        <span class="btn-text">{updatePending ? 'updating' : 'update'}</span>
      </Button.Root>
    {/if}
    <Button.Root class="pill ghost" onclick={onRefresh}>
      <RefreshCw size="16" />
      <span class="btn-text">refresh</span>
    </Button.Root>
    <Button.Root class="pill ghost logout-button" aria-label="Log out" onclick={onLogout}>
      <LogOut size="16" />
    </Button.Root>
  </div>
</header>

<style>
  .topbar {
    position: relative;
    z-index: 1;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 18px;
    margin-bottom: 28px;
  }

  .eyebrow {
    width: fit-content;
    margin-bottom: 8px;
    padding: 4px 9px;
    border: 1px solid var(--line);
    border-radius: 999px;
    color: var(--muted);
    font-size: 0.68rem;
    font-weight: 700;
  }

  h1 {
    max-width: min(760px, 74vw);
    font-size: clamp(1.85rem, 3.2vw, 3.55rem);
    line-height: 1.02;
    overflow-wrap: anywhere;
  }

  .top-actions {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  :global(.logout-button) {
    width: 40px;
    padding: 0;
    justify-content: center;
  }

  :global(.update-button) {
    background: var(--accent-2);
    border-color: var(--accent-2);
  }

  :global(.update-button:disabled) {
    opacity: 0.55;
    cursor: not-allowed;
  }

  @media (max-width: 980px) {
    .topbar {
      align-items: center;
      flex-direction: row;
      margin-bottom: 20px;
    }

    .eyebrow {
      display: none;
    }

    h1 {
      max-width: calc(100vw - 102px);
      font-size: 1.45rem;
    }
  }

  @media (max-width: 560px) {
    .top-actions {
      width: auto;
    }

    .btn-text {
      display: none;
    }

    h1 {
      max-width: calc(100vw - 82px);
      font-size: 1.15rem;
      line-height: 1.18;
    }
  }
</style>
