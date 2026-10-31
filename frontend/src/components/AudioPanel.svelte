<script>
  // Audio input settings, live meters and master modulation.
  import { onMount } from 'svelte';
  import { app, live, save, audioDevices } from '../lib/state.svelte.js';

  let devices = $state([]);
  const refresh = async () => (devices = await audioDevices().catch(() => ['sim', 'default']));
  onMount(refresh);

  let a = $derived(app.show.settings.audio);
  let st = $derived(app.status?.audio);
  let opts = $derived(devices.includes(a.device) ? devices : [a.device, ...devices]);
  const meters = ['level', 'bass', 'mid', 'treble', 'beat'];
</script>

<div class="card">
  <div class="row">
    <label class="inline"><input type="checkbox" bind:checked={a.enabled} onchange={save} /> Audio input enabled</label>
    <span class="spacer"></span>
    {#if a.enabled}
      {#if st?.running}<span class="ok">● capturing</span>
      {:else if st?.error}<span class="err">● {st.error}</span>
      {:else}<span class="hint">starting…</span>{/if}
    {/if}
  </div>
  <div class="grid3">
    <label>
      <span>Input device</span>
      <span class="row">
        <select class="grow" bind:value={a.device} onchange={save}>
          {#each opts as d (d)}<option value={d}>{d === 'sim' ? 'sim (built-in test beat)' : d}</option>{/each}
        </select>
        <button class="icon" title="Rescan devices" onclick={refresh}>↻</button>
      </span>
    </label>
    <label>
      <span>Sensitivity ×{a.gain.toFixed(1)}</span>
      <input type="range" min="0.2" max="5" step="0.1" bind:value={a.gain} oninput={save} />
    </label>
    <label class="inline" title="Lock the tempo to beats detected in the audio">
      <input type="checkbox" bind:checked={a.autoBpm} onchange={save} /> Auto BPM from audio
      {#if live.audio?.bpm}<small>(hearing ~{live.audio.bpm.toFixed(0)})</small>{/if}
    </label>
  </div>

  {#if live.audio}
    <div class="meters">
      {#each meters as m (m)}
        <div class="meter">
          <span class="name">{m}</span>
          <span class="bar"><span style="width: {Math.round(live.audio[m] * 100)}%"></span></span>
        </div>
      {/each}
    </div>
    <div class="bands" title="Spectrum (bass → treble)">
      {#each live.audio.bands as b, i (i)}<span style="height: {Math.round(b * 100)}%"></span>{/each}
    </div>
  {/if}
</div>

<div class="card">
  <h3>Master modulation</h3>
  <p class="hint">Make the master brightness follow a signal: e.g. <b>beat</b> or <b>pulse</b> to flash on the beat, <b>level</b> to breathe with the music.</p>
  <div class="grid3">
    <label>
      <span>Signal</span>
      <select bind:value={app.show.settings.masterMod.source} onchange={save}>
        <option value="">off</option>
        {#each app.meta?.modSources ?? [] as s (s)}<option value={s}>{s}</option>{/each}
      </select>
    </label>
    <label>
      <span>Amount {Math.round(app.show.settings.masterMod.amount * 100)}%</span>
      <input type="range" min="0" max="1" step="0.01" bind:value={app.show.settings.masterMod.amount} oninput={save} />
    </label>
  </div>
</div>

<style>
  .ok {
    color: var(--ok);
    font-size: 0.8rem;
  }
  .err {
    font-size: 0.8rem;
  }
  small {
    color: var(--muted);
  }
  .meters {
    margin-top: 0.6rem;
  }
  .meter {
    display: grid;
    grid-template-columns: 3.5rem 1fr;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.75rem;
    color: var(--muted);
    margin: 2px 0;
  }
  .bar {
    height: 0.5rem;
    background: var(--line);
    border-radius: 3px;
    overflow: hidden;
  }
  .bar span {
    display: block;
    height: 100%;
    background: var(--ok);
  }
  .bands {
    display: flex;
    align-items: flex-end;
    gap: 2px;
    height: 3rem;
    margin-top: 0.4rem;
    background: #0c0e13;
    padding: 2px;
    border-radius: 4px;
  }
  .bands span {
    flex: 1;
    background: linear-gradient(to top, #ff0040, #2060ff);
    min-height: 1px;
  }
</style>
