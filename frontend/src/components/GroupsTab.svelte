<script>
  // Groups: named collections of strip sections that layers can target.
  import { app, save, uid } from '../lib/state.svelte.js';
  import GroupCard from './GroupCard.svelte';
  import { ask } from '../lib/confirm.svelte.js';

  function addGroup() {
    app.show.groups.push({ id: uid(), name: `Group ${app.show.groups.length + 1}`, sections: [] });
    save();
  }

  async function removeGroup(group) {
    if (!(await ask(`Delete group "${group.name}"? Layers using it will target all pixels.`, { confirmLabel: 'Delete', danger: true }))) return;
    for (const l of app.show.layers) if (l.groupId === group.id) l.groupId = '';
    app.show.groups = app.show.groups.filter((g) => g.id !== group.id);
    save();
  }
</script>

<section>
  <h2>Groups</h2>
  <p class="hint">Drag across a strip to add that range to a group as a section. Blue bars show the group's current sections.</p>
  {#each app.show.groups as group (group.id)}
    <GroupCard {group} onremove={() => removeGroup(group)} />
  {:else}
    <div class="card empty">
      <p>No groups yet: layers target all pixels.</p>
      <p class="hint">Add a group, then pick which pixels belong to it.</p>
    </div>
  {/each}
  <div class="row">
    <button class="primary" onclick={addGroup}>+ New group</button>
  </div>
</section>

<style>
  .empty p {
    margin: 0.2rem 0;
  }
</style>
