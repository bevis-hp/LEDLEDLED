<script>
  // Groups: named collections of strip sections that layers can target.
  // Drag across a strip to add a section to the selected group.
  import { app, save, uid, defaultParams } from '../lib/state.svelte.js';
  import StripCanvas from './StripCanvas.svelte';
  import { ask } from '../lib/confirm.svelte.js';

  let selectedId = $state('');
  let group = $derived(app.show.groups.find((g) => g.id === selectedId) ?? app.show.groups[0]);

  function stripName(id) {
    return app.show.strips.find((s) => s.id === id)?.name ?? '(missing strip)';
  }
  function stripPixels(id) {
    return app.show.strips.find((s) => s.id === id)?.pixels ?? 0;
  }

  function addGroup() {
    const g = { id: uid(), name: `Group ${app.show.groups.length + 1}`, sections: [] };
    app.show.groups.push(g);
    selectedId = g.id;
    save();
  }

  async function removeGroup() {
    if (!group || !(await ask(`Delete group "${group.name}"? Layers using it will target all pixels.`, { confirmLabel: 'Delete', danger: true }))) return;
    for (const l of app.show.layers) if (l.groupId === group.id) l.groupId = '';
    app.show.groups = app.show.groups.filter((g) => g.id !== group.id);
    selectedId = '';
    save();
  }

  function addSection(stripId, start, end) {
    if (!group) return;
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
    });
    save();
  }
</script>

<section>
  <div class="row">
    <select bind:value={selectedId} disabled={!app.show.groups.length}>
      {#each app.show.groups as g (g.id)}
        <option value={g.id}>{g.name}</option>
      {:else}
        <option value="">No groups</option>
      {/each}
    </select>
    <button class="primary" onclick={addGroup}>+ New group</button>
  </div>
</section>

{#if group}
  <section class="card">
    <div class="row">
      <input class="grow" bind:value={group.name} onchange={save} placeholder="Group name" />
      <button onclick={addColourLayer} title="Add a solid colour layer for this group">+ Colour layer</button>
      <button class="danger" onclick={removeGroup}>Delete</button>
    </div>

    <h3>Select pixels</h3>
    <p class="hint">Drag across a strip to add that range as a section. Blue bars show the group's current sections.</p>
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
        <button
          class="icon danger"
          onclick={() => {
            group.sections.splice(i, 1);
            save();
          }}>✕</button
        >
      </div>
    {:else}
      <p class="hint">No sections yet.</p>
    {/each}
    {#if group.sections.length}
      <p class="hint">
        {group.sections.reduce((n, s) => n + (s.end - s.start + 1), 0)} pixels across
        {new Set(group.sections.map((s) => stripName(s.stripId))).size} strip(s)
      </p>
    {/if}
  </section>
{/if}

<style>
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
