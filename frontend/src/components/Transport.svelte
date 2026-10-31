<script>
  // Header transport: tempo (BPM + tap), beat indicator, timeline play
  // state and a small audio meter.
  import { app, live, tap, setBpm, transport } from '../lib/state.svelte.js';

  let beatInBar = $derived(((Math.floor(live.beat) % 4) + 4) % 4);
  let flash = $derived(live.beat - Math.floor(live.beat) < 0.15);

  function onBpm(e) {
    const v = Number(e.currentTarget.value);
    if (v > 0) setBpm(v);
  }
</script>

<div class="transport">
  <label class="bpm" title="Tempo (beats per minute)">
    <input type="number" min="20" max="300" step="0.1" value={live.bpm.toFixed(1)} onchange={onBpm} />
    <span>BPM</span>
  </label>
  <button class="tap" class:flash onpointerdown={tap} title="Tap on the beat (2+ taps set the tempo; 1 tap re-syncs the downbeat)">TAP</button>
  <div class="beats" title="Beat in bar">
    {#each [0, 1, 2, 3] as b (b)}<span class:on={b === beatInBar}></span>{/each}
  </div>
  {#if app.show?.settings.audio.autoBpm && live.audio?.bpm}<span class="auto" title="Following detected tempo">auto</span>{/if}

  <span class="spacer"></span>

  <button class="icon" title="Stop (back to start)" onclick={() => transport('stop')}>■</button>
  <button
    class="icon"
    class:on={live.timeline.playing}
    title={live.timeline.playing ? 'Pause timeline' : 'Play timeline'}
    onclick={() => transport(live.timeline.playing ? 'pause' : 'play')}>{live.timeline.playing ? '❚❚' : '▶'}</button
  >
  <span class="pos mono">{live.timeline.pos.toFixed(1)}/{live.timeline.length}</span>

  {#if live.audio}
    <span class="meter" title="Audio level">
      <span style="width: {Math.round(live.audio.level * 100)}%"></span>
      <i class:hit={live.audio.beat > 0.5}></i>
    </span>
  {/if}
</div>

<style>
  .transport {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    margin: 0.3rem 0;
    flex-wrap: wrap;
  }
  .bpm {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    font-size: 0.75rem;
    color: var(--muted);
  }
  .bpm input {
    width: 4.6rem;
    text-align: right;
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
  .auto {
    font-size: 0.7rem;
    color: var(--ok);
  }
  .pos {
    font-size: 0.75rem;
    color: var(--muted);
    min-width: 4.2rem;
  }
  .meter {
    position: relative;
    width: 3.5rem;
    height: 0.6rem;
    background: var(--line);
    border-radius: 3px;
    overflow: hidden;
  }
  .meter span {
    position: absolute;
    inset: 0 auto 0 0;
    background: var(--ok);
  }
  .meter i {
    position: absolute;
    right: 0;
    top: 0;
    bottom: 0;
    width: 0.5rem;
  }
  .meter i.hit {
    background: var(--warn);
  }
</style>
