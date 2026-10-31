// Live preview frames from the engine. Kept outside Svelte's reactivity:
// canvases pull the latest frame on each animation frame, which is far
// cheaper than re-rendering components 20 times a second.
//
// Frame format (little-endian), see engine.encodePreview:
//   u8 type=1 | u16 stripCount | u16 pixels[stripCount] | RGB bytes...

let strips = [];

export function setFrame(buf) {
  const view = new DataView(buf);
  if (view.getUint8(0) !== 1) return;
  const count = view.getUint16(1, true);
  let off = 3 + count * 2;
  const next = new Array(count);
  for (let i = 0; i < count; i++) {
    const n = view.getUint16(3 + i * 2, true);
    next[i] = new Uint8Array(buf, off, n * 3);
    off += n * 3;
  }
  strips = next;
}

/** RGB bytes for strip i in the latest frame, or undefined. */
export function stripPixels(i) {
  return strips[i];
}
