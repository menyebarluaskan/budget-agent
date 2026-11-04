<script lang="ts">
	import type { MonthPoint } from '$lib/api/types';
	import { barPercent, chartMax } from '$lib/charts';
	import { formatMoney, formatMonthShort, formatMonthYear, formatSignedMoney } from '$lib/format';

	let { points, currency }: { points: MonthPoint[]; currency: string } = $props();

	const max = $derived(
		chartMax(points.flatMap((point) => [point.income_minor, point.expense_minor]))
	);
</script>

{#if points.length === 0}
	<p class="text-sm text-ink-muted">No monthly history yet.</p>
{:else}
	<figure>
		<figcaption class="sr-only">
			Monthly income and spending for the last {points.length} months.
		</figcaption>
		<div class="flex items-end gap-2" aria-hidden="true">
			{#each points as point (point.month)}
				<div class="flex min-w-0 flex-1 flex-col items-center gap-2">
					<div class="flex h-36 w-full items-end justify-center gap-1">
						<div
							class="w-2/5 rounded-t-md bg-positive"
							style:height="{barPercent(point.income_minor, max)}%"
							title="Income {formatMoney(point.income_minor, currency)}"
						></div>
						<div
							class="w-2/5 rounded-t-md bg-negative"
							style:height="{barPercent(point.expense_minor, max)}%"
							title="Expenses {formatMoney(point.expense_minor, currency)}"
						></div>
					</div>
					<p class="truncate text-[11px] text-ink-muted">{formatMonthShort(point.month)}</p>
				</div>
			{/each}
		</div>
		<div class="mt-4 flex flex-wrap gap-4 text-xs text-ink-muted">
			<span class="inline-flex items-center gap-1.5">
				<span class="size-2.5 rounded-full bg-positive" aria-hidden="true"></span>
				Income
			</span>
			<span class="inline-flex items-center gap-1.5">
				<span class="size-2.5 rounded-full bg-negative" aria-hidden="true"></span>
				Expenses
			</span>
		</div>
		<ul class="sr-only">
			{#each points as point (point.month)}
				<li>
					{formatMonthYear(point.month)}: income {formatMoney(point.income_minor, currency)},
					expenses {formatMoney(point.expense_minor, currency)}, net {formatSignedMoney(
						point.net_minor,
						currency
					)}.
				</li>
			{/each}
		</ul>
	</figure>
{/if}
