import { describe, expect, it } from 'vitest';
import { barPercent, chartMax } from './charts';

describe('chart helpers', () => {
	it('finds a non-zero chart maximum', () => {
		expect(chartMax([])).toBe(1);
		expect(chartMax([0, -4, 12, 8])).toBe(12);
		expect(chartMax([0, -4])).toBe(1);
	});

	it('scales bars and keeps small values visible', () => {
		expect(barPercent(0, 100)).toBe(0);
		expect(barPercent(-5, 100)).toBe(0);
		expect(barPercent(50, 100)).toBe(50);
		expect(barPercent(1, 1000)).toBe(2);
		expect(barPercent(200, 100)).toBe(100);
		expect(barPercent(10, 0)).toBe(0);
	});
});
