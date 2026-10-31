<script>
  // Draws one strip's live pixels from the preview stream. In selectable
  // mode (Groups tab) it also shows highlighted ranges, lets the user drag
  // to select a range, and can zoom in for precise picking:
  //   - −/+ buttons, two-finger pinch, or ctrl/⌘ + scroll (trackpad pinch)
  //   - pan with two fingers, the overview bar underneath, or by dragging a
  //     selection towards either edge
  // Only the visible part is drawn, so deep zoom stays cheap.
  import { onMount } from 'svelte';
  import { stripPixels } from '../lib/preview.js';

  /** @type {{ index: number, pixels: number, highlights?: {start:number,end:number}[], selectable?: boolean, onselect?: (start:number,end:number)=>void }} */
  let { index, pixels, highlights = [], selectable = false, onselect } = $props();

  let scroller;
  let canvas;
  let overview;
  let viewW = $state(300); // visible width in CSS px
  let zoom = $state(1);
  let drag = $state(null); // {from, to} pixel indices while selecting

  const PX_H = 22; // pixel row height
  const RULER_H = 13; // pixel-number ruler height (selectable only)
  const HL_H = 6; // highlight bar height
  const OV_H = 10; // overview bar height
  const MAX_PX_W = 48; // max on-screen width of one pixel when zoomed

  let cssH = $derived((selectable ? RULER_H : 0) + PX_H + (highlights.length || selectable ? HL_H + 2 : 0));
  let maxZoom = $derived(Math.max(1, (MAX_PX_W * pixels) / Math.max(viewW, 1)));
  let totalW = $derived(viewW * zoom);

  // ---- Coordinates ----
  const scrollX = () => scroller?.scrollLeft ?? 0;
  function pixelAtClientX(clientX) {
    const r = canvas.getBoundingClientRect();
    const x = Math.min(Math.max(clientX - r.left, 0), r.width - 1) + scrollX();
    return Math.min(pixels - 1, Math.max(0, Math.floor((x / totalW) * pixels)));
  }

  // Zoom keeping the content under viewport x (CSS px) fixed.
  function setZoom(z, anchorX = viewW / 2) {
    z = Math.min(maxZoom, Math.max(1, z));
    const frac = (scrollX() + anchorX) / totalW;
    zoom = z;
    // Wait for the spacer to resize before scrolling.
    requestAnimationFrame(() => {
      if (scroller) scroller.scrollLeft = frac * viewW * z - anchorX;
    });
  }

  // ---- Pointer handling: one pointer selects, two pointers pinch/pan ----
  const pointers = new Map();
  let pinch = null; // {dist, zoom, midX, scroll}
  let lastX = 0; // last clientX while selecting (for edge auto-scroll)

  function mid() {
    const [a, b] = [...pointers.values()];
    return { x: (a.x + b.x) / 2, dist: Math.hypot(a.x - b.x, a.y - b.y) };
  }

  function down(ev) {
    if (!selectable || !pixels) return;
    canvas.setPointerCapture(ev.pointerId);
    pointers.set(ev.pointerId, { x: ev.clientX, y: ev.clientY });
    if (pointers.size === 2) {
      drag = null; // second finger: switch to pinch
      const m = mid();
      const r = canvas.getBoundingClientRect();
      pinch = { dist: m.dist, zoom, anchor: (scrollX() + m.x - r.left) / totalW, left: r.left };
    } else if (pointers.size === 1) {
      const p = pixelAtClientX(ev.clientX);
      lastX = ev.clientX;
      drag = { from: p, to: p };
    }
  }

  function move(ev) {
    if (!pointers.has(ev.pointerId)) return;
    pointers.set(ev.pointerId, { x: ev.clientX, y: ev.clientY });
    if (pinch && pointers.size === 2) {
      const m = mid();
      const z = Math.min(maxZoom, Math.max(1, pinch.zoom * (m.dist / Math.max(pinch.dist, 1))));
      zoom = z;
      // Keep the pinched content under the fingers' midpoint (pans too).
      const x = m.x - pinch.left;
      requestAnimationFrame(() => scroller && (scroller.scrollLeft = pinch ? pinch.anchor * viewW * z - x : scroller.scrollLeft));
    } else if (drag) {
      lastX = ev.clientX;
      drag.to = pixelAtClientX(ev.clientX);
    }
  }

  function up(ev) {
    pointers.delete(ev.pointerId);
    if (pointers.size < 2) pinch = null;
    if (!drag || pointers.size) return;
    const { from, to } = drag;
    drag = null;
    onselect?.(Math.min(from, to), Math.max(from, to));
  }

  function cancel(ev) {
    pointers.delete(ev.pointerId);
    pinch = null;
    drag = null;
  }

  // Overview bar: drag to pan.
  function panTo(ev) {
    const r = overview.getBoundingClientRect();
    const frac = Math.min(1, Math.max(0, (ev.clientX - r.left) / r.width));
    scroller.scrollLeft = frac * totalW - viewW / 2;
  }
  let panning = false;

  onMount(() => {
    const ro = new ResizeObserver(() => (viewW = scroller.clientWidth || 300));
    ro.observe(scroller);

    // ctrl/⌘ + wheel (and trackpad pinch, which browsers send as ctrl+wheel)
    // zooms; needs a non-passive listener to stop the page zooming.
    const wheel = (ev) => {
      if (!selectable || !(ev.ctrlKey || ev.metaKey)) return;
      ev.preventDefault();
      const r = canvas.getBoundingClientRect();
      setZoom(zoom * Math.exp(-ev.deltaY * 0.01), ev.clientX - r.left);
    };
    scroller.addEventListener('wheel', wheel, { passive: false });

    const ctx = canvas.getContext('2d');
    let raf;
    const draw = () => {
      raf = requestAnimationFrame(draw);
      const dpr = window.devicePixelRatio || 1;
      const w = viewW;
      const top = selectable ? RULER_H : 0;
      const h = cssH;
      if (canvas.width !== Math.round(w * dpr) || canvas.height !== Math.round(h * dpr)) {
        canvas.width = Math.round(w * dpr);
        canvas.height = Math.round(h * dpr);
      }
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      ctx.fillStyle = '#05060a';
      ctx.fillRect(0, 0, w, h);
      if (!pixels) return;

      // Edge auto-scroll while selecting.
      if (drag && zoom > 1) {
        const r = canvas.getBoundingClientRect();
        const edge = 32;
        const dx = lastX - r.left < edge ? -1 : r.right - lastX < edge ? 1 : 0;
        if (dx) {
          scroller.scrollLeft += dx * 8;
          drag.to = pixelAtClientX(lastX);
        }
      }

      const off = scrollX();
      const pw = (w * zoom) / pixels;
      const i0 = Math.max(0, Math.floor(off / pw));
      const i1 = Math.min(pixels - 1, Math.ceil((off + w) / pw));
      const gap = pw > 4 ? 1 : 0;
      const rgb = stripPixels(index);
      for (let i = i0; i <= i1; i++) {
        ctx.fillStyle = rgb && i * 3 + 2 < rgb.length ? `rgb(${rgb[i * 3]},${rgb[i * 3 + 1]},${rgb[i * 3 + 2]})` : '#1a1c22';
        ctx.fillRect(i * pw - off, top, Math.max(pw - gap, 0.5), PX_H);
      }

      // Pixel-number ruler: pick a label step that leaves room for text.
      if (selectable) {
        const step = [1, 2, 5, 10, 20, 25, 50, 100, 200, 500, 1000].find((s) => s * pw >= 30) ?? 1000;
        ctx.fillStyle = '#8a92a3';
        ctx.font = '10px system-ui, sans-serif';
        ctx.textAlign = 'center';
        for (let i = Math.ceil(i0 / step) * step; i <= i1; i += step) {
          const x = (i + 0.5) * pw - off;
          ctx.fillText(String(i), Math.min(Math.max(x, 8), w - 8), 10);
          ctx.fillRect(x, RULER_H - 2, 1, 2);
        }
      }

      const hy = top + PX_H + 2;
      ctx.fillStyle = '#4f8cff';
      for (const hl of highlights) {
        ctx.fillRect(hl.start * pw - off, hy, (hl.end - hl.start + 1) * pw, HL_H);
      }
      if (drag) {
        const a = Math.min(drag.from, drag.to);
        const b = Math.max(drag.from, drag.to);
        ctx.fillStyle = 'rgba(255,255,255,0.25)';
        ctx.fillRect(a * pw - off, top, (b - a + 1) * pw, PX_H);
        ctx.fillStyle = '#ffd34f';
        ctx.fillRect(a * pw - off, hy, (b - a + 1) * pw, HL_H);
      }

      // Overview: whole strip + the visible window.
      if (overview && zoom > 1) {
        if (overview.width !== Math.round(w * dpr)) {
          overview.width = Math.round(w * dpr);
          overview.height = Math.round(OV_H * dpr);
        }
        const octx = overview.getContext('2d');
        octx.setTransform(dpr, 0, 0, dpr, 0, 0);
        octx.fillStyle = '#0c0e13';
        octx.fillRect(0, 0, w, OV_H);
        const opw = w / pixels;
        if (rgb) {
          for (let i = 0; i < pixels && i * 3 + 2 < rgb.length; i += Math.max(1, Math.floor(pixels / w))) {
            octx.fillStyle = `rgb(${rgb[i * 3]},${rgb[i * 3 + 1]},${rgb[i * 3 + 2]})`;
            octx.fillRect(i * opw, 2, Math.max(opw, 1), OV_H - 4);
          }
        }
        octx.fillStyle = '#4f8cff';
        for (const hl of highlights) octx.fillRect(hl.start * opw, OV_H - 2, (hl.end - hl.start + 1) * opw, 2);
        octx.strokeStyle = '#ffffff';
        octx.lineWidth = 1.5;
        octx.strokeRect((off / (w * zoom)) * w + 0.75, 0.75, w / zoom - 1.5, OV_H - 1.5);
      }
    };
    draw();
    return () => {
      cancelAnimationFrame(raf);
      ro.disconnect();
      scroller.removeEventListener('wheel', wheel);
    };
  });
</script>

<div class="strip">
  {#if selectable}
    <div class="zoom">
      {#if drag}<span class="drag-label">{Math.min(drag.from, drag.to)} – {Math.max(drag.from, drag.to)}</span>{/if}
      <button class="small" title="Zoom out" disabled={zoom <= 1} onclick={() => setZoom(zoom / 2)}>−</button>
      <span class="z mono">{zoom < 10 ? zoom.toFixed(1) : Math.round(zoom)}×</span>
      <button class="small" title="Zoom in" disabled={zoom >= maxZoom} onclick={() => setZoom(zoom * 2)}>+</button>
      <button class="small" title="Fit whole strip" disabled={zoom === 1} onclick={() => setZoom(1)}>fit</button>
    </div>
  {/if}
  <div class="scroller" bind:this={scroller}>
    <div class="spacer" style="width: {totalW}px">
      <canvas
        bind:this={canvas}
        class:selectable
        style:width="{viewW}px"
        style:height="{cssH}px"
        onpointerdown={down}
        onpointermove={move}
        onpointerup={up}
        onpointercancel={cancel}
      ></canvas>
    </div>
  </div>
  {#if selectable && zoom > 1}
    <canvas
      class="overview"
      bind:this={overview}
      style="height: {OV_H}px"
      title="Drag to pan"
      onpointerdown={(e) => {
        panning = true;
        overview.setPointerCapture(e.pointerId);
        panTo(e);
      }}
      onpointermove={(e) => panning && panTo(e)}
      onpointerup={() => (panning = false)}
    ></canvas>
  {/if}
</div>

<style>
  .strip {
    position: relative;
  }
  .scroller {
    overflow-x: auto;
    overflow-y: hidden;
    scrollbar-width: thin;
    border-radius: 4px;
  }
  .spacer {
    min-width: 100%;
  }
  canvas {
    display: block;
    position: sticky; /* stays in view while the spacer scrolls */
    left: 0;
    border-radius: 4px;
  }
  canvas.selectable {
    cursor: crosshair;
    touch-action: none; /* drags select; two fingers pinch/pan */
  }
  .overview {
    position: static;
    width: 100%;
    margin-top: 3px;
    cursor: ew-resize;
    touch-action: none;
  }
  .zoom {
    display: flex;
    justify-content: flex-end;
    align-items: center;
    gap: 0.25rem;
    margin-bottom: 0.2rem;
  }
  .zoom button {
    min-width: 1.9rem;
  }
  .z {
    font-size: 0.75rem;
    color: var(--muted);
    min-width: 2.4rem;
    text-align: center;
  }
  .drag-label {
    margin-right: auto;
    font-size: 12px;
    color: #ffd34f;
  }
</style>
