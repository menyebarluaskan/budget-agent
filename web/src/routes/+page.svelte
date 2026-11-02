<svelte:head>
	<title>Integrated Personal Budget</title>
	<meta
		name="description"
		content="A self-hosted, mobile-first personal budgeting app with an API, AI drafts, and MCP access."
	/>
</svelte:head>

<main class="mx-auto flex min-h-dvh w-full max-w-2xl flex-col justify-center gap-8 px-6 py-12">
	<div class="space-y-4">
		<p class="text-sm font-semibold tracking-wide text-sky-700 uppercase dark:text-sky-400">
			Self-hosted budgeting
		</p>
		<h1 class="text-4xl font-bold tracking-tight text-balance sm:text-5xl">
			Integrated Personal Budget
		</h1>
		<p class="max-w-prose text-lg leading-relaxed text-slate-600 dark:text-slate-300">
			The repository scaffold is running. Accounts, transactions, budgets, receipts, and the
			assistant arrive in the next milestones.
		</p>
	</div>

	<section
		class="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm dark:border-slate-800 dark:bg-slate-900"
		aria-labelledby="status-heading"
	>
		<h2 id="status-heading" class="text-lg font-semibold">Foundation status</h2>
		<ul class="mt-4 space-y-3 text-sm text-slate-600 dark:text-slate-300">
			<li class="flex items-center gap-3">
				<span class="size-2.5 rounded-full bg-emerald-500" aria-hidden="true"></span>
				Backend health endpoints and graceful shutdown
			</li>
			<li class="flex items-center gap-3">
				<span class="size-2.5 rounded-full bg-emerald-500" aria-hidden="true"></span>
				SvelteKit static build with Tailwind CSS
			</li>
			<li class="flex items-center gap-3">
				<span class="size-2.5 rounded-full bg-amber-500" aria-hidden="true"></span>
				Database, domain API, and embedded UI are next
			</li>
		</ul>
	</section>

	<footer class="text-sm text-slate-500 dark:text-slate-400">
		API liveness: <code class="font-mono">/healthz</code> · readiness:
		<code class="font-mono">/readyz</code>
	</footer>
</main>
