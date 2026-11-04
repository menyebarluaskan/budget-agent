<script lang="ts">
	import { resolve } from '$app/paths';
	import { onMount } from 'svelte';
	import { getDashboard, getSpendingAnalytics } from '$lib/api/client';
	import type { Dashboard, SpendingAnalytics } from '$lib/api/types';
	import ButtonLink from '$lib/components/ButtonLink.svelte';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import Icon from '$lib/components/Icon.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import MonthlyTrendChart from '$lib/components/charts/MonthlyTrendChart.svelte';
	import TagSpendingChart from '$lib/components/charts/TagSpendingChart.svelte';
	import BudgetProgressList from '$lib/components/dashboard/BudgetProgressList.svelte';
	import RecentTransactionList from '$lib/components/dashboard/RecentTransactionList.svelte';
	import StatCard from '$lib/components/dashboard/StatCard.svelte';
	import {
		formatDateRange,
		formatMoney,
		formatMonthYear,
		formatSignedMoney,
		monthRange,
		shiftMonth
	} from '$lib/format';

	let cursor = $state(new Date());
	let dashboard = $state<Dashboard | null>(null);
	let analytics = $state<SpendingAnalytics | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let latestRequest = 0;

	const range = $derived(monthRange(cursor.getFullYear(), cursor.getMonth()));
	const monthLabel = $derived(formatMonthYear(range.from));
	const isCurrentMonth = $derived.by(() => {
		const now = new Date();
		return cursor.getFullYear() === now.getFullYear() && cursor.getMonth() === now.getMonth();
	});
	const isEmpty = $derived(
		dashboard !== null &&
			dashboard.account_count === 0 &&
			dashboard.recent_transactions.length === 0 &&
			dashboard.budgets.length === 0
	);
	const periodLabel = $derived(
		dashboard
			? formatDateRange(dashboard.from, dashboard.to)
			: formatDateRange(range.from, range.to)
	);
	const netTone = $derived(
		!dashboard || dashboard.net_minor === 0
			? 'muted'
			: dashboard.net_minor > 0
				? 'positive'
				: 'negative'
	);

	async function load() {
		const request = ++latestRequest;
		loading = true;
		error = null;
		try {
			const [data, spending] = await Promise.all([
				getDashboard(range),
				getSpendingAnalytics(range)
			]);
			if (request === latestRequest) {
				dashboard = data;
				analytics = spending;
			}
		} catch (cause) {
			if (request === latestRequest) {
				error = cause instanceof Error ? cause.message : 'The dashboard could not be loaded.';
			}
		} finally {
			if (request === latestRequest) {
				loading = false;
			}
		}
	}

	function goToMonth(delta: number) {
		cursor = shiftMonth(cursor, delta);
		void load();
	}

	function goToCurrentMonth() {
		cursor = new Date();
		void load();
	}

	onMount(() => {
		void load();
	});
</script>

<PageHeader
	title="Overview"
	description="Balance, monthly spending, and budget progress at a glance."
>
	{#snippet actions()}
		<ButtonLink href="/transactions/new">Add transaction</ButtonLink>
	{/snippet}
</PageHeader>

<div class="mt-6 flex flex-wrap items-center justify-between gap-3">
	<div class="flex items-center gap-1">
		<button
			type="button"
			class="grid size-10 place-items-center rounded-xl border border-line bg-surface text-ink-muted transition-colors hover:bg-surface-muted hover:text-ink"
			aria-label="Previous month"
			onclick={() => goToMonth(-1)}
		>
			<Icon name="chevron-left" class="size-5" />
		</button>
		<p class="min-w-40 text-center text-sm font-semibold" aria-live="polite">{monthLabel}</p>
		<button
			type="button"
			class="grid size-10 place-items-center rounded-xl border border-line bg-surface text-ink-muted transition-colors hover:bg-surface-muted hover:text-ink"
			aria-label="Next month"
			onclick={() => goToMonth(1)}
		>
			<Icon name="chevron-right" class="size-5" />
		</button>
	</div>

	{#if !isCurrentMonth}
		<button
			type="button"
			class="min-h-10 rounded-xl border border-line bg-surface px-3 text-sm font-medium text-ink-muted transition-colors hover:bg-surface-muted hover:text-ink"
			onclick={goToCurrentMonth}
		>
			This month
		</button>
	{/if}
</div>

<p class="sr-only" role="status">{loading ? 'Loading dashboard' : 'Dashboard updated'}</p>

{#if error}
	<div role="alert" class="mt-6 rounded-card border border-negative/40 bg-negative/5 p-5">
		<h2 class="text-sm font-semibold text-negative">Couldn't load the dashboard</h2>
		<p class="mt-1 text-sm text-ink-muted">{error}</p>
		<button
			type="button"
			class="mt-4 inline-flex min-h-10 items-center justify-center rounded-xl bg-accent px-4 text-sm font-semibold text-on-accent transition-colors hover:bg-accent-strong"
			onclick={() => void load()}
		>
			Try again
		</button>
	</div>
{/if}

{#if loading && !dashboard}
	<div class="mt-6 space-y-4" aria-hidden="true">
		<div class="h-40 animate-pulse rounded-card border border-line bg-surface"></div>
		<div class="grid gap-4 sm:grid-cols-3">
			<div class="h-28 animate-pulse rounded-card border border-line bg-surface"></div>
			<div class="h-28 animate-pulse rounded-card border border-line bg-surface"></div>
			<div class="h-28 animate-pulse rounded-card border border-line bg-surface"></div>
		</div>
		<div class="grid gap-4 lg:grid-cols-2">
			<div class="h-64 animate-pulse rounded-card border border-line bg-surface"></div>
			<div class="h-64 animate-pulse rounded-card border border-line bg-surface"></div>
		</div>
	</div>
{:else if dashboard}
	{#if isEmpty}
		<div class="mt-6">
			<EmptyState
				icon="home"
				title="No data to show yet"
				description="Add an account and your first transaction to bring the dashboard to life."
			>
				{#snippet actions()}
					<ButtonLink href="/transactions/new">Add transaction</ButtonLink>
					<ButtonLink href="/accounts" variant="secondary">View accounts</ButtonLink>
				{/snippet}
			</EmptyState>
		</div>
	{:else}
		<div
			class="mt-6 space-y-6 transition-opacity {loading ? 'opacity-60' : ''}"
			aria-busy={loading}
		>
			<section class="rounded-card border border-line bg-surface p-6 shadow-card">
				<div class="flex flex-wrap items-end justify-between gap-4">
					<div>
						<p class="text-sm font-medium text-ink-muted">Total balance</p>
						<p class="mt-2 text-3xl font-semibold tracking-tight tabular-nums sm:text-4xl">
							{formatMoney(dashboard.total_balance_minor, dashboard.currency)}
						</p>
						<p class="mt-1 text-xs text-ink-muted">
							{dashboard.account_count}
							{dashboard.account_count === 1 ? 'account' : 'accounts'} on file
						</p>
					</div>
					<div class="rounded-xl bg-surface-muted px-4 py-3 text-right">
						<p class="text-xs font-medium tracking-wide text-ink-muted uppercase">
							Net this period
						</p>
						<p
							class="mt-1 text-lg font-semibold tabular-nums {dashboard.net_minor > 0
								? 'text-positive'
								: dashboard.net_minor < 0
									? 'text-negative'
									: 'text-ink-muted'}"
						>
							{formatSignedMoney(dashboard.net_minor, dashboard.currency)}
						</p>
					</div>
				</div>
			</section>

			<section aria-labelledby="period-heading">
				<div class="mb-3 flex flex-wrap items-baseline justify-between gap-2">
					<h2 id="period-heading" class="text-base font-semibold tracking-tight">This period</h2>
					<p class="text-xs text-ink-muted">{periodLabel}</p>
				</div>
				<div class="grid gap-4 sm:grid-cols-3">
					<StatCard
						label="Income"
						value={formatMoney(dashboard.income_minor, dashboard.currency)}
						tone="positive"
					/>
					<StatCard
						label="Expenses"
						value={formatMoney(dashboard.expense_minor, dashboard.currency)}
						tone="negative"
					/>
					<StatCard
						label="Net"
						value={formatSignedMoney(dashboard.net_minor, dashboard.currency)}
						hint="Income minus expenses"
						tone={netTone}
					/>
				</div>
			</section>

			{#if analytics}
				<div class="grid items-start gap-6 lg:grid-cols-2">
					<section
						class="rounded-card border border-line bg-surface shadow-card"
						aria-labelledby="tag-spending-heading"
					>
						<div class="flex items-baseline justify-between gap-3 border-b border-line px-5 py-4">
							<h2 id="tag-spending-heading" class="text-base font-semibold tracking-tight">
								Spending by tag
							</h2>
							<p class="text-xs text-ink-muted">{periodLabel}</p>
						</div>
						<div class="p-5">
							<TagSpendingChart
								tags={analytics.tags}
								untaggedMinor={analytics.untagged_minor}
								totalMinor={analytics.total_spent_minor}
								currency={analytics.currency}
							/>
						</div>
					</section>

					<section
						class="rounded-card border border-line bg-surface shadow-card"
						aria-labelledby="trend-heading"
					>
						<div class="flex items-baseline justify-between gap-3 border-b border-line px-5 py-4">
							<h2 id="trend-heading" class="text-base font-semibold tracking-tight">
								Monthly trend
							</h2>
							<p class="text-xs text-ink-muted">Last {analytics.trend.length} months</p>
						</div>
						<div class="p-5">
							<MonthlyTrendChart points={analytics.trend} currency={analytics.currency} />
						</div>
					</section>
				</div>
			{/if}

			<div class="grid items-start gap-6 lg:grid-cols-2">
				<section
					class="rounded-card border border-line bg-surface shadow-card"
					aria-labelledby="budgets-heading"
				>
					<div class="flex items-center justify-between gap-3 border-b border-line px-5 py-4">
						<h2 id="budgets-heading" class="text-base font-semibold tracking-tight">
							Budget progress
						</h2>
						<a
							href={resolve('/budgets')}
							class="text-sm font-medium text-accent-strong hover:underline"
						>
							View budgets
						</a>
					</div>
					<div class="p-5">
						{#if dashboard.budgets.length === 0}
							<p class="text-sm text-ink-muted">
								No active budgets for this period.
								<a
									href={resolve('/budgets')}
									class="font-medium text-accent-strong hover:underline"
								>
									Create a budget
								</a>
							</p>
						{:else}
							<BudgetProgressList budgets={dashboard.budgets} currency={dashboard.currency} />
						{/if}
					</div>
				</section>

				<section
					class="rounded-card border border-line bg-surface shadow-card"
					aria-labelledby="recent-heading"
				>
					<div class="flex items-center justify-between gap-3 border-b border-line px-5 py-4">
						<h2 id="recent-heading" class="text-base font-semibold tracking-tight">
							Recent transactions
						</h2>
						<a
							href={resolve('/transactions')}
							class="text-sm font-medium text-accent-strong hover:underline"
						>
							View all
						</a>
					</div>
					<div class="p-5">
						{#if dashboard.recent_transactions.length === 0}
							<p class="text-sm text-ink-muted">
								No transactions in this period.
								<a
									href={resolve('/transactions/new')}
									class="font-medium text-accent-strong hover:underline"
								>
									Add one
								</a>
							</p>
						{:else}
							<RecentTransactionList
								transactions={dashboard.recent_transactions}
								currency={dashboard.currency}
							/>
						{/if}
					</div>
				</section>
			</div>
		</div>
	{/if}
{/if}
