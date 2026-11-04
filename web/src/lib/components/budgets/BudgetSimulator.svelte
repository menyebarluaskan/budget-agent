<script lang="ts">
	import { untrack } from 'svelte';
	import type { BudgetSimulationResult } from '$lib/simulation';
	import { simulateBudget } from '$lib/simulation';
	import {
		currencyFractionDigits,
		formatMinorForInput,
		formatMoney,
		parseMoneyToMinor
	} from '$lib/format';

	let {
		amountMinor,
		spentMinor,
		currency,
		periodLabel
	}: {
		amountMinor: number;
		spentMinor: number;
		currency: string;
		periodLabel: string;
	} = $props();

	const digits = $derived(currencyFractionDigits(currency));
	let budgetInput = $state(untrack(() => formatMinorForInput(amountMinor, digits)));
	let extraInput = $state('0');

	$effect(() => {
		budgetInput = formatMinorForInput(amountMinor, digits);
		extraInput = '0';
	});

	const parsedBudget = $derived(parseMoneyToMinor(budgetInput, digits));
	const parsedExtra = $derived(
		extraInput.trim() === '' ? 0 : parseMoneyToMinor(extraInput, digits)
	);
	const inputError = $derived.by(() => {
		if (parsedBudget === null || parsedBudget <= 0) {
			return 'Enter a budget limit greater than zero.';
		}
		if (parsedExtra === null || parsedExtra < 0) {
			return `Enter an extra spend of zero or more with at most ${digits} decimal places.`;
		}
		return null;
	});
	const simulation = $derived.by<BudgetSimulationResult | null>(() => {
		if (inputError !== null || parsedBudget === null || parsedExtra === null) {
			return null;
		}
		return simulateBudget({
			budgetMinor: parsedBudget,
			spentMinor,
			extraSpendMinor: parsedExtra
		});
	});

	function reset() {
		budgetInput = formatMinorForInput(amountMinor, digits);
		extraInput = '0';
	}
</script>

<div class="space-y-5">
	<p class="text-sm text-ink-muted">
		Try a different limit or an extra purchase for {periodLabel}. This is a local simulation;
		nothing is saved or sent to the API.
	</p>

	<div class="grid gap-4 sm:grid-cols-2">
		<div>
			<label for="simulation-budget" class="text-sm font-medium">Budget limit</label>
			<input
				id="simulation-budget"
				name="simulation_budget"
				type="text"
				inputmode="decimal"
				autocomplete="off"
				class="mt-1.5 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-base tabular-nums outline-none focus:border-accent"
				bind:value={budgetInput}
			/>
		</div>
		<div>
			<label for="simulation-extra" class="text-sm font-medium">Extra hypothetical spend</label>
			<input
				id="simulation-extra"
				name="simulation_extra"
				type="text"
				inputmode="decimal"
				autocomplete="off"
				class="mt-1.5 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-base tabular-nums outline-none focus:border-accent"
				bind:value={extraInput}
			/>
		</div>
	</div>

	<p class="text-xs text-ink-muted">
		Already spent {formatMoney(spentMinor, currency)} in this period.
	</p>

	<div aria-live="polite">
		{#if inputError}
			<p role="alert" class="text-sm font-medium text-negative">{inputError}</p>
		{:else if simulation}
			<dl class="grid gap-3 rounded-xl bg-surface-muted p-4 sm:grid-cols-3">
				<div>
					<dt class="text-xs font-medium tracking-wide text-ink-muted uppercase">
						Projected spend
					</dt>
					<dd class="mt-1 text-lg font-semibold tabular-nums">
						{formatMoney(simulation.projectedSpentMinor, currency)}
					</dd>
				</div>
				<div>
					<dt class="text-xs font-medium tracking-wide text-ink-muted uppercase">
						{simulation.overBudget ? 'Over budget' : 'Remaining'}
					</dt>
					<dd
						class="mt-1 text-lg font-semibold tabular-nums {simulation.overBudget
							? 'text-negative'
							: 'text-positive'}"
					>
						{formatMoney(Math.abs(simulation.remainingMinor), currency)}
					</dd>
				</div>
				<div>
					<dt class="text-xs font-medium tracking-wide text-ink-muted uppercase">Budget used</dt>
					<dd class="mt-1 text-lg font-semibold tabular-nums">{simulation.percentUsed}%</dd>
				</div>
			</dl>

			<div
				class="mt-3 h-2 overflow-hidden rounded-full bg-surface-muted"
				role="progressbar"
				aria-label="Simulated budget used"
				aria-valuemin="0"
				aria-valuemax="100"
				aria-valuenow={simulation.barPercent}
				aria-valuetext="{simulation.percentUsed}% of {formatMoney(parsedBudget ?? 0, currency)}"
			>
				<div
					class="h-full rounded-full {simulation.overBudget ? 'bg-negative' : 'bg-accent'}"
					style:width="{simulation.barPercent}%"
				></div>
			</div>
		{/if}
	</div>

	<button
		type="button"
		class="inline-flex min-h-10 items-center justify-center rounded-xl border border-line bg-surface px-4 text-sm font-semibold text-ink transition-colors hover:bg-surface-muted"
		onclick={reset}
	>
		Reset simulation
	</button>
</div>
