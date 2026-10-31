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
    if (msg.type === 'status') app.status = msg.status;
    else if (msg.type === 'show' && !inflight) app.show = msg.show;
  };
}
