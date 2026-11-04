<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { onMount } from 'svelte';
	import { createTransaction, listAccounts, listTags } from '$lib/api/client';
	import type { Account, Tag, TransactionInput } from '$lib/api/types';
	import ButtonLink from '$lib/components/ButtonLink.svelte';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import TransactionForm from '$lib/components/transactions/TransactionForm.svelte';

	let accounts = $state<Account[]>([]);
	let tags = $state<Tag[]>([]);
	let loading = $state(true);
	let saving = $state(false);
	let error = $state<string | null>(null);

	async function load() {
		loading = true;
		error = null;
		try {
			const [accountList, tagList] = await Promise.all([listAccounts(), listTags()]);
			accounts = accountList;
			tags = tagList;
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Accounts and tags could not be loaded.';
		} finally {
			loading = false;
		}
	}

	async function save(input: TransactionInput) {
		saving = true;
		error = null;
		try {
			const created = await createTransaction(input);
			await goto(resolve('/transactions/[id]', { id: created.id }));
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'The transaction could not be saved.';
		} finally {
			saving = false;
		}
	}

	onMount(() => {
		void load();
	});
</script>

<PageHeader
	title="New transaction"
	description="Record an expense or income with a signed amount, account, tags, and date."
/>

{#if loading}
	<div class="mt-6 space-y-4" aria-hidden="true">
		<div class="h-64 animate-pulse rounded-card border border-line bg-surface"></div>
		<div class="h-64 animate-pulse rounded-card border border-line bg-surface"></div>
	</div>
{:else if error && accounts.length === 0}
	<div role="alert" class="mt-6 rounded-card border border-negative/40 bg-negative/5 p-5">
		<h2 class="text-sm font-semibold text-negative">Couldn't load accounts</h2>
		<p class="mt-1 text-sm text-ink-muted">{error}</p>
		<button
			type="button"
			class="mt-4 inline-flex min-h-10 items-center justify-center rounded-xl bg-accent px-4 text-sm font-semibold text-on-accent transition-colors hover:bg-accent-strong"
			onclick={() => void load()}
		>
			Try again
		</button>
	</div>
{:else if accounts.length === 0}
	<div class="mt-6">
		<EmptyState
			icon="accounts"
			title="Add an account first"
			description="Transactions belong to an account. Create a wallet, savings, or credit account to get started."
		>
			{#snippet actions()}
				<ButtonLink href="/accounts">Go to accounts</ButtonLink>
				<ButtonLink href="/transactions" variant="secondary">Back to transactions</ButtonLink>
			{/snippet}
		</EmptyState>
	</div>
{:else}
	<div class="mt-6">
		<TransactionForm
			{accounts}
			{tags}
			submitLabel="Save transaction"
			busy={saving}
			{error}
			onSubmit={save}
			onTagsChange={(next) => (tags = next)}
		/>
	</div>
{/if}
