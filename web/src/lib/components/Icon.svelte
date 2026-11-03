<script lang="ts">
	import { iconPaths, type IconName } from '$lib/icons';

	let { name, class: className = 'size-5' }: { name: IconName; class?: string } = $props();
</script>

<svg
	xmlns="http://www.w3.org/2000/svg"
	viewBox="0 0 24 24"
	fill="none"
	stroke="currentColor"
	stroke-width="1.6"
	stroke-linecap="round"
	stroke-linejoin="round"
	class={className}
	aria-hidden="true"
	focusable="false"
>
	{#each iconPaths[name] as path (path)}
		<path d={path} />
	{/each}
</svg>
