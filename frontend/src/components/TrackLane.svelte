<script>
  // A canvas lane for one keyframe track: draws the interpolated value
  // (a curve for numbers, a gradient for colours), the keys and the
  // playhead. Drag keys to move them; tap empty space to seek.
  import { evalTrack, snapBeat } from '../lib/keyframes.js';

  /** @type {{ track: any, length: number, pos: number, range?: [number, number], selected: number, snap: number, onselect: (i:number)=>void, onchange: ()=>void, onseek: (beat:number)=>void }} */
  let { track, length, pos, range = [0, 1], selected, snap, onselect, onchange, onseek } = $props();

  let canvas;
  let drag = null; // index of the key being dragged
  const H = 44;

  const beatAt = (ev) => {
    const r = canvas.getBoundingClientRect();
    return Math.max(0, Math.min(length, ((ev.clientX - r.left) / r.width) * length));
  };

  function down(ev) {
    const r = canvas.getBoundingClientRect();
    const x = ev.clientX - r.left;
    // Nearest key within 12px.
    let hit = -1;
    let best = 12;
    track.keys.forEach((k, i) => {
      const d = Math.abs((k.beat / length) * r.width - x);
      if (d < best) [hit, best] = [i, d];
    });
    if (hit >= 0) {
      onselect(hit);
      drag = hit;
      canvas.setPointerCapture(ev.pointerId);
    } else {
      onseek(snapBeat(beatAt(ev), snap));
    }
  }

  function move(ev) {
    if (drag === null) return;
    track.keys[drag].beat = snapBeat(beatAt(ev), snap);
  }

  function up() {
    if (drag === null) return;
    const key = track.keys[drag];
    drag = null;
    track.keys.sort((a, b) => a.beat - b.beat);
    onselect(track.keys.indexOf(key));
    onchange();
  }

  $effect(() => {
    // Reading these makes the effect redraw when they change.
    const keys = track.keys.map((k) => [k.beat, k.value, k.ease]);
    const p = pos;
    const sel = selected;
    if (!canvas || !keys) return;

    const dpr = window.devicePixelRatio || 1;
    const w = canvas.clientWidth;
    canvas.width = Math.round(w * dpr);
    canvas.height = Math.round(H * dpr);
    const ctx = canvas.getContext('2d');
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.fillStyle = '#0c0e13';
    ctx.fillRect(0, 0, w, H);

    // Beat grid, stronger every bar (4 beats).
    for (let b = 0; b <= length; b++) {
      ctx.fillStyle = b % 4 === 0 ? '#2c313c' : '#181b22';
      ctx.fillRect(Math.round((b / length) * w), 0, 1, H);
    }

    if (track.keys.length) {
      const isColor = typeof track.keys[0].value === 'string';
      if (isColor) {
        for (let x = 0; x < w; x += 2) {
          ctx.fillStyle = evalTrack(track, (x / w) * length);
          ctx.fillRect(x, H / 2 - 6, 2, 12);
        }
      } else {
        const [lo, hi] = range;
        ctx.strokeStyle = '#ffb547';
        ctx.lineWidth = 1.5;
        ctx.beginPath();
        for (let x = 0; x <= w; x += 2) {
          const v = evalTrack(track, (x / w) * length);
          const y = H - 6 - ((v - lo) / (hi - lo || 1)) * (H - 12);
          x === 0 ? ctx.moveTo(x, y) : ctx.lineTo(x, y);
        }
        ctx.stroke();
      }
    }

    track.keys.forEach((k, i) => {
      const x = (k.beat / length) * w;
      ctx.save();
      ctx.translate(x, H / 2);
      ctx.rotate(Math.PI / 4);
      ctx.fillStyle = i === sel ? '#ffffff' : '#ffb547';
      ctx.strokeStyle = '#000';
      ctx.fillRect(-5, -5, 10, 10);
      ctx.strokeRect(-5, -5, 10, 10);
      ctx.restore();
    });

    ctx.fillStyle = '#4f8cff';
    ctx.fillRect(Math.round((p / length) * w), 0, 2, H);
  });
</script>

<canvas
  bind:this={canvas}
  style="height: {H}px"
  onpointerdown={down}
  onpointermove={move}
  onpointerup={up}
  onpointercancel={up}
></canvas>

<style>
  canvas {
    display: block;
    width: 100%;
    border-radius: 4px;
    touch-action: none;
    cursor: pointer;
  }
</style>
