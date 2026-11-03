<script lang="ts">
	import type { BudgetProgress } from '$lib/api/types';
	import { formatDateRange, formatMoney } from '$lib/format';

	let { budgets, currency }: { budgets: BudgetProgress[]; currency: string } = $props();

	function rawPercent(budget: BudgetProgress): number {
		if (budget.amount_minor <= 0) {
			return 0;
		}
		return Math.round((budget.spent_minor / budget.amount_minor) * 100);
	}

	function barPercent(budget: BudgetProgress): number {
		return Math.min(100, Math.max(0, rawPercent(budget)));
	}

	function barClass(budget: BudgetProgress): string {
		if (budget.spent_minor > budget.amount_minor) {
			return 'bg-negative';
		}
		if (rawPercent(budget) >= 80) {
			return 'bg-warning';
		}
		return 'bg-accent';
	}
</script>

<ul class="space-y-5">
	{#each budgets as budget (budget.id)}
		{@const percent = rawPercent(budget)}
		{@const width = barPercent(budget)}
		<li>
			<div class="flex items-start justify-between gap-3">
				<div class="min-w-0">
					<p class="truncate text-sm font-semibold">{budget.name}</p>
					<p class="mt-0.5 text-xs text-ink-muted">
						{formatDateRange(budget.period_start, budget.period_end)} · {budget.period}
					</p>
				</div>
				<p class="shrink-0 text-right text-sm font-semibold tabular-nums">
					{formatMoney(budget.spent_minor, currency)}
					<span class="font-normal text-ink-muted"
						>/ {formatMoney(budget.amount_minor, currency)}</span
					>
				</p>
			</div>

			<div
				class="mt-2 h-2 overflow-hidden rounded-full bg-surface-muted"
				role="progressbar"
				aria-label="{budget.name} budget used"
				aria-valuemin="0"
				aria-valuemax="100"
				aria-valuenow={width}
				aria-valuetext="{Math.max(0, percent)}% of {formatMoney(budget.amount_minor, currency)}"
			>
				<div class="h-full rounded-full {barClass(budget)}" style:width="{width}%"></div>
			</div>

			<div class="mt-1.5 flex items-center justify-between gap-3 text-xs">
				<p
					class="tabular-nums {budget.remaining_minor < 0
						? 'font-medium text-negative'
						: 'text-ink-muted'}"
				>
					{budget.remaining_minor < 0
						? `Over by ${formatMoney(-budget.remaining_minor, currency)}`
						: `${formatMoney(budget.remaining_minor, currency)} left`}
				</p>
				<p class="text-ink-muted tabular-nums">{Math.max(0, percent)}%</p>
			</div>
		</li>
	{/each}
</ul>
