<script>
  // Palettes: built-in gradients (read-only) and the show's own, editable.
  // Tap the bar to add a stop, drag stops to move them, tap a stop to edit.
  import { app, save, uid } from '../lib/state.svelte.js';
  import { cssGradient, colorAt } from '../lib/palettes.js';

  // Selected stop, by palette id + index (the show object is replaced after
  // every save, so references would go stale).
  let sel = $state({ id: '', index: -1 });
  let selPalette = $derived(app.show.palettes.find((p) => p.id === sel.id));
  let selStop = $derived(selPalette?.stops[sel.index]);

  function newPalette(from) {
    const p = {
      id: uid(),
      name: from ? `${from.name} copy` : `Palette ${app.show.palettes.length + 1}`,
      stops: from ? from.stops.map((s) => ({ ...s })) : [
        { pos: 0, color: '#ff0040' },
        { pos: 1, color: '#2060ff' },
      ],
    };
    app.show.palettes.push(p);
    sel = { id: p.id, index: 0 };
    save();
  }

  function removePalette(p) {
    if (!confirm(`Delete palette "${p.name}"? Layers using it go back to their own colours.`)) return;
    app.show.palettes = app.show.palettes.filter((x) => x.id !== p.id);
    sel = { id: '', index: -1 };
    save();
  }

  // ---- Stop dragging on the bar ----
  let dragging = null; // {palette, stop}

  const posAt = (ev, bar) => {
    const r = bar.getBoundingClientRect();
    return Math.round(Math.max(0, Math.min(1, (ev.clientX - r.left) / r.width)) * 1000) / 1000;
  };

  function barDown(ev, p) {
    const bar = ev.currentTarget;
    const stopEl = ev.target.closest('[data-stop]');
    let stop;
    if (stopEl) {
      stop = p.stops[Number(stopEl.dataset.stop)];
    } else {
      // Add a stop with the colour already at that point.
      const pos = posAt(ev, bar);
      stop = { pos, color: colorAt(p.stops, pos) };
      p.stops.push(stop);
      stop = p.stops[p.stops.length - 1];
    }
    dragging = { p, stop, bar };
    sel = { id: p.id, index: p.stops.indexOf(stop) };
    bar.setPointerCapture(ev.pointerId);
  }

  function barMove(ev) {
    if (dragging) dragging.stop.pos = posAt(ev, dragging.bar);
  }

  function barUp() {
    if (!dragging) return;
    const { p, stop } = dragging;
    dragging = null;
    p.stops.sort((a, b) => a.pos - b.pos);
    sel = { id: p.id, index: p.stops.indexOf(stop) };
    save();
  }

  function removeStop() {
    if (selPalette.stops.length <= 1) return;
    selPalette.stops.splice(sel.index, 1);
    sel = { id: sel.id, index: -1 };
    save();
  }
</script>

<section>
  <div class="row">
    <h2 class="grow">Your palettes</h2>
    <button class="primary" onclick={() => newPalette()}>+ New palette</button>
  </div>
  <p class="hint">
    Use a palette from any effect's <b>Palette</b> setting (Gradient, Noise, Sparkle, VU, Spectrum, Palette scroll). Tap the bar to add a
    colour stop, drag stops to move them, tap one to change its colour.
  </p>

  {#each app.show.palettes as p (p.id)}
    <div class="card">
      <div class="row">
        <input class="grow name" bind:value={p.name} onchange={save} />
        <button class="small" onclick={() => newPalette(p)}>Duplicate</button>
        <button class="icon danger" title="Delete palette" onclick={() => removePalette(p)}>✕</button>
      </div>
      <div
        class="bar editable"
        style="background: {cssGradient(p.stops)}"
        role="slider"
        tabindex="0"
        aria-label="Palette stops"
        aria-valuenow={sel.id === p.id ? sel.index : -1}
        onpointerdown={(e) => barDown(e, p)}
        onpointermove={barMove}
        onpointerup={barUp}
        onpointercancel={barUp}
      >
        {#each p.stops as s, i (i)}
          <span class="stop" class:selected={sel.id === p.id && sel.index === i} data-stop={i} style="left: {s.pos * 100}%; background: {s.color}"></span>
        {/each}
      </div>
      {#if sel.id === p.id && selStop}
        <div class="row stopedit">
          <input type="color" bind:value={selStop.color} oninput={save} />
          <label class="inline">at <input type="number" min="0" max="1" step="0.01" bind:value={selStop.pos} onchange={() => { const s = selStop; selPalette.stops.sort((a, b) => a.pos - b.pos); sel.index = selPalette.stops.indexOf(s); save(); }} /></label>
          <span class="mono hint">{selStop.color}</span>
          <span class="spacer"></span>
          <button class="small danger" disabled={p.stops.length <= 1} onclick={removeStop}>Remove stop</button>
        </div>
      {/if}
    </div>
  {:else}
    <p class="hint">No palettes of your own yet. Start a new one or duplicate a built-in below.</p>
  {/each}
</section>

<section>
  <h2>Built-in</h2>
  <div class="builtins">
    {#each app.meta?.builtinPalettes ?? [] as p (p.id)}
      <div class="builtin">
        <div class="bar" style="background: {cssGradient(p.stops)}"></div>
        <div class="row small">
          <span class="grow">{p.name}</span>
          <button class="small" onclick={() => newPalette(p)}>Duplicate</button>
        </div>
      </div>
    {/each}
  </div>
</section>

<style>
  .name {
    font-weight: 600;
  }
  .bar {
    position: relative;
    height: 2.2rem;
    border-radius: 6px;
    margin-top: 0.5rem;
    border: 1px solid var(--line);
  }
  .bar.editable {
    height: 2.8rem;
    cursor: copy;
    touch-action: none;
  }
  .stop {
    position: absolute;
    top: 50%;
    width: 1.3rem;
    height: 1.3rem;
    margin: -0.65rem 0 0 -0.65rem;
    border-radius: 50%;
    border: 2px solid #fff;
    box-shadow: 0 0 0 1px #000;
    cursor: grab;
  }
  .stop.selected {
    width: 1.7rem;
    height: 1.7rem;
    margin: -0.85rem 0 0 -0.85rem;
    border-color: var(--accent);
  }
  .stopedit {
    margin-top: 0.5rem;
  }
  .stopedit input[type='number'] {
    width: 4.5rem;
  }
  .stopedit input[type='color'] {
    width: 3rem;
    height: 2.2rem;
  }
  .builtins {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(12rem, 1fr));
    gap: 0.75rem;
  }
  .builtin .bar {
    margin: 0 0 0.3rem;
  }
</style>
