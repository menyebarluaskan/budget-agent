<script lang="ts">
	import type { Snippet } from 'svelte';

	let { title, description, actions }: { title: string; description?: string; actions?: Snippet } =
		$props();
</script>

<header class="flex flex-wrap items-end justify-between gap-4">
	<div class="min-w-0 space-y-1">
		<h1 class="text-2xl font-semibold tracking-tight text-balance sm:text-3xl">{title}</h1>
		{#if description}
			<p class="max-w-2xl text-sm text-pretty text-ink-muted sm:text-base">{description}</p>
		{/if}
	</div>
	{#if actions}
		<div class="flex flex-wrap items-center gap-2">
			{@render actions()}
		</div>
	{/if}
</header>
