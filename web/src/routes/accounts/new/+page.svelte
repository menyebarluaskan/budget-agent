<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { onMount } from 'svelte';
	import { createAccount, listAccounts } from '$lib/api/client';
	import type { AccountInput, AccountUpdate } from '$lib/api/types';
	import AccountForm from '$lib/components/accounts/AccountForm.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import ButtonLink from '$lib/components/ButtonLink.svelte';

	let loading = $state(true);
	let saving = $state(false);
	let error = $state<string | null>(null);

	async function load() {
		loading = true;
		error = null;
		try {
			await listAccounts();
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Accounts could not be loaded.';
		} finally {
			loading = false;
		}
	}

	async function save(input: AccountInput | AccountUpdate) {
		saving = true;
		error = null;
		try {
			const created = await createAccount(input as AccountInput);
			await goto(resolve('/accounts/[id]', { id: created.id }));
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'The account could not be saved.';
		} finally {
			saving = false;
		}
	}

	onMount(() => void load());
</script>

<PageHeader title="New account" description="Create a wallet, savings, or credit account." />

{#if loading}
	<div
		class="mt-6 h-64 animate-pulse rounded-card border border-line bg-surface"
		aria-hidden="true"
	></div>
{:else}
	<div class="mt-6">
		<AccountForm submitLabel="Save account" busy={saving} {error} onSubmit={save} />
	</div>
	<div class="mt-6">
		<ButtonLink href="/accounts" variant="secondary">Back to accounts</ButtonLink>
	</div>
{/if}
