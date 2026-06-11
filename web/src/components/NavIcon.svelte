<script>
    export let icon = null;
    export let label = "";
    export let size = 18;

    function kindOf(value) {
        if (!value) return "none";
        if (typeof value === "string") {
            return /^\s*(<svg|<\?xml)/i.test(value) ? "svg" : "image";
        }
        return "component";
    }

    $: kind = kindOf(icon);
    $: dimension = /^\d+$/.test(String(size)) ? `${size}px` : String(size);
</script>

{#if kind === "component"}
    <svelte:component this={icon} {size} />
{:else if kind === "svg"}
    <span
        class="nav-icon"
        style="--nav-icon-size:{dimension}"
        aria-hidden="true">{@html icon}</span
    >
{:else if kind === "image"}
    <img
        class="nav-icon"
        src={icon}
        alt={label ? `${label} icon` : ""}
        style="--nav-icon-size:{dimension}"
    />
{/if}

<style>
    .nav-icon {
        width: var(--nav-icon-size, 18px);
        height: var(--nav-icon-size, 18px);
        flex: 0 0 auto;
        display: inline-flex;
        align-items: center;
        justify-content: center;
        object-fit: contain;
    }

    .nav-icon :global(svg) {
        width: 100%;
        height: 100%;
        display: block;
    }
</style>
