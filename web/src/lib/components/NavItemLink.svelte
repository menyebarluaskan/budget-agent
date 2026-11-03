<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { isActiveHref, type NavItem } from '$lib/navigation';
	import Icon from './Icon.svelte';

	let { item, variant = 'sidebar' }: { item: NavItem; variant?: 'sidebar' | 'bottom' } = $props();

	const active = $derived(isActiveHref(page.url.pathname, item.href));
</script>

{#if variant === 'sidebar'}
	<a
		href={resolve(item.href)}
		aria-current={active ? 'page' : undefined}
		class="flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-colors {active
			? 'bg-accent/10 text-accent-strong'
			: 'text-ink-muted hover:bg-surface-muted hover:text-ink'}"
	>
		<Icon name={item.icon} class="size-5 shrink-0" />
		<span class="truncate">{item.label}</span>
	</a>
{:else}
	<a
		href={resolve(item.href)}
		aria-current={active ? 'page' : undefined}
		class="flex min-w-0 flex-1 flex-col items-center gap-1 rounded-lg px-1 py-2 text-[0.6875rem] font-medium transition-colors {active
			? 'text-accent-strong'
			: 'text-ink-muted hover:text-ink'}"
	>
		<Icon name={item.icon} class="size-5 shrink-0" />
		<span class="w-full truncate text-center">{item.label}</span>
	</a>
{/if}
