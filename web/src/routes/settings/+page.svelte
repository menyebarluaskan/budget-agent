<script lang="ts">
	import EmptyState from '$lib/components/EmptyState.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
</script>

<PageHeader
	title="Settings"
	description="API tokens, AI status, and display preferences for this workspace."
/>

<div class="mt-6">
	<EmptyState
		icon="settings"
		title="Settings are on the way"
		description="Token management, AI provider status, and workspace preferences will live here."
	/>
</div>
