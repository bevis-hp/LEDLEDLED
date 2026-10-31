<script>
  // Host health for unattended installations: Pi temperature and power
  // problems, uptime, load and free disk.
  import { app } from '../lib/state.svelte.js';

  let sys = $derived(app.status?.system);

  function dur(s) {
    if (!s) return '–';
    const d = Math.floor(s / 86400);
    const h = Math.floor((s % 86400) / 3600);
    const m = Math.floor((s % 3600) / 60);
    return d ? `${d}d ${h}h` : h ? `${h}h ${m}m` : `${m}m`;
  }
  const gb = (b) => (b / 1e9).toFixed(1) + ' GB';
  const tempLevel = (t) => (t >= 80 ? 'bad' : t >= 70 ? 'warn' : 'ok');
</script>

{#if sys}
  <div class="card system">
    <div class="chips">
      {#if sys.cpuTempC}
        <span class="chip {tempLevel(sys.cpuTempC)}" title="CPU temperature. The Pi slows down above ~80 °C: add a heatsink or airflow.">
          {sys.cpuTempC.toFixed(0)} °C
        </span>
      {/if}
      {#if sys.throttle}
        {#if sys.throttle.underVoltageNow}
          <span class="chip bad" title="The Pi's supply is too low right now. Use a proper 5.1 V / 3 A supply and don't power the Pi from the LED supply rail without a regulator.">⚡ under-voltage</span>
        {:else if sys.throttle.underVoltageEver}
          <span class="chip warn" title="The supply dipped below spec at some point since boot.">⚡ under-voltage since boot</span>
        {:else}
          <span class="chip ok" title="No supply problems since boot">⚡ power OK</span>
        {/if}
        {#if sys.throttle.throttledNow}<span class="chip bad">throttled</span>{/if}
      {/if}
      {#if sys.systemUptime}<span class="chip" title="System uptime">up {dur(sys.systemUptime)}</span>{/if}
      <span class="chip" title="Time since LEDLEDLED started">app {dur(sys.appUptime)}</span>
      {#if sys.load1}<span class="chip" title="1-minute load average (4 cores on a Pi 4)">load {sys.load1.toFixed(2)}</span>{/if}
      {#if sys.diskFreeBytes}
        <span class="chip" class:warn={sys.diskFreeBytes < 5e8} title="Free space for shows and the library">{gb(sys.diskFreeBytes)} free</span>
      {/if}
    </div>
    <p class="hint">{sys.hostname} · LEDLEDLED {sys.version}</p>
  </div>
{/if}

<style>
  .system {
    padding: 0.6rem 0.75rem;
  }
  .chips {
    display: flex;
    flex-wrap: wrap;
    gap: 0.35rem;
  }
  .chip {
    font-size: 0.8rem;
    padding: 0.15rem 0.5rem;
    border-radius: 999px;
    border: 1px solid var(--line);
    color: var(--muted);
  }
  .chip.ok {
    color: var(--ok);
    border-color: #1f5a3a;
  }
  .chip.warn {
    color: var(--warn);
    border-color: #6a4d1c;
  }
  .chip.bad {
    color: var(--danger);
    border-color: #6a2228;
  }
  p {
    margin: 0.4rem 0 0;
  }
</style>
