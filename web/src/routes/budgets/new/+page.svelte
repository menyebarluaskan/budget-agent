<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { onMount } from 'svelte';
	import { createBudget, listAccounts, listTags } from '$lib/api/client';
	import type { Account, BudgetInput, Tag } from '$lib/api/types';
	import BudgetForm from '$lib/components/budgets/BudgetForm.svelte';
	import ButtonLink from '$lib/components/ButtonLink.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';

	let accounts = $state<Account[]>([]);
	let tags = $state<Tag[]>([]);
	let loading = $state(true);
	let saving = $state(false);
	let error = $state<string | null>(null);

	async function load() {
		loading = true;
		error = null;
		try {
			const [a, t] = await Promise.all([listAccounts(), listTags()]);
			accounts = a;
			tags = t;
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Budgets could not be loaded.';
		} finally {
			loading = false;
		}
	}

	async function save(input: BudgetInput) {
		saving = true;
		error = null;
		try {
			const created = await createBudget(input);
			await goto(resolve('/budgets/[id]', { id: created.id }));
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'The budget could not be saved.';
		} finally {
			saving = false;
		}
	}

	onMount(() => void load());
</script>

<PageHeader
	title="New budget"
	description="Create a tag-based budget with a weekly or monthly limit."
/>

{#if loading}
	<div
		class="mt-6 h-64 animate-pulse rounded-card border border-line bg-surface"
		aria-hidden="true"
	></div>
{:else}
	<div class="mt-6">
		<BudgetForm {accounts} {tags} submitLabel="Save budget" busy={saving} {error} onSubmit={save} />
	</div>
	<div class="mt-6">
		<ButtonLink href="/budgets" variant="secondary">Back to budgets</ButtonLink>
	</div>
{/if}
