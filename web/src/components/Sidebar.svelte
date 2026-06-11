<script>
  import { Switch, Tabs } from 'bits-ui';
  import { PanelLeftClose, PanelLeftOpen } from '@lucide/svelte';
  import BrandLogo from './BrandLogo.svelte';
  import NavIcon from './NavIcon.svelte';

  export let navItems = [];
  export let sidebarOpen = true;
</script>

<aside class="sidebar">
  <div class="brand-row">
    <BrandLogo />
    <span class="brand-name">yrax</span>
    <Switch.Root class="collapse-button" aria-label="Toggle sidebar" bind:checked={sidebarOpen}>
      {#if sidebarOpen}<PanelLeftClose size="17" />{:else}<PanelLeftOpen size="17" />{/if}
    </Switch.Root>
  </div>

  <Tabs.List class="side-nav" aria-label="Myrax navigation">
    {#each navItems as item}
      <Tabs.Trigger class="side-link" value={item.value} title={sidebarOpen ? undefined : item.label}>
        <NavIcon icon={item.icon} label={item.label} size={18} />
        {#if sidebarOpen}<span>{item.label}</span>{/if}
      </Tabs.Trigger>
    {/each}
  </Tabs.List>
</aside>

<style>
  .sidebar {
    position: sticky;
    top: 0;
    height: 100vh;
    display: flex;
    flex-direction: column;
    padding: 20px 14px;
    border-right: 1px solid var(--line-soft);
    background: #111111;
  }

  .brand-row {
    min-height: 42px;
    display: flex;
    align-items: center;
    gap: 2px;
    margin-bottom: 34px;
    color: var(--text);
    font-size: 1.12rem;
    font-weight: 800;
  }

  :global(.shell:not(.sidebar-open)) .brand-row {
    justify-content: center;
    gap: 0;
  }

  :global(.shell:not(.sidebar-open)) :global(.brand-logo) {
    display: none;
  }

  :global(.shell:not(.sidebar-open)) .brand-name {
    display: none;
  }

  :global(.shell:not(.sidebar-open) .collapse-button) {
    margin-left: 0;
  }

  :global(.collapse-button) {
    width: 34px;
    height: 34px;
    display: grid;
    place-items: center;
    margin-left: auto;
    border: 1px solid var(--line);
    border-radius: 12px;
    color: var(--muted);
    background: #161616;
  }

  :global(.side-nav) {
    display: grid;
    gap: 8px;
  }

  :global(.side-link) {
    width: 100%;
    min-height: 42px;
    display: flex;
    align-items: center;
    justify-content: flex-start;
    gap: 10px;
    padding: 0 12px;
    border: 1px solid transparent;
    border-radius: 999px;
    color: var(--muted);
    background: transparent;
    font-size: 0.78rem;
    font-weight: 700;
  }

  :global(.side-link[data-state="active"]) {
    color: #101010;
    border-color: #f5f5f5;
    background: #f5f5f5;
  }

  .sidebar:not(:has(:global(.side-link span))) :global(.side-link) {
    justify-content: center;
    padding: 0;
  }

  @media (max-width: 980px) {
    .sidebar {
      position: static;
      height: 64px;
      flex-direction: row;
      align-items: center;
      justify-content: space-between;
      padding: 12px 14px;
      border-right: 0;
      border-bottom: 1px solid var(--line-soft);
    }

    .brand-row {
      width: 100%;
      min-height: 40px;
      margin-bottom: 0;
    }

    :global(.shell:not(.sidebar-open)) .brand-row {
      justify-content: flex-start;
      gap: 2px;
    }

    :global(.shell:not(.sidebar-open)) :global(.brand-logo) {
      display: block;
    }

    :global(.shell:not(.sidebar-open)) .brand-name {
      display: inline;
    }

    .brand-row span {
      display: inline;
    }

    :global(.collapse-button),
    :global(.side-nav) {
      display: none;
    }
  }
</style>
