import { describe, expect, it } from 'vitest';
import { simulateBudget } from './simulation';

describe('simulateBudget', () => {
	it('projects spending and the remaining amount', () => {
		expect(
			simulateBudget({ budgetMinor: 500000, spentMinor: 200000, extraSpendMinor: 50000 })
		).toEqual({
			projectedSpentMinor: 250000,
			remainingMinor: 250000,
			percentUsed: 50,
			barPercent: 50,
			overBudget: false
		});
	});

	it('reports an over-budget projection', () => {
		const result = simulateBudget({
			budgetMinor: 100000,
			spentMinor: 90000,
			extraSpendMinor: 25000
		});

		expect(result.overBudget).toBe(true);
		expect(result.projectedSpentMinor).toBe(115000);
		expect(result.remainingMinor).toBe(-15000);
		expect(result.percentUsed).toBe(115);
		expect(result.barPercent).toBe(100);
	});

	it('ignores negative extra spending and handles a zero budget', () => {
		expect(
			simulateBudget({ budgetMinor: 100000, spentMinor: 25000, extraSpendMinor: -5000 })
		).toMatchObject({
			projectedSpentMinor: 25000,
			percentUsed: 25,
			overBudget: false
		});

		expect(simulateBudget({ budgetMinor: 0, spentMinor: 0, extraSpendMinor: 0 })).toMatchObject({
			percentUsed: 0,
			barPercent: 0,
			overBudget: false
		});
	});
});
