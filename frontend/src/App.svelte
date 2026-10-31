<script>
  import { app, save, setOutput, setBlackout } from './lib/state.svelte.js';
  import LiveTab from './components/LiveTab.svelte';
  import TimelineTab from './components/TimelineTab.svelte';
  import Transport from './components/Transport.svelte';
  import GroupsTab from './components/GroupsTab.svelte';
  import SetupTab from './components/SetupTab.svelte';

  const tabs = [
    { id: 'live', label: 'Live' },
    { id: 'timeline', label: 'Timeline' },
    { id: 'groups', label: 'Groups' },
    { id: 'setup', label: 'Setup' },
  ];
  // Remember the tab across reloads (handy when reconnecting on a phone).
  let tab = $state(localStorage.getItem('tab') || 'live');
  $effect(() => localStorage.setItem('tab', tab));
</script>

<header>
  <div class="bar">
    <h1>LEDLEDLED</h1>
    <span class="dot" class:ok={app.connected} title={app.connected ? 'Connected' : 'Disconnected'}></span>
    {#if app.status}<span class="fps">{app.status.fps.toFixed(0)} fps</span>{/if}
    <span class="spacer"></span>
    {#if app.status}
      <button class:on={app.status.outputOn} onclick={() => setOutput(!app.status.outputOn)} title="Stop sending to release the AirPixels (needed to open their config page)">
        Output {app.status.outputOn ? 'ON' : 'OFF'}
      </button>
      <button class:alert={app.status.blackout} onclick={() => setBlackout(!app.status.blackout)}>
        {app.status.blackout ? 'Blacked out' : 'Blackout'}
      </button>
    {/if}
  </div>
  {#if app.show}
    <Transport />
    <label class="master">
      <span>Master {Math.round(app.show.settings.brightness * 100)}%</span>
      <input type="range" min="0" max="1" step="0.01" bind:value={app.show.settings.brightness} oninput={save} />
    </label>
  {/if}
  <nav>
    {#each tabs as t (t.id)}
      <button class:active={tab === t.id} onclick={() => (tab = t.id)}>{t.label}</button>
    {/each}
  </nav>
</header>

{#if app.error}
  <div class="error" role="alert">
    <pre>{app.error}</pre>
    <button class="icon" onclick={() => (app.error = '')}>✕</button>
  </div>
{/if}

<main>
  {#if !app.show || !app.meta}
    <p class="hint">Connecting…</p>
  {:else if tab === 'live'}
    <LiveTab />
  {:else if tab === 'timeline'}
    <TimelineTab />
  {:else if tab === 'groups'}
    <GroupsTab />
  {:else}
    <SetupTab />
  {/if}
</main>

<style>
  header {
    position: sticky;
    top: 0;
    z-index: 10;
    background: var(--bg-2);
    border-bottom: 1px solid var(--line);
    padding: 0.5rem 0.75rem 0;
  }
  .bar {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
  h1 {
    font-size: 1.05rem;
    margin: 0;
    letter-spacing: 0.08em;
  }
  .dot {
    width: 0.6rem;
    height: 0.6rem;
    border-radius: 50%;
    background: var(--danger);
  }
  .dot.ok {
    background: var(--ok);
  }
  .fps {
    font-size: 0.75rem;
    color: var(--muted);
  }
  .master {
    display: grid;
    grid-template-columns: 7rem 1fr;
    align-items: center;
    gap: 0.5rem;
    margin: 0.4rem 0;
    font-size: 0.85rem;
  }
  nav {
    display: flex;
    gap: 0.25rem;
  }
  nav button {
    flex: 1;
    border-radius: 6px 6px 0 0;
    border-bottom: none;
  }
  nav button.active {
    background: var(--bg);
    color: var(--accent);
  }
  .error {
    display: flex;
    gap: 0.5rem;
    align-items: flex-start;
    background: #3a1418;
    border-bottom: 1px solid var(--danger);
    padding: 0.5rem 0.75rem;
  }
  .error pre {
    flex: 1;
    margin: 0;
    white-space: pre-wrap;
    font-size: 0.8rem;
  }
  main {
    padding: 0.75rem;
    max-width: 900px;
    margin: 0 auto;
  }
</style>
