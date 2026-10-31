<script>
  // Renders the dialog opened with ask() from lib/confirm.svelte.js.
  import { dialog, answer } from '../lib/confirm.svelte.js';

  let okBtn = $state();
  $effect(() => {
    if (dialog.open) okBtn?.focus();
  });
</script>

<svelte:window onkeydown={(e) => dialog.open && e.key === 'Escape' && answer(false)} />

{#if dialog.open}
  <div class="backdrop" role="presentation" onclick={(e) => e.target === e.currentTarget && answer(false)}>
    <div class="dialog" role="alertdialog" aria-modal="true" aria-labelledby="confirm-msg">
      <p id="confirm-msg">{dialog.message}</p>
      <div class="row">
        <span class="spacer"></span>
        <button onclick={() => answer(false)}>Cancel</button>
        <button bind:this={okBtn} class={dialog.danger ? 'alert' : 'primary'} onclick={() => answer(true)}>{dialog.confirmLabel}</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 100;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 1rem;
  }
  .dialog {
    background: var(--card);
    border: 1px solid var(--line);
    border-radius: 10px;
    padding: 1rem;
    width: min(26rem, 100%);
    box-shadow: 0 10px 40px rgba(0, 0, 0, 0.5);
  }
  p {
    margin: 0 0 1rem;
    line-height: 1.4;
  }
</style>
