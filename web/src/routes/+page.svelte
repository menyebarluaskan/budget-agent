<script lang="ts">
	import ButtonLink from '$lib/components/ButtonLink.svelte';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
</script>

<PageHeader
	title="Overview"
	description="Balance, monthly spending, and budget progress at a glance."
/>

<div class="mt-6">
	<EmptyState
		icon="home"
		title="No data to show yet"
		description="Add an account and your first transaction to bring the dashboard to life."
	>
		{#snippet actions()}
			<ButtonLink href="/transactions/new">Add transaction</ButtonLink>
			<ButtonLink href="/accounts" variant="secondary">View accounts</ButtonLink>
		{/snippet}
	</EmptyState>
</div>
