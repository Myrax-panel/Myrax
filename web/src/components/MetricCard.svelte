<script>
    import { percent } from "../lib/format.js";

    export let title = "";
    export let icon = null;
    export let value = 0;
    export let variant = "";
    export let metaLabel = "";
    export let metaValue = "";
    export let history = [];
    export let warnAt = 80;
    export let critAt = 92;

    const RADIUS = 58;
    const CIRCUMFERENCE = 2 * Math.PI * RADIUS;

    $: clamped = Math.max(0, Math.min(Number(value) || 0, 100));
    $: tone = clamped >= critAt ? "crit" : clamped >= warnAt ? "warn" : "";
    $: dashOffset = CIRCUMFERENCE * (1 - clamped / 100);
    $: sparkLine = buildLine(history);
    $: sparkArea = sparkLine ? `${sparkLine} L100,30 L0,30 Z` : "";

    function buildLine(samples) {
        const list = (samples || []).map((item) =>
            Math.max(0, Math.min(Number(item) || 0, 100)),
        );
        if (list.length < 2) {
            return "";
        }
        const step = 100 / (list.length - 1);
        return `M${list
            .map(
                (sample, index) =>
                    `${(index * step).toFixed(2)},${(29 - (sample / 100) * 26).toFixed(2)}`,
            )
            .join(" L")}`;
    }
</script>

<article
    class:light={variant === "light"}
    class:warm={variant === "warm"}
    class:warn={tone === "warn"}
    class:crit={tone === "crit"}
    class="metric-card"
>
    <div class="metric-head">
        <span>{title}</span>
        {#if tone}<em class="alert-badge"
                >{tone === "crit" ? "high" : "elevated"}</em
            >{/if}
        {#if icon}<svelte:component this={icon} size="19" />{/if}
    </div>

    <div class="ring-wrap">
        <svg
            class="ring"
            viewBox="0 0 132 132"
            role="img"
            aria-label={`${title} ${percent(clamped)}`}
        >
            <circle class="ring-track" cx="66" cy="66" r={RADIUS} />
            <circle
                class="ring-value"
                cx="66"
                cy="66"
                r={RADIUS}
                style={`stroke-dasharray:${CIRCUMFERENCE};stroke-dashoffset:${dashOffset}`}
            />
        </svg>
        <strong>{percent(clamped)}</strong>
    </div>

    <svg
        class="spark"
        viewBox="0 0 100 30"
        preserveAspectRatio="none"
        aria-hidden="true"
    >
        {#if sparkArea}
            <path class="spark-area" d={sparkArea} />
            <path class="spark-line" d={sparkLine} />
        {:else}
            <line class="spark-idle" x1="0" y1="29" x2="100" y2="29" />
        {/if}
    </svg>

    <div class="kv">
        <span>{metaLabel}</span>
        <strong>{metaValue}</strong>
    </div>
</article>

<style>
    .metric-card {
        min-height: var(--card-min, 318px);
        display: grid;
        grid-template-rows: auto 1fr auto auto;
        gap: 12px;
        padding: 20px;
    }

    .metric-card.light {
        background: #d8f0ff;
        color: #111111;
    }

    .metric-card.warm {
        background: #fff8c8;
        color: #111111;
    }

    .metric-card.warn {
        background: #fff6bd;
        color: #111111;
    }

    .metric-card.crit {
        background: #ffd8d6;
        color: #111111;
    }

    .metric-head {
        display: flex;
        align-items: center;
        gap: 10px;
        color: inherit;
        font-size: 0.86rem;
        font-weight: 800;
    }

    .metric-head :global(svg:last-child) {
        margin-left: auto;
    }

    .alert-badge {
        padding: 3px 10px;
        border-radius: 999px;
        background: #101010;
        color: #ffd8d6;
        font-size: 0.6rem;
        font-style: normal;
        font-weight: 800;
    }

    .metric-card.warn .alert-badge {
        color: #fff6bd;
    }

    .metric-head :global(svg) {
        color: currentColor;
    }

    .ring-wrap {
        display: grid;
        place-items: center;
        align-self: center;
    }

    .ring-wrap > * {
        grid-area: 1 / 1;
    }

    .ring {
        width: 158px;
        height: 158px;
    }

    .ring-track,
    .ring-value {
        fill: none;
        stroke: currentColor;
        stroke-width: 11;
    }

    .ring-track {
        opacity: 0.14;
    }

    .ring-value {
        stroke-linecap: round;
        transform: rotate(-90deg);
        transform-origin: center;
        transition: stroke-dashoffset 700ms cubic-bezier(0.3, 0.9, 0.3, 1);
    }

    .ring-wrap strong {
        font-size: 1.85rem;
    }

    .spark {
        width: 100%;
        height: 38px;
        display: block;
    }

    .spark-area {
        fill: currentColor;
        opacity: 0.12;
    }

    .spark-line {
        fill: none;
        stroke: currentColor;
        stroke-width: 1.7;
        stroke-linejoin: round;
        stroke-linecap: round;
        vector-effect: non-scaling-stroke;
    }

    .spark-idle {
        stroke: currentColor;
        stroke-width: 1.5;
        opacity: 0.2;
        vector-effect: non-scaling-stroke;
    }

    .kv {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 14px;
        color: var(--muted);
        font-size: 0.72rem;
    }

    .metric-card.light .kv,
    .metric-card.warm .kv,
    .metric-card.warn .kv,
    .metric-card.crit .kv {
        color: #3d4148;
    }

    .kv strong {
        color: inherit;
    }
</style>
