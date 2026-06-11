<script>
  import { Button } from 'bits-ui';
  import { dismissToast, toasts } from '../lib/toast.js';

  function runAction(toast) {
    toast.onAction?.();
    dismissToast(toast.id);
  }
</script>

<div class="toast-host" aria-live="polite" aria-relevant="additions removals">
  {#each $toasts as toast (toast.id)}
    <article class:error={toast.tone === 'error'} class="toast">
      <div>
        {#if toast.title}
          <strong>{toast.title}</strong>
        {/if}
        {#if toast.message}
          <p>{toast.message}</p>
        {/if}
      </div>
      {#if toast.actionLabel}
        <Button.Root class="toast-action" onclick={() => runAction(toast)}>
          {toast.actionLabel}
        </Button.Root>
      {/if}
    </article>
  {/each}
</div>

<style>
  .toast-host {
    position: fixed;
    right: 18px;
    bottom: 18px;
    z-index: 120;
    width: min(440px, calc(100vw - 32px));
    display: grid;
    gap: 10px;
    pointer-events: none;
  }

  .toast {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 14px;
    padding: 12px;
    border: 1px solid var(--line);
    border-radius: 18px;
    color: var(--text);
    background: var(--panel);
    box-shadow: 0 18px 70px rgb(0 0 0 / 38%);
    pointer-events: auto;
  }

  .toast.error {
    border-color: #5a3030;
    color: #ffd8d6;
  }

  strong {
    display: block;
    font-size: 0.78rem;
  }

  p {
    margin-top: 4px;
    color: var(--muted);
    font-size: 0.72rem;
    line-height: 1.45;
  }

  :global(.toast-action) {
    min-height: 38px;
    padding: 0 14px;
    border: 1px solid #f5f5f5;
    border-radius: 999px;
    color: #101010;
    background: #f5f5f5;
    font-size: 0.74rem;
    font-weight: 900;
    white-space: nowrap;
  }

  @media (max-width: 980px) {
    .toast-host {
      right: 12px;
      bottom: 86px;
      width: calc(100vw - 24px);
    }
  }
</style>
