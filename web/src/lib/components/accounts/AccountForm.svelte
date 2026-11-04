<script lang="ts">
	import { untrack } from 'svelte';
	import type { Account, AccountInput, AccountType, AccountUpdate } from '$lib/api/types';
	import {
		currencyFractionDigits,
		formatMinorForInput,
		formatMoney,
		parseMoneyToMinor
	} from '$lib/format';

	let {
		account = null,
		submitLabel,
		busy = false,
		error = null,
		onCancel,
		onSubmit
	}: {
		account?: Account | null;
		submitLabel: string;
		busy?: boolean;
		error?: string | null;
		onCancel?: () => void;
		onSubmit: (input: AccountInput | AccountUpdate) => void | Promise<void>;
	} = $props();

	const editing = account !== null;
	const initial = untrack(() => {
		const currency = account?.currency ?? 'IDR';
		const digits = currencyFractionDigits(currency);
		return {
			name: account?.name ?? '',
			type: (account?.type ?? 'wallet') as AccountType,
			currency,
			openingBalance: account ? formatMinorForInput(account.opening_balance_minor, digits) : '0',
			archived: account?.archived ?? false
		};
	});

	let name = $state(initial.name);
	let type = $state<AccountType>(initial.type);
	let currency = $state(initial.currency);
	let openingBalance = $state(initial.openingBalance);
	let archived = $state(initial.archived);
	let localError = $state<string | null>(null);

	const normalizedCurrency = $derived(currency.trim().toUpperCase());
	const digits = $derived(
		normalizedCurrency.length === 3 ? currencyFractionDigits(normalizedCurrency) : 2
	);
	const parsedBalance = $derived(parseMoneyToMinor(openingBalance || '0', digits));
	const preview = $derived(
		parsedBalance === null ? null : formatMoney(parsedBalance, normalizedCurrency || 'IDR')
	);
	const visibleError = $derived(localError ?? error);

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		localError = null;
		if (name.trim() === '') {
			localError = 'Enter an account name.';
			return;
		}
		if (normalizedCurrency.length !== 3) {
			localError = 'Use a three-letter ISO currency code, for example IDR.';
			return;
		}
		if (parsedBalance === null) {
			localError = `Enter an opening balance with at most ${digits} decimal places.`;
			return;
		}

		if (account) {
			await onSubmit({
				name: name.trim(),
				archived,
				opening_balance_minor: parsedBalance
			});
			return;
		}
		await onSubmit({
			name: name.trim(),
			type,
			currency: normalizedCurrency,
			opening_balance_minor: parsedBalance
		});
	}
</script>

<form class="space-y-5" onsubmit={submit} novalidate>
	{#if visibleError}
		<div role="alert" class="rounded-card border border-negative/40 bg-negative/5 p-4">
			<p class="text-sm font-semibold text-negative">Couldn't save the account</p>
			<p class="mt-1 text-sm text-ink-muted">{visibleError}</p>
		</div>
	{/if}

	<div>
		<label for="account-name" class="text-sm font-medium">Name</label>
		<input
			id="account-name"
			name="name"
			type="text"
			maxlength="100"
			autocomplete="off"
			class="mt-1.5 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-sm outline-none placeholder:text-ink-muted focus:border-accent"
			placeholder="Everyday wallet"
			bind:value={name}
			required
		/>
	</div>

	<div class="grid gap-4 sm:grid-cols-2">
		<div>
			<label for="account-type" class="text-sm font-medium">Type</label>
			<select
				id="account-type"
				name="type"
				class="mt-1.5 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-sm outline-none focus:border-accent disabled:opacity-60"
				bind:value={type}
				disabled={editing}
				required
			>
				<option value="wallet">Wallet</option>
				<option value="savings">Savings</option>
				<option value="credit">Credit</option>
			</select>
			{#if editing}
				<p class="mt-1 text-xs text-ink-muted">The account type cannot be changed.</p>
			{/if}
		</div>

		<div>
			<label for="account-currency" class="text-sm font-medium">Currency</label>
			<input
				id="account-currency"
				name="currency"
				type="text"
				maxlength="3"
				autocomplete="off"
				class="mt-1.5 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-sm uppercase outline-none placeholder:text-ink-muted focus:border-accent disabled:opacity-60"
				placeholder="IDR"
				bind:value={currency}
				disabled={editing}
				required
			/>
		</div>
	</div>

	<div>
		<label for="account-opening" class="text-sm font-medium">Opening balance</label>
		<input
			id="account-opening"
			name="opening_balance"
			type="text"
			inputmode="decimal"
			autocomplete="off"
			class="mt-1.5 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-base tabular-nums outline-none placeholder:text-ink-muted focus:border-accent"
			placeholder="0"
			bind:value={openingBalance}
		/>
		<p class="mt-1.5 min-h-5 text-xs text-ink-muted" aria-live="polite">
			{#if preview}
				{preview}
			{:else}
				Use a leading minus for existing credit-card debt.
			{/if}
		</p>
	</div>

	{#if editing}
		<label class="flex items-center gap-2 text-sm font-medium">
			<input type="checkbox" class="size-4 accent-[var(--token-accent)]" bind:checked={archived} />
			Archived (hidden from new transactions and totals)
		</label>
	{/if}

	<div class="flex flex-wrap items-center gap-3">
		<button
			type="submit"
			class="inline-flex min-h-11 items-center justify-center rounded-xl bg-accent px-5 text-sm font-semibold text-on-accent transition-colors hover:bg-accent-strong disabled:cursor-not-allowed disabled:opacity-60"
			disabled={busy}
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
