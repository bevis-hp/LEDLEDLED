// Shared application state and server communication.
//
// The whole show lives in `app.show` as deep reactive state. Components bind
// straight to it and call `save()` after an edit; save() PUTs the complete
// show back to the server, coalescing rapid edits (e.g. slider drags) so at
// most one request is in flight at a time.
import { setFrame } from './preview.js';

export const app = $state({
  show: null, // the show document (see internal/show/show.go)
  meta: null, // effects, colour orders, blend modes, protocols
  status: null, // engine runtime status, pushed every second
  connected: false,
  error: '', // last validation / network error
});

// High-rate values pushed ~20x a second (beat clock, playhead, audio).
// Kept separate from `app` so they only re-render what uses them.
export const live = $state({
  bpm: 120,
  beat: 0,
  timeline: { playing: false, pos: 0, length: 16 },
  audio: null,
});

// Per-device UI preferences.
export const ui = $state({
  snap: Number(localStorage.getItem('snap') ?? 0.25), // keyframe snap in beats (0 = off)
});

// Identifies this browser tab so the server doesn't echo our own edits back.
const clientId = Math.random().toString(36).slice(2, 10);

// Short random IDs. crypto.randomUUID needs a secure context (https), which
// the Pi's plain-http UI isn't, so keep it simple.
export function uid() {
  return Math.random().toString(16).slice(2, 10);
}

async function api(method, path, body) {
  const res = await fetch(path, {
    method,
    headers: { 'Content-Type': 'application/json', 'X-Client-Id': clientId },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const text = await res.text();
  const data = text ? JSON.parse(text) : null;
  if (!res.ok) throw new Error(data?.error || res.statusText);
  return data;
}

// ---- Saving -------------------------------------------------------------

let inflight = false;
let again = false;

export function save() {
  if (inflight) {
    again = true;
    return;
  }
  flush();
}

async function flush() {
  inflight = true;
  try {
    const applied = await api('PUT', '/api/show', $state.snapshot(app.show));
    app.error = '';
    // Adopt the server's normalised copy (fills IDs/defaults) unless the
    // user has already made further edits we're about to send.
    if (!again) app.show = applied;
  } catch (e) {
    app.error = e.message;
  } finally {
    inflight = false;
    if (again) {
      again = false;
      flush();
    }
  }
}

// ---- Runtime controls ---------------------------------------------------

export async function setOutput(on) {
  app.status = await api('POST', '/api/output', { on });
}

export async function setBlackout(on) {
  app.status = await api('POST', '/api/blackout', { on });
}

export async function identify(stripId, seconds = 6) {
  await api('POST', '/api/identify', { stripId, seconds });
}

// ---- Tempo & timeline transport ----------------------------------------

export async function tap() {
  app.status = await api('POST', '/api/tempo/tap');
}

export async function setBpm(bpm) {
  app.status = await api('POST', '/api/tempo', { bpm });
}

/** action: play | pause | stop | seek */
export async function transport(action, beat = 0) {
  await api('POST', '/api/transport', { action, beat });
}

export async function audioDevices() {
  return api('GET', '/api/audio/devices');
}

// ---- Library (shows and layer presets stored on the Pi) -----------------

const libPath = (kind, name) => `/api/library/${kind}/${encodeURIComponent(name)}`;

export const library = {
  list: (kind) => api('GET', `/api/library/${kind}`),
  get: (kind, name) => api('GET', libPath(kind, name)),
  put: (kind, name, data) => api('PUT', libPath(kind, name), data),
  remove: (kind, name) => api('DELETE', libPath(kind, name)),
  downloadUrl: (kind, name) => libPath(kind, name) + '?download=1',
  saveCurrent: (name) => api('POST', libPath('shows', name) + '/save'),
  async load(name) {
    app.show = await api('POST', libPath('shows', name) + '/load');
  },
};

/** Default parameter values for an effect, from the server's schema. */
export function defaultParams(effectName) {
  const def = app.meta?.effects.find((e) => e.name === effectName);
  return Object.fromEntries((def?.params ?? []).map((p) => [p.name, p.default]));
}

// ---- Connection ---------------------------------------------------------

async function load() {
  try {
    const [meta, show] = await Promise.all([api('GET', '/api/meta'), api('GET', '/api/show')]);
    app.meta = meta;
    app.show = show;
  } catch (e) {
    app.error = 'Could not load from server: ' + e.message;
  }
}

export function connect() {
  load();
  const proto = location.protocol === 'https:' ? 'wss' : 'ws';
  const ws = new WebSocket(`${proto}://${location.host}/ws?client=${clientId}`);
  ws.binaryType = 'arraybuffer';

  ws.onopen = () => {
    app.connected = true;
  };
  ws.onclose = () => {
    app.connected = false;
    setTimeout(connect, 1500); // reconnect (and reload) after Wi-Fi drops
  };
  ws.onmessage = (ev) => {
    if (typeof ev.data !== 'string') {
      setFrame(ev.data);
      return;
    }
    const msg = JSON.parse(ev.data);
    if (msg.type === 'live') {
      live.bpm = msg.bpm;
      live.beat = msg.beat;
      live.timeline = msg.timeline;
      live.audio = msg.audio ?? null;
    } else if (msg.type === 'status') app.status = msg.status;
    else if (msg.type === 'show' && !inflight) app.show = msg.show;
  };
}
