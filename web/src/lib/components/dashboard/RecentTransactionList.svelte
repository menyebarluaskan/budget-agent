<script lang="ts">
	import type { Transaction } from '$lib/api/types';
	import { formatDate, formatSignedMoney } from '$lib/format';

	let { transactions, currency }: { transactions: Transaction[]; currency: string } = $props();

	function amountClass(transaction: Transaction): string {
		if (transaction.transfer_id) {
			return 'text-ink-muted';
		}
		if (transaction.amount_minor > 0) {
			return 'text-positive';
		}
		if (transaction.amount_minor < 0) {
			return 'text-negative';
		}
		return 'text-ink-muted';
	}
</script>

<ul class="divide-y divide-line">
	{#each transactions as transaction (transaction.id)}
		<li class="flex items-center justify-between gap-3 py-3 first:pt-0 last:pb-0">
			<div class="min-w-0">
				<p class="truncate text-sm font-medium">
					{transaction.description || 'Untitled transaction'}
				</p>
				<p class="mt-0.5 flex flex-wrap items-center gap-1.5 text-xs text-ink-muted">
					<span>{formatDate(transaction.occurred_at)}</span>
					{#if transaction.transfer_id}
						<span
							class="rounded-full bg-surface-muted px-2 py-0.5 text-[0.65rem] font-medium tracking-wide uppercase"
						>
							Transfer
						</span>
					{/if}
				</p>
			</div>
			<p class="shrink-0 text-sm font-semibold tabular-nums {amountClass(transaction)}">
				{formatSignedMoney(transaction.amount_minor, currency)}
			</p>
		</li>
	{/each}
</ul>
