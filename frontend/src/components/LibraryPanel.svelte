<script>
  // Library: named shows and layer presets saved as JSON files on the Pi.
  import { onMount } from 'svelte';
  import { app, library } from '../lib/state.svelte.js';

  let shows = $state([]);
  let presets = $state([]);
  let name = $state('');

  async function refresh() {
    try {
      [shows, presets] = await Promise.all([library.list('shows'), library.list('presets')]);
    } catch (e) {
      app.error = e.message;
    }
  }
  onMount(refresh);

  async function run(fn) {
    try {
      await fn();
      app.error = '';
    } catch (e) {
      app.error = e.message;
    }
    refresh();
  }

  const saveCurrent = () =>
    run(async () => {
      const n = name.trim();
      if (!n) return;
      if (shows.some((s) => s.name === n) && !confirm(`Overwrite "${n}"?`)) return;
      await library.saveCurrent(n);
      name = '';
    });

  const load = (n) =>
    run(async () => {
      if (confirm(`Load "${n}"? The current show is replaced on every connected device (save it first if needed).`)) {
        await library.load(n);
      }
    });

  const remove = (kind, n) =>
    run(async () => {
      if (confirm(`Delete "${n}"?`)) await library.remove(kind, n);
    });

  // Upload a .json file from this device into the library.
  function upload(kind) {
    return (ev) =>
      run(async () => {
        const file = ev.currentTarget.files?.[0];
        ev.currentTarget.value = '';
        if (!file) return;
        const n = file.name.replace(/\.json$/i, '').replace(/[^A-Za-z0-9 _.()-]/g, '_').replace(/^[^A-Za-z0-9]+/, '').slice(0, 64) || 'upload';
        await library.put(kind, n, JSON.parse(await file.text()));
      });
  }

  const when = (iso) => new Date(iso).toLocaleString();
</script>

<div class="card">
  <h3>Shows</h3>
  <div class="row">
    <input class="grow" bind:value={name} placeholder="Name for the current show" onkeydown={(e) => e.key === 'Enter' && saveCurrent()} />
    <button class="primary" onclick={saveCurrent} disabled={!name.trim()}>Save current</button>
  </div>
  {#each shows as s (s.name)}
    <div class="row entry">
      <span class="grow"><b>{s.name}</b> <small>{when(s.modified)}</small></span>
      <button class="small" onclick={() => load(s.name)}>Load</button>
      <a class="button small" href={library.downloadUrl('shows', s.name)} download>⤓</a>
      <button class="small danger" onclick={() => remove('shows', s.name)}>✕</button>
    </div>
  {:else}
    <p class="hint">No saved shows yet.</p>
  {/each}
  <label class="button small upload">Upload show .json <input type="file" accept="application/json,.json" onchange={upload('shows')} hidden /></label>
</div>

<div class="card">
  <h3>Layer presets</h3>
  <p class="hint">Save a layer (effect, settings, modulators and keyframes) with "Save as preset" on the Live tab, then add it to any show.</p>
  {#each presets as p (p.name)}
    <div class="row entry">
      <span class="grow"><b>{p.name}</b> <small>{when(p.modified)}</small></span>
      <a class="button small" href={library.downloadUrl('presets', p.name)} download>⤓</a>
      <button class="small danger" onclick={() => remove('presets', p.name)}>✕</button>
    </div>
  {:else}
    <p class="hint">No presets yet.</p>
  {/each}
  <label class="button small upload">Upload preset .json <input type="file" accept="application/json,.json" onchange={upload('presets')} hidden /></label>
</div>

<style>
  .entry {
    border-top: 1px solid var(--line);
    padding: 0.3rem 0;
  }
  small {
    color: var(--muted);
    font-size: 0.75rem;
  }
  a.button {
    text-decoration: none;
  }
  .upload {
    margin-top: 0.5rem;
  }
</style>
