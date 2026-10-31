<script>
  // Live view: preview of every strip plus the layer stack.
  import { onMount } from 'svelte';
  import { app, save, defaultParams, uid, library } from '../lib/state.svelte.js';
  import StripCanvas from './StripCanvas.svelte';
  import LayerCard from './LayerCard.svelte';

  let presets = $state([]);
  const refreshPresets = async () => (presets = await library.list('presets').catch(() => []));
  onMount(refreshPresets);

  const power = (id) => app.status?.strips?.find((s) => s.id === id);

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
      tempoSync: false,
      mods: [],
    });
    save();
  }

  // Presets carry a layer and its keyframe tracks; give them fresh IDs.
  async function addFromPreset(ev) {
    const name = ev.currentTarget.value;
    ev.currentTarget.value = '';
    if (!name) return;
    try {
      const p = await library.get('presets', name);
      const id = uid();
      const groupOk = app.show.groups.some((g) => g.id === p.layer.groupId);
      app.show.layers.push({ ...p.layer, id, name: p.layer.name || name, groupId: groupOk ? p.layer.groupId : '', mods: p.layer.mods ?? [] });
      for (const t of p.tracks ?? []) app.show.timeline.tracks.push({ ...t, layerId: id });
      save();
    } catch (e) {
      app.error = e.message;
    }
  }

  function move(i, dir) {
    const l = app.show.layers;
    const j = i + dir;
    if (j < 0 || j >= l.length) return;
    [l[i], l[j]] = [l[j], l[i]];
    save();
  }

  function remove(i) {
    const id = app.show.layers[i].id;
    app.show.layers.splice(i, 1);
    app.show.timeline.tracks = app.show.timeline.tracks.filter((t) => t.layerId !== id);
    save();
  }
</script>

<section>
  <h2>Preview</h2>
  {#each app.show.strips as s, i (s.id)}
    {@const pw = power(s.id)}
    <div class="strip-row">
      <div class="strip-name">
        {s.name} <small>{s.pixels}px</small>
        {#if pw}
          <small class="power" class:limited={pw.limited} title="Estimated peak over the last second">
            {pw.amps.toFixed(2)} A · {pw.watts.toFixed(1)} W
            {#if pw.limited}· limited from {pw.requestedAmps.toFixed(2)} A{/if}
          </small>
        {/if}
      </div>
      <StripCanvas index={i} pixels={s.pixels} />
    </div>
  {:else}
    <p class="hint">No strips yet. Add controllers and strips in Setup.</p>
  {/each}
</section>

<section>
  <h2>Layers</h2>
  <p class="hint">
    Listed bottom to top: each layer paints over the ones above it in this list. ◇ adds a keyframe at the timeline playhead.
  </p>
  {#each app.show.layers as layer, i (layer.id)}
    <LayerCard {layer} index={i} count={app.show.layers.length} onmove={(d) => move(i, d)} onremove={() => remove(i)} onpreset={refreshPresets} />
  {/each}
  <div class="row">
    <button class="primary" onclick={addLayer}>+ Add layer</button>
    {#if presets.length}
      <select onchange={addFromPreset} onfocus={refreshPresets}>
        <option value="">+ Add from preset…</option>
        {#each presets as p (p.name)}<option value={p.name}>{p.name}</option>{/each}
      </select>
    {/if}
  </div>
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
  .power {
    float: right;
  }
  .power.limited {
    color: var(--warn);
  }
</style>
