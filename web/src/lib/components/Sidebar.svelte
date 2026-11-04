<script lang="ts">
	import { resolve } from '$app/paths';
	import { primaryNav } from '$lib/navigation';
	import Brand from './Brand.svelte';
	import Icon from './Icon.svelte';
	import NavItemLink from './NavItemLink.svelte';
	import WorkspaceSwitcher from './WorkspaceSwitcher.svelte';
</script>

<aside
	class="hidden border-r border-line bg-surface lg:sticky lg:top-0 lg:flex lg:h-dvh lg:flex-col"
>
	<div class="flex h-20 items-center px-5">
		<Brand />
	</div>
	<WorkspaceSwitcher />

	<nav class="flex-1 space-y-1 overflow-y-auto px-3 py-4" aria-label="Primary">
		{#each primaryNav as item (item.href)}
			<NavItemLink {item} />
		{/each}
	</nav>

	<div class="space-y-3 border-t border-line p-3">
		<a
			href={resolve('/transactions/new')}
			class="flex items-center justify-center gap-2 rounded-xl bg-accent px-3 py-3 text-sm font-semibold text-on-accent transition-colors hover:bg-accent-strong"
		>
			<Icon name="plus" class="size-4" />
			Add transaction
		</a>
		<p class="px-2 text-xs text-ink-muted">Self-hosted · Tailnet only</p>
	</div>
</aside>
