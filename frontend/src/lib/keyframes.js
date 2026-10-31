// Keyframe helpers. The show's timeline holds tracks of keys
// ({beat, value, ease}) per layer parameter; these functions edit them in
// place on the reactive show and evaluate them for display. Evaluation
// mirrors internal/timeline on the server.

const EPS = 1e-3;

const eases = {
  linear: (t) => t,
  smooth: (t) => t * t * (3 - 2 * t),
  in: (t) => t * t,
  out: (t) => 1 - (1 - t) * (1 - t),
  step: () => 0,
};

export function findTrack(show, layerId, param) {
  return show.timeline.tracks.find((t) => t.layerId === layerId && t.param === param);
}

export function keyIndexAt(track, beat) {
  return track ? track.keys.findIndex((k) => Math.abs(k.beat - beat) < EPS) : -1;
}

/** Add or replace the key at `beat`, creating the track if needed. */
export function setKey(show, layerId, param, beat, value) {
  let track = findTrack(show, layerId, param);
  if (!track) {
    track = { layerId, param, keys: [] };
    show.timeline.tracks.push(track);
    track = show.timeline.tracks[show.timeline.tracks.length - 1]; // reactive proxy
  }
  const i = keyIndexAt(track, beat);
  if (i >= 0) {
    track.keys[i].value = value;
  } else {
    track.keys.push({ beat, value, ease: 'linear' });
    track.keys.sort((a, b) => a.beat - b.beat);
  }
}

/** Remove the key at `beat`; drop the track when it becomes empty. */
export function removeKeyAt(show, layerId, param, beat) {
  const track = findTrack(show, layerId, param);
  const i = keyIndexAt(track, beat);
  if (i < 0) return;
  track.keys.splice(i, 1);
  if (!track.keys.length) removeTrack(show, track);
}

export function removeTrack(show, track) {
  show.timeline.tracks = show.timeline.tracks.filter((t) => t !== track);
}

function hexToRgb(h) {
  const v = parseInt(h.slice(1), 16);
  return [(v >> 16) & 255, (v >> 8) & 255, v & 255];
}
function rgbToHex([r, g, b]) {
  return '#' + [r, g, b].map((c) => Math.round(c).toString(16).padStart(2, '0')).join('');
}

/** Value of a track at `beat` (number or "#rrggbb"). */
export function evalTrack(track, beat) {
  const keys = track.keys;
  if (!keys.length) return undefined;
  if (beat <= keys[0].beat) return keys[0].value;
  const last = keys[keys.length - 1];
  if (beat >= last.beat) return last.value;
  let i = 1;
  while (keys[i].beat <= beat) i++;
  const a = keys[i - 1];
  const b = keys[i];
  const span = b.beat - a.beat;
  const u = span > 0 ? (eases[a.ease] ?? eases.linear)((beat - a.beat) / span) : 0;
  if (typeof a.value === 'number') return a.value + (b.value - a.value) * u;
  const ca = hexToRgb(a.value);
  const cb = hexToRgb(b.value);
  return rgbToHex(ca.map((c, k) => c + (cb[k] - c) * u));
}

/** Round a beat to the snap grid (0 = no snapping). */
export function snapBeat(beat, snap) {
  return snap > 0 ? Math.round(beat / snap) * snap : beat;
}
