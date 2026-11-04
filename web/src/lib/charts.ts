/** Largest non-negative amount, with a floor so charts never divide by zero. */
export function chartMax(values: number[]): number {
	let max = 0;
	for (const value of values) {
		if (Number.isFinite(value) && value > max) {
			max = value;
		}
	}
	return Math.max(1, max);
}

/** Scale an amount to a bar height percentage and keep small bars visible. */
export function barPercent(value: number, max: number): number {
	if (!Number.isFinite(value) || value <= 0 || !Number.isFinite(max) || max <= 0) {
		return 0;
	}
	return Math.min(100, Math.max(2, Math.round((value / max) * 100)));
}
