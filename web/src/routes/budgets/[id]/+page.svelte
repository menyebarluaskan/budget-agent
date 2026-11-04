<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import {
		getBudget,
		getBudgetTrend,
		listAccounts,
		listTags,
		updateBudget,
		deleteBudget as removeBudget
	} from '$lib/api/client';
	import type { Account, Budget, BudgetTrend, BudgetUpdate, Tag } from '$lib/api/types';
	import BudgetBurnDown from '$lib/components/budgets/BudgetBurnDown.svelte';
	import BudgetForm from '$lib/components/budgets/BudgetForm.svelte';
	import BudgetMeter from '$lib/components/budgets/BudgetMeter.svelte';
	import BudgetSimulator from '$lib/components/budgets/BudgetSimulator.svelte';
	import ButtonLink from '$lib/components/ButtonLink.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import { formatDateRange } from '$lib/format';

	let budget = $state<Budget | null>(null);
	let trend = $state<BudgetTrend | null>(null);
	let trendError = $state<string | null>(null);
	let accounts = $state<Account[]>([]);
	let tags = $state<Tag[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let saving = $state(false);
	let deleting = $state(false);

	const currency = $derived(
		trend?.currency ??
			accounts.find((account) => account.id === budget?.account_id)?.currency ??
			accounts.find((account) => !account.archived)?.currency ??
			'IDR'
	);
	const budgetTags = $derived.by(() => {
		const byID = new Map(tags.map((tag) => [tag.id, tag]));
		return (budget?.tag_ids ?? [])
			.map((id) => byID.get(id))
			.filter((tag): tag is Tag => tag !== undefined);
	});
	const currentPeriod = $derived(
		trend && trend.periods.length > 0 ? trend.periods[trend.periods.length - 1] : null
	);
	const currentPeriodLabel = $derived(
		currentPeriod ? formatDateRange(currentPeriod.start, currentPeriod.end) : 'the current period'
	);

	async function load(id: string) {
		loading = true;
		error = null;
		trendError = null;
		try {
			budget = await getBudget(id);
			try {
				trend = await getBudgetTrend(id);
			} catch (cause) {
				trend = null;
				trendError =
					cause instanceof Error ? cause.message : 'The budget trend could not be loaded.';
			}
			try {
				const [a, t] = await Promise.all([listAccounts(), listTags()]);
				accounts = a;
				tags = t;
			} catch {
				// Account and tag options are optional for this view.
			}
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
	<div
		class="mt-6 h-64 animate-pulse rounded-card border border-line bg-surface"
		aria-hidden="true"
	></div>
{:else if error && !budget}
	<div role="alert" class="mt-6 rounded-card border border-negative/40 bg-negative/5 p-5">
		<h2 class="text-sm font-semibold text-negative">Couldn't load budget</h2>
		<p class="mt-1 text-sm text-ink-muted">{error}</p>
	</div>
{:else if budget}
	<PageHeader
		title={budget.name}
		description="Review spending, try a what-if simulation, then edit the budget."
	/>

	{#if budgetTags.length > 0}
		<ul class="mt-4 flex flex-wrap gap-2" aria-label="Budget tags">
			{#each budgetTags as tag (tag.id)}
				<li
					class="inline-flex items-center gap-2 rounded-full border border-line bg-surface px-3 py-1.5 text-sm text-ink-muted"
				>
					<span
						class="size-2.5 rounded-full"
						style:background-color={tag.color || 'var(--token-accent)'}
						aria-hidden="true"
					></span>
					{tag.name}
				</li>
			{/each}
		</ul>
	{/if}

	{#if trendError}
		<div role="alert" class="mt-6 rounded-card border border-warning/40 bg-warning/5 p-4">
			<p class="text-sm font-semibold text-warning">Charts unavailable</p>
			<p class="mt-1 text-sm text-ink-muted">{trendError}</p>
		</div>
	{/if}

	{#if trend}
		<div class="mt-6 grid items-start gap-6 lg:grid-cols-2">
			<section
				class="rounded-card border border-line bg-surface shadow-card"
				aria-labelledby="burn-down-heading"
			>
				<div class="flex items-baseline justify-between gap-3 border-b border-line px-5 py-4">
					<h2 id="burn-down-heading" class="text-base font-semibold tracking-tight">
						Spending by period
					</h2>
					<p class="text-xs text-ink-muted">
						{trend.periods.length}
						{trend.periods.length === 1 ? 'period' : 'periods'}
					</p>
				</div>
				<div class="p-5">
					{#if currentPeriod}
						<BudgetMeter
							progress={{
								id: budget.id,
								name: budget.name,
								period: budget.period,
								amount_minor: currentPeriod.amount_minor,
								spent_minor: currentPeriod.spent_minor,
								remaining_minor: currentPeriod.remaining_minor,
								rollover: budget.rollover,
								period_start: currentPeriod.start,
								period_end: currentPeriod.end
							}}
							currency={trend.currency}
						/>
						<div class="mt-6">
							<BudgetBurnDown
								periods={trend.periods}
								amountMinor={trend.amount_minor}
								currency={trend.currency}
							/>
						</div>
					{:else}
						<BudgetBurnDown
							periods={trend.periods}
							amountMinor={trend.amount_minor}
							currency={trend.currency}
						/>
					{/if}
				</div>
			</section>

			<section
				class="rounded-card border border-line bg-surface shadow-card"
				aria-labelledby="simulation-heading"
			>
				<div class="border-b border-line px-5 py-4">
					<h2 id="simulation-heading" class="text-base font-semibold tracking-tight">
						What-if simulation
					</h2>
				</div>
				<div class="p-5">
					<BudgetSimulator
						amountMinor={trend.amount_minor}
						spentMinor={currentPeriod?.spent_minor ?? 0}
						currency={trend.currency}
						periodLabel={currentPeriodLabel}
					/>
				</div>
			</section>
		</div>
	{/if}

	<section class="mt-6" aria-labelledby="edit-budget-heading">
		<h2 id="edit-budget-heading" class="text-base font-semibold tracking-tight">Edit budget</h2>
		<div class="mt-3">
			<BudgetForm
				{budget}
				{accounts}
				{tags}
				{currency}
				submitLabel="Save changes"
				busy={saving}
				{error}
				onSubmit={save}
			/>
		</div>
	</section>
	<div class="mt-6 flex flex-wrap items-center justify-between gap-3">
		<ButtonLink href="/budgets" variant="secondary">Back to budgets</ButtonLink>
		<button
			type="button"
			class="inline-flex min-h-11 items-center justify-center rounded-xl border border-negative/40 bg-surface px-4 text-sm font-semibold text-negative transition-colors hover:bg-negative/5 disabled:cursor-wait disabled:opacity-60"
			disabled={deleting}
			onclick={() => void remove()}
		>
			{deleting ? 'Deleting…' : 'Delete budget'}
		</button>
	</div>
{/if}
