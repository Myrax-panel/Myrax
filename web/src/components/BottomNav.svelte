<script>
  import { Tabs } from 'bits-ui';
  import NavIcon from './NavIcon.svelte';

  export let navItems = [];
</script>

<Tabs.List class="bottom-nav" aria-label="Mobile navigation">
  {#each navItems as item}
    <Tabs.Trigger class="bottom-link" value={item.value} aria-label={item.label}>
      <NavIcon icon={item.icon} label={item.label} size={18} />
    </Tabs.Trigger>
  {/each}
</Tabs.List>

<style>
  :global(.bottom-nav) {
    display: none;
  }

  @media (max-width: 980px) {
    :global(.bottom-nav) {
      position: fixed;
      left: 50%;
      bottom: 16px;
      z-index: 80;
      width: min(468px, calc(100vw - 20px));
      display: grid;
      grid-auto-flow: column;
      grid-auto-columns: 1fr;
      gap: 4px;
      padding: 6px;
      border: 1px solid var(--line);
      border-radius: 999px;
      background: rgb(17 17 17 / 85%);
      box-shadow: 0 12px 40px rgb(0 0 0 / 50%), inset 0 1px 1px rgb(255 255 255 / 10%);
      transform: translateX(-50%);
      backdrop-filter: blur(20px);
    }

    :global(.bottom-link) {
      min-width: 0;
      height: 38px;
      display: grid;
      place-items: center;
      border: 0;
      border-radius: 999px;
      color: var(--muted);
      background: transparent;
      transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
    }

    :global(.bottom-link[data-state="active"]) {
      color: #101010;
      background: #f5f5f5;
      box-shadow: 0 4px 12px rgb(0 0 0 / 25%);
    }

    :global(.bottom-link:not([data-state="active"]):hover) {
      color: var(--text);
      background: rgb(255 255 255 / 5%);
    }

    :global(.bottom-link:active) {
      transform: scale(0.95);
    }
  }

  @media (max-width: 560px) {
    :global(.bottom-nav) {
      bottom: 16px;
      width: calc(100vw - 20px);
      max-width: 420px;
      gap: 4px;
      padding: 6px;
    }

    :global(.bottom-link) {
      width: auto;
    }
  }
</style>
