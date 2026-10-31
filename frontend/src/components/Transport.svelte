<script>
  // Header transport: tempo (BPM + tap), beat indicator, timeline play
  // state. The audio level is drawn by App as a line under the header.
  import { app, live, tap, setBpm, transport } from '../lib/state.svelte.js';

  let beatInBar = $derived(((Math.floor(live.beat) % 4) + 4) % 4);
  let flash = $derived(live.beat - Math.floor(live.beat) < 0.15);

  function onBpm(e) {
    const v = Number(e.currentTarget.value);
    if (v > 0) setBpm(v);
  }
</script>

<div class="transport">
  <label class="bpm" class:auto={app.show?.settings.audio.autoBpm && live.audio?.bpm} title="Tempo in beats per minute (green outline = following the audio)">
    <input type="number" min="20" max="300" step="0.1" value={live.bpm.toFixed(1)} onchange={onBpm} aria-label="BPM" />
    <span>bpm</span>
  </label>
  <button class="tap" class:flash onpointerdown={tap} title="Tap on the beat (2+ taps set the tempo; 1 tap re-syncs the downbeat)">TAP</button>
  <div class="beats" title="Beat in bar">
    {#each [0, 1, 2, 3] as b (b)}<span class:on={b === beatInBar}></span>{/each}
  </div>

  <span class="spacer"></span>

  <button class="icon" title="Stop (back to start)" onclick={() => transport('stop')}>■</button>
  <button
    class="icon"
    class:on={live.timeline.playing}
    title={live.timeline.playing ? 'Pause timeline' : 'Play timeline'}
    onclick={() => transport(live.timeline.playing ? 'pause' : 'play')}>{live.timeline.playing ? '❚❚' : '▶'}</button
  >
  <span class="pos mono">{live.timeline.pos.toFixed(1)}/{live.timeline.length}</span>


</div>

<style>
  .transport {
    display: flex;
    align-items: center;
    gap: 0.35rem;
    margin: 0.3rem 0 0.1rem;
  }
  .bpm {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    font-size: 0.75rem;
    color: var(--muted);
  }
  .bpm input {
    width: 4.4rem;
    padding-inline: 0.4rem;
    text-align: right;
    font-variant-numeric: tabular-nums;
    appearance: textfield; /* no spinner arrows eating the width */
    -moz-appearance: textfield;
  }
  .bpm input::-webkit-inner-spin-button,
  .bpm input::-webkit-outer-spin-button {
    -webkit-appearance: none;
    margin: 0;
  }
  .bpm.auto input {
    border-color: var(--ok);
  }
  .tap {
    font-weight: 700;
    min-width: 4rem;
    letter-spacing: 0.08em;
    touch-action: manipulation; /* no double-tap zoom delay */
  }
  .tap.flash {
    border-color: var(--accent);
    color: var(--accent);
  }
  .beats {
    display: flex;
    gap: 3px;
  }
  .beats span {
    width: 0.55rem;
    height: 0.55rem;
    border-radius: 50%;
    background: var(--line);
  }
  .beats span.on {
    background: var(--accent);
  }
  .pos {
    font-size: 0.7rem;
    color: var(--muted);
    white-space: nowrap;
  }
</style>
