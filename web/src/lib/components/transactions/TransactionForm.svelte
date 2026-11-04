<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import type { Account, Tag, Transaction, TransactionInput } from '$lib/api/types';
	import { untrack } from 'svelte';
	import TagPicker from '$lib/components/tags/TagPicker.svelte';
	import {
		currencyFractionDigits,
		dateOnlyFromTimestamp,
		formatMinorForInput,
		formatSignedMoney,
		occurredAtFromDate,
		parseMoneyToMinor,
		todayISODate
	} from '$lib/format';

	let {
		accounts,
		tags,
		transaction = null,
		submitLabel,
		busy = false,
		error = null,
		cancelLabel = 'Cancel',
		onCancel,
		onSubmit,
		onTagsChange
	}: {
		accounts: Account[];
		tags: Tag[];
		transaction?: Transaction | null;
		submitLabel: string;
		busy?: boolean;
		error?: string | null;
		cancelLabel?: string;
		onCancel?: () => void;
		onSubmit: (input: TransactionInput) => void | Promise<void>;
		onTagsChange?: (tags: Tag[]) => void;
	} = $props();

	const accountOptions = $derived(
		accounts.filter((account) => !account.archived || account.id === transaction?.account_id)
	);
	const initial = untrack(() => {
		const initialDigits = currencyFractionDigits(
			accounts.find((account) => account.id === transaction?.account_id)?.currency ?? 'IDR'
		);
		return {
			accountID: transaction?.account_id ?? accountOptions[0]?.id ?? '',
			amount: transaction ? formatMinorForInput(transaction.amount_minor, initialDigits) : '',
			date: transaction ? dateOnlyFromTimestamp(transaction.occurred_at) : todayISODate(),
			description: transaction?.description ?? '',
			notes: transaction?.notes ?? '',
			tagIDs: transaction ? [...transaction.tag_ids] : []
		};
	});

	let accountID = $state(initial.accountID);
	let amount = $state(initial.amount);
	let date = $state(initial.date);
	let description = $state(initial.description);
	let notes = $state(initial.notes);
	let selectedTagIDs = $state<string[]>(initial.tagIDs);
	let tagOptions = $derived(tags);
	let localError = $state<string | null>(null);

	const selectedAccount = $derived(accounts.find((account) => account.id === accountID));
	const currency = $derived(selectedAccount?.currency ?? 'IDR');
	const digits = $derived(currencyFractionDigits(currency));
	const parsedAmount = $derived(parseMoneyToMinor(amount, digits));
	const amountPreview = $derived(
		parsedAmount === null ? null : formatSignedMoney(parsedAmount, currency)
	);
	const visibleError = $derived(localError ?? error);

	function updateSelectedTags(ids: string[]) {
		selectedTagIDs = ids;
	}

	function updateTags(next: Tag[]) {
		tagOptions = next;
		onTagsChange?.(next);
	}

	function addSign(sign: '+' | '-') {
		const value = amount.trim();
		if (value.startsWith('+') || value.startsWith('-')) {
			amount = sign + value.slice(1);
		} else if (value !== '') {
			amount = sign + value;
		}
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		localError = null;
		if (!accountID) {
			localError = 'Choose an account before saving.';
			return;
		}
		if (parsedAmount === null) {
			localError = `Enter an amount with at most ${digits} decimal places.`;
			return;
		}
		if (parsedAmount === 0) {
			localError = 'Enter an amount other than zero.';
			return;
		}
		if (!date) {
			localError = 'Choose a date.';
			return;
		}

		await onSubmit({
			account_id: accountID,
			amount_minor: parsedAmount,
			occurred_at: occurredAtFromDate(date),
			description: description.trim(),
			notes: notes.trim(),
			tag_ids: selectedTagIDs
		});
	}

	function cancel() {
		if (onCancel) {
			onCancel();
			return;
		}
		void goto(resolve('/transactions'));
	}
</script>

<form class="space-y-6" onsubmit={submit} novalidate>
	{#if visibleError}
		<div role="alert" class="rounded-card border border-negative/40 bg-negative/5 p-4">
			<p class="text-sm font-semibold text-negative">Couldn't save the transaction</p>
			<p class="mt-1 text-sm text-ink-muted">{visibleError}</p>
		</div>
	{/if}

	<section class="rounded-card border border-line bg-surface p-5 shadow-card sm:p-6">
		<h2 class="text-base font-semibold tracking-tight">Amount and account</h2>
		<p class="mt-1 text-sm text-ink-muted">
			Use a leading <span class="font-mono">-</span> for spending and
			<span class="font-mono">+</span> for income.
		</p>

		<div class="mt-4 grid gap-4 sm:grid-cols-2">
			<div class="sm:col-span-2">
				<label for="transaction-amount" class="text-sm font-medium">Amount</label>
				<div class="mt-1.5 flex gap-2">
					<input
						id="transaction-amount"
						name="amount"
						type="text"
						inputmode="decimal"
						autocomplete="off"
						class="min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-base tabular-nums outline-none placeholder:text-ink-muted focus:border-accent"
						placeholder={digits === 0 ? '-25000' : '-12.50'}
						bind:value={amount}
						required
					/>
					<button
						type="button"
						class="min-h-11 shrink-0 rounded-xl border border-line bg-surface px-3 text-sm font-semibold text-negative transition-colors hover:bg-surface-muted"
						onclick={() => addSign('-')}
					>
						Expense
					</button>
					<button
						type="button"
						class="min-h-11 shrink-0 rounded-xl border border-line bg-surface px-3 text-sm font-semibold text-positive transition-colors hover:bg-surface-muted"
						onclick={() => addSign('+')}
					>
						Income
					</button>
				</div>
				<p class="mt-1.5 min-h-5 text-xs text-ink-muted" aria-live="polite">
					{#if amountPreview}
						{amountPreview}
					{:else}
						{currency} · {digits} decimal {digits === 1 ? 'place' : 'places'}
					{/if}
				</p>
			</div>

			<div>
				<label for="transaction-account" class="text-sm font-medium">Account</label>
				<select
					id="transaction-account"
					name="account_id"
					class="mt-1.5 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-sm outline-none focus:border-accent"
					bind:value={accountID}
					required
				>
					{#if accountOptions.length === 0}
						<option value="">No accounts available</option>
					{/if}
					{#each accountOptions as account (account.id)}
						<option value={account.id}>
							{account.name}{account.archived ? ' (archived)' : ''} · {account.currency}
						</option>
					{/each}
				</select>
			</div>

			<div>
				<label for="transaction-date" class="text-sm font-medium">Date</label>
				<input
					id="transaction-date"
					name="occurred_at"
					type="date"
					class="mt-1.5 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-sm outline-none focus:border-accent"
					bind:value={date}
					required
				/>
			</div>
		</div>
	</section>

	<section class="rounded-card border border-line bg-surface p-5 shadow-card sm:p-6">
		<h2 class="text-base font-semibold tracking-tight">Details</h2>

		<div class="mt-4 space-y-4">
			<div>
				<label for="transaction-description" class="text-sm font-medium">Description</label>
				<input
					id="transaction-description"
					name="description"
					type="text"
					maxlength="200"
					autocomplete="off"
					class="mt-1.5 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-sm outline-none placeholder:text-ink-muted focus:border-accent"
					placeholder="Coffee, groceries, salary…"
					bind:value={description}
				/>
			</div>

			<div>
				<label for="transaction-notes" class="text-sm font-medium">Notes</label>
				<textarea
					id="transaction-notes"
					name="notes"
					rows="3"
					maxlength="2000"
					class="mt-1.5 w-full rounded-xl border border-line bg-surface px-3 py-2 text-sm outline-none placeholder:text-ink-muted focus:border-accent"
					placeholder="Optional context"
					bind:value={notes}></textarea>
			</div>

			<TagPicker
				tags={tagOptions}
				selectedIDs={selectedTagIDs}
				onSelectedChange={updateSelectedTags}
				onTagsChange={updateTags}
			/>
		</div>
	</section>

	<div class="flex flex-wrap items-center gap-3">
		<button
			type="submit"
			class="inline-flex min-h-11 items-center justify-center rounded-xl bg-accent px-5 text-sm font-semibold text-on-accent transition-colors hover:bg-accent-strong disabled:cursor-not-allowed disabled:opacity-60"
			disabled={busy || accountOptions.length === 0}
		>
			{busy ? 'Saving…' : submitLabel}
		</button>
		<button
			type="button"
			class="inline-flex min-h-11 items-center justify-center rounded-xl border border-line bg-surface px-4 text-sm font-semibold text-ink transition-colors hover:bg-surface-muted"
			onclick={cancel}
		>
			{cancelLabel}
		</button>
	</div>
</form>
