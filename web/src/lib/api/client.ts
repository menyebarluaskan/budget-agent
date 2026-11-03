import type {
	Account,
	Dashboard,
	DateRange,
	Tag,
	Transaction,
	TransactionFilters,
	TransactionInput,
	TransactionPage,
	TransactionUpdate
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

/** Fetch the dashboard read model for an inclusive date range. */
export function getDashboard(range: DateRange, fetcher: Fetcher = fetch): Promise<Dashboard> {
	const params = new URLSearchParams({ from: range.from, to: range.to });
	return requestJSON<Dashboard>(`/api/v1/dashboard?${params.toString()}`, fetcher);
}

/** Fetch accounts with their current balances. */
export function listAccounts(fetcher: Fetcher = fetch): Promise<Account[]> {
	return requestJSON<{ accounts?: Account[] } | Account[]>('/api/v1/accounts', fetcher).then(
		(body) => (Array.isArray(body) ? body : (body.accounts ?? []))
	);
}

/** Fetch every tag in the workspace. */
export function listTags(fetcher: Fetcher = fetch): Promise<Tag[]> {
	return requestJSON<{ tags?: Tag[] } | Tag[]>('/api/v1/tags', fetcher).then((body) =>
		Array.isArray(body) ? body : (body.tags ?? [])
	);
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
export async function deleteTransaction(id: string, fetcher: Fetcher = fetch): Promise<void> {
	const response = await fetcher(`/api/v1/transactions/${encodeURIComponent(id)}`, {
		method: 'DELETE',
		headers: { Accept: 'application/json' }
	});
	if (!response.ok) {
		throw new ApiError(response.status, await errorMessage(response));
	}
}
