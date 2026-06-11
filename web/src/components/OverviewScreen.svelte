<script>
    import { onMount } from "svelte";
    import { ScrollArea } from "bits-ui";
    import {
        Cpu,
        HardDrive,
        ListTree,
        MemoryStick,
        Network,
        Server,
    } from "@lucide/svelte";
    import {
        byteRate,
        bytes,
        duration,
        intervalMs,
        percent,
    } from "../lib/format.js";
    import { metricHistory } from "../lib/history.js";
    import MetricCard from "./MetricCard.svelte";

    export let stats = null;
    export let refreshRate = "2s";

    let topProcesses = [];
    let processesTimer = null;
    let mounted = false;

    $: cpuUsage = stats?.cpu?.usage || 0;
    $: memoryUsage = stats?.memory?.usage || 0;
    $: primaryDisk = stats?.disks?.[0];
    $: diskUsage = primaryDisk?.usage || 0;
    $: history = $metricHistory;

    $: totalRx = (stats?.network || []).reduce(
        (sum, item) => sum + (item.rxRate || 0),
        0,
    );
    $: totalTx = (stats?.network || []).reduce(
        (sum, item) => sum + (item.txRate || 0),
        0,
    );
    $: netScale = Math.max(
        ...(history.netRx || []),
        ...(history.netTx || []),
        1,
    );
    $: rxLine = sparkPath(history.netRx || [], netScale);
    $: txLine = sparkPath(history.netTx || [], netScale);
    $: rxArea = rxLine ? `${rxLine} L100,30 L0,30 Z` : "";

    $: if (mounted) {
        refreshRate;
        restartProcessesTimer();
    }

    function sparkPath(samples, max) {
        if (samples.length < 2) {
            return "";
        }
        const step = 100 / (samples.length - 1);
        return `M${samples
            .map(
                (sample, index) =>
                    `${(index * step).toFixed(2)},${(29 - (sample / max) * 26).toFixed(2)}`,
            )
            .join(" L")}`;
    }

    async function loadTopProcesses() {
        try {
            const response = await fetch("/api/processes?limit=30");
            if (!response.ok) {
                return;
            }
            const payload = await response.json();
            topProcesses = [...payload]
                .sort((a, b) => Number(b.cpu || 0) - Number(a.cpu || 0))
                .slice(0, 3);
        } catch {
            // Keep the previous list on transient errors; the timer retries shortly.
        }
    }

    function restartProcessesTimer() {
        clearInterval(processesTimer);
        processesTimer = setInterval(loadTopProcesses, intervalMs(refreshRate));
    }

    onMount(() => {
        mounted = true;
        loadTopProcesses();
        restartProcessesTimer();
        return () => clearInterval(processesTimer);
    });
</script>

<section class="metrics">
    <MetricCard
        title="CPU"
        icon={Cpu}
        value={cpuUsage}
        history={history.cpu || []}
        warnAt={75}
        critAt={90}
        metaLabel="load"
        metaValue={stats?.cpu?.loadAverage?.[0]?.toFixed?.(2) || "0.00"}
    />

    <MetricCard
        title="RAM"
        icon={MemoryStick}
        value={memoryUsage}
        variant="light"
        history={history.ram || []}
        warnAt={80}
        critAt={92}
        metaLabel="used"
        metaValue={bytes(stats?.memory?.used)}
    />

    <MetricCard
        title="Disk"
        icon={HardDrive}
        value={diskUsage}
        variant="warm"
        history={history.disk || []}
        warnAt={75}
        critAt={85}
        metaLabel={primaryDisk?.mountpoint || "/"}
        metaValue={bytes(primaryDisk?.used)}
    />
</section>

<section class="lower-grid">
    <article class="panel interfaces">
        <div class="panel-head">
            <Network size="18" />
            <h2>Network</h2>
        </div>

        <div class="net-summary">
            <div class="net-rates">
                <strong>&darr; {byteRate(totalRx)}</strong>
                <strong class="tx">&uarr; {byteRate(totalTx)}</strong>
            </div>
            <svg
                class="net-spark"
                viewBox="0 0 100 30"
                preserveAspectRatio="none"
                aria-hidden="true"
            >
                {#if rxArea}
                    <path class="rx-area" d={rxArea} />
                    <path class="rx-line" d={rxLine} />
                    <path class="tx-line" d={txLine} />
                {:else}
                    <line class="spark-idle" x1="0" y1="29" x2="100" y2="29" />
                {/if}
            </svg>
        </div>

        <ScrollArea.Root class="scroll">
            <ScrollArea.Viewport class="viewport">
                {#if stats?.network?.length}
                    {#each stats.network as adapter}
                        <div class="row">
                            <span>{adapter.name}</span>
                            <strong class="traffic">
                                <span>rx {byteRate(adapter.rxRate)}</span>
                                <span>tx {byteRate(adapter.txRate)}</span>
                            </strong>
                        </div>
                    {/each}
                {:else}
                    <div class="empty">no data</div>
                {/if}
            </ScrollArea.Viewport>
            <ScrollArea.Scrollbar class="scrollbar" orientation="vertical">
                <ScrollArea.Thumb class="thumb" />
            </ScrollArea.Scrollbar>
        </ScrollArea.Root>
    </article>

    <article class="panel">
        <div class="panel-head">
            <ListTree size="18" />
            <h2>Top processes</h2>
        </div>
        {#if topProcesses.length}
            {#each topProcesses as process}
                <div class="row">
                    <span class="proc-name">{process.name || process.pid}</span>
                    <strong
                        >{percent(process.cpu)} &middot; {bytes(
                            process.rss,
                        )}</strong
                    >
                </div>
            {/each}
        {:else}
            <div class="empty">no data</div>
        {/if}
    </article>

    <article class="panel">
        <div class="panel-head">
            <Server size="18" />
            <h2>Host</h2>
        </div>
        <div class="table">
            <div>
                <span>os</span><strong>{stats?.host?.os || "Linux"}</strong>
            </div>
            <div>
                <span>kernel</span><strong>{stats?.host?.kernel || "-"}</strong>
            </div>
            <div>
                <span>uptime</span><strong
                    >{duration(stats?.host?.uptime)}</strong
                >
            </div>
            <div>
                <span>cores</span><strong>{stats?.cpu?.cores || 0}</strong>
            </div>
        </div>
    </article>
</section>

<style>
    .metrics {
        display: grid;
        grid-template-columns: repeat(3, minmax(220px, 1fr));
        gap: 16px;
    }

    .lower-grid {
        display: grid;
        grid-template-columns: 1.1fr 0.95fr 0.75fr;
        gap: 16px;
        margin-top: 16px;
    }

    .panel {
        padding: 18px;
    }

    .panel-head {
        display: flex;
        align-items: center;
        gap: 10px;
        margin-bottom: 12px;
        color: var(--text);
    }

    .net-summary {
        margin-bottom: 12px;
    }

    .net-rates {
        display: flex;
        align-items: baseline;
        gap: 16px;
        margin-bottom: 8px;
    }

    .net-rates strong {
        color: var(--text);
        font-size: 0.95rem;
    }

    .net-rates .tx {
        color: var(--muted);
    }

    .net-spark {
        width: 100%;
        height: 44px;
        display: block;
        color: var(--text);
    }

    .rx-area {
        fill: currentColor;
        opacity: 0.1;
    }

    .rx-line {
        fill: none;
        stroke: currentColor;
        stroke-width: 1.7;
        stroke-linejoin: round;
        stroke-linecap: round;
        vector-effect: non-scaling-stroke;
    }

    .tx-line {
        fill: none;
        stroke: var(--muted);
        stroke-width: 1.5;
        stroke-linejoin: round;
        stroke-linecap: round;
        vector-effect: non-scaling-stroke;
        opacity: 0.8;
    }

    .spark-idle {
        stroke: currentColor;
        stroke-width: 1.5;
        opacity: 0.2;
        vector-effect: non-scaling-stroke;
    }

    :global(.scroll) {
        height: 132px;
        overflow: hidden;
    }

    :global(.viewport) {
        height: 100%;
        padding-right: 12px;
    }

    .empty {
        height: 132px;
        display: grid;
        place-items: center;
        border: 1px dashed var(--line);
        border-radius: 18px;
        color: var(--muted);
        font-size: 0.78rem;
    }

    .traffic {
        display: inline-flex;
        justify-content: flex-end;
        gap: 12px;
    }

    .proc-name {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    :global(.scrollbar) {
        width: 7px;
        background: #252525;
    }

    :global(.thumb) {
        border-radius: 999px;
        background: #f5f5f5;
    }

    @media (max-width: 980px) {
        .metrics,
        .lower-grid {
            grid-template-columns: 1fr;
        }
    }
</style>
