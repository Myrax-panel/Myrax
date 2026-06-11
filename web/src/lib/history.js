import { writable } from "svelte/store";

const MAX_SAMPLES = 48;

export const metricHistory = writable({});

export function pushMetrics(samples) {
  metricHistory.update((map) => {
    const next = { ...map };
    for (const [key, value] of Object.entries(samples)) {
      const series = [...(next[key] || []), Number(value) || 0];
      next[key] = series.slice(-MAX_SAMPLES);
    }
    return next;
  });
}
