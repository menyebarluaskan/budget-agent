import type {
	Account,
	AccountInput,
	AccountUpdate,
	Budget,
	BudgetInput,
	BudgetSummary,
	BudgetTrend,
	BudgetUpdate,
	Dashboard,
	DateRange,
	SpendingAnalytics,
	Tag,
	TagInput,
	TagUpdate,
	Transaction,
	TransactionFilters,
	TransactionInput,
	TransactionPage,
	TransactionUpdate,
	Transfer,
	TransferInput
} from './types';

export class ApiError extends Error {
	readonly status: number;

	constructor(status: number, message: string) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
	}
}

type Fetcher = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;

async function requestJSON<T>(path: string, fetcher: Fetcher, init?: RequestInit): Promise<T> {
	const response = await fetcher(path, {
		...init,
		headers: {
			Accept: 'application/json',
			...(init?.body ? { 'Content-Type': 'application/json' } : {}),
			...init?.headers
		}
	});
	if (!response.ok) {
		throw new ApiError(response.status, await errorMessage(response));
	}
	return (await response.json()) as T;
}

async function requestVoid(path: string, fetcher: Fetcher, init?: RequestInit): Promise<void> {
	const response = await fetcher(path, {
		...init,
		headers: { Accept: 'application/json', ...init?.headers }
	});
	if (!response.ok) {
		throw new ApiError(response.status, await errorMessage(response));
	}
}

async function errorMessage(response: Response): Promise<string> {
	try {
		const body = (await response.json()) as { detail?: unknown; title?: unknown };
		if (typeof body.detail === 'string' && body.detail !== '') {
			return body.detail;
		}
		if (typeof body.title === 'string' && body.title !== '') {
			return body.title;
		}
	} catch {
		// The response was not JSON; fall through to the status text.
	}
	return `Request failed with status ${response.status}`;
}

function unwrapList<T>(body: Record<string, T[] | undefined> | T[], key: string): T[] {
	return Array.isArray(body) ? body : (body[key] ?? []);
}

/** Fetch the dashboard read model for an inclusive date range. */
export function getDashboard(range: DateRange, fetcher: Fetcher = fetch): Promise<Dashboard> {
	const params = new URLSearchParams({ from: range.from, to: range.to });
	return requestJSON<Dashboard>(`/api/v1/dashboard?${params.toString()}`, fetcher);
}

/** Fetch spending by tag and the monthly trend for the chart panels. */
export function getSpendingAnalytics(
	range: DateRange,
	months = 6,
	fetcher: Fetcher = fetch
): Promise<SpendingAnalytics> {
	const params = new URLSearchParams({
		from: range.from,
		to: range.to,
		months: String(months)
	});
	return requestJSON<SpendingAnalytics>(`/api/v1/analytics/spending?${params.toString()}`, fetcher);
}

/** Fetch accounts with their current balances. */
export function listAccounts(fetcher: Fetcher = fetch): Promise<Account[]> {
	return requestJSON<{ accounts?: Account[] } | Account[]>('/api/v1/accounts', fetcher).then(
		(body) => unwrapList<Account>(body, 'accounts')
	);
}

/** Create a wallet, savings, or credit account. */
export function createAccount(input: AccountInput, fetcher: Fetcher = fetch): Promise<Account> {
	return requestJSON<Account>('/api/v1/accounts', fetcher, {
		method: 'POST',
		body: JSON.stringify(input)
	});
}

/** Update an account name, archived flag, or opening balance. */
export function updateAccount(
	id: string,
	input: AccountUpdate,
	fetcher: Fetcher = fetch
): Promise<Account> {
	return requestJSON<Account>(`/api/v1/accounts/${encodeURIComponent(id)}`, fetcher, {
		method: 'PATCH',
		body: JSON.stringify(input)
	});
}

/** Delete an account and, by cascade, its transactions. */
export function deleteAccount(id: string, fetcher: Fetcher = fetch): Promise<void> {
	return requestVoid(`/api/v1/accounts/${encodeURIComponent(id)}`, fetcher, { method: 'DELETE' });
}

/** Fetch every tag in the workspace. */
export function listTags(fetcher: Fetcher = fetch): Promise<Tag[]> {
	return requestJSON<{ tags?: Tag[] } | Tag[]>('/api/v1/tags', fetcher).then((body) =>
		unwrapList<Tag>(body, 'tags')
	);
}

/** Create a tag with a unique name and a hex color. */
export function createTag(input: TagInput, fetcher: Fetcher = fetch): Promise<Tag> {
	return requestJSON<Tag>('/api/v1/tags', fetcher, {
		method: 'POST',
		body: JSON.stringify(input)
	});
}

/** Rename or recolor a tag. */
export function updateTag(id: string, input: TagUpdate, fetcher: Fetcher = fetch): Promise<Tag> {
	return requestJSON<Tag>(`/api/v1/tags/${encodeURIComponent(id)}`, fetcher, {
		method: 'PATCH',
		body: JSON.stringify(input)
	});
}

/** Fetch one cursor-paginated transaction page. */
export function listTransactions(
	filters: TransactionFilters = {},
	fetcher: Fetcher = fetch
): Promise<TransactionPage> {
	const params = new URLSearchParams();
	if (filters.accountId) params.set('account_id', filters.accountId);
	for (const tagID of filters.tagIds ?? []) {
		params.append('tag', tagID);
	}
	if (filters.from) params.set('from', filters.from);
	if (filters.to) params.set('to', filters.to);
	if (filters.minAmountMinor !== undefined) {
		params.set('min_amount_minor', String(filters.minAmountMinor));
	}
	if (filters.maxAmountMinor !== undefined) {
		params.set('max_amount_minor', String(filters.maxAmountMinor));
	}
	if (filters.q) params.set('q', filters.q);
	if (filters.source) params.set('source', filters.source);
	if (filters.includeTransfers) params.set('include_transfers', 'true');
	if (filters.sort) params.set('sort', filters.sort);
	if (filters.limit !== undefined) params.set('limit', String(filters.limit));
	if (filters.cursor) params.set('cursor', filters.cursor);

	const query = params.toString();
	return requestJSON<TransactionPage>(`/api/v1/transactions${query ? `?${query}` : ''}`, fetcher);
}

/** Fetch a single transaction with its tags. */
export function getTransaction(id: string, fetcher: Fetcher = fetch): Promise<Transaction> {
	return requestJSON<Transaction>(`/api/v1/transactions/${encodeURIComponent(id)}`, fetcher);
}

/** Create a manual transaction. */
export function createTransaction(
	input: TransactionInput,
	fetcher: Fetcher = fetch
): Promise<Transaction> {
	return requestJSON<Transaction>('/api/v1/transactions', fetcher, {
		method: 'POST',
		body: JSON.stringify(input)
	});
}

/** Update an existing transaction, leaving omitted fields unchanged. */
export function updateTransaction(
	id: string,
	input: TransactionUpdate,
	fetcher: Fetcher = fetch
): Promise<Transaction> {
	return requestJSON<Transaction>(`/api/v1/transactions/${encodeURIComponent(id)}`, fetcher, {
		method: 'PATCH',
		body: JSON.stringify(input)
	});
}

/** Delete a transaction. */
export function deleteTransaction(id: string, fetcher: Fetcher = fetch): Promise<void> {
	return requestVoid(`/api/v1/transactions/${encodeURIComponent(id)}`, fetcher, {
		method: 'DELETE'
	});
}

/** Move money between two same-currency accounts. */
export function createTransfer(input: TransferInput, fetcher: Fetcher = fetch): Promise<Transfer> {
	return requestJSON<Transfer>('/api/v1/transfers', fetcher, {
		method: 'POST',
		body: JSON.stringify(input)
	});
}

/** Fetch a single account. */
export function getAccount(id: string, fetcher: Fetcher = fetch): Promise<Account> {
	return requestJSON<Account>(`/api/v1/accounts/${encodeURIComponent(id)}`, fetcher);
}

/** Fetch a budget summary (spent per tag). */
export function getBudgetSummary(id: string, fetcher: Fetcher = fetch): Promise<BudgetSummary[]> {
	return requestJSON<{ summary?: BudgetSummary[] } | BudgetSummary[]>(
		`/api/v1/budgets/${encodeURIComponent(id)}/summary`,
		fetcher
	).then((body) => unwrapList<BudgetSummary>(body, 'summary'));
}
/** Fetch every budget in the workspace. */
export function listBudgets(fetcher: Fetcher = fetch): Promise<Budget[]> {
	return requestJSON<{ budgets?: Budget[] } | Budget[]>('/api/v1/budgets', fetcher).then((body) =>
		unwrapList<Budget>(body, 'budgets')
	);
}

/** Fetch one budget with its tags. */
export function getBudget(id: string, fetcher: Fetcher = fetch): Promise<Budget> {
	return requestJSON<Budget>(`/api/v1/budgets/${encodeURIComponent(id)}`, fetcher);
}

/** Fetch spending for the most recent periods of a budget. */
export function getBudgetTrend(
	id: string,
	options: { periods?: number; to?: string } = {},
	fetcher: Fetcher = fetch
): Promise<BudgetTrend> {
	const params = new URLSearchParams();
	if (options.periods !== undefined) params.set('periods', String(options.periods));
	if (options.to) params.set('to', options.to);
	const query = params.toString();
	return requestJSON<BudgetTrend>(
		`/api/v1/budgets/${encodeURIComponent(id)}/trend${query ? `?${query}` : ''}`,
		fetcher
	);
}

/** Create a weekly or monthly tag-based budget. */
export function createBudget(input: BudgetInput, fetcher: Fetcher = fetch): Promise<Budget> {
	return requestJSON<Budget>('/api/v1/budgets', fetcher, {
		method: 'POST',
		body: JSON.stringify(input)
	});
}

/** Update a budget, leaving omitted fields unchanged. */
export function updateBudget(
	id: string,
	input: BudgetUpdate,
	fetcher: Fetcher = fetch
): Promise<Budget> {
	return requestJSON<Budget>(`/api/v1/budgets/${encodeURIComponent(id)}`, fetcher, {
		method: 'PATCH',
		body: JSON.stringify(input)
	});
}

/** Delete a budget. Tag and transaction data is unaffected. */
export function deleteBudget(id: string, fetcher: Fetcher = fetch): Promise<void> {
	return requestVoid(`/api/v1/budgets/${encodeURIComponent(id)}`, fetcher, { method: 'DELETE' });
}
