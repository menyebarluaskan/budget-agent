<script lang="ts">
	import { untrack } from 'svelte';
	import type { Account, TransferInput } from '$lib/api/types';
	import {
		currencyFractionDigits,
		formatSignedMoney,
		occurredAtFromDate,
		parseMoneyToMinor,
		todayISODate
	} from '$lib/format';

	let {
		accounts,
		submitLabel,
		busy = false,
		error = null,
		onCancel,
		onSubmit
	}: {
		accounts: Account[];
		submitLabel: string;
		busy?: boolean;
		error?: string | null;
		onCancel?: () => void;
		onSubmit: (input: TransferInput) => void | Promise<void>;
	} = $props();

	const usable = $derived(accounts.filter((account) => !account.archived));
	const initial = untrack(() => {
		const from = usable[0];
		const to = usable.find(
			(account) => account.id !== from?.id && account.currency === from?.currency
		);
		return { from: from?.id ?? '', to: to?.id ?? '' };
	});

	let fromAccountID = $state(initial.from);
	let toAccountID = $state(initial.to);
	let amount = $state('');
	let date = $state(todayISODate());
	let description = $state('');
	let notes = $state('');
	let localError = $state<string | null>(null);

	const fromAccount = $derived(accounts.find((account) => account.id === fromAccountID));
	const currency = $derived(fromAccount?.currency ?? 'IDR');
	const digits = $derived(currencyFractionDigits(currency));
	const eligibleTo = $derived(
		usable.filter(
			(account) => account.id !== fromAccountID && account.currency === fromAccount?.currency
		)
	);
	const parsedAmount = $derived(parseMoneyToMinor(amount, digits));
	const amountPreview = $derived(
		parsedAmount === null || parsedAmount <= 0 ? null : formatSignedMoney(parsedAmount, currency)
	);
	const visibleError = $derived(localError ?? error);

	$effect(() => {
		if (!eligibleTo.some((account) => account.id === toAccountID)) {
			toAccountID = eligibleTo[0]?.id ?? '';
		}
	});

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		localError = null;
		if (!fromAccountID || !toAccountID) {
			localError = 'Choose both a source and a destination account.';
			return;
		}
		if (fromAccountID === toAccountID) {
			localError = 'Choose two different accounts.';
			return;
		}
		if (parsedAmount === null || parsedAmount <= 0) {
			localError = 'Enter a positive amount to move.';
			return;
		}
		if (!date) {
			localError = 'Choose a date.';
			return;
		}

		await onSubmit({
			from_account_id: fromAccountID,
			to_account_id: toAccountID,
			amount_minor: parsedAmount,
			occurred_at: occurredAtFromDate(date),
			description: description.trim(),
			notes: notes.trim()
		});
	}
</script>

<form class="space-y-5" onsubmit={submit} novalidate>
	{#if visibleError}
		<div role="alert" class="rounded-card border border-negative/40 bg-negative/5 p-4">
			<p class="text-sm font-semibold text-negative">Couldn't save the transfer</p>
			<p class="mt-1 text-sm text-ink-muted">{visibleError}</p>
		</div>
	{/if}

	<p class="text-sm text-ink-muted">
		A transfer writes a negative and a positive transaction with the same transfer ID. Both accounts
		must use the same currency.
	</p>

	<div class="grid gap-4 sm:grid-cols-2">
		<div>
			<label for="transfer-from" class="text-sm font-medium">From</label>
			<select
				id="transfer-from"
				name="from_account_id"
				class="mt-1.5 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-sm outline-none focus:border-accent"
				bind:value={fromAccountID}
				required
			>
				{#each usable as account (account.id)}
					<option value={account.id}>{account.name} · {account.currency}</option>
				{/each}
			</select>
		</div>

		<div>
			<label for="transfer-to" class="text-sm font-medium">To</label>
			<select
				id="transfer-to"
				name="to_account_id"
				class="mt-1.5 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-sm outline-none focus:border-accent"
				bind:value={toAccountID}
				required
			>
				{#if eligibleTo.length === 0}
					<option value="">No matching account</option>
				{/if}
				{#each eligibleTo as account (account.id)}
					<option value={account.id}>{account.name} · {account.currency}</option>
				{/each}
			</select>
		</div>

		<div>
			<label for="transfer-amount" class="text-sm font-medium">Amount</label>
			<input
				id="transfer-amount"
				name="amount"
				type="text"
				inputmode="decimal"
				autocomplete="off"
				class="mt-1.5 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-base tabular-nums outline-none placeholder:text-ink-muted focus:border-accent"
				placeholder={digits === 0 ? '25000' : '25.00'}
				bind:value={amount}
				required
			/>
			<p class="mt-1.5 min-h-5 text-xs text-ink-muted" aria-live="polite">
				{amountPreview ?? `${currency} · positive amount`}
			</p>
		</div>

		<div>
			<label for="transfer-date" class="text-sm font-medium">Date</label>
			<input
				id="transfer-date"
				name="occurred_at"
				type="date"
				class="mt-1.5 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-sm outline-none focus:border-accent"
				bind:value={date}
				required
			/>
		</div>
	</div>

	<div>
		<label for="transfer-description" class="text-sm font-medium">Description</label>
		<input
			id="transfer-description"
			name="description"
			type="text"
			maxlength="200"
			autocomplete="off"
			class="mt-1.5 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-sm outline-none placeholder:text-ink-muted focus:border-accent"
			placeholder="Move to savings"
			bind:value={description}
		/>
	</div>

	<div>
		<label for="transfer-notes" class="text-sm font-medium">Notes</label>
		<textarea
			id="transfer-notes"
			name="notes"
			rows="2"
			maxlength="2000"
			class="mt-1.5 w-full rounded-xl border border-line bg-surface px-3 py-2 text-sm outline-none placeholder:text-ink-muted focus:border-accent"
			placeholder="Optional context"
			bind:value={notes}></textarea>
	</div>

	<div class="flex flex-wrap items-center gap-3">
		<button
			type="submit"
			class="inline-flex min-h-11 items-center justify-center rounded-xl bg-accent px-5 text-sm font-semibold text-on-accent transition-colors hover:bg-accent-strong disabled:cursor-not-allowed disabled:opacity-60"
			disabled={busy || eligibleTo.length === 0}
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
