<script>
  // One effect layer: target group, blend, params (with keyframes) and
  // audio/tempo modulators.
  import { app, live, ui, save, defaultParams, library } from '../lib/state.svelte.js';
  import { findTrack, keyIndexAt, setKey, removeKeyAt, evalTrack, snapBeat } from '../lib/keyframes.js';
  import ParamControl from './ParamControl.svelte';

  /** @type {{ layer: any, index: number, count: number, onmove: (dir:number)=>void, onremove: ()=>void, onpreset?: ()=>void }} */
  let { layer, index, count, onmove, onremove, onpreset } = $props();

  // Opacity is animatable/modulatable like an effect parameter.
  const opacityDef = { name: 'opacity', label: 'Opacity', type: 'number', min: 0, max: 1, step: 0.01 };

  let effectDef = $derived(app.meta?.effects.find((e) => e.name === layer.effect));
  let numericParams = $derived([opacityDef, ...(effectDef?.params.filter((p) => p.type === 'number') ?? [])]);
  let open = $state(true);
  const groupName = (id) => (id ? (app.show.groups.find((g) => g.id === id)?.name ?? '?') : 'All pixels');
  let keyedCount = $derived(app.show.timeline.tracks.filter((t) => t.layerId === layer.id && t.keys.length).length);
  let showMods = $state(false);

  const keyBeat = () => snapBeat(live.timeline.pos, ui.snap);
  const track = (name) => {
    const t = findTrack(app.show, layer.id, name);
    return t && t.keys.length ? t : null;
  };

  // With keyframes, show the animated value at the playhead.
  function getValue(name) {
    const t = track(name);
    if (t) return evalTrack(t, live.timeline.pos);
    return name === 'opacity' ? layer.opacity : layer.params[name];
  }

  // Editing an animated parameter records a key at the playhead.
  function setValue(name, v) {
    if (track(name)) setKey(app.show, layer.id, name, keyBeat(), v);
    else if (name === 'opacity') layer.opacity = v;
    else layer.params[name] = v;
    save();
  }

  function keyState(name) {
    const t = track(name);
    if (!t) return 'none';
    return keyIndexAt(t, keyBeat()) >= 0 ? 'on' : 'track';
  }

  function toggleKey(name) {
    const b = keyBeat();
    if (keyState(name) === 'on') removeKeyAt(app.show, layer.id, name, b);
    else setKey(app.show, layer.id, name, b, getValue(name));
    save();
  }

  const modFor = (name) => layer.mods.find((m) => m.param === name)?.source ?? null;

  function changeEffect() {
    layer.params = defaultParams(layer.effect);
    save();
  }

  function addMod() {
    const p = numericParams.find((d) => !layer.mods.some((m) => m.param === d.name)) ?? opacityDef;
    layer.mods.push({ param: p.name, source: 'bass', min: p.min ?? 0, max: p.max ?? 1 });
    showMods = true;
    save();
  }

  // Reset a modulator's range to the parameter's full range.
  function modParamChanged(m) {
    const d = numericParams.find((p) => p.name === m.param);
    if (d) [m.min, m.max] = [d.min, d.max];
    save();
  }

  // Inline name field instead of prompt(), which some in-app browsers block.
  let presetName = $state(null); // null = field hidden

  async function saveAsPreset() {
    const name = presetName?.trim();
    if (!name) return;
    presetName = null;
    const tracks = app.show.timeline.tracks.filter((t) => t.layerId === layer.id);
    try {
      await library.put('presets', name, $state.snapshot({ layer, tracks }));
      onpreset?.();
    } catch (e) {
      app.error = e.message;
    }
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

  {#if !open}
    <p class="summary">
      {effectDef?.label ?? layer.effect} → {groupName(layer.groupId)}
      {#if layer.blend !== 'normal'}· {layer.blend}{/if}
      {#if layer.opacity < 1}· {Math.round(layer.opacity * 100)}%{/if}
      {#if layer.tempoSync}· ♩{/if}
      {#if keyedCount}<span class="tag key">◆ {keyedCount}</span>{/if}
      {#if layer.mods.length}<span class="tag mod">~ {layer.mods.length}</span>{/if}
    </p>
  {:else}
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
      <label class="inline" title="Feed the effect beats instead of seconds: speeds and rates become per-beat and follow the tempo">
        <input type="checkbox" bind:checked={layer.tempoSync} onchange={save} /> Tempo sync ♩
      </label>
    </div>

    {#if effectDef?.description}<p class="hint">{effectDef.description}</p>{/if}
    <ParamControl
      def={opacityDef}
      value={getValue('opacity')}
      onchange={(v) => setValue('opacity', v)}
      keyState={keyState('opacity')}
      onkey={() => toggleKey('opacity')}
      mod={modFor('opacity')}
    />
    {#each effectDef?.params ?? [] as p (p.name)}
      <ParamControl
        def={p}
        value={getValue(p.name)}
        onchange={(v) => setValue(p.name, v)}
        keyState={keyState(p.name)}
        onkey={() => toggleKey(p.name)}
        mod={modFor(p.name)}
      />
    {/each}

    <div class="row footer">
      <button class="small" onclick={() => (showMods = !showMods)}>
        {showMods ? '▾' : '▸'} Modulators ({layer.mods.length})
      </button>
      <span class="spacer"></span>
      {#if presetName === null}
        <button class="small" onclick={() => (presetName = layer.name)}>Save as preset</button>
      {:else}
        <input class="preset-name" bind:value={presetName} placeholder="Preset name" onkeydown={(e) => e.key === 'Enter' && saveAsPreset()} />
        <button class="small primary" onclick={saveAsPreset} disabled={!presetName.trim()}>Save</button>
        <button class="small" onclick={() => (presetName = null)}>Cancel</button>
      {/if}
    </div>

    {#if showMods}
      <div class="mods">
        <p class="hint">Drive a parameter from audio (level, bass, mid, treble, beat) or the tempo clock (pulse, phase). Value = min + (max − min) × signal.</p>
        {#each layer.mods as m, i (i)}
          <div class="row mod">
            <select bind:value={m.param} onchange={() => modParamChanged(m)}>
              {#each numericParams as p (p.name)}<option value={p.name}>{p.label}</option>{/each}
            </select>
            <span>←</span>
            <select bind:value={m.source} onchange={save}>
              {#each app.meta?.modSources ?? [] as s (s)}<option value={s}>{s}</option>{/each}
            </select>
            <label class="inline">min <input type="number" step="any" bind:value={m.min} onchange={save} /></label>
            <label class="inline">max <input type="number" step="any" bind:value={m.max} onchange={save} /></label>
            <button
              class="icon danger"
              onclick={() => {
                layer.mods.splice(i, 1);
                save();
              }}>✕</button
            >
          </div>
        {/each}
        <button class="small" onclick={addMod}>+ Add modulator</button>
      </div>
    {/if}
  {/if}
</div>

<style>
  .summary {
    margin: -0.2rem 0 0 2.75rem;
    font-size: 0.8rem;
    color: var(--muted);
  }
  .tag {
    margin-left: 0.4rem;
    font-size: 0.75rem;
  }
  .tag.key {
    color: var(--warn);
  }
  .tag.mod {
    color: var(--accent);
  }
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
  .footer {
    margin-top: 0.5rem;
  }
  .preset-name {
    min-height: 1.8rem;
    width: 10rem;
    font-size: 0.8rem;
  }
  .mods {
    margin-top: 0.4rem;
    padding: 0.5rem;
    border: 1px dashed var(--line);
    border-radius: 6px;
  }
  .mod {
    margin-bottom: 0.3rem;
  }
  .mod input[type='number'] {
    width: 4.5rem;
  }
</style>
