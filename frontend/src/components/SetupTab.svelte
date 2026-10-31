<script>
  // Setup: pixel controllers (AirPixels), strips, and global settings.
  import { app, save, uid, identify } from '../lib/state.svelte.js';
  import { ask } from '../lib/confirm.svelte.js';
  import AudioPanel from './AudioPanel.svelte';
  import LibraryPanel from './LibraryPanel.svelte';
  import SystemCard from './SystemCard.svelte';

  // Typical draw per pixel (one addressable unit). 12V WS2811 strips drive
  // 3 LEDs per pixel. Measure your own strip at full white to refine.
  const powerPresets = [
    { name: 'WS2812B 5V', volts: 5, ma: 20, idle: 1 },
    { name: 'SK6812 RGBW 5V', volts: 5, ma: 20, idle: 1 },
    { name: 'WS2811 12V (3 LEDs/px)', volts: 12, ma: 20, idle: 2 },
    { name: 'WS2815 12V', volts: 12, ma: 12, idle: 2 },
  ];
  function applyPowerPreset(s, ev) {
    const p = powerPresets[Number(ev.currentTarget.value)];
    ev.currentTarget.value = '';
    if (!p) return;
    Object.assign(s, { volts: p.volts, milliampsPerChannel: p.ma, idleMilliamps: p.idle });
    save();
  }
  // Expanded cards by id. Setup is done once for an installation, so cards
  // start collapsed to a summary line; newly added ones open.
  let expanded = $state({});
  const toggle = (id) => (expanded[id] = !expanded[id]);

  const powerFor = (id) => app.status?.strips?.find((p) => p.id === id);
  // Worst case: every channel at full.
  const maxDraw = (s) => (s.pixels * (s.colorOrder.length * s.milliampsPerChannel + s.idleMilliamps)) / 1000;

  const statusFor = (id) => app.status?.controllers?.find((c) => c.id === id);
  const controllerName = (id) => app.show.controllers.find((c) => c.id === id)?.name ?? 'no controller';
  // Health dot: grey = disabled, red = send errors, green = sending.
  function health(c, st) {
    if (c.disabled) return 'off';
    if (st?.lastError) return 'bad';
    return st?.packets ? 'ok' : 'off';
  }

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
    const id = uid();
    expanded[id] = true;
    app.show.controllers.push({
      id,
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

  async function removeController(i) {
    const c = app.show.controllers[i];
    if (!(await ask(`Delete controller "${c.name}"? Its strips will be unassigned.`, { confirmLabel: 'Delete', danger: true }))) return;
    for (const s of app.show.strips) if (s.controllerId === c.id) s.controllerId = '';
    app.show.controllers.splice(i, 1);
    save();
  }

  function addStrip() {
    const ctrl = app.show.controllers.find((c) => !app.show.strips.some((s) => s.controllerId === c.id));
    const id = uid();
    expanded[id] = true;
    app.show.strips.push({
      id,
      name: `Strip ${app.show.strips.length + 1}`,
      controllerId: ctrl?.id ?? app.show.controllers[0]?.id ?? '',
      pixels: 200,
      colorOrder: 'GRB',
      brightness: 1,
      reverse: false,
      whiteMode: 'auto',
      volts: 5,
      milliampsPerChannel: 20,
      idleMilliamps: 1,
      maxAmps: 0,
    });
    save();
  }

  async function removeStrip(i) {
    const s = app.show.strips[i];
    if (!(await ask(`Delete strip "${s.name}"? Group sections on it will be removed.`, { confirmLabel: 'Delete', danger: true }))) return;
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
  <h2>System</h2>
  <SystemCard />
</section>

<section>
  <h2>Controllers</h2>
  <p class="hint">
    One per AirPixel. Set the same IP, protocol, start universe and universe size on the AirPixel's own config page.
  </p>
  {#each app.show.controllers as c, i (c.id)}
    {@const st = statusFor(c.id)}
    <div class="card" class:muted={c.disabled}>
      <div class="row">
        <button class="icon" aria-expanded={!!expanded[c.id]} title={expanded[c.id] ? 'Collapse' : 'Edit'} onclick={() => toggle(c.id)}>{expanded[c.id] ? '▾' : '▸'}</button>
        <span class="health {health(c, st)}" title={st?.lastError || (c.disabled ? 'Disabled' : 'Sending')}></span>
        <input class="grow name" bind:value={c.name} onchange={save} aria-label="Controller name" />
        <label class="inline"><input type="checkbox" checked={!c.disabled} onchange={(e) => { c.disabled = !e.currentTarget.checked; save(); }} /> on</label>
        <button class="icon danger" title="Delete controller" onclick={() => removeController(i)}>✕</button>
      </div>
      {#if !expanded[c.id]}
        <p class="summary">
          {c.ip} · {c.protocol === 'artnet' ? 'Art-Net' : 'sACN'}
          {c.startUniverse}{st?.universes > 1 ? `–${c.startUniverse + st.universes - 1}` : ''}
          {#if st}· {st.channels} ch{/if}
          {#if st?.errors}<span class="err"> · {st.errors} errors</span>{/if}
          {#if st?.warnings?.length}<span class="warn"> · ⚠ too many pixels</span>{/if}
        </p>
      {:else}
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
      {/if}
    </div>
  {/each}
  <button class="primary" onclick={addController}>+ Add controller</button>
</section>

<section>
  <h2>Strips</h2>
  <p class="hint">Strips on the same controller are chained in this order (first strip starts at channel 1).</p>
  {#each app.show.strips as s, i (s.id)}
    {@const pw = powerFor(s.id)}
    <div class="card">
      <div class="row">
        <button class="icon" aria-expanded={!!expanded[s.id]} title={expanded[s.id] ? 'Collapse' : 'Edit'} onclick={() => toggle(s.id)}>{expanded[s.id] ? '▾' : '▸'}</button>
        <input class="grow name" bind:value={s.name} onchange={save} aria-label="Strip name" />
        <button onclick={() => identify(s.id)} title="Flash a test pattern: first pixel green, last red, every 10th white">Identify</button>
        {#if expanded[s.id]}
          <button class="icon" title="Move up" disabled={i === 0} onclick={() => moveStrip(i, -1)}>↑</button>
          <button class="icon" title="Move down" disabled={i === app.show.strips.length - 1} onclick={() => moveStrip(i, 1)}>↓</button>
          <button class="icon danger" title="Delete strip" onclick={() => removeStrip(i)}>✕</button>
        {/if}
      </div>
      {#if !expanded[s.id]}
        <p class="summary">
          {controllerName(s.controllerId)} · {s.pixels} px · {s.colorOrder}{s.reverse ? ' · reversed' : ''}
          · {s.volts} V{#if pw}{' · '}<span class:warn={pw.limited}>{pw.amps.toFixed(2)} A{pw.limited ? ' (limited)' : ''}</span>{/if}
        </p>
      {:else}
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

      <details class="power">
        <summary>
          Power: {s.volts} V
          {#if s.maxAmps > 0}· limit {s.maxAmps} A{/if}
          {#if pw}
            · now <b class:warn={pw.limited}>{pw.amps.toFixed(2)} A / {pw.watts.toFixed(1)} W</b>
            {#if pw.limited}<span class="warn">(limited from {pw.requestedAmps.toFixed(2)} A)</span>{/if}
          {/if}
        </summary>
        <div class="grid3">
          <label>
            <span>Preset</span>
            <select onchange={(e) => applyPowerPreset(s, e)}>
              <option value="">Choose strip type…</option>
              {#each powerPresets as p, i (p.name)}<option value={i}>{p.name}</option>{/each}
            </select>
          </label>
          <label>
            <span>Supply voltage</span>
            <select bind:value={s.volts} onchange={save}>
              <option value={5}>5 V</option>
              <option value={12}>12 V</option>
              <option value={24}>24 V</option>
            </select>
          </label>
          <label><span>mA per channel (full, per pixel)</span><input type="number" min="0" step="0.5" bind:value={s.milliampsPerChannel} onchange={save} /></label>
          <label><span>Idle mA per pixel</span><input type="number" min="0" step="0.1" bind:value={s.idleMilliamps} onchange={save} /></label>
          <label><span>Max amps (0 = no limit)</span><input type="number" min="0" step="0.1" bind:value={s.maxAmps} onchange={save} /></label>
        </div>
        <p class="hint">
          Full-on worst case ≈ {maxDraw(s).toFixed(2)} A ({(maxDraw(s) * s.volts).toFixed(0)} W). Set the max to what your supply and
          power injection can carry; brightness is scaled down automatically to stay under it.
        </p>
      </details>
      {/if}
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
  <h2>Audio &amp; tempo</h2>
  <AudioPanel />
</section>

<section>
  <h2>Library</h2>
  <LibraryPanel />
  <div class="row">
    <button onclick={exportShow}>Download current show</button>
    <label class="button">Import show file (replaces current) <input type="file" accept="application/json" onchange={importShow} hidden /></label>
  </div>
</section>

<style>
  .name {
    font-weight: 600;
  }
  .card.muted {
    opacity: 0.6;
  }
  .summary {
    margin: 0.35rem 0 0 2.75rem;
    font-size: 0.8rem;
    color: var(--muted);
  }
  .health {
    width: 0.6rem;
    height: 0.6rem;
    border-radius: 50%;
    background: var(--line);
    flex: none;
  }
  .health.ok {
    background: var(--ok);
  }
  .health.bad {
    background: var(--danger);
  }
  .power {
    margin-top: 0.6rem;
    font-size: 0.85rem;
  }
  .power summary {
    cursor: pointer;
    color: var(--muted);
  }
  .warn {
    color: var(--warn);
  }
</style>
