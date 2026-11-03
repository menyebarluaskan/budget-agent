<script lang="ts">
	import ButtonLink from '$lib/components/ButtonLink.svelte';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
</script>

<PageHeader
	title="New transaction"
	description="Record an expense or income with a signed amount, account, tags, and date."
/>

<div class="mt-6">
	<EmptyState
		icon="plus"
		title="Quick add is coming next"
		description="The entry form is part of the transaction flow. Until then, use the API to create records."
	>
		{#snippet actions()}
			<ButtonLink href="/transactions" variant="secondary">Back to transactions</ButtonLink>
		{/snippet}
	</EmptyState>
</div>
