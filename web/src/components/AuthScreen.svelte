<script>
  import { Button } from 'bits-ui';

  export let configured = true;
  export let error = '';
  export let pending = false;
  export let onLogin = async () => {};

  let username = '';
  let password = '';

  async function submit() {
    await onLogin(username, password);
  }
</script>

<main class="auth-screen">
  <section class="auth-panel">
    <div class="auth-copy">
      <h1>Sign in</h1>
    </div>

    {#if !configured}
      <div class="auth-error">Authentication is not configured. Run installer or `myrax configure` on the server.</div>
    {:else}
      <form onsubmit={(event) => { event.preventDefault(); submit(); }}>
        <label>
          <span>login</span>
          <input bind:value={username} autocomplete="username" required />
        </label>
        <label>
          <span>password</span>
          <input bind:value={password} autocomplete="current-password" type="password" required />
        </label>
        {#if error}
          <div class="auth-error">{error}</div>
        {/if}
        <Button.Root class="auth-button" type="submit" disabled={pending}>
          {pending ? 'checking' : 'enter'}
        </Button.Root>
      </form>
    {/if}
  </section>
</main>

<style>
  .auth-screen {
    min-height: 100vh;
    display: grid;
    place-items: center;
    padding: 28px;
    background: var(--bg);
  }

  .auth-panel {
    width: min(440px, 100%);
    display: grid;
    gap: 22px;
    padding: 24px;
    border: 1px solid var(--line);
    border-radius: 24px;
    background: color-mix(in srgb, var(--panel) 94%, transparent);
    box-shadow: 0 22px 80px rgb(0 0 0 / 38%);
  }

  .auth-copy {
    display: grid;
    gap: 8px;
  }

  label span {
    color: var(--muted);
    font-size: 0.72rem;
    font-weight: 700;
  }

  .auth-copy h1 {
    font-size: clamp(2.2rem, 8vw, 4.2rem);
  }

  form {
    display: grid;
    gap: 12px;
  }

  label {
    display: grid;
    gap: 7px;
  }

  input {
    width: 100%;
    height: 54px;
    padding: 0 14px;
    border: 1px solid var(--line);
    border-radius: 14px;
    color: var(--text);
    background: var(--panel-2);
    outline: none;
  }

  input:focus {
    border-color: var(--text);
  }

  :global(.auth-button) {
    height: 54px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border: 1px solid #f5f5f5;
    border-radius: 14px;
    color: #101010;
    background: #f5f5f5;
    font-weight: 900;
    transition: background 150ms ease, color 150ms ease;
  }

  :global(.auth-button:hover:not(:disabled)) {
    color: var(--text);
    background: #2a2b2f;
    border-color: var(--line);
  }

  :global(.auth-button:disabled) {
    opacity: 0.55;
    cursor: not-allowed;
  }

  .auth-error {
    padding: 11px 12px;
    border: 1px solid #5a3030;
    border-radius: 14px;
    color: #ffd8d6;
    background: #201414;
    font-size: 0.72rem;
    line-height: 1.45;
  }
</style>
