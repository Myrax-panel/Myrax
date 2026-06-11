import { writable } from 'svelte/store';

let toastId = 0;

export const toasts = writable([]);

export function showToast(toast) {
  const id = ++toastId;
  const entry = {
    id,
    title: '',
    message: '',
    actionLabel: '',
    onAction: null,
    timeout: 5000,
    tone: 'default',
    ...toast
  };

  toasts.update((items) => [...items, entry]);

  if (entry.timeout > 0) {
    window.setTimeout(() => dismissToast(id), entry.timeout);
  }

  return id;
}

export function dismissToast(id) {
  toasts.update((items) => items.filter((item) => item.id !== id));
}
