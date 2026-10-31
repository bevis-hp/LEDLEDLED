<script>
  // Renders the right control for one effect parameter (from its schema).
  /** @type {{ def: any, params: Record<string, any>, onchange: () => void }} */
  let { def, params, onchange } = $props();
</script>

<label class="param">
  <span class="label">{def.label}</span>
  {#if def.type === 'number'}
    <input
      type="range"
      min={def.min}
      max={def.max}
      step={def.step || 0.01}
      bind:value={params[def.name]}
      oninput={onchange}
    />
    <span class="value">{Number(params[def.name]).toFixed(def.step >= 1 ? 0 : 2)}</span>
  {:else if def.type === 'color'}
    <input type="color" bind:value={params[def.name]} oninput={onchange} />
    <span class="value mono">{params[def.name]}</span>
  {:else if def.type === 'bool'}
    <input type="checkbox" bind:checked={params[def.name]} onchange={onchange} />
  {/if}
</label>

<style>
  .param {
    display: grid;
    grid-template-columns: 8.5rem 1fr 3.5rem;
    align-items: center;
    gap: 0.5rem;
    margin: 0.25rem 0;
  }
  .label {
    font-size: 0.85rem;
    color: var(--muted);
  }
  .value {
    font-size: 0.8rem;
    text-align: right;
    color: var(--muted);
  }
  input[type='color'] {
    width: 100%;
    height: 2rem;
  }
</style>
