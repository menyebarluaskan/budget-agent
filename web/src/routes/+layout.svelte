<script lang="ts">
	import '../app.css';
	import { page } from '$app/state';
	import type { Snippet } from 'svelte';
	import BottomNav from '$lib/components/BottomNav.svelte';
	import MobileHeader from '$lib/components/MobileHeader.svelte';
	import Sidebar from '$lib/components/Sidebar.svelte';
	import { findActiveNavItem } from '$lib/navigation';

	let { children }: { children: Snippet } = $props();

	const activeItem = $derived(findActiveNavItem(page.url.pathname));
	const title = $derived(activeItem ? `${activeItem.label} · Budget Agent` : 'Budget Agent');
</script>

<svelte:head>
	<title>{title}</title>
	<meta
		name="description"
		content="A self-hosted, mobile-first personal budgeting app with an API, AI drafts, and MCP access."
	/>
</svelte:head>

<a
	href="#main-content"
	class="fixed top-3 left-3 z-50 -translate-y-20 rounded-xl bg-accent px-4 py-2 text-sm font-semibold text-on-accent transition-transform focus:translate-y-0"
>
	Skip to content
</a>

<div class="lg:grid lg:min-h-dvh lg:grid-cols-[17rem_minmax(0,1fr)]">
	<Sidebar />
	<div class="flex min-w-0 flex-col">
		<MobileHeader />
		<main
			id="main-content"
			tabindex="-1"
			class="mx-auto w-full max-w-5xl flex-1 px-4 pt-6 pb-24 sm:px-6 lg:px-10 lg:pt-10 lg:pb-12"
		>
			{@render children()}
		</main>
	</div>
</div>

<BottomNav />
