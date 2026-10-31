<script>
  // One parameter control, built from the effect's schema.
  //
  // `keyState` drives the keyframe button: 'none' (not animated), 'track'
  // (animated, no key at the playhead) or 'on' (key at the playhead).
  // `mod` names the modulation source driving it, if any.
  import { allPalettes, findPalette, cssGradient } from '../lib/palettes.js';

  /** @type {{ def: any, value: any, onchange: (v:any)=>void, keyState?: string, onkey?: ()=>void, mod?: string|null }} */
  let { def, value, onchange, keyState = 'none', onkey, mod = null } = $props();

  let animatable = $derived(def.type === 'number' || def.type === 'color');
  const fmt = (v) => Number(v).toFixed(def.step >= 1 ? 0 : 2);
</script>

<div class="param" class:animated={keyState !== 'none'}>
  <span class="label">
    {def.label}
    {#if mod}<span class="mod" title="Driven by a modulator">~{mod}</span>{/if}
  </span>
  <span class="control">
    {#if def.type === 'number'}
      <input type="range" min={def.min} max={def.max} step={def.step || 0.01} {value} oninput={(e) => onchange(Number(e.currentTarget.value))} />
    {:else if def.type === 'color'}
      <input type="color" {value} oninput={(e) => onchange(e.currentTarget.value)} />
    {:else if def.type === 'bool'}
      <input type="checkbox" checked={value} onchange={(e) => onchange(e.currentTarget.checked)} />
    {:else if def.type === 'select'}
      <select {value} onchange={(e) => onchange(e.currentTarget.value)}>
        {#each def.options as o (o)}<option value={o}>{o}</option>{/each}
      </select>
    {:else if def.type === 'palette'}
      <select {value} onchange={(e) => onchange(e.currentTarget.value)}>
        <option value="">(use colours)</option>
        {#each allPalettes() as p (p.id)}<option value={p.id}>{p.name}</option>{/each}
      </select>
    {/if}
  </span>
  {#if def.type === 'palette'}
    <span class="swatch" style="background: {value ? cssGradient(findPalette(value)?.stops) : 'transparent'}"></span>
  {:else}
    <span class="value mono">{def.type === 'number' ? fmt(value) : def.type === 'color' ? value : ''}</span>
  {/if}
  {#if animatable && onkey}
    <button
      class="key"
      class:on={keyState === 'on'}
      class:track={keyState === 'track'}
      title={keyState === 'on' ? 'Remove keyframe at playhead' : 'Add keyframe at playhead'}
      onclick={onkey}>{keyState === 'on' ? '◆' : '◇'}</button
    >
  {:else}
    <span></span>
  {/if}
</div>

<style>
  .param {
    display: grid;
    grid-template-columns: minmax(5.5rem, 8rem) 1fr 3.2rem 2rem;
    align-items: center;
    gap: 0.4rem;
    margin: 0.2rem 0;
  }
  .label {
    font-size: 0.8rem;
    line-height: 1.15;
    color: var(--muted);
  }
  .animated .label {
    color: var(--warn);
  }
  .mod {
    font-size: 0.7rem;
    color: var(--accent);
    margin-left: 0.2rem;
  }
  .control > * {
    width: 100%;
  }
  input[type='checkbox'] {
    width: 1.1rem;
  }
  input[type='color'] {
    height: 2rem;
  }
  .value {
    font-size: 0.75rem;
    text-align: right;
    color: var(--muted);
    overflow: hidden;
  }
  .swatch {
    height: 1.4rem;
    border-radius: 4px;
  }
  .key {
    min-width: 2rem;
    min-height: 2rem;
    padding: 0;
    color: var(--muted);
  }
  .key.track {
    color: var(--warn);
  }
  .key.on {
    color: var(--warn);
    border-color: var(--warn);
  }
</style>
