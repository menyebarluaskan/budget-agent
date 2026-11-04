<script lang="ts">
	import { resolve } from '$app/paths';
	import { onMount } from 'svelte';
	import { deleteAccount as removeAccount, listAccounts, updateAccount } from '$lib/api/client';
	import type { Account } from '$lib/api/types';
	import ButtonLink from '$lib/components/ButtonLink.svelte';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import { formatMoney } from '$lib/format';

	let accounts = $state<Account[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	async function load() {
		loading = true;
		error = null;
		try {
			accounts = await listAccounts();
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Accounts could not be loaded.';
		} finally {
			loading = false;
		}
	}

	async function archive(account: Account) {
		try {
			await updateAccount(account.id, { archived: !account.archived });
			await load();
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Account could not be updated.';
		}
	}

	async function remove(account: Account) {
		if (!window.confirm('Delete this account and all its transactions? This cannot be undone.'))
			return;
		try {
			await removeAccount(account.id);
			await load();
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Account could not be deleted.';
		}
	}

	onMount(() => void load());
</script>

<PageHeader
	title="Accounts"
	description="Track wallets, savings, and credit balances in one place."
/>

<div class="mt-4 flex justify-end">
	<ButtonLink href="/accounts/new">Add account</ButtonLink>
</div>

{#if loading}
	<div class="mt-6 space-y-3" aria-hidden="true">
		<div class="h-20 animate-pulse rounded-card border border-line bg-surface"></div>
		<div class="h-20 animate-pulse rounded-card border border-line bg-surface"></div>
	</div>
{:else if error && accounts.length === 0}
	<div role="alert" class="mt-6 rounded-card border border-negative/40 bg-negative/5 p-5">
		<h2 class="text-sm font-semibold text-negative">Couldn't load accounts</h2>
		<p class="mt-1 text-sm text-ink-muted">{error}</p>
		<button
			type="button"
			class="mt-4 inline-flex min-h-10 items-center justify-center rounded-xl bg-accent px-4 text-sm font-semibold text-on-accent transition-colors hover:bg-accent-strong"
			onclick={() => void load()}>Try again</button
		>
	</div>
{:else if accounts.length === 0}
	<div class="mt-6">
		<EmptyState
			icon="accounts"
			title="No accounts yet"
			description="Add a wallet, savings account, or credit account."
		>
			{#snippet actions()}<ButtonLink href="/accounts/new">Add account</ButtonLink>{/snippet}
		</EmptyState>
	</div>
{:else}
	<section class="mt-6" aria-labelledby="account-list">
		<ul class="space-y-3">
			{#each accounts as account (account.id)}
				<li class="rounded-card border border-line bg-surface p-4 shadow-card">
					<div class="flex items-start justify-between gap-3">
						<div class="min-w-0">
							<a
								href={resolve(`/accounts/${account.id}`)}
								class="text-sm font-semibold hover:underline">{account.name}</a
							>
							<p class="mt-1 text-xs text-ink-muted">
								{account.type} · {account.currency}{account.archived ? ' · archived' : ''}
							</p>
						</div>
						<p class="text-sm font-semibold tabular-nums">
							{formatMoney(account.balance_minor, account.currency)}
						</p>
					</div>
					<div class="mt-3 flex flex-wrap gap-2">
						<ButtonLink href={`/accounts/${account.id}`} variant="secondary">Edit</ButtonLink>
						<button
							type="button"
							class="inline-flex min-h-9 items-center justify-center rounded-xl border border-line bg-surface px-3 text-xs font-semibold text-ink transition-colors hover:bg-surface-muted"
							onclick={() => void archive(account)}
							>{account.archived ? 'Unarchive' : 'Archive'}</button
						>
						<button
							type="button"
							class="inline-flex min-h-9 items-center justify-center rounded-xl border border-negative/40 bg-surface px-3 text-xs font-semibold text-negative transition-colors hover:bg-negative/5"
							onclick={() => void remove(account)}>Delete</button
						>
					</div>
				</li>
			{/each}
		</ul>
	</section>
{/if}
