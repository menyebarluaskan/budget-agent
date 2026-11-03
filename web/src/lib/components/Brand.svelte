<script lang="ts">
	import { resolve } from '$app/paths';
	import Icon from './Icon.svelte';
</script>

<a
	href={resolve('/')}
	class="flex items-center gap-3 rounded-xl focus-visible:outline-offset-4"
	aria-label="Budget Agent home"
>
	<span class="grid size-9 shrink-0 place-items-center rounded-xl bg-accent text-on-accent">
		<Icon name="budgets" class="size-5" />
	</span>
	<span class="flex min-w-0 flex-col leading-tight">
		<span class="truncate text-sm font-semibold tracking-tight">Budget Agent</span>
		<span class="truncate text-xs text-ink-muted">Integrated budget</span>
	</span>
</a>
