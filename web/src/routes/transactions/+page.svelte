<script lang="ts">
	import ButtonLink from '$lib/components/ButtonLink.svelte';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
</script>

<PageHeader title="Transactions" description="Search and filter spending, income, and transfers." />

<div class="mt-6">
	<EmptyState
		icon="transactions"
		title="No transactions yet"
		description="Record spending and income to see them here. Filters, sorting, and tags arrive with this view."
	>
		{#snippet actions()}
			<ButtonLink href="/transactions/new">Add transaction</ButtonLink>
		{/snippet}
	</EmptyState>
</div>
