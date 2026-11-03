import type { DateRange } from './api/types';

const DEFAULT_LOCALE = 'en-US';
const DATE_ONLY = /^\d{4}-\d{2}-\d{2}$/;

/** Number of decimal places used by a currency (for example, 0 for IDR). */
export function currencyFractionDigits(currency: string, locale = DEFAULT_LOCALE): number {
	try {
		const options = new Intl.NumberFormat(locale, {
			style: 'currency',
			currency
		}).resolvedOptions();
		return options.maximumFractionDigits ?? 2;
	} catch {
		return 2;
	}
}

/** Format integer minor units as a localized currency amount. */
export function formatMoney(minor: number, currency: string, locale = DEFAULT_LOCALE): string {
	const digits = currencyFractionDigits(currency, locale);
	const major = minor / 10 ** digits;
	try {
		return new Intl.NumberFormat(locale, { style: 'currency', currency })
			.format(major)
			.replace(/[\u00a0\u202f]/g, ' ');
	} catch {
		return `${currency} ${major.toFixed(digits)}`;
	}
}

/** Format a signed amount, adding an explicit `+` for income. */
export function formatSignedMoney(
	minor: number,
	currency: string,
	locale = DEFAULT_LOCALE
): string {
	const formatted = formatMoney(minor, currency, locale);
	return minor > 0 ? `+${formatted}` : formatted;
}

function toDate(value: string): Date | null {
	const date = DATE_ONLY.test(value) ? new Date(`${value}T00:00:00Z`) : new Date(value);
	return Number.isNaN(date.getTime()) ? null : date;
}

/** Format an ISO date or RFC 3339 timestamp for display. */
export function formatDate(value: string, locale = DEFAULT_LOCALE): string {
	const date = toDate(value);
	if (!date) {
		return value;
	}
	return new Intl.DateTimeFormat(locale, {
		day: 'numeric',
		month: 'short',
		year: 'numeric',
		timeZone: DATE_ONLY.test(value) ? 'UTC' : undefined
	}).format(date);
}

/** Format a date as `November 2026`. */
export function formatMonthYear(value: string, locale = DEFAULT_LOCALE): string {
	const date = toDate(value);
	if (!date) {
		return value;
	}
	return new Intl.DateTimeFormat(locale, {
		month: 'long',
		year: 'numeric',
		timeZone: 'UTC'
	}).format(date);
}

/** Format an inclusive date range, collapsing a single month to `Nov 1–30, 2026`. */
export function formatDateRange(from: string, to: string, locale = DEFAULT_LOCALE): string {
	if (from === to) {
		return formatDate(from, locale);
	}

	const fromDate = toDate(from);
	const toDateValue = toDate(to);
	if (!fromDate || !toDateValue) {
		return `${from} – ${to}`;
	}

	if (from.slice(0, 7) === to.slice(0, 7)) {
		const month = new Intl.DateTimeFormat(locale, { month: 'short', timeZone: 'UTC' }).format(
			toDateValue
		);
		const fromDay = new Intl.DateTimeFormat(locale, { day: 'numeric', timeZone: 'UTC' }).format(
			fromDate
		);
		const toDay = new Intl.DateTimeFormat(locale, { day: 'numeric', timeZone: 'UTC' }).format(
			toDateValue
		);
		const year = new Intl.DateTimeFormat(locale, { year: 'numeric', timeZone: 'UTC' }).format(
			toDateValue
		);
		return `${month} ${fromDay}–${toDay}, ${year}`;
	}

	return `${formatDate(from, locale)} – ${formatDate(to, locale)}`;
}

function toISODate(date: Date): string {
	return date.toISOString().slice(0, 10);
}

/** Inclusive first and last day of a calendar month. */
export function monthRange(year: number, monthIndex: number): DateRange {
	return {
		from: toISODate(new Date(Date.UTC(year, monthIndex, 1))),
		to: toISODate(new Date(Date.UTC(year, monthIndex + 1, 0)))
	};
}

/** Move a cursor by whole months, clamped to the first day of the month. */
export function shiftMonth(date: Date, delta: number): Date {
	return new Date(date.getFullYear(), date.getMonth() + delta, 1);
}
