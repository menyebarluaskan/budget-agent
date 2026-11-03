<script lang="ts">
	import ButtonLink from '$lib/components/ButtonLink.svelte';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
</script>

<PageHeader
	title="Budgets"
	description="Group transactions by tag and track a weekly or monthly limit."
/>

<div class="mt-6">
	<EmptyState
		icon="budgets"
		title="No budgets yet"
		description="Budgets show what is left in a period and carry unused amounts forward when you choose."
	>
		{#snippet actions()}
			<ButtonLink href="/transactions" variant="secondary">Review transactions</ButtonLink>
		{/snippet}
	</EmptyState>
</div>
