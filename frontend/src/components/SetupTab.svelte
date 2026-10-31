<script>
  // Setup: pixel controllers (AirPixels), strips, and global settings.
  import { app, save, uid, identify } from '../lib/state.svelte.js';

  const statusFor = (id) => app.status?.controllers?.find((c) => c.id === id);

  function addController() {
    const n = app.show.controllers.length;
    const last = app.show.controllers[n - 1];
    // Guess the next IP / universe block from the last controller.
    let ip = '10.43.0.11';
    if (last) {
      const parts = last.ip.split('.');
      parts[3] = String(Math.min(254, Number(parts[3]) + 1));
      ip = parts.join('.');
    }
    app.show.controllers.push({
      id: uid(),
      name: `AirPixel ${n + 1}`,
      ip,
      protocol: last?.protocol ?? 'sacn',
      startUniverse: last ? last.startUniverse + (last.maxUniverses || 4) : 1,
      universeSize: 0,
      maxUniverses: 4,
      disabled: false,
    });
    save();
  }

  function removeController(i) {
    const c = app.show.controllers[i];
    if (!confirm(`Delete controller "${c.name}"? Its strips will be unassigned.`)) return;
    for (const s of app.show.strips) if (s.controllerId === c.id) s.controllerId = '';
    app.show.controllers.splice(i, 1);
    save();
  }

  function addStrip() {
    const ctrl = app.show.controllers.find((c) => !app.show.strips.some((s) => s.controllerId === c.id));
    app.show.strips.push({
      id: uid(),
      name: `Strip ${app.show.strips.length + 1}`,
      controllerId: ctrl?.id ?? app.show.controllers[0]?.id ?? '',
      pixels: 200,
      colorOrder: 'GRB',
      brightness: 1,
      reverse: false,
      whiteMode: 'auto',
    });
    save();
  }

  function removeStrip(i) {
    const s = app.show.strips[i];
    if (!confirm(`Delete strip "${s.name}"? Group sections on it will be removed.`)) return;
    for (const g of app.show.groups) g.sections = g.sections.filter((sec) => sec.stripId !== s.id);
    app.show.strips.splice(i, 1);
    save();
  }

  function moveStrip(i, dir) {
    const s = app.show.strips;
    const j = i + dir;
    if (j < 0 || j >= s.length) return;
    [s[i], s[j]] = [s[j], s[i]];
    save();
  }

  // Art-Net port-address -> "net.subnet.universe" for display.
  const artnetLabel = (pa) => `Net ${(pa >> 8) & 0x7f} · Sub ${(pa >> 4) & 0xf} · Uni ${pa & 0xf}`;

  function exportShow() {
    const blob = new Blob([JSON.stringify($state.snapshot(app.show), null, 2)], { type: 'application/json' });
    const a = document.createElement('a');
    a.href = URL.createObjectURL(blob);
    a.download = 'ledledled-show.json';
    a.click();
    URL.revokeObjectURL(a.href);
  }

  async function importShow(ev) {
    const file = ev.currentTarget.files?.[0];
    if (!file) return;
    try {
      app.show = JSON.parse(await file.text());
      save();
    } catch (e) {
      app.error = 'Import failed: ' + e.message;
    }
    ev.currentTarget.value = '';
  }
</script>

<section>
  <h2>Controllers</h2>
  <p class="hint">
    One per AirPixel. Set the same IP, protocol, start universe and universe size on the AirPixel's own config page.
  </p>
  {#each app.show.controllers as c, i (c.id)}
    {@const st = statusFor(c.id)}
    <div class="card" class:muted={c.disabled}>
      <div class="row">
        <input class="grow name" bind:value={c.name} onchange={save} />
        <label class="inline"><input type="checkbox" checked={!c.disabled} onchange={(e) => { c.disabled = !e.currentTarget.checked; save(); }} /> enabled</label>
        <button class="icon danger" title="Delete controller" onclick={() => removeController(i)}>✕</button>
      </div>
      <div class="grid3">
        <label><span>IP address</span><input bind:value={c.ip} onchange={save} inputmode="decimal" /></label>
        <label>
          <span>Protocol</span>
          <select bind:value={c.protocol} onchange={() => { if (c.protocol === 'sacn' && c.startUniverse < 1) c.startUniverse = 1; save(); }}>
            <option value="sacn">sACN (E1.31)</option>
            <option value="artnet">Art-Net</option>
          </select>
        </label>
        <label>
          <span>{c.protocol === 'artnet' ? 'Start port-address' : 'Start universe'}</span>
          <input type="number" min={c.protocol === 'artnet' ? 0 : 1} bind:value={c.startUniverse} onchange={save} />
        </label>
        <label>
          <span>Universe size (0 = auto)</span>
          <input type="number" min="0" max="512" bind:value={c.universeSize} onchange={save} />
        </label>
        <label>
          <span>Max universes</span>
          <input type="number" min="1" max="64" bind:value={c.maxUniverses} onchange={save} />
        </label>
        <label>
          <span>UDP port (0 = default)</span>
          <input type="number" min="0" max="65535" bind:value={c.port} onchange={save} />
        </label>
      </div>
      {#if c.protocol === 'artnet'}
        <p class="hint">Starts at {artnetLabel(c.startUniverse)}</p>
      {/if}
      {#if st}
        <p class="hint">
          Sending {st.universes} universe{st.universes === 1 ? '' : 's'}
          ({c.protocol === 'artnet' ? 'port-address' : 'universe'} {c.startUniverse}–{c.startUniverse + Math.max(st.universes - 1, 0)})
          × {st.universeSize} ch · {st.channels} channels used · {st.packets} packets
          {#if st.errors}<span class="err"> · {st.errors} errors</span>{/if}
        </p>
        {#if st.lastError}<p class="err">{st.lastError}</p>{/if}
        {#each st.warnings ?? [] as w}<p class="warn">⚠ {w}</p>{/each}
      {/if}
    </div>
  {/each}
  <button class="primary" onclick={addController}>+ Add controller</button>
</section>

<section>
  <h2>Strips</h2>
  <p class="hint">Strips on the same controller are chained in this order (first strip starts at channel 1).</p>
  {#each app.show.strips as s, i (s.id)}
    <div class="card">
      <div class="row">
        <input class="grow name" bind:value={s.name} onchange={save} />
        <button onclick={() => identify(s.id)} title="Flash a test pattern: first pixel green, last red, every 10th white">Identify</button>
        <button class="icon" disabled={i === 0} onclick={() => moveStrip(i, -1)}>↑</button>
        <button class="icon" disabled={i === app.show.strips.length - 1} onclick={() => moveStrip(i, 1)}>↓</button>
        <button class="icon danger" title="Delete strip" onclick={() => removeStrip(i)}>✕</button>
      </div>
      <div class="grid3">
        <label>
          <span>Controller</span>
          <select bind:value={s.controllerId} onchange={save}>
            <option value="">(none)</option>
            {#each app.show.controllers as c (c.id)}
              <option value={c.id}>{c.name}</option>
            {/each}
          </select>
        </label>
        <label><span>Pixels</span><input type="number" min="0" max="2048" bind:value={s.pixels} onchange={save} /></label>
        <label>
          <span>Colour order</span>
          <select bind:value={s.colorOrder} onchange={save}>
            {#each app.meta?.colorOrders ?? [] as o (o)}
              <option value={o}>{o}{o === 'GRB' ? ' (WS2812)' : o === 'GRBW' ? ' (SK6812 RGBW)' : ''}</option>
            {/each}
          </select>
        </label>
        <label>
          <span>Brightness {Math.round(s.brightness * 100)}%</span>
          <input type="range" min="0" max="1" step="0.01" bind:value={s.brightness} oninput={save} />
        </label>
        <label class="inline"><input type="checkbox" bind:checked={s.reverse} onchange={save} /> reversed (data enters at far end)</label>
        {#if s.colorOrder.includes('W')}
          <label>
            <span>White channel</span>
            <select bind:value={s.whiteMode} onchange={save}>
              <option value="auto">auto (extract from RGB)</option>
              <option value="off">off</option>
            </select>
          </label>
        {/if}
      </div>
    </div>
  {/each}
  <button class="primary" onclick={addStrip}>+ Add strip</button>
</section>

<section>
  <h2>Output settings</h2>
  <div class="card grid3">
    <label><span>Frame rate (fps)</span><input type="number" min="1" max="120" bind:value={app.show.settings.fps} onchange={save} /></label>
    <label><span>Gamma (1 = linear)</span><input type="number" min="1" max="4" step="0.1" bind:value={app.show.settings.gamma} onchange={save} /></label>
    <label><span>sACN priority</span><input type="number" min="1" max="200" bind:value={app.show.settings.priority} onchange={save} /></label>
    <label><span>sACN source name</span><input bind:value={app.show.settings.sourceName} onchange={save} /></label>
    <label>
      <span>Send from IP (blank = auto)</span>
      <input bind:value={app.show.settings.bindIp} onchange={save} placeholder="e.g. 10.43.0.1" />
    </label>
  </div>
</section>

<section>
  <h2>Backup</h2>
  <div class="row">
    <button onclick={exportShow}>Export show</button>
    <label class="button">Import show <input type="file" accept="application/json" onchange={importShow} hidden /></label>
  </div>
</section>

<style>
  .name {
    font-weight: 600;
  }
  .card.muted {
    opacity: 0.6;
  }
</style>
