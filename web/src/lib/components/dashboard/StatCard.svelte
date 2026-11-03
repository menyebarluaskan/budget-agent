<script lang="ts">
	let {
		label,
		value,
		hint,
		tone = 'default'
	}: {
		label: string;
		value: string;
		hint?: string;
		tone?: 'default' | 'positive' | 'negative' | 'muted';
	} = $props();

	const toneClass = $derived(
		tone === 'positive'
			? 'text-positive'
			: tone === 'negative'
				? 'text-negative'
				: tone === 'muted'
					? 'text-ink-muted'
					: 'text-ink'
	);
</script>

<div class="rounded-card border border-line bg-surface p-5 shadow-card">
	<p class="text-sm font-medium text-ink-muted">{label}</p>
	<p class="mt-2 text-2xl font-semibold tracking-tight tabular-nums {toneClass}">{value}</p>
	{#if hint}
		<p class="mt-1 text-xs text-ink-muted">{hint}</p>
	{/if}
</div>
