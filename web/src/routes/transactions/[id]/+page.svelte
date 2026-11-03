<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import {
		ApiError,
		deleteTransaction as removeTransaction,
		getTransaction,
		listAccounts,
		listTags,
		updateTransaction
	} from '$lib/api/client';
	import type { Account, Tag, Transaction, TransactionInput } from '$lib/api/types';
	import ButtonLink from '$lib/components/ButtonLink.svelte';
	import Icon from '$lib/components/Icon.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import TransactionForm from '$lib/components/transactions/TransactionForm.svelte';
	import { formatDate, formatSignedMoney } from '$lib/format';

	let transaction = $state<Transaction | null>(null);
	let accounts = $state<Account[]>([]);
	let tags = $state<Tag[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let actionError = $state<string | null>(null);
	let notFound = $state(false);
	let editing = $state(false);
	let saving = $state(false);
	let deleting = $state(false);

	const account = $derived(
		transaction ? accounts.find((item) => item.id === transaction?.account_id) : undefined
	);
	const currency = $derived(account?.currency ?? 'IDR');
	const tagLookup = $derived(new Map(tags.map((tag) => [tag.id, tag])));
	const transactionTags = $derived<Tag[]>(
		transaction
			? transaction.tag_ids
					.map((id) => tagLookup.get(id))
					.filter((tag): tag is Tag => tag !== undefined)
			: []
	);

	async function load(id: string) {
		loading = true;
		error = null;
		notFound = false;
		editing = false;
		try {
			const result = await getTransaction(id);
			transaction = result;
			try {
				const [accountList, tagList] = await Promise.all([listAccounts(), listTags()]);
				accounts = accountList;
				tags = tagList;
			} catch {
				// The transaction is still useful without lookup labels.
			}
		} catch (cause) {
			if (cause instanceof ApiError && cause.status === 404) {
				notFound = true;
			} else {
				error = cause instanceof Error ? cause.message : 'The transaction could not be loaded.';
			}
		} finally {
			loading = false;
		}
	}

	async function save(input: TransactionInput) {
		if (!transaction) {
			return;
		}
		saving = true;
		actionError = null;
		try {
			transaction = await updateTransaction(transaction.id, {
				account_id: input.account_id,
				amount_minor: input.amount_minor,
				occurred_at: input.occurred_at,
				description: input.description,
				notes: input.notes,
				tag_ids: input.tag_ids
			});
			editing = false;
		} catch (cause) {
			actionError = cause instanceof Error ? cause.message : 'The transaction could not be saved.';
		} finally {
			saving = false;
		}
	}

	async function remove() {
		if (!transaction) {
			return;
		}
		if (!window.confirm('Delete this transaction? This cannot be undone.')) {
			return;
		}
		deleting = true;
		actionError = null;
		try {
			await removeTransaction(transaction.id);
			await goto(resolve('/transactions'));
		} catch (cause) {
			actionError =
				cause instanceof Error ? cause.message : 'The transaction could not be deleted.';
		} finally {
			deleting = false;
		}
	}

	$effect(() => {
		const id = page.params.id;
		if (id) {
			void load(id);
		}
	});
</script>

<div class="mb-4">
	<a
		href={resolve('/transactions')}
		class="inline-flex items-center gap-1 text-sm font-medium text-ink-muted transition-colors hover:text-ink"
	>
		<Icon name="chevron-left" class="size-4" />
		Back to transactions
	</a>
</div>

{#if loading}
	<PageHeader title="Transaction" />
	<div class="mt-6 space-y-4" aria-hidden="true">
		<div class="h-40 animate-pulse rounded-card border border-line bg-surface"></div>
		<div class="h-56 animate-pulse rounded-card border border-line bg-surface"></div>
	</div>
{:else if notFound}
	<PageHeader title="Transaction not found" description="It may have been deleted already." />
	<div class="mt-6">
		<ButtonLink href="/transactions">Back to transactions</ButtonLink>
	</div>
{:else if error}
	<PageHeader title="Transaction" />
	<div role="alert" class="mt-6 rounded-card border border-negative/40 bg-negative/5 p-5">
		<h2 class="text-sm font-semibold text-negative">Couldn't load the transaction</h2>
		<p class="mt-1 text-sm text-ink-muted">{error}</p>
		<button
			type="button"
			class="mt-4 inline-flex min-h-10 items-center justify-center rounded-xl bg-accent px-4 text-sm font-semibold text-on-accent transition-colors hover:bg-accent-strong"
			onclick={() => void load(page.params.id ?? '')}
		>
			Try again
		</button>
	</div>
{:else if transaction}
	<PageHeader
		title={transaction.description || 'Untitled transaction'}
		description={formatDate(transaction.occurred_at)}
	>
		{#snippet actions()}
			{#if !editing}
				<button
					type="button"
					class="inline-flex min-h-11 items-center justify-center rounded-xl bg-accent px-4 text-sm font-semibold text-on-accent transition-colors hover:bg-accent-strong"
					onclick={() => {
						editing = true;
						actionError = null;
					}}
				>
					Edit
				</button>
			{/if}
		{/snippet}
	</PageHeader>

	{#if actionError}
		<div role="alert" class="mt-4 rounded-card border border-negative/40 bg-negative/5 p-4">
			<p class="text-sm text-ink-muted">{actionError}</p>
		</div>
	{/if}

	{#if editing}
		<div class="mt-6">
			<TransactionForm
				{accounts}
				{tags}
				{transaction}
				submitLabel="Save changes"
				cancelLabel="Stop editing"
				onCancel={() => {
					editing = false;
				}}
				busy={saving}
				error={actionError}
				onSubmit={save}
			/>
		</div>
	{:else}
		<section class="mt-6 rounded-card border border-line bg-surface p-6 shadow-card">
			<p class="text-sm font-medium text-ink-muted">Amount</p>
			<p
				class="mt-1 text-3xl font-semibold tracking-tight tabular-nums sm:text-4xl {transaction.amount_minor >
				0
					? 'text-positive'
					: transaction.amount_minor < 0
						? 'text-negative'
						: 'text-ink-muted'}"
			>
				{formatSignedMoney(transaction.amount_minor, currency)}
			</p>
			<div class="mt-4 flex flex-wrap items-center gap-2 text-xs text-ink-muted">
				{#if transaction.transfer_id}
					<span
						class="rounded-full bg-surface-muted px-2 py-1 text-[0.65rem] font-medium tracking-wide uppercase"
					>
						Transfer
					</span>
				{/if}
				<span
					class="rounded-full bg-surface-muted px-2 py-1 text-[0.65rem] font-medium tracking-wide uppercase"
				>
					{transaction.source}
				</span>
			</div>
		</section>

		<section class="mt-6 rounded-card border border-line bg-surface p-6 shadow-card">
			<h2 class="text-base font-semibold tracking-tight">Details</h2>
			<dl class="mt-4 grid gap-x-6 gap-y-4 text-sm sm:grid-cols-2">
				<div>
					<dt class="text-ink-muted">Account</dt>
					<dd class="mt-0.5 font-medium">{account?.name ?? transaction.account_id}</dd>
				</div>
				<div>
					<dt class="text-ink-muted">Date</dt>
					<dd class="mt-0.5 font-medium">{formatDate(transaction.occurred_at)}</dd>
				</div>
				<div>
					<dt class="text-ink-muted">Source</dt>
					<dd class="mt-0.5 font-medium capitalize">{transaction.source}</dd>
				</div>
				<div>
					<dt class="text-ink-muted">Updated</dt>
					<dd class="mt-0.5 font-medium">{formatDate(transaction.updated_at)}</dd>
				</div>
				<div class="sm:col-span-2">
					<dt class="text-ink-muted">Tags</dt>
					<dd class="mt-1">
						{#if transactionTags.length > 0}
							<span class="flex flex-wrap gap-2">
								{#each transactionTags as tag (tag?.id)}
									{#if tag}
										<span
											class="inline-flex items-center gap-1.5 rounded-full border border-line px-2.5 py-1 text-xs"
										>
											<span
												class="size-2 rounded-full"
												style:background-color={tag.color || 'var(--token-accent)'}
												aria-hidden="true"
											></span>
											{tag.name}
										</span>
									{/if}
								{/each}
							</span>
						{:else}
							<span class="text-ink-muted">No tags</span>
						{/if}
					</dd>
				</div>
				{#if transaction.notes}
					<div class="sm:col-span-2">
						<dt class="text-ink-muted">Notes</dt>
						<dd class="mt-0.5 whitespace-pre-wrap">{transaction.notes}</dd>
					</div>
				{/if}
			</dl>
		</section>

		<section class="mt-6 rounded-card border border-dashed border-line bg-surface p-6">
			<h2 class="text-base font-semibold tracking-tight">Receipt</h2>
			<p class="mt-1 text-sm text-ink-muted">
				No receipt attached. Receipt upload and OCR arrive with the receipt flow.
			</p>
		</section>

		<div class="mt-6 flex flex-wrap items-center justify-between gap-3">
			<ButtonLink href="/transactions" variant="secondary">Back to transactions</ButtonLink>
			<button
				type="button"
				class="inline-flex min-h-11 items-center justify-center rounded-xl border border-negative/40 bg-surface px-4 text-sm font-semibold text-negative transition-colors hover:bg-negative/5 disabled:cursor-wait disabled:opacity-60"
				disabled={deleting}
				onclick={() => void remove()}
			>
				{deleting ? 'Deleting…' : 'Delete transaction'}
			</button>
		</div>
	{/if}
{:else}
	<p class="text-sm text-ink-muted">No transaction selected.</p>
{/if}
