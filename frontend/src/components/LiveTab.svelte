<script>
  // Live view: preview of every strip plus the layer stack.
  import { app, save, defaultParams, uid } from '../lib/state.svelte.js';
  import StripCanvas from './StripCanvas.svelte';
  import LayerCard from './LayerCard.svelte';

  function addLayer() {
    const effect = 'solid';
    app.show.layers.push({
      id: uid(),
      name: `Layer ${app.show.layers.length + 1}`,
      groupId: '',
      effect,
      params: defaultParams(effect),
      opacity: 1,
      blend: 'normal',
      muted: false,
    });
    save();
  }

  function move(i, dir) {
    const l = app.show.layers;
    const j = i + dir;
    if (j < 0 || j >= l.length) return;
    [l[i], l[j]] = [l[j], l[i]];
    save();
  }

  function remove(i) {
    app.show.layers.splice(i, 1);
    save();
  }
</script>

<section>
  <h2>Preview</h2>
  {#each app.show.strips as s, i (s.id)}
    <div class="strip-row">
      <div class="strip-name">{s.name} <small>{s.pixels}px</small></div>
      <StripCanvas index={i} pixels={s.pixels} />
    </div>
  {:else}
    <p class="hint">No strips yet. Add controllers and strips in Setup.</p>
  {/each}
</section>

<section>
  <h2>Layers</h2>
  <p class="hint">Listed bottom to top: each layer paints over the ones above it in this list.</p>
  {#each app.show.layers as layer, i (layer.id)}
    <LayerCard {layer} index={i} count={app.show.layers.length} onmove={(d) => move(i, d)} onremove={() => remove(i)} />
  {/each}
  <button class="primary" onclick={addLayer}>+ Add layer</button>
</section>

<style>
  .strip-row {
    margin-bottom: 0.6rem;
  }
  .strip-name {
    font-size: 0.85rem;
    margin-bottom: 0.2rem;
  }
  .strip-name small {
    color: var(--muted);
  }
</style>
