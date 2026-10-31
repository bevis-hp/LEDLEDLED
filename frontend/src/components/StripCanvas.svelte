<script>
  // Draws one strip's live pixels from the preview stream. Optionally shows
  // highlighted ranges underneath and lets the user drag to select a range.
  import { onMount } from 'svelte';
  import { stripPixels } from '../lib/preview.js';

  /** @type {{ index: number, pixels: number, highlights?: {start:number,end:number}[], selectable?: boolean, onselect?: (start:number,end:number)=>void }} */
  let { index, pixels, highlights = [], selectable = false, onselect } = $props();

  let canvas;
  let drag = $state(null); // {from, to} pixel indices while dragging

  const PX_H = 22; // pixel row height (CSS px)
  const HL_H = 6; // highlight bar height

  function pixelAt(ev) {
    const r = canvas.getBoundingClientRect();
    const x = Math.min(Math.max(ev.clientX - r.left, 0), r.width - 1);
    return Math.floor((x / r.width) * pixels);
  }

  function down(ev) {
    if (!selectable || !pixels) return;
    canvas.setPointerCapture(ev.pointerId);
    const p = pixelAt(ev);
    drag = { from: p, to: p };
  }
  function move(ev) {
    if (drag) drag.to = pixelAt(ev);
  }
  function up() {
    if (!drag) return;
    const { from, to } = drag;
    drag = null;
    onselect?.(Math.min(from, to), Math.max(from, to));
  }

  onMount(() => {
    const ctx = canvas.getContext('2d');
    let raf;
    const draw = () => {
      raf = requestAnimationFrame(draw);
      const dpr = window.devicePixelRatio || 1;
      const w = canvas.clientWidth;
      const h = PX_H + (highlights.length || selectable ? HL_H + 2 : 0);
      if (canvas.width !== Math.round(w * dpr) || canvas.height !== Math.round(h * dpr)) {
        canvas.width = Math.round(w * dpr);
        canvas.height = Math.round(h * dpr);
        canvas.style.height = h + 'px';
      }
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      ctx.fillStyle = '#05060a';
      ctx.fillRect(0, 0, w, h);
      if (!pixels) return;

      const pw = w / pixels;
      const gap = pw > 4 ? 1 : 0;
      const rgb = stripPixels(index);
      for (let i = 0; i < pixels; i++) {
        if (rgb && i * 3 + 2 < rgb.length) {
          ctx.fillStyle = `rgb(${rgb[i * 3]},${rgb[i * 3 + 1]},${rgb[i * 3 + 2]})`;
        } else {
          ctx.fillStyle = '#1a1c22';
        }
        ctx.fillRect(i * pw, 0, Math.max(pw - gap, 0.5), PX_H);
      }

      ctx.fillStyle = '#4f8cff';
      for (const hl of highlights) {
        ctx.fillRect(hl.start * pw, PX_H + 2, (hl.end - hl.start + 1) * pw, HL_H);
      }
      if (drag) {
        const a = Math.min(drag.from, drag.to);
        const b = Math.max(drag.from, drag.to);
        ctx.fillStyle = 'rgba(255,255,255,0.25)';
        ctx.fillRect(a * pw, 0, (b - a + 1) * pw, h);
        ctx.fillStyle = '#ffd34f';
        ctx.fillRect(a * pw, PX_H + 2, (b - a + 1) * pw, HL_H);
      }
    };
    draw();
    return () => cancelAnimationFrame(raf);
  });
</script>

<div class="strip">
  <canvas
    bind:this={canvas}
    class:selectable
    onpointerdown={down}
    onpointermove={move}
    onpointerup={up}
    onpointercancel={() => (drag = null)}
  ></canvas>
  {#if drag}
    <span class="drag-label">{Math.min(drag.from, drag.to)} – {Math.max(drag.from, drag.to)}</span>
  {/if}
</div>

<style>
  .strip {
    position: relative;
  }
  canvas {
    display: block;
    width: 100%;
    border-radius: 4px;
  }
  canvas.selectable {
    cursor: crosshair;
    touch-action: none; /* let drags select instead of scrolling */
  }
  .drag-label {
    position: absolute;
    right: 4px;
    top: -18px;
    font-size: 12px;
    color: #ffd34f;
  }
</style>
