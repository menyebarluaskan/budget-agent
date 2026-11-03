import type { Pathname } from '$app/types';
import type { IconName } from './icons';

export interface NavItem {
	href: Pathname;
	label: string;
	icon: IconName;
}

/** Primary navigation shown in the sidebar and the mobile bottom bar. */
export const primaryNav: NavItem[] = [
	{ href: '/', label: 'Overview', icon: 'home' },
	{ href: '/transactions', label: 'Transactions', icon: 'transactions' },
	{ href: '/budgets', label: 'Budgets', icon: 'budgets' },
	{ href: '/accounts', label: 'Accounts', icon: 'accounts' },
	{ href: '/settings', label: 'Settings', icon: 'settings' }
];

/**
 * Match the current path against a navigation item. The root matches exactly;
 * every other item also owns its nested routes, so `/transactions/new` keeps
 * the Transactions item active.
 */
export function isActiveHref(pathname: string, href: string): boolean {
	if (href === '/') {
		return pathname === '/';
	}

	return pathname === href || pathname.startsWith(`${href}/`);
}

export function findActiveNavItem(
	pathname: string,
	items: NavItem[] = primaryNav
): NavItem | undefined {
	return items.find((item) => isActiveHref(pathname, item.href));
}
