<script lang="ts">
	import { resolve } from '$app/paths';
	import Brand from './Brand.svelte';
	import Icon from './Icon.svelte';
	import WorkspaceSwitcher from './WorkspaceSwitcher.svelte';
</script>

<header
	class="sticky top-0 z-20 flex h-16 items-center justify-between gap-4 border-b border-line bg-canvas/90 px-4 backdrop-blur lg:hidden"
>
	<Brand />
	<WorkspaceSwitcher />
	<a
		href={resolve('/transactions/new')}
		class="grid size-11 shrink-0 place-items-center rounded-xl bg-accent text-on-accent transition-colors hover:bg-accent-strong"
		aria-label="Add transaction"
	>
		<Icon name="plus" class="size-5" />
	</a>
</header>
