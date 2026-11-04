<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { getAccount, updateAccount, deleteAccount as removeAccount } from '$lib/api/client';
	import type { Account, AccountUpdate } from '$lib/api/types';
	import AccountForm from '$lib/components/accounts/AccountForm.svelte';
	import ButtonLink from '$lib/components/ButtonLink.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';

	let account = $state<Account | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let saving = $state(false);
	let deleting = $state(false);

	async function load(id: string) {
		loading = true;
		error = null;
		try {
			account = await getAccount(id);
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Account could not be loaded.';
		} finally {
			loading = false;
		}
	}

	async function save(input: AccountUpdate) {
		if (!account) return;
		saving = true;
		error = null;
		try {
			account = await updateAccount(account.id, input);
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Account could not be saved.';
		} finally {
			saving = false;
		}
	}

	async function remove() {
		if (!account) return;
		if (!window.confirm('Delete this account and all its transactions? This cannot be undone.'))
			return;
		deleting = true;
		try {
			await removeAccount(account.id);
			await goto(resolve('/accounts'));
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Account could not be deleted.';
		} finally {
			deleting = false;
		}
	}

	$effect(() => {
		const id = page.params.id;
		if (id) void load(id);
	});
</script>

{#if loading}
	<div
		class="mt-6 h-64 animate-pulse rounded-card border border-line bg-surface"
		aria-hidden="true"
	></div>
{:else if error && !account}
	<div role="alert" class="mt-6 rounded-card border border-negative/40 bg-negative/5 p-5">
		<h2 class="text-sm font-semibold text-negative">Couldn't load account</h2>
		<p class="mt-1 text-sm text-ink-muted">{error}</p>
	</div>
{:else if account}
	<PageHeader title={account.name} description="Edit account details." />
	<div class="mt-6">
		<AccountForm {account} submitLabel="Save changes" busy={saving} {error} onSubmit={save} />
	</div>
	<div class="mt-6 flex flex-wrap items-center justify-between gap-3">
		<ButtonLink href="/accounts" variant="secondary">Back to accounts</ButtonLink>
		<button
			type="button"
			class="inline-flex min-h-11 items-center justify-center rounded-xl border border-negative/40 bg-surface px-4 text-sm font-semibold text-negative transition-colors hover:bg-negative/5 disabled:cursor-wait disabled:opacity-60"
			disabled={deleting}
			onclick={() => void remove()}
		>
			{deleting ? 'Deleting…' : 'Delete account'}
		</button>
	</div>
{/if}
