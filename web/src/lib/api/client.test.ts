import { describe, expect, it, vi } from 'vitest';
import { ApiError, getDashboard } from './client';
import type { Dashboard } from './types';

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
