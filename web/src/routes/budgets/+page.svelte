<script lang="ts">
	import { resolve } from '$app/paths';
	import { onMount } from 'svelte';
	import { deleteBudget as removeBudget, listBudgets } from '$lib/api/client';
	import type { Budget } from '$lib/api/types';
	import ButtonLink from '$lib/components/ButtonLink.svelte';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';

	let budgets = $state<Budget[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	async function load() {
		loading = true;
		error = null;
		try {
			budgets = await listBudgets();
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Budgets could not be loaded.';
		} finally {
			loading = false;
		}
	}

	async function remove(budget: Budget) {
		if (!window.confirm('Delete this budget? Tag and transaction data is unaffected.')) return;
		try {
			await removeBudget(budget.id);
			await load();
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Budget could not be deleted.';
		}
	}

	onMount(() => void load());
</script>

<PageHeader
	title="Budgets"
	description="Group transactions by tag and track a weekly or monthly limit."
/>

<div class="mt-4 flex justify-end">
	<ButtonLink href="/budgets/new">Add budget</ButtonLink>
</div>

{#if loading}
	<div class="mt-6 space-y-3" aria-hidden="true">
		<div class="h-20 animate-pulse rounded-card border border-line bg-surface"></div>
		<div class="h-20 animate-pulse rounded-card border border-line bg-surface"></div>
	</div>
{:else if error && budgets.length === 0}
	<div role="alert" class="mt-6 rounded-card border border-negative/40 bg-negative/5 p-5">
		<h2 class="text-sm font-semibold text-negative">Couldn't load budgets</h2>
		<p class="mt-1 text-sm text-ink-muted">{error}</p>
		<button
			type="button"
			class="mt-4 inline-flex min-h-10 items-center justify-center rounded-xl bg-accent px-4 text-sm font-semibold text-on-accent transition-colors hover:bg-accent-strong"
			onclick={() => void load()}>Try again</button
		>
	</div>
{:else if budgets.length === 0}
	<div class="mt-6">
		<EmptyState
			icon="budgets"
			title="No budgets yet"
			description="Budgets show what is left in a period and carry unused amounts forward when you choose."
		>
			{#snippet actions()}<ButtonLink href="/budgets/new">Add budget</ButtonLink>{/snippet}
		</EmptyState>
	</div>
{:else}
	<section class="mt-6" aria-labelledby="budget-list">
		<ul class="space-y-3">
			{#each budgets as budget (budget.id)}
				<li class="rounded-card border border-line bg-surface p-4 shadow-card">
					<div class="flex items-start justify-between gap-3">
						<div class="min-w-0">
							<a
								href={resolve(`/budgets/${budget.id}`)}
								class="text-sm font-semibold hover:underline">{budget.name}</a
							>
							<p class="mt-1 text-xs text-ink-muted">
								{budget.period} · {budget.tag_ids.length} tag(s)
							</p>
						</div>
						<p class="text-sm font-semibold tabular-nums">{budget.amount_minor}</p>
					</div>
					<div class="mt-3 flex flex-wrap gap-2">
						<ButtonLink href={`/budgets/${budget.id}`} variant="secondary">Edit</ButtonLink>
						<button
							type="button"
							class="inline-flex min-h-9 items-center justify-center rounded-xl border border-negative/40 bg-surface px-3 text-xs font-semibold text-negative transition-colors hover:bg-negative/5"
							onclick={() => void remove(budget)}>Delete</button
						>
					</div>
				</li>
			{/each}
		</ul>
	</section>
{/if}
