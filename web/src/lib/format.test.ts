import { describe, expect, it } from 'vitest';
import {
	currencyFractionDigits,
	formatDate,
	formatDateRange,
	formatMoney,
	formatMonthYear,
	formatSignedMoney,
	monthRange,
	shiftMonth
} from './format';

describe('money formatting', () => {
	it('respects currency minor units', () => {
		expect(currencyFractionDigits('IDR')).toBe(0);
		expect(currencyFractionDigits('USD')).toBe(2);
		expect(currencyFractionDigits('JPY')).toBe(0);
	});

	it('formats integer minor units', () => {
		expect(formatMoney(100_000, 'IDR')).toBe('IDR 100,000');
		expect(formatMoney(12_345, 'USD')).toBe('$123.45');
		expect(formatMoney(1_234, 'JPY')).toBe('¥1,234');
		expect(formatMoney(-2_500, 'USD')).toBe('-$25.00');
	});

	it('falls back for unknown currency codes', () => {
		expect(formatMoney(1_234, 'BAD')).toBe('BAD 12.34');
	});

	it('marks positive amounts as income', () => {
		expect(formatSignedMoney(10_000, 'IDR')).toBe('+IDR 10,000');
		expect(formatSignedMoney(-30_000, 'IDR')).toBe('-IDR 30,000');
	});
});

describe('date formatting', () => {
	it('formats date-only values without a timezone shift', () => {
		expect(formatDate('2026-11-05')).toBe('Nov 5, 2026');
		expect(formatMonthYear('2026-11-05')).toBe('November 2026');
	});

	it('collapses a range within one month', () => {
		expect(formatDateRange('2026-11-01', '2026-11-30')).toBe('Nov 1–30, 2026');
		expect(formatDateRange('2026-11-05', '2026-11-05')).toBe('Nov 5, 2026');
	});

	it('keeps both months when a range spans a boundary', () => {
		expect(formatDateRange('2026-11-01', '2026-12-31')).toBe('Nov 1, 2026 – Dec 31, 2026');
	});
});

describe('month ranges', () => {
	it('returns the inclusive first and last day', () => {
		expect(monthRange(2026, 10)).toEqual({ from: '2026-11-01', to: '2026-11-30' });
		expect(monthRange(2024, 1)).toEqual({ from: '2024-02-01', to: '2024-02-29' });
	});

	it('shifts across year boundaries', () => {
		const january = new Date(2026, 0, 15);
		expect(shiftMonth(january, -1).getFullYear()).toBe(2025);
		expect(shiftMonth(january, -1).getMonth()).toBe(11);
	});
});
