<script>
  export let page = null;

  // Plugin pages may mount/render into the host node and return a cleanup function.
  function mount(node) {
    let cleanup = null;
    node.innerHTML = '';

    if (typeof page?.mount === 'function') {
      cleanup = page.mount(node);
    } else if (typeof page?.render === 'function') {
      cleanup = page.render(node);
    } else if (page?.html) {
      node.innerHTML = page.html;
    } else {
      node.innerHTML = `<section class="plugin-empty"><h2>${page?.label || 'Plugin'}</h2></section>`;
    }

    return {
      destroy() {
        if (typeof cleanup === 'function') {
          cleanup();
        }
        node.innerHTML = '';
      }
    };
  }
</script>

<div class="plugin-host" use:mount></div>

<style>
  .plugin-host {
    min-height: 420px;
  }

  :global(.plugin-empty) {
    padding: 22px;
    border: 1px solid var(--line);
    border-radius: 26px;
    background: var(--panel);
  }
</style>
