<script lang="ts">
	import EmptyState from '$lib/components/EmptyState.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
</script>

<PageHeader
	title="Accounts"
	description="Track wallets, savings, and credit balances in one place."
/>

<div class="mt-6">
	<EmptyState
		icon="accounts"
		title="No accounts yet"
		description="Add a wallet, savings account, or credit account. Balances and transfers appear here."
	/>
</div>
