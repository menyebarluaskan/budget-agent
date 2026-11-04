export interface BudgetSimulationInput {
	budgetMinor: number;
	spentMinor: number;
	extraSpendMinor: number;
}

export interface BudgetSimulationResult {
	projectedSpentMinor: number;
	remainingMinor: number;
	percentUsed: number;
	barPercent: number;
	overBudget: boolean;
}

/**
 * Project a budget period after a hypothetical extra spend. The result is
 * display-only; callers never persist it.
 */
export function simulateBudget(input: BudgetSimulationInput): BudgetSimulationResult {
	const budgetMinor = Math.max(0, Math.trunc(input.budgetMinor) || 0);
	const spentMinor = Math.max(0, Math.trunc(input.spentMinor) || 0);
	const extraSpendMinor = Math.max(0, Math.trunc(input.extraSpendMinor) || 0);
	const projectedSpentMinor = spentMinor + extraSpendMinor;
	const remainingMinor = budgetMinor - projectedSpentMinor;
	const percentUsed = budgetMinor > 0 ? Math.round((projectedSpentMinor / budgetMinor) * 100) : 0;

	return {
		projectedSpentMinor,
		remainingMinor,
		percentUsed,
		barPercent: Math.min(100, Math.max(0, percentUsed)),
		overBudget: projectedSpentMinor > budgetMinor
	};
}
