<script lang="ts">
	import { resolve } from '$app/paths';
	import type { Pathname } from '$app/types';
	import type { Snippet } from 'svelte';

	let {
		href,
		variant = 'primary',
		children
	}: {
		href: Pathname;
		variant?: 'primary' | 'secondary';
		children: Snippet;
	} = $props();

	const classes = $derived(
		variant === 'primary'
			? 'bg-accent text-on-accent hover:bg-accent-strong'
			: 'border border-line bg-surface text-ink hover:bg-surface-muted'
	);
</script>

<a
	href={resolve(href)}
	class="inline-flex min-h-11 items-center justify-center rounded-xl px-4 py-2 text-sm font-semibold transition-colors {classes}"
>
	{@render children()}
</a>
