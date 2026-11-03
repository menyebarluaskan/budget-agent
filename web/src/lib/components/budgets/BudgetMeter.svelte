<script lang="ts">
	import type { BudgetProgress } from '$lib/api/types';
	import { formatMoney } from '$lib/format';

	let { progress, currency }: { progress: BudgetProgress; currency: string } = $props();

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

	const percent = $derived(rawPercent(progress));
	const width = $derived(barPercent(progress));
</script>

<div>
	<div class="flex items-baseline justify-between gap-3 text-sm">
		<p class="font-semibold tabular-nums">
			{formatMoney(progress.spent_minor, currency)}
			<span class="font-normal text-ink-muted"
				>/ {formatMoney(progress.amount_minor, currency)}</span
			>
		</p>
		<p class="shrink-0 text-xs text-ink-muted tabular-nums">{Math.max(0, percent)}%</p>
	</div>

	<div
		class="mt-2 h-2 overflow-hidden rounded-full bg-surface-muted"
		role="progressbar"
		aria-label="{progress.name} budget used"
		aria-valuemin="0"
		aria-valuemax="100"
		aria-valuenow={width}
		aria-valuetext="{Math.max(0, percent)}% of {formatMoney(progress.amount_minor, currency)}"
	>
		<div class="h-full rounded-full {barClass(progress)}" style:width="{width}%"></div>
	</div>

	<p
		class="mt-1.5 text-xs tabular-nums {progress.remaining_minor < 0
			? 'font-medium text-negative'
			: 'text-ink-muted'}"
	>
		{progress.remaining_minor < 0
			? `Over by ${formatMoney(-progress.remaining_minor, currency)}`
			: `${formatMoney(progress.remaining_minor, currency)} left`}
	</p>
</div>
