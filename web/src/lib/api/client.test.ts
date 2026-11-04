import { describe, expect, it, vi } from 'vitest';
import {
	ApiError,
	createAccount,
	createBudget,
	createTag,
	createTransaction,
	createTransfer,
	deleteAccount,
	deleteBudget,
	deleteTransaction,
	getBudget,
	getBudgetTrend,
	getDashboard,
	getSpendingAnalytics,
	getTransaction,
	listAccounts,
	listBudgets,
	listTags,
	listTransactions,
	updateAccount,
	updateBudget,
	updateTag,
	updateTransaction
} from './client';
import type {
	Account,
	Budget,
	BudgetTrend,
	Dashboard,
	SpendingAnalytics,
	Tag,
	Transaction
} from './types';

const dashboard: Dashboard = {
	from: '2026-11-01',
	to: '2026-11-30',
	currency: 'IDR',
	total_balance_minor: 75_000,
	account_count: 1,
	income_minor: 10_000,
	expense_minor: 25_000,
	net_minor: -15_000,
	budgets: [],
	recent_transactions: []
};

function response(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'Content-Type': 'application/json' }
	});
}

describe('getDashboard', () => {
	it('requests the selected range and returns the dashboard', async () => {
		const fetcher = vi.fn(async () => response(dashboard));

		const result = await getDashboard({ from: '2026-11-01', to: '2026-11-30' }, fetcher);

		expect(result).toEqual(dashboard);
		expect(fetcher).toHaveBeenCalledWith('/api/v1/dashboard?from=2026-11-01&to=2026-11-30', {
			headers: { Accept: 'application/json' }
		});
	});

	it('surfaces the problem title on failure', async () => {
		const fetcher = vi.fn(async () =>
			response({ title: 'from must not be after to', status: 400 }, 400)
		);

		await expect(getDashboard({ from: '2026-11-30', to: '2026-11-01' }, fetcher)).rejects.toEqual(
			new ApiError(400, 'from must not be after to')
		);
	});

	it('falls back to a status message when the error is not JSON', async () => {
		const fetcher = vi.fn(async () => new Response('upstream unavailable', { status: 502 }));

		await expect(getDashboard({ from: '2026-11-01', to: '2026-11-30' }, fetcher)).rejects.toEqual(
			new ApiError(502, 'Request failed with status 502')
		);
	});
});

describe('getSpendingAnalytics', () => {
	const analytics: SpendingAnalytics = {
		from: '2026-11-01',
		to: '2026-11-30',
		currency: 'IDR',
		total_spent_minor: 25000,
		untagged_minor: 0,
		tags: [{ tag_id: 'food', name: 'Food', color: '#f97316', spent_minor: 25000 }],
		trend: [{ month: '2026-11', income_minor: 0, expense_minor: 25000, net_minor: -25000 }]
	};

	it('requests the range and month window', async () => {
		const fetcher = vi.fn(async () => response(analytics));

		const result = await getSpendingAnalytics({ from: '2026-11-01', to: '2026-11-30' }, 6, fetcher);

		expect(result).toEqual(analytics);
		expect(fetcher).toHaveBeenCalledWith(
			'/api/v1/analytics/spending?from=2026-11-01&to=2026-11-30&months=6',
			{ headers: { Accept: 'application/json' } }
		);
	});
});

const transaction: Transaction = {
	id: 't1',
	workspace_id: 'default',
	account_id: 'w1',
	amount_minor: -2500,
	occurred_at: '2026-11-05T12:00:00Z',
	description: 'Coffee',
	notes: '',
	source: 'manual',
	tag_ids: ['food'],
	created_at: '2026-11-05T12:00:00Z',
	updated_at: '2026-11-05T12:00:00Z'
};

describe('transaction API', () => {
	it('serializes every list filter', async () => {
		const fetcher = vi.fn(async () =>
			response({ transactions: [transaction], next_cursor: 'abc' })
		);

		const result = await listTransactions(
			{
				accountId: 'w1',
				tagIds: ['food', 'transport'],
				from: '2026-11-01',
				to: '2026-11-30',
				minAmountMinor: -5000,
				maxAmountMinor: 10000,
				q: 'coffee',
				source: 'manual',
				includeTransfers: true,
				sort: 'amount_minor',
				limit: 20,
				cursor: 'next'
			},
			fetcher
		);

		expect(result.next_cursor).toBe('abc');
		expect(fetcher).toHaveBeenCalledWith(
			'/api/v1/transactions?account_id=w1&tag=food&tag=transport&from=2026-11-01&to=2026-11-30&min_amount_minor=-5000&max_amount_minor=10000&q=coffee&source=manual&include_transfers=true&sort=amount_minor&limit=20&cursor=next',
			{ headers: { Accept: 'application/json' } }
		);
	});

	it('omits empty list filters', async () => {
		const fetcher = vi.fn(async () => response({ transactions: [] }));

		await listTransactions({}, fetcher);

		expect(fetcher).toHaveBeenCalledWith('/api/v1/transactions', {
			headers: { Accept: 'application/json' }
		});
	});

	it('gets one transaction', async () => {
		const fetcher = vi.fn(async () => response(transaction));

		await expect(getTransaction('t 1', fetcher)).resolves.toEqual(transaction);
		expect(fetcher).toHaveBeenCalledWith('/api/v1/transactions/t%201', {
			headers: { Accept: 'application/json' }
		});
	});

	it('creates a transaction as JSON', async () => {
		const fetcher = vi.fn(async () => response(transaction, 201));

		await createTransaction(
			{
				account_id: 'w1',
				amount_minor: -2500,
				occurred_at: '2026-11-05T12:00:00Z',
				description: 'Coffee',
				notes: '',
				tag_ids: ['food']
			},
			fetcher
		);

		expect(fetcher).toHaveBeenCalledWith('/api/v1/transactions', {
			method: 'POST',
			headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
			body: JSON.stringify({
				account_id: 'w1',
				amount_minor: -2500,
				occurred_at: '2026-11-05T12:00:00Z',
				description: 'Coffee',
				notes: '',
				tag_ids: ['food']
			})
		});
	});

	it('patches a transaction', async () => {
		const fetcher = vi.fn(async () => response(transaction));

		await updateTransaction('t1', { description: 'Tea' }, fetcher);

		expect(fetcher).toHaveBeenCalledWith('/api/v1/transactions/t1', {
			method: 'PATCH',
			headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
			body: JSON.stringify({ description: 'Tea' })
		});
	});

	it('deletes a transaction without parsing a body', async () => {
		const fetcher = vi.fn(async () => new Response(null, { status: 204 }));

		await expect(deleteTransaction('t1', fetcher)).resolves.toBeUndefined();
		expect(fetcher).toHaveBeenCalledWith('/api/v1/transactions/t1', {
			method: 'DELETE',
			headers: { Accept: 'application/json' }
		});
	});
});

describe('lookup API', () => {
	it('unwraps accounts', async () => {
		const account: Account = {
			id: 'w1',
			workspace_id: 'default',
			name: 'Wallet',
			type: 'wallet',
			currency: 'IDR',
			opening_balance_minor: 0,
			archived: false,
			created_at: '2026-01-01T00:00:00Z',
			updated_at: '2026-01-01T00:00:00Z',
			balance_minor: 10000
		};
		const fetcher = vi.fn(async () => response({ accounts: [account] }));

		await expect(listAccounts(fetcher)).resolves.toEqual([account]);
	});

	it('unwraps tags', async () => {
		const tag: Tag = {
			id: 'food',
			workspace_id: 'default',
			name: 'Food',
			color: '#f97316',
			created_at: '2026-01-01T00:00:00Z'
		};
		const fetcher = vi.fn(async () => response({ tags: [tag] }));

		await expect(listTags(fetcher)).resolves.toEqual([tag]);
	});

	it('creates a tag', async () => {
		const tag: Tag = {
			id: 'food',
			workspace_id: 'default',
			name: 'Food',
			color: '#f97316',
			created_at: '2026-01-01T00:00:00Z'
		};
		const fetcher = vi.fn(async () => response(tag, 201));

		await expect(createTag({ name: 'Food', color: '#f97316' }, fetcher)).resolves.toEqual(tag);
		expect(fetcher).toHaveBeenCalledWith('/api/v1/tags', {
			method: 'POST',
			body: JSON.stringify({ name: 'Food', color: '#f97316' }),
			headers: { Accept: 'application/json', 'Content-Type': 'application/json' }
		});
	});

	it('updates a tag name and color', async () => {
		const tag: Tag = {
			id: 'food',
			workspace_id: 'default',
			name: 'Groceries',
			color: '#22c55e',
			created_at: '2026-01-01T00:00:00Z'
		};
		const fetcher = vi.fn(async () => response(tag));

		await expect(
			updateTag('food', { name: 'Groceries', color: '#22c55e' }, fetcher)
		).resolves.toEqual(tag);
		expect(fetcher).toHaveBeenCalledWith('/api/v1/tags/food', {
			method: 'PATCH',
			body: JSON.stringify({ name: 'Groceries', color: '#22c55e' }),
			headers: { Accept: 'application/json', 'Content-Type': 'application/json' }
		});
	});
});

const accountFixture: Account = {
	id: 'w1',
	workspace_id: 'default',
	name: 'Wallet',
	type: 'wallet',
	currency: 'IDR',
	opening_balance_minor: 0,
	archived: false,
	created_at: '2026-01-01T00:00:00Z',
	updated_at: '2026-01-01T00:00:00Z',
	balance_minor: 10000
};

describe('account API', () => {
	it('creates an account', async () => {
		const fetcher = vi.fn(async () => response(accountFixture, 201));

		await createAccount(
			{ name: 'Wallet', type: 'wallet', currency: 'IDR', opening_balance_minor: 0 },
			fetcher
		);

		expect(fetcher).toHaveBeenCalledWith('/api/v1/accounts', {
			method: 'POST',
			headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
			body: JSON.stringify({
				name: 'Wallet',
				type: 'wallet',
				currency: 'IDR',
				opening_balance_minor: 0
			})
		});
	});

	it('updates an account', async () => {
		const fetcher = vi.fn(async () => response(accountFixture));

		await updateAccount('w1', { archived: true }, fetcher);

		expect(fetcher).toHaveBeenCalledWith('/api/v1/accounts/w1', {
			method: 'PATCH',
			headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
			body: JSON.stringify({ archived: true })
		});
	});

	it('deletes an account', async () => {
		const fetcher = vi.fn(async () => new Response(null, { status: 204 }));

		await expect(deleteAccount('w1', fetcher)).resolves.toBeUndefined();
		expect(fetcher).toHaveBeenCalledWith('/api/v1/accounts/w1', {
			method: 'DELETE',
			headers: { Accept: 'application/json' }
		});
	});
});

describe('transfer API', () => {
	it('posts both legs as one transfer', async () => {
		const payload = {
			from_account_id: 'w1',
			to_account_id: 'w2',
			amount_minor: 15000,
			occurred_at: '2026-11-05T12:00:00Z',
			description: 'Savings',
			notes: ''
		};
		const transfer = { transfer_id: 'tr1', from: transaction, to: transaction };
		const fetcher = vi.fn(async () => response(transfer, 201));

		await expect(createTransfer(payload, fetcher)).resolves.toEqual(transfer);
		expect(fetcher).toHaveBeenCalledWith('/api/v1/transfers', {
			method: 'POST',
			headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
			body: JSON.stringify(payload)
		});
	});
});

const budgetFixture: Budget = {
	id: 'b1',
	workspace_id: 'default',
	name: 'Groceries',
	amount_minor: 500000,
	period: 'monthly',
	start_date: '2026-11-01',
	end_date: null,
	rollover: true,
	account_id: null,
	tag_ids: ['food'],
	created_at: '2026-11-01T00:00:00Z',
	updated_at: '2026-11-01T00:00:00Z'
};

describe('budget API', () => {
	it('lists budgets', async () => {
		const fetcher = vi.fn(async () => response({ budgets: [budgetFixture] }));

		await expect(listBudgets(fetcher)).resolves.toEqual([budgetFixture]);
	});

	it('gets one budget', async () => {
		const fetcher = vi.fn(async () => response(budgetFixture));

		await expect(getBudget('b 1', fetcher)).resolves.toEqual(budgetFixture);
		expect(fetcher).toHaveBeenCalledWith('/api/v1/budgets/b%201', {
			headers: { Accept: 'application/json' }
		});
	});

	it('gets a budget trend with optional parameters', async () => {
		const trend: BudgetTrend = {
			budget_id: 'b1',
			name: 'Groceries',
			period: 'monthly',
			currency: 'IDR',
			amount_minor: 500000,
			periods: [
				{
					start: '2026-11-01',
					end: '2026-11-30',
					spent_minor: 250000,
					amount_minor: 500000,
					remaining_minor: 250000
				}
			]
		};
		const fetcher = vi.fn(async () => response(trend));

		await expect(getBudgetTrend('b 1', { periods: 6, to: '2026-11-30' }, fetcher)).resolves.toEqual(
			trend
		);
		expect(fetcher).toHaveBeenCalledWith('/api/v1/budgets/b%201/trend?periods=6&to=2026-11-30', {
			headers: { Accept: 'application/json' }
		});
	});

	it('creates a budget', async () => {
		const payload = {
			name: 'Groceries',
			amount_minor: 500000,
			period: 'monthly' as const,
			start_date: '2026-11-01',
			rollover: true,
			tag_ids: ['food']
		};
		const fetcher = vi.fn(async () => response(budgetFixture, 201));

		await createBudget(payload, fetcher);

		expect(fetcher).toHaveBeenCalledWith('/api/v1/budgets', {
			method: 'POST',
			headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
			body: JSON.stringify(payload)
		});
	});

	it('updates a budget', async () => {
		const fetcher = vi.fn(async () => response(budgetFixture));

		await updateBudget('b1', { amount_minor: 600000, end_date: '' }, fetcher);

		expect(fetcher).toHaveBeenCalledWith('/api/v1/budgets/b1', {
			method: 'PATCH',
			headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
			body: JSON.stringify({ amount_minor: 600000, end_date: '' })
		});
	});

	it('deletes a budget', async () => {
		const fetcher = vi.fn(async () => new Response(null, { status: 204 }));

		await expect(deleteBudget('b1', fetcher)).resolves.toBeUndefined();
		expect(fetcher).toHaveBeenCalledWith('/api/v1/budgets/b1', {
			method: 'DELETE',
			headers: { Accept: 'application/json' }
		});
	});
});
