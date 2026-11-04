<script>
  // One group on the Groups tab: drag across a strip to add a section.
  import { app, save, uid, defaultParams } from '../lib/state.svelte.js';
  import StripCanvas from './StripCanvas.svelte';

  /** @type {{ group: any, onremove: () => void }} */
  let { group, onremove } = $props();

  let open = $state(true);

  const stripName = (id) => app.show.strips.find((s) => s.id === id)?.name ?? '(missing strip)';
  const stripPixels = (id) => app.show.strips.find((s) => s.id === id)?.pixels ?? 0;
  let pixelCount = $derived(group.sections.reduce((n, s) => n + (s.end - s.start + 1), 0));
  let stripCount = $derived(new Set(group.sections.map((s) => s.stripId)).size);

  function addSection(stripId, start, end) {
    group.sections.push({ stripId, start, end, reverse: false });
    save();
  }

  function moveSection(i, dir) {
    const s = group.sections;
    const j = i + dir;
    if (j < 0 || j >= s.length) return;
    [s[i], s[j]] = [s[j], s[i]];
    save();
  }

  function removeSection(i) {
    group.sections.splice(i, 1);
    save();
  }

  // Shortcut for "colour this group": add a solid colour layer targeting it.
  function addColourLayer() {
    app.show.layers.push({
      id: uid(),
      name: `${group.name} colour`,
      groupId: group.id,
      effect: 'solid',
      params: defaultParams('solid'),
      opacity: 1,
      blend: 'normal',
      muted: false,
      tempoSync: false,
      mods: [],
    });
    save();
  }
</script>

<div class="card group">
  <div class="row head">
    <button class="icon" title={open ? 'Collapse' : 'Expand'} onclick={() => (open = !open)}>{open ? '▾' : '▸'}</button>
    <input class="name" bind:value={group.name} onchange={save} placeholder="Group name" />
    <button class="small" onclick={addColourLayer} title="Add a solid colour layer for this group">+ Colour layer</button>
    <button class="icon danger" title="Delete group" onclick={onremove}>✕</button>
  </div>

  {#if !open}
    <p class="summary">
      {#if group.sections.length}
        {pixelCount} pixels across {stripCount} strip(s) · {group.sections.length} section(s)
      {:else}
        No sections yet
      {/if}
    </p>
  {:else}
    <h3>Select pixels</h3>
    {#each app.show.strips as s, i (s.id)}
      <div class="strip-row">
        <div class="row small">
          <span class="grow">{s.name} <small>{s.pixels}px</small></span>
          <button class="small" onclick={() => addSection(s.id, 0, s.pixels - 1)}>+ whole strip</button>
        </div>
        <StripCanvas
          index={i}
          pixels={s.pixels}
          selectable
          highlights={group.sections.filter((sec) => sec.stripId === s.id)}
          onselect={(a, b) => addSection(s.id, a, b)}
        />
      </div>
    {:else}
      <p class="hint">No strips yet. Add controllers and strips in Setup.</p>
    {/each}

    <h3>Sections <small>(effects run through these in order)</small></h3>
    {#each group.sections as sec, i (i)}
      <div class="row section">
        <select bind:value={sec.stripId} onchange={save}>
          {#each app.show.strips as s (s.id)}
            <option value={s.id}>{s.name}</option>
          {/each}
        </select>
        <label class="inline">from <input type="number" min="0" bind:value={sec.start} onchange={save} /></label>
        <label class="inline">to <input type="number" min="0" max={stripPixels(sec.stripId) - 1} bind:value={sec.end} onchange={save} /></label>
        <label class="inline"><input type="checkbox" bind:checked={sec.reverse} onchange={save} /> reverse</label>
        <span class="spacer"></span>
        <button class="icon" disabled={i === 0} onclick={() => moveSection(i, -1)}>↑</button>
        <button class="icon" disabled={i === group.sections.length - 1} onclick={() => moveSection(i, 1)}>↓</button>
        <button class="icon danger" onclick={() => removeSection(i)}>✕</button>
      </div>
    {:else}
      <p class="hint">No sections yet.</p>
    {/each}
    {#if group.sections.length}
      <p class="hint">
        {pixelCount} pixels across
        {new Set(group.sections.map((s) => stripName(s.stripId))).size} strip(s)
      </p>
    {/if}
  {/if}
</div>

<style>
  .head {
    margin-bottom: 0.5rem;
  }
  .name {
    flex: 1;
    min-width: 0;
    font-weight: 600;
  }
  .summary {
    margin: -0.2rem 0 0 2.75rem;
    font-size: 0.8rem;
    color: var(--muted);
  }
  .strip-row {
    margin-bottom: 0.7rem;
  }
  .section input[type='number'] {
    width: 4.5rem;
  }
  small {
    color: var(--muted);
    font-weight: normal;
  }
</style>
