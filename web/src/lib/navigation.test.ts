import { existsSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import { findActiveNavItem, isActiveHref, primaryNav } from './navigation';

describe('primary navigation', () => {
	it('uses unique, non-empty hrefs and labels', () => {
		const hrefs = primaryNav.map((item) => item.href);

		expect(new Set(hrefs).size).toBe(hrefs.length);
		expect(hrefs.every((href) => href.startsWith('/'))).toBe(true);
		expect(primaryNav.every((item) => item.label.trim().length > 0)).toBe(true);
	});

	it('has a prerenderable page for every item', () => {
		for (const item of primaryNav) {
			const route = item.href === '/' ? '' : item.href;
			const page = new URL(`../routes${route}/+page.svelte`, import.meta.url);

			expect(existsSync(page), `missing page for ${item.href}`).toBe(true);
		}
	});
});

describe('isActiveHref', () => {
	it('matches only the exact root path', () => {
		expect(isActiveHref('/', '/')).toBe(true);
		expect(isActiveHref('/transactions', '/')).toBe(false);
		expect(isActiveHref('/anything', '/')).toBe(false);
	});

	it('matches a section and its nested routes', () => {
		expect(isActiveHref('/transactions', '/transactions')).toBe(true);
		expect(isActiveHref('/transactions/new', '/transactions')).toBe(true);
		expect(isActiveHref('/transactions-archive', '/transactions')).toBe(false);
	});
});

describe('findActiveNavItem', () => {
	it('returns the section that owns the current path', () => {
		expect(findActiveNavItem('/transactions/new')?.href).toBe('/transactions');
		expect(findActiveNavItem('/')?.href).toBe('/');
		expect(findActiveNavItem('/unknown')).toBeUndefined();
	});
});
