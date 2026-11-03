<script lang="ts">
	import { primaryNav } from '$lib/navigation';
	import NavItemLink from './NavItemLink.svelte';
</script>

<nav
	class="fixed inset-x-0 bottom-0 z-20 border-t border-line bg-surface/95 pb-[env(safe-area-inset-bottom)] backdrop-blur lg:hidden"
	aria-label="Primary"
>
	<div class="mx-auto flex w-full max-w-lg items-stretch justify-around gap-1 px-2 py-1">
		{#each primaryNav as item (item.href)}
			<NavItemLink {item} variant="bottom" />
		{/each}
	</div>
</nav>
