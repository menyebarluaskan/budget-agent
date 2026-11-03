<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { IconName } from '$lib/icons';
	import Icon from './Icon.svelte';

	let {
		title,
		description,
		icon = 'sparkles',
		actions
	}: { title: string; description: string; icon?: IconName; actions?: Snippet } = $props();
</script>

<section
	class="rounded-card border border-dashed border-line bg-surface p-8 text-center shadow-card"
>
	<span
		class="mx-auto grid size-12 place-items-center rounded-full bg-accent/10 text-accent-strong"
	>
		<Icon name={icon} class="size-6" />
	</span>
	<h2 class="mt-4 text-base font-semibold tracking-tight">{title}</h2>
	<p class="mx-auto mt-2 max-w-md text-sm text-pretty text-ink-muted">{description}</p>
	{#if actions}
		<div class="mt-5 flex flex-wrap items-center justify-center gap-2">
			{@render actions()}
		</div>
	{/if}
</section>
