<script lang="ts">
	import type { BudgetPeriodSpending } from '$lib/api/types';
	import { barPercent, chartMax } from '$lib/charts';
	import { formatDateRange, formatDayMonth, formatMoney } from '$lib/format';

	let {
		periods,
		amountMinor,
		currency
	}: {
		periods: BudgetPeriodSpending[];
		amountMinor: number;
		currency: string;
	} = $props();

	const max = $derived(chartMax([amountMinor, ...periods.map((period) => period.spent_minor)]));
</script>

{#if periods.length === 0}
	<p class="text-sm text-ink-muted">This budget has no completed periods yet.</p>
{:else}
	<figure>
		<figcaption class="sr-only">
			Spending compared with the budget limit for each of the last {periods.length} periods.
		</figcaption>
		<div class="flex items-end gap-2" aria-hidden="true">
			{#each periods as period (period.start)}
				<div class="flex min-w-0 flex-1 flex-col items-center gap-2">
					<div class="flex h-36 w-full items-end justify-center gap-1">
						<div
							class="w-2/5 rounded-t-md {period.spent_minor > period.amount_minor
								? 'bg-negative'
								: 'bg-accent'}"
							style:height="{barPercent(period.spent_minor, max)}%"
							title="Spent {formatMoney(period.spent_minor, currency)}"
						></div>
						<div
							class="w-2/5 rounded-t-md bg-line"
							style:height="{barPercent(period.amount_minor, max)}%"
							title="Budget {formatMoney(period.amount_minor, currency)}"
						></div>
					</div>
					<p class="truncate text-[11px] text-ink-muted">{formatDayMonth(period.start)}</p>
				</div>
			{/each}
		</div>
		<div class="mt-4 flex flex-wrap gap-4 text-xs text-ink-muted">
			<span class="inline-flex items-center gap-1.5">
				<span class="size-2.5 rounded-full bg-accent" aria-hidden="true"></span>
				Spent
			</span>
			<span class="inline-flex items-center gap-1.5">
				<span class="size-2.5 rounded-full bg-line" aria-hidden="true"></span>
				Budget
			</span>
		</div>
		<ul class="sr-only">
			{#each periods as period (period.start)}
				<li>
					{formatDateRange(period.start, period.end)}: spent {formatMoney(
						period.spent_minor,
						currency
					)}
					of {formatMoney(period.amount_minor, currency)};
					{period.remaining_minor < 0
						? `over by ${formatMoney(-period.remaining_minor, currency)}`
						: `${formatMoney(period.remaining_minor, currency)} left`}.
				</li>
			{/each}
		</ul>
	</figure>
{/if}
