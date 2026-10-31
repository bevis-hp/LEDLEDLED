// Palette helpers shared by the palette editor and parameter controls.
import { app } from './state.svelte.js';

/** Built-in palettes followed by the show's own. */
export function allPalettes() {
  return [...(app.meta?.builtinPalettes ?? []), ...(app.show?.palettes ?? [])];
}

export function isBuiltin(id) {
  return (app.meta?.builtinPalettes ?? []).some((p) => p.id === id);
}

export function findPalette(id) {
  return allPalettes().find((p) => p.id === id);
}

/** CSS linear-gradient for a list of stops. */
export function cssGradient(stops) {
  if (!stops?.length) return '#000';
  if (stops.length === 1) return stops[0].color;
  const sorted = [...stops].sort((a, b) => a.pos - b.pos);
  return `linear-gradient(to right, ${sorted.map((s) => `${s.color} ${(s.pos * 100).toFixed(1)}%`).join(', ')})`;
}

const toRgb = (h) => {
  const v = parseInt(h.slice(1), 16);
  return [(v >> 16) & 255, (v >> 8) & 255, v & 255];
};
const toHex = (c) => '#' + c.map((x) => Math.round(x).toString(16).padStart(2, '0')).join('');

/** Colour of a palette at u (0..1), matching the server's interpolation. */
export function colorAt(stops, u) {
  const s = [...stops].sort((a, b) => a.pos - b.pos);
  if (!s.length) return '#000000';
  if (u <= s[0].pos) return s[0].color;
  for (let i = 1; i < s.length; i++) {
    if (u <= s[i].pos) {
      const span = s[i].pos - s[i - 1].pos;
      const t = span > 0 ? (u - s[i - 1].pos) / span : 0;
      const a = toRgb(s[i - 1].color);
      const b = toRgb(s[i].color);
      return toHex(a.map((x, k) => x + (b[k] - x) * t));
    }
  }
  return s[s.length - 1].color;
}
