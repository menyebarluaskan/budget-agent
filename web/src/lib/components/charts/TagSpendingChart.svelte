<script lang="ts">
	import type { TagSpending } from '$lib/api/types';
	import { barPercent, chartMax } from '$lib/charts';
	import { formatMoney } from '$lib/format';

	let {
		tags,
		untaggedMinor = 0,
		totalMinor = 0,
		currency
	}: {
		tags: TagSpending[];
		untaggedMinor?: number;
		totalMinor?: number;
		currency: string;
	} = $props();

	interface Row {
		key: string;
		name: string;
		color: string;
		spentMinor: number;
	}

	const rows = $derived.by<Row[]>(() => {
		const list: Row[] = tags.map((tag) => ({
			key: tag.tag_id,
			name: tag.name,
			color: tag.color,
			spentMinor: tag.spent_minor
		}));
		if (untaggedMinor > 0) {
			list.push({
				key: 'untagged',
				name: 'Untagged',
				color: '',
				spentMinor: untaggedMinor
			});
		}
		return list;
	});

	const max = $derived(chartMax(rows.map((row) => row.spentMinor)));
</script>

{#if rows.length === 0}
	<p class="text-sm text-ink-muted">No spending in this period.</p>
{:else}
	<div class="space-y-4">
		<ul class="space-y-3">
			{#each rows as row (row.key)}
				{@const width = barPercent(row.spentMinor, max)}
				<li>
					<div class="flex items-baseline justify-between gap-3 text-sm">
						<span class="flex min-w-0 items-center gap-2">
							<span
								class="size-2.5 shrink-0 rounded-full"
								style:background-color={row.color || 'var(--token-accent)'}
								aria-hidden="true"
							></span>
							<span class="truncate">{row.name}</span>
						</span>
						<span class="shrink-0 font-semibold tabular-nums">
							{formatMoney(row.spentMinor, currency)}
						</span>
					</div>
					<div class="mt-1.5 h-2 overflow-hidden rounded-full bg-surface-muted" aria-hidden="true">
						<div
							class="h-full rounded-full"
							style:width="{width}%"
							style:background-color={row.color || 'var(--token-accent)'}
						></div>
					</div>
				</li>
			{/each}
		</ul>
		<p class="text-xs text-ink-muted">
			Total spending: {formatMoney(totalMinor, currency)}. Transactions with several tags count in
			each tag.
		</p>
	</div>
{/if}
