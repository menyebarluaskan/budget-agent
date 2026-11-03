<script lang="ts">
	import { resolve } from '$app/paths';
	import { afterNavigate, goto } from '$app/navigation';
	import { page } from '$app/state';
	import { SvelteURLSearchParams } from 'svelte/reactivity';
	import { listAccounts, listTags, listTransactions } from '$lib/api/client';
	import type {
		Account,
		Tag,
		Transaction,
		TransactionFilters,
		TransactionSort
	} from '$lib/api/types';
	import ButtonLink from '$lib/components/ButtonLink.svelte';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import Icon from '$lib/components/Icon.svelte';
	import PageHeader from '$lib/components/PageHeader.svelte';
	import { formatDate, formatSignedMoney } from '$lib/format';

	const sortOptions: { value: TransactionSort; label: string }[] = [
		{ value: '-occurred_at', label: 'Newest first' },
		{ value: 'occurred_at', label: 'Oldest first' },
		{ value: '-amount_minor', label: 'Largest amount' },
		{ value: 'amount_minor', label: 'Smallest amount' }
	];
	const sourceOptions = ['manual', 'whatsapp', 'mcp', 'receipt', 'import'];

	let accounts = $state<Account[]>([]);
	let tags = $state<Tag[]>([]);
	let lookupsLoaded = false;

	let transactions = $state<Transaction[]>([]);
	let nextCursor = $state('');
	let loading = $state(true);
	let loadingMore = $state(false);
	let error = $state<string | null>(null);
	let request = 0;

	let accountID = $state('');
	let tagID = $state('');
	let from = $state('');
	let to = $state('');
	let q = $state('');
	let source = $state('');
	let type = $state('');
	let includeTransfers = $state(false);
	let sort = $state<TransactionSort>('-occurred_at');

	const accountNames = $derived(new Map(accounts.map((account) => [account.id, account.name])));
	const accountCurrencies = $derived(
		new Map(accounts.map((account) => [account.id, account.currency]))
	);
	const tagLookup = $derived(new Map(tags.map((tag) => [tag.id, tag])));
	const hasActiveFilters = $derived.by(() => {
		const params = page.url.searchParams;
		return (
			['account', 'tag', 'from', 'to', 'q', 'source', 'type', 'transfers'].some((key) =>
				params.has(key)
			) ||
			(params.has('sort') && params.get('sort') !== '-occurred_at')
		);
	});

	function syncFromURL() {
		const params = page.url.searchParams;
		accountID = params.get('account') ?? '';
		tagID = params.get('tag') ?? '';
		from = params.get('from') ?? '';
		to = params.get('to') ?? '';
		q = params.get('q') ?? '';
		source = params.get('source') ?? '';
		type = params.get('type') ?? '';
		includeTransfers = params.get('transfers') === '1';
		const requestedSort = params.get('sort') as TransactionSort | null;
		sort = sortOptions.some((option) => option.value === requestedSort)
			? (requestedSort as TransactionSort)
			: '-occurred_at';
	}

	function currentFilters(): TransactionFilters {
		const filters: TransactionFilters = { sort, includeTransfers, limit: 50 };
		if (accountID) filters.accountId = accountID;
		if (tagID) filters.tagIds = [tagID];
		if (from) filters.from = from;
		if (to) filters.to = to;
		if (q.trim()) filters.q = q.trim();
		if (source) filters.source = source;
		if (type === 'spending') filters.maxAmountMinor = -1;
		if (type === 'income') filters.minAmountMinor = 1;
		return filters;
	}

	async function loadLookups() {
		if (lookupsLoaded) {
			return;
		}
		try {
			const [accountList, tagList] = await Promise.all([listAccounts(), listTags()]);
			accounts = accountList;
			tags = tagList;
			lookupsLoaded = true;
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'Accounts and tags could not be loaded.';
		}
	}

	async function loadFirstPage() {
		const current = ++request;
		loading = true;
		error = null;
		nextCursor = '';
		void loadLookups();
		try {
			const result = await listTransactions(currentFilters());
			if (current !== request) {
				return;
			}
			transactions = result.transactions;
			nextCursor = result.next_cursor ?? '';
		} catch (cause) {
			if (current === request) {
				error = cause instanceof Error ? cause.message : 'Transactions could not be loaded.';
			}
		} finally {
			if (current === request) {
				loading = false;
			}
		}
	}

	async function loadMore() {
		if (!nextCursor || loadingMore) {
			return;
		}
		const current = request;
		const cursor = nextCursor;
		loadingMore = true;
		try {
			const result = await listTransactions({ ...currentFilters(), cursor });
			if (current !== request) {
				return;
			}
			transactions = [...transactions, ...result.transactions];
			nextCursor = result.next_cursor ?? '';
		} catch (cause) {
			if (current === request) {
				error = cause instanceof Error ? cause.message : 'More transactions could not be loaded.';
			}
		} finally {
			if (current === request) {
				loadingMore = false;
			}
		}
	}

	function applyFilters(event: SubmitEvent) {
		event.preventDefault();
		const params = new SvelteURLSearchParams();
		if (accountID) params.set('account', accountID);
		if (tagID) params.set('tag', tagID);
		if (from) params.set('from', from);
		if (to) params.set('to', to);
		if (q.trim()) params.set('q', q.trim());
		if (source) params.set('source', source);
		if (type) params.set('type', type);
		if (includeTransfers) params.set('transfers', '1');
		if (sort !== '-occurred_at') params.set('sort', sort);
		const query = params.toString();
		if (query) {
			void goto(resolve(`/transactions?${query}`));
		} else {
			void goto(resolve('/transactions'));
		}
	}

	function clearFilters() {
		void goto(resolve('/transactions'));
	}

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

	afterNavigate(() => {
		syncFromURL();
		void loadFirstPage();
	});
</script>

<PageHeader title="Transactions" description="Search and filter spending, income, and transfers.">
	{#snippet actions()}
		<ButtonLink href="/transactions/new">Add transaction</ButtonLink>
	{/snippet}
</PageHeader>

<form
	class="mt-6 rounded-card border border-line bg-surface p-4 shadow-card sm:p-5"
	aria-label="Transaction filters"
	onsubmit={applyFilters}
>
	<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
		<div class="sm:col-span-2">
			<label for="filter-q" class="text-sm font-medium">Search</label>
			<input
				id="filter-q"
				type="search"
				class="mt-1.5 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-sm outline-none placeholder:text-ink-muted focus:border-accent"
				placeholder="Description or notes"
				bind:value={q}
			/>
		</div>

		<div>
			<label for="filter-account" class="text-sm font-medium">Account</label>
			<select
				id="filter-account"
				class="mt-1.5 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-sm outline-none focus:border-accent"
				bind:value={accountID}
			>
				<option value="">All accounts</option>
				{#each accounts as account (account.id)}
					<option value={account.id}>{account.name}</option>
				{/each}
			</select>
		</div>

		<div>
			<label for="filter-tag" class="text-sm font-medium">Tag</label>
			<select
				id="filter-tag"
				class="mt-1.5 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-sm outline-none focus:border-accent"
				bind:value={tagID}
			>
				<option value="">All tags</option>
				{#each tags as tag (tag.id)}
					<option value={tag.id}>{tag.name}</option>
				{/each}
			</select>
		</div>

		<div>
			<label for="filter-from" class="text-sm font-medium">From</label>
			<input
				id="filter-from"
				type="date"
				class="mt-1.5 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-sm outline-none focus:border-accent"
				bind:value={from}
			/>
		</div>

		<div>
			<label for="filter-to" class="text-sm font-medium">To</label>
			<input
				id="filter-to"
				type="date"
				class="mt-1.5 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-sm outline-none focus:border-accent"
				bind:value={to}
			/>
		</div>

		<div>
			<label for="filter-type" class="text-sm font-medium">Type</label>
			<select
				id="filter-type"
				class="mt-1.5 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-sm outline-none focus:border-accent"
				bind:value={type}
			>
				<option value="">All amounts</option>
				<option value="spending">Spending</option>
				<option value="income">Income</option>
			</select>
		</div>

		<div>
			<label for="filter-source" class="text-sm font-medium">Source</label>
			<select
				id="filter-source"
				class="mt-1.5 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-sm outline-none focus:border-accent"
				bind:value={source}
			>
				<option value="">All sources</option>
				{#each sourceOptions as option (option)}
					<option value={option}>{option}</option>
				{/each}
			</select>
		</div>

		<div>
			<label for="filter-sort" class="text-sm font-medium">Sort</label>
			<select
				id="filter-sort"
				class="mt-1.5 min-h-11 w-full rounded-xl border border-line bg-surface px-3 text-sm outline-none focus:border-accent"
				bind:value={sort}
			>
				{#each sortOptions as option (option.value)}
					<option value={option.value}>{option.label}</option>
				{/each}
			</select>
		</div>

		<label class="flex min-h-11 items-center gap-2 self-end text-sm font-medium">
			<input
				type="checkbox"
				class="size-4 accent-[var(--token-accent)]"
				bind:checked={includeTransfers}
			/>
			Include transfers
		</label>
	</div>

	<div class="mt-4 flex flex-wrap items-center gap-2">
		<button
			type="submit"
			class="inline-flex min-h-11 items-center justify-center rounded-xl bg-accent px-5 text-sm font-semibold text-on-accent transition-colors hover:bg-accent-strong"
		>
			Apply filters
		</button>
		{#if hasActiveFilters}
			<button
				type="button"
				class="inline-flex min-h-11 items-center justify-center rounded-xl border border-line bg-surface px-4 text-sm font-semibold text-ink transition-colors hover:bg-surface-muted"
				onclick={clearFilters}
			>
				Clear
			</button>
		{/if}
	</div>
</form>

<p class="sr-only" role="status">
	{loading ? 'Loading transactions' : `${transactions.length} transactions shown`}
</p>

{#if error}
	<div role="alert" class="mt-6 rounded-card border border-negative/40 bg-negative/5 p-5">
		<h2 class="text-sm font-semibold text-negative">Couldn't load transactions</h2>
		<p class="mt-1 text-sm text-ink-muted">{error}</p>
		<button
			type="button"
			class="mt-4 inline-flex min-h-10 items-center justify-center rounded-xl bg-accent px-4 text-sm font-semibold text-on-accent transition-colors hover:bg-accent-strong"
			onclick={() => void loadFirstPage()}
		>
			Try again
		</button>
	</div>
{/if}

{#if loading && transactions.length === 0}
	<div class="mt-6 space-y-3" aria-hidden="true">
		<div class="h-20 animate-pulse rounded-card border border-line bg-surface"></div>
		<div class="h-20 animate-pulse rounded-card border border-line bg-surface"></div>
		<div class="h-20 animate-pulse rounded-card border border-line bg-surface"></div>
	</div>
{:else if !error && transactions.length === 0}
	<div class="mt-6">
		<EmptyState
			icon="transactions"
			title={hasActiveFilters ? 'No matching transactions' : 'No transactions yet'}
			description={hasActiveFilters
				? 'Try a wider date range, another account or tag, or clear the filters.'
				: 'Record spending and income to see them here, grouped by the date they happened.'}
		>
			{#snippet actions()}
				{#if hasActiveFilters}
					<button
						type="button"
						class="inline-flex min-h-11 items-center justify-center rounded-xl border border-line bg-surface px-4 text-sm font-semibold text-ink transition-colors hover:bg-surface-muted"
						onclick={clearFilters}
					>
						Clear filters
					</button>
				{/if}
				<ButtonLink href="/transactions/new">Add transaction</ButtonLink>
			{/snippet}
		</EmptyState>
	</div>
{:else if transactions.length > 0}
	<section class="mt-6" aria-labelledby="transaction-results">
		<div class="mb-3 flex flex-wrap items-baseline justify-between gap-2">
			<h2 id="transaction-results" class="text-base font-semibold tracking-tight">
				{hasActiveFilters ? 'Filtered transactions' : 'All transactions'}
			</h2>
			<p class="text-xs text-ink-muted">{transactions.length} shown</p>
		</div>

		<div class="overflow-hidden rounded-card border border-line bg-surface shadow-card">
			<ul class="divide-y divide-line">
				{#each transactions as transaction (transaction.id)}
					<li>
						<a
							href={resolve('/transactions/[id]', { id: transaction.id })}
							class="flex items-center justify-between gap-3 px-4 py-3 transition-colors hover:bg-surface-muted sm:px-5"
						>
							<span class="min-w-0">
								<span class="block truncate text-sm font-medium">
									{transaction.description || 'Untitled transaction'}
								</span>
								<span
									class="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-ink-muted"
								>
									<span>{formatDate(transaction.occurred_at)}</span>
									{#if accountNames.get(transaction.account_id)}
										<span aria-hidden="true">·</span>
										<span>{accountNames.get(transaction.account_id)}</span>
									{/if}
									{#if transaction.transfer_id}
										<span
											class="rounded-full bg-surface-muted px-2 py-0.5 text-[0.65rem] font-medium tracking-wide uppercase"
										>
											Transfer
										</span>
									{/if}
									{#if transaction.source !== 'manual'}
										<span
											class="rounded-full bg-surface-muted px-2 py-0.5 text-[0.65rem] font-medium tracking-wide uppercase"
										>
											{transaction.source}
										</span>
									{/if}
									{#each transaction.tag_ids as tagID (tagID)}
										{@const tag = tagLookup.get(tagID)}
										{#if tag}
											<span class="inline-flex items-center gap-1">
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
							</span>
							<span class="flex shrink-0 items-center gap-2">
								<span class="text-sm font-semibold tabular-nums {amountClass(transaction)}">
									{formatSignedMoney(
										transaction.amount_minor,
										accountCurrencies.get(transaction.account_id) ?? 'IDR'
									)}
								</span>
								<Icon name="chevron-right" class="size-4 text-ink-muted" />
							</span>
						</a>
					</li>
				{/each}
			</ul>
		</div>

		{#if nextCursor}
			<div class="mt-4 flex justify-center">
				<button
					type="button"
					class="inline-flex min-h-11 items-center justify-center rounded-xl border border-line bg-surface px-5 text-sm font-semibold text-ink transition-colors hover:bg-surface-muted disabled:cursor-wait disabled:opacity-60"
					disabled={loadingMore}
					onclick={() => void loadMore()}
				>
					{loadingMore ? 'Loading…' : 'Load more'}
				</button>
			</div>
		{/if}
	</section>
{/if}
