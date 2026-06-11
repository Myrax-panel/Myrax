<script>
  import { AlertDialog } from 'bits-ui';

  export let actions = [];
  export let pendingAction = '';
  export let onRunAction = () => {};
</script>

<section class="action-grid">
  {#each actions as action}
    <AlertDialog.Root>
      <article class="action-tile">
        <div class="action-header">
          <svelte:component this={action.icon} size="20" />
          <h2>{action.label}</h2>
        </div>
        <AlertDialog.Trigger class="danger" disabled={!!pendingAction}>
          {pendingAction === action.id ? 'sending' : 'run'}
        </AlertDialog.Trigger>
      </article>
      <AlertDialog.Portal>
        <AlertDialog.Overlay class="overlay" />
        <AlertDialog.Content class="dialog">
          <AlertDialog.Title>{action.label} server?</AlertDialog.Title>
          <AlertDialog.Description>System command will run now.</AlertDialog.Description>
          <div class="dialog-row">
            <AlertDialog.Cancel class="dialog-button">cancel</AlertDialog.Cancel>
            <AlertDialog.Action class="dialog-button danger-fill" onclick={() => onRunAction(action.id)}>
              confirm
            </AlertDialog.Action>
          </div>
        </AlertDialog.Content>
      </AlertDialog.Portal>
    </AlertDialog.Root>
  {/each}
</section>

<style>
  .action-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(220px, 360px));
    gap: 16px;
  }

  .action-tile {
    min-height: 124px;
    display: grid;
    align-content: space-between;
    justify-items: start;
    padding: 20px;
  }

  .action-header {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .action-tile :global(svg) {
    color: var(--text);
  }

  :global(.danger),
  :global(.dialog-button) {
    min-height: 40px;
    padding: 0 15px;
    border: 1px solid var(--line);
    border-radius: 999px;
    color: var(--text);
    background: #202124;
    font-size: 0.78rem;
    font-weight: 800;
  }

  :global(.danger) {
    color: #101010;
    background: #ffd8d6;
  }

  :global(.overlay) {
    position: fixed;
    inset: 0;
    z-index: 20;
    background: rgb(0 0 0 / 72%);
  }

  :global(.dialog) {
    position: fixed;
    top: 50%;
    left: 50%;
    z-index: 30;
    width: min(390px, calc(100vw - 32px));
    padding: 20px;
    border: 1px solid var(--line);
    border-radius: 24px;
    background: #161616;
    transform: translate(-50%, -50%);
  }

  :global(.dialog [data-dialog-description]) {
    display: block;
    margin-top: 10px;
    color: var(--muted);
    font-size: 0.78rem;
  }

  .dialog-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
    margin-top: 18px;
  }

  :global(.danger-fill) {
    color: #101010;
    background: #ffd8d6;
  }

  @media (max-width: 980px) {
    .action-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
