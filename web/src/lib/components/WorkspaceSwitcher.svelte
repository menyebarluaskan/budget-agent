<script lang="ts">
	import { getWorkspaces, hasMultiple, setActiveWorkspace } from '$lib/workspace.svelte';
	import { getActiveWorkspaceId } from '$lib/activeWorkspace';

	const workspaces = $derived(getWorkspaces());
	const multiple = $derived(hasMultiple());
	const activeId = $derived(getActiveWorkspaceId());

	function change(event: Event) {
		const target = event.target as HTMLSelectElement;
		setActiveWorkspace(target.value);
		// reload to apply new workspace context
		if (typeof window !== 'undefined') window.location.reload();
	}
</script>

{#if multiple}
	<label class="block px-3 py-2">
		<span class="sr-only">Workspace</span>
		<select
			class="w-full rounded-lg border border-line bg-surface px-2 py-2 text-sm"
			value={activeId ?? ''}
			onchange={change}
		>
			{#each workspaces as w (w.id)}
				<option value={w.id}>{w.name}</option>
			{/each}
		</select>
	</label>
{/if}
