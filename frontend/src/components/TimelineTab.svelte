<script>
  // Timeline: one looping sequence measured in beats. Tracks animate layer
  // parameters between keyframes; the playhead follows the tempo.
  import { app, live, ui, save, transport } from '../lib/state.svelte.js';
  import { setKey, evalTrack, removeTrack, snapBeat, findTrack } from '../lib/keyframes.js';
  import TrackLane from './TrackLane.svelte';

  const snaps = [
    [0, 'off'],
    [1, '1 beat'],
    [0.5, '½'],
    [0.25, '¼'],
    [0.125, '⅛'],
  ];
  $effect(() => localStorage.setItem('snap', String(ui.snap)));

  // Selected keyframe, by identity rather than object reference, because the
  // show object is replaced whenever the server sends back its copy.
  let sel = $state({ layerId: '', param: '', key: -1 });
  let selTrack = $derived(sel.layerId ? findTrack(app.show, sel.layerId, sel.param) : null);
  const isSel = (t) => t.layerId === sel.layerId && t.param === sel.param;
  const clearSel = () => (sel = { layerId: '', param: '', key: -1 });

  const layerOf = (id) => app.show.layers.find((l) => l.id === id);
  const opacityDef = { name: 'opacity', label: 'Opacity', type: 'number', min: 0, max: 1 };

  function paramDef(track) {
    if (track.param === 'opacity') return opacityDef;
    const l = layerOf(track.layerId);
    const eff = app.meta?.effects.find((e) => e.name === l?.effect);
    return eff?.params.find((p) => p.name === track.param);
  }

  // Seeking is throttled so dragging across the ruler doesn't flood the Pi.
  let seekTimer = null;
  let seekTo = 0;
  function seek(beat) {
    seekTo = beat;
    if (seekTimer) return;
    seekTimer = setTimeout(() => {
      seekTimer = null;
      transport('seek', seekTo);
    }, 50);
  }

  // Ruler: drag to scrub.
  let ruler;
  let scrubbing = false;
  function rulerBeat(ev) {
    const r = ruler.getBoundingClientRect();
    return snapBeat(Math.max(0, Math.min(1, (ev.clientX - r.left) / r.width)) * app.show.timeline.lengthBeats, ui.snap);
  }

  // Quick-add: key any animatable parameter at the playhead.
  let addLayer = $state('');
  let addParam = $state('opacity');
  let addParams = $derived.by(() => {
    const l = layerOf(addLayer);
    const eff = app.meta?.effects.find((e) => e.name === l?.effect);
    return [opacityDef, ...(eff?.params.filter((p) => p.type === 'number' || p.type === 'color') ?? [])];
  });
  function quickKey() {
    const l = layerOf(addLayer);
    if (!l) return;
    const value = addParam === 'opacity' ? l.opacity : l.params[addParam];
    setKey(app.show, l.id, addParam, snapBeat(live.timeline.pos, ui.snap), value);
    save();
  }

  function deleteKey() {
    const t = selTrack;
    t.keys.splice(sel.key, 1);
    if (!t.keys.length) removeTrack(app.show, t);
    clearSel();
    save();
  }

  let selKey = $derived(selTrack?.keys[sel.key]);
  let selDef = $derived(selTrack ? paramDef(selTrack) : null);
</script>

<section class="card">
  <div class="row">
    <button class="icon" title="Stop" onclick={() => transport('stop')}>■</button>
    <button class="icon" class:on={live.timeline.playing} onclick={() => transport(live.timeline.playing ? 'pause' : 'play')}>
      {live.timeline.playing ? '❚❚' : '▶'}
    </button>
    <span class="mono pos">beat {live.timeline.pos.toFixed(2)}</span>
    <span class="spacer"></span>
    <label class="inline">Length <input type="number" min="1" max="1024" step="1" bind:value={app.show.timeline.lengthBeats} onchange={save} /> beats</label>
    <label class="inline"><input type="checkbox" checked={!app.show.timeline.oneShot} onchange={(e) => { app.show.timeline.oneShot = !e.currentTarget.checked; save(); }} /> loop</label>
    <label class="inline">
      Snap
      <select bind:value={ui.snap}>
        {#each snaps as [v, label] (v)}<option value={v}>{label}</option>{/each}
      </select>
    </label>
  </div>

  <!-- Ruler / scrubber -->
  <div
    class="ruler"
    bind:this={ruler}
    role="slider"
    tabindex="0"
    aria-label="Timeline position"
    aria-valuenow={live.timeline.pos}
    onpointerdown={(e) => {
      scrubbing = true;
      ruler.setPointerCapture(e.pointerId);
      seek(rulerBeat(e));
    }}
    onpointermove={(e) => scrubbing && seek(rulerBeat(e))}
    onpointerup={() => (scrubbing = false)}
  >
    {#each Array.from({ length: Math.floor(app.show.timeline.lengthBeats / 4) + 1 }, (_, i) => i * 4) as b (b)}
      <span class="bar" style="left: {(b / app.show.timeline.lengthBeats) * 100}%">{b / 4 + 1}</span>
    {/each}
    <span class="head" style="left: {(live.timeline.pos / app.show.timeline.lengthBeats) * 100}%"></span>
  </div>
</section>

<section>
  <h2>Tracks</h2>
  <p class="hint">
    Move the playhead, then press ◇ next to a parameter on the Live tab (or use the picker below) to add a keyframe. Once a
    parameter has keys, changing it records a key at the playhead. Drag ◆ to move keys; tap a lane to seek.
  </p>

  {#each app.show.timeline.tracks as t (t.layerId + t.param)}
    {@const def = paramDef(t)}
    <div class="track">
      <div class="row small">
        <span class="grow"><b>{layerOf(t.layerId)?.name ?? '?'}</b> › {def?.label ?? t.param}</span>
        <span class="mono value">{(() => { const v = evalTrack(t, live.timeline.pos); return typeof v === 'number' ? v.toFixed(2) : v; })()}</span>
        <button
          class="icon danger small"
          title="Delete track"
          onclick={() => {
            removeTrack(app.show, t);
            clearSel();
            save();
          }}>✕</button
        >
      </div>
      <TrackLane
        track={t}
        length={app.show.timeline.lengthBeats}
        pos={live.timeline.pos}
        range={[def?.min ?? 0, def?.max ?? 1]}
        selected={isSel(t) ? sel.key : -1}
        snap={ui.snap}
        onselect={(i) => (sel = { layerId: t.layerId, param: t.param, key: i })}
        onchange={save}
        onseek={seek}
      />
    </div>
  {:else}
    <p class="hint">No keyframes yet.</p>
  {/each}

  {#if selKey}
    <div class="card keyedit">
      <b>Keyframe</b>
      <label class="inline">beat <input type="number" min="0" max={app.show.timeline.lengthBeats} step={ui.snap || 0.01} bind:value={selKey.beat} onchange={() => { const k = selKey; selTrack.keys.sort((a, b) => a.beat - b.beat); sel.key = selTrack.keys.indexOf(k); save(); }} /></label>
      {#if typeof selKey.value === 'string'}
        <input type="color" bind:value={selKey.value} oninput={save} />
      {:else}
        <label class="inline">value <input type="number" step="any" min={selDef?.min} max={selDef?.max} bind:value={selKey.value} onchange={save} /></label>
      {/if}
      <label class="inline">
        ease
        <select bind:value={selKey.ease} onchange={save}>
          {#each app.meta?.eases ?? [] as e (e)}<option value={e}>{e}</option>{/each}
        </select>
      </label>
      <button class="danger" onclick={deleteKey}>Delete key</button>
    </div>
  {/if}
</section>

<section class="card">
  <h3>Add a keyframe</h3>
  <div class="row">
    <select bind:value={addLayer}>
      <option value="">Layer…</option>
      {#each app.show.layers as l (l.id)}<option value={l.id}>{l.name}</option>{/each}
    </select>
    <select bind:value={addParam} disabled={!addLayer}>
      {#each addParams as p (p.name)}<option value={p.name}>{p.label}</option>{/each}
    </select>
    <button disabled={!addLayer} onclick={quickKey}>◆ Key at playhead (current value)</button>
  </div>
</section>

<style>
  .pos {
    font-size: 0.85rem;
  }
  input[type='number'] {
    width: 4.5rem;
  }
  .ruler {
    position: relative;
    height: 2rem;
    margin-top: 0.6rem;
    background: #0c0e13;
    border-radius: 4px;
    touch-action: none;
    cursor: ew-resize;
    overflow: hidden;
  }
  .bar {
    position: absolute;
    top: 0;
    bottom: 0;
    border-left: 1px solid var(--line);
    font-size: 0.7rem;
    color: var(--muted);
    padding-left: 3px;
  }
  .head {
    position: absolute;
    top: 0;
    bottom: 0;
    width: 2px;
    background: var(--accent);
  }
  .track {
    margin-bottom: 0.7rem;
  }
  .track .value {
    font-size: 0.75rem;
    color: var(--muted);
  }
  .keyedit {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.6rem;
  }
</style>
