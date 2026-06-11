<main class="boot-loader" aria-busy="true" aria-live="polite">
  <div class="loader-card" aria-label="Loading panel">
    <div class="loader-grid">
      <span></span>
      <span></span>
      <span></span>
      <span></span>
    </div>
    <div class="loader-copy">
      <strong>checking session</strong>
      <small>secure panel bootstrap</small>
    </div>
  </div>
</main>

<style>
  .boot-loader {
    min-height: 100vh;
    display: grid;
    place-items: center;
    padding: 24px;
    background: var(--bg);
  }

  .loader-card {
    width: min(360px, 100%);
    display: grid;
    grid-template-columns: auto 1fr;
    align-items: center;
    gap: 16px;
    padding: 16px;
    border: 1px solid var(--line);
    border-radius: 24px;
    background: var(--panel);
    box-shadow: 0 22px 80px rgb(0 0 0 / 34%);
  }

  .loader-grid {
    width: 64px;
    height: 64px;
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 8px;
    padding: 8px;
    border: 1px solid var(--line-soft);
    border-radius: 18px;
    background: var(--panel-2);
  }

  .loader-grid span {
    border-radius: 9px;
    background: var(--text);
    animation: boot-cell 900ms ease-in-out infinite;
  }

  .loader-grid span:nth-child(2) {
    animation-delay: 120ms;
    background: var(--accent);
  }

  .loader-grid span:nth-child(3) {
    animation-delay: 240ms;
    background: var(--accent-2);
  }

  .loader-grid span:nth-child(4) {
    animation-delay: 360ms;
    background: var(--accent-3);
  }

  .loader-copy {
    min-width: 0;
    display: grid;
    gap: 6px;
  }

  .loader-copy strong {
    color: var(--text);
    font-size: 0.9rem;
  }

  .loader-copy small {
    overflow: hidden;
    color: var(--muted);
    font-size: 0.68rem;
    font-weight: 700;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  @keyframes boot-cell {
    0%,
    100% {
      opacity: 0.34;
      transform: scale(0.82);
    }

    45% {
      opacity: 1;
      transform: scale(1);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .loader-grid span {
      animation: none;
    }
  }
</style>
