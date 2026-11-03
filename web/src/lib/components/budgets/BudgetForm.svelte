<script lang="ts">
	import { untrack } from 'svelte';
	import type { Account, Budget, BudgetInput, BudgetPeriod, Tag } from '$lib/api/types';
	import {
		currencyFractionDigits,
		formatMoney,
		occurredAtFromDate,
		parseMoneyToMinor,
		todayISODate
	} from '$lib/format';

	let {
		budget = null,
		accounts,
		tags,
		currency = 'IDR',
		submitLabel,
		busy = false,
		error = null,
		onCancel,
		onSubmit
	}: {
		budget?: Budget | null;
		accounts: Account[];
		tags: Tag[];
		currency?: string;
		submitLabel: string;
		busy?: boolean;
		error?: string | null;
		onCancel?: () => void;
		onSubmit: (input: BudgetInput) => void | Promise<void>;
	} = $props();

	const digits = $derived(currencyFractionDigits(currency));
	const editing = budget !== null;
	const initial = untrack(() => ({
		name: budget?.name ?? '',
		amount: budget ? String(budget.amount_minor / 10 ** currencyFractionDigits(currency)) : '',
		period: (budget?.period ?? 'monthly') as BudgetPeriod,
		startDate: budget?.start_date ?? todayISODate(),
		endDate: budget?.end_date ?? '',
		rollover: budget?.rollover ?? false,
		accountID: budget?.account_id ?? '',
		tagIDs: budget ? [...budget.tag_ids] : []
	}));

	let name = $state(initial.name);
	let amount = $state(initial.amount);
	let period = $state<BudgetPeriod>(initial.period);
	let startDate = $state(initial.startDate);
	let endDate = $state(initial.endDate ?? '');
	let rollover = $state(initial.rollover);
	let accountID = $state(initial.accountID ?? '');
	let selectedTagIDs = $state<string[]>(initial.tagIDs);
	let localError = $state<string | null>(null);

	const parsedAmount = $derived(parseMoneyToMinor(amount, digits));
	const amountPreview = $derived(
		parsedAmount === null || parsedAmount <= 0 ? null : formatMoney(parsedAmount, currency)
	);
	const visibleError = $derived(localError ?? error);
	const usableAccounts = $derived(accounts.filter((account) => !account.archived));
	const clearEndDate = $derived(editing && Boolean(budget?.end_date) && endDate === '');

	function toggleTag(tagID: string) {
		selectedTagIDs = selectedTagIDs.includes(tagID)
			? selectedTagIDs.filter((id) => id !== tagID)
			: [...selectedTagIDs, tagID];
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		localError = null;
		if (name.trim() === '') {
			localError = 'Enter a budget name.';
			return;
		}
		if (parsedAmount === null || parsedAmount <= 0) {
			localError = `Enter a positive amount with at most ${digits} decimal places.`;
			return;
		}
		if (!startDate) {
			localError = 'Choose a start date.';
			return;
		}
		if (endDate && endDate < startDate) {
			localError = 'The end date must be on or after the start date.';
			return;
		}
		if (selectedTagIDs.length === 0) {
			localError = 'Choose at least one tag to track.';
			return;
		}

		const input: BudgetInput = {
			name: name.trim(),
			amount_minor: parsedAmount,
			period,
			start_date: startDate,
			rollover,
			account_id: accountID || null,
			tag_ids: selectedTagIDs
		};
		if (endDate) {
			input.end_date = endDate;
		} else if (clearEndDate) {
			input.end_date = '';
		}
		await onSubmit(input);
	}
</script>

<form class="space-y-5" onsubmit={submit} novalidate>
	{#if visibleError}
		<div role="alert" class="rounded-card border border-negative/40 bg-negative/5 p-4">
			<p class="text-sm font-semibold text-negative">Couldn't save the budget</p>
			<p class="mt-1 text-sm text-ink-muted">{visibleError}</p>
		</div>
	{/if}

	<div>
		<label for="budget-name" class="text-sm font-medium">Name</label>
		<input
			id="budget-name"
			name="name"
			type="text"
			maxlength="100"
			autocomplete="off"
			class="mt-1.5 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-sm outline-none placeholder:text-ink-muted focus:border-accent"
			placeholder="Groceries"
			bind:value={name}
			required
		/>
	</div>

	<div class="grid gap-4 sm:grid-cols-2">
		<div>
			<label for="budget-amount" class="text-sm font-medium">Limit</label>
			<input
				id="budget-amount"
				name="amount"
				type="text"
				inputmode="decimal"
				autocomplete="off"
				class="mt-1.5 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-base tabular-nums outline-none placeholder:text-ink-muted focus:border-accent"
				placeholder={digits === 0 ? '500000' : '500.00'}
				bind:value={amount}
				required
			/>
			<p class="mt-1.5 min-h-5 text-xs text-ink-muted" aria-live="polite">
				{amountPreview ?? `${currency} · positive amount`}
			</p>
		</div>

		<div>
			<label for="budget-period" class="text-sm font-medium">Period</label>
			<select
				id="budget-period"
				name="period"
				class="mt-1.5 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-sm outline-none focus:border-accent"
				bind:value={period}
				required
			>
				<option value="monthly">Monthly</option>
				<option value="weekly">Weekly</option>
			</select>
		</div>

		<div>
			<label for="budget-start" class="text-sm font-medium">Start date</label>
			<input
				id="budget-start"
				name="start_date"
				type="date"
				class="mt-1.5 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-sm outline-none focus:border-accent"
				bind:value={startDate}
				required
			/>
		</div>

		<div>
			<label for="budget-end" class="text-sm font-medium">End date (optional)</label>
			<input
				id="budget-end"
				name="end_date"
				type="date"
				class="mt-1.5 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-sm outline-none focus:border-accent"
				bind:value={endDate}
			/>
		</div>
	</div>

	<div>
		<label for="budget-account" class="text-sm font-medium">Account (optional)</label>
		<select
			id="budget-account"
			name="account_id"
			class="mt-1.5 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-sm outline-none focus:border-accent"
			bind:value={accountID}
		>
			<option value="">All accounts</option>
			{#each usableAccounts as account (account.id)}
				<option value={account.id}>{account.name} · {account.currency}</option>
			{/each}
		</select>
	</div>

	<fieldset>
		<legend class="text-sm font-medium">Tags</legend>
		{#if tags.length === 0}
			<p class="mt-1.5 text-sm text-ink-muted">
				No tags yet. Create tags before adding a tag-based budget.
			</p>
		{:else}
			<div class="mt-2 flex flex-wrap gap-2">
				{#each tags as tag (tag.id)}
					<label
						class="inline-flex min-h-9 cursor-pointer items-center gap-2 rounded-full border px-3 text-sm transition-colors {selectedTagIDs.includes(
							tag.id
						)
							? 'border-accent bg-accent/10 text-ink'
							: 'border-line bg-surface text-ink-muted hover:bg-surface-muted'}"
					>
						<input
							type="checkbox"
							class="size-4 accent-[var(--token-accent)]"
							checked={selectedTagIDs.includes(tag.id)}
							onchange={() => toggleTag(tag.id)}
						/>
						<span
							class="size-2.5 rounded-full"
							style:background-color={tag.color || 'var(--token-accent)'}
							aria-hidden="true"
						></span>
						{tag.name}
					</label>
				{/each}
			</div>
		{/if}
	</fieldset>

	<label class="flex items-center gap-2 text-sm font-medium">
		<input type="checkbox" class="size-4 accent-[var(--token-accent)]" bind:checked={rollover} />
		Carry unused amounts into the next period
	</label>

	<div class="flex flex-wrap items-center gap-3">
		<button
			type="submit"
			class="inline-flex min-h-11 items-center justify-center rounded-xl bg-accent px-5 text-sm font-semibold text-on-accent transition-colors hover:bg-accent-strong disabled:cursor-not-allowed disabled:opacity-60"
			disabled={busy || tags.length === 0}
		>
			{busy ? 'Saving…' : submitLabel}
		</button>
		{#if onCancel}
			<button
				type="button"
				class="inline-flex min-h-11 items-center justify-center rounded-xl border border-line bg-surface px-4 text-sm font-semibold text-ink transition-colors hover:bg-surface-muted"
				onclick={onCancel}
			>
				Cancel
			</button>
		{/if}
	</div>
</form>
