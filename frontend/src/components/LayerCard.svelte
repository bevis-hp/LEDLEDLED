<script>
  // One effect layer: which group it paints, how it blends and its params.
  import { app, save, defaultParams } from '../lib/state.svelte.js';
  import ParamControl from './ParamControl.svelte';

  /** @type {{ layer: any, index: number, count: number, onmove: (dir:number)=>void, onremove: ()=>void }} */
  let { layer, index, count, onmove, onremove } = $props();

  let effectDef = $derived(app.meta?.effects.find((e) => e.name === layer.effect));
  let open = $state(true);

  function changeEffect() {
    layer.params = defaultParams(layer.effect);
    save();
  }
</script>

<div class="card layer" class:muted={layer.muted}>
  <div class="row head">
    <button class="icon" title={open ? 'Collapse' : 'Expand'} onclick={() => (open = !open)}>{open ? '▾' : '▸'}</button>
    <input class="name" bind:value={layer.name} onchange={save} placeholder="Layer name" />
    <button
      class="icon"
      class:on={!layer.muted}
      title={layer.muted ? 'Unmute' : 'Mute'}
      onclick={() => {
        layer.muted = !layer.muted;
        save();
      }}>{layer.muted ? '○' : '●'}</button
    >
    <button class="icon" title="Move up" disabled={index === 0} onclick={() => onmove(-1)}>↑</button>
    <button class="icon" title="Move down" disabled={index === count - 1} onclick={() => onmove(1)}>↓</button>
    <button class="icon danger" title="Delete layer" onclick={onremove}>✕</button>
  </div>

  {#if open}
    <div class="grid2">
      <label>
        <span>Effect</span>
        <select bind:value={layer.effect} onchange={changeEffect}>
          {#each app.meta?.effects ?? [] as e (e.name)}
            <option value={e.name}>{e.label}</option>
          {/each}
        </select>
      </label>
      <label>
        <span>Group</span>
        <select bind:value={layer.groupId} onchange={save}>
          <option value="">All pixels</option>
          {#each app.show.groups as g (g.id)}
            <option value={g.id}>{g.name || g.id}</option>
          {/each}
        </select>
      </label>
      <label>
        <span>Blend</span>
        <select bind:value={layer.blend} onchange={save}>
          {#each app.meta?.blendModes ?? [] as b (b)}
            <option value={b}>{b}</option>
          {/each}
        </select>
      </label>
      <label>
        <span>Opacity {Math.round(layer.opacity * 100)}%</span>
        <input type="range" min="0" max="1" step="0.01" bind:value={layer.opacity} oninput={save} />
      </label>
    </div>

    {#if effectDef}
      {#if effectDef.description}<p class="hint">{effectDef.description}</p>{/if}
      {#each effectDef.params as p (p.name)}
        <ParamControl def={p} params={layer.params} onchange={save} />
      {/each}
    {/if}
  {/if}
</div>

<style>
  .layer.muted {
    opacity: 0.55;
  }
  .head {
    margin-bottom: 0.5rem;
  }
  .name {
    flex: 1;
    min-width: 0;
    font-weight: 600;
  }
</style>
