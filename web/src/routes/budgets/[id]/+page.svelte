<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { getBudget, listAccounts, listTags, updateBudget, deleteBudget as removeBudget } from '$lib/api/client';
	import type { Account, Budget, BudgetUpdate, Tag } from '$lib/api/types';
	import BudgetForm from '$lib/components/budgets/BudgetForm.svelte';
	import ButtonLink from '$lib/components/ButtonLink.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';

	let budget = $state<Budget | null>(null);
	let accounts = $state<Account[]>([]);
	let tags = $state<Tag[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let saving = $state(false);
	let deleting = $state(false);

	async function load(id: string) {
		loading = true;
		error = null;
		try {
			budget = await getBudget(id);
			try {
				const [a, t] = await Promise.all([listAccounts(), listTags()]);
				accounts = a;
				tags = t;
			} catch {}
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Budget could not be loaded.';
		} finally {
			loading = false;
		}
	}

	async function save(input: BudgetUpdate) {
		if (!budget) return;
		saving = true;
		error = null;
		try {
			budget = await updateBudget(budget.id, input);
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Budget could not be saved.';
		} finally {
			saving = false;
		}
	}

	async function remove() {
		if (!budget) return;
		if (!window.confirm('Delete this budget? Tag and transaction data is unaffected.')) return;
		deleting = true;
		try {
			await removeBudget(budget.id);
			await goto(resolve('/budgets'));
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Budget could not be deleted.';
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
	<div class="mt-6 h-64 animate-pulse rounded-card border border-line bg-surface" aria-hidden="true"></div>
{:else if error && !budget}
	<div role="alert" class="mt-6 rounded-card border border-negative/40 bg-negative/5 p-5">
		<h2 class="text-sm font-semibold text-negative">Couldn't load budget</h2>
		<p class="mt-1 text-sm text-ink-muted">{error}</p>
	</div>
{:else if budget}
	<PageHeader title={budget.name} description="Edit budget details." />
	<div class="mt-6">
		<BudgetForm budget={budget} accounts={accounts} tags={tags} submitLabel="Save changes" busy={saving} error={error} onSubmit={save} />
	</div>
	<div class="mt-6 flex flex-wrap items-center justify-between gap-3">
		<ButtonLink href="/budgets" variant="secondary">Back to budgets</ButtonLink>
		<button type="button" class="inline-flex min-h-11 items-center justify-center rounded-xl border border-negative/40 bg-surface px-4 text-sm font-semibold text-negative transition-colors hover:bg-negative/5 disabled:cursor-wait disabled:opacity-60" disabled={deleting} onclick={() => void remove()}>
			{deleting ? 'Deleting…' : 'Delete budget'}
		</button>
	</div>
{/if}