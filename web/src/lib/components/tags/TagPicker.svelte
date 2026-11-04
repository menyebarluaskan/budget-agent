<script lang="ts">
	import { tick } from 'svelte';
	import { createTag, updateTag } from '$lib/api/client';
	import type { Tag } from '$lib/api/types';
	import Icon from '$lib/components/Icon.svelte';
	import { nextTagColor, sortTags, TAG_COLORS, tagNameExists } from '$lib/tags';

	let {
		tags,
		selectedIDs,
		onSelectedChange,
		onTagsChange
	}: {
		tags: Tag[];
		selectedIDs: string[];
		onSelectedChange: (ids: string[]) => void;
		onTagsChange: (tags: Tag[]) => void;
	} = $props();

	let query = $state('');
	let open = $state(false);
	let creating = $state(false);
	let saving = $state(false);
	let pickerError = $state<string | null>(null);
	let editorOpen = $state(false);
	let editingTagID = $state<string | null>(null);
	let draftName = $state('');
	let draftColor = $state(TAG_COLORS[0].value);
	let editorError = $state<string | null>(null);

	let containerEl = $state<HTMLDivElement>();
	let inputEl = $state<HTMLInputElement>();
	let nameInputEl = $state<HTMLInputElement>();

	const selectedTags = $derived(
		selectedIDs
			.map((id) => tags.find((tag) => tag.id === id))
			.filter((tag): tag is Tag => tag !== undefined)
	);
	const trimmedQuery = $derived(query.trim());
	const normalizedQuery = $derived(trimmedQuery.toLocaleLowerCase());
	const matchingTags = $derived(
		tags.filter(
			(tag) =>
				!selectedIDs.includes(tag.id) &&
				(normalizedQuery === '' || tag.name.toLocaleLowerCase().includes(normalizedQuery))
		)
	);
	const exactMatch = $derived(
		trimmedQuery === ''
			? undefined
			: tags.find((tag) => tag.name.toLocaleLowerCase() === normalizedQuery)
	);
	const canCreate = $derived(trimmedQuery !== '' && exactMatch === undefined);
	const editingTag = $derived(
		editingTagID ? (tags.find((tag) => tag.id === editingTagID) ?? null) : null
	);

	$effect(() => {
		if (editingTagID && !editingTag) {
			closeEditor();
		}
	});

	$effect(() => {
		if (!containerEl) {
			return;
		}
		function handlePointerDown(event: PointerEvent) {
			if (event.target instanceof Node && containerEl?.contains(event.target)) {
				return;
			}
			open = false;
			closeEditor();
		}
		document.addEventListener('pointerdown', handlePointerDown);
		return () => document.removeEventListener('pointerdown', handlePointerDown);
	});

	function errorMessage(cause: unknown, fallback: string): string {
		return cause instanceof Error ? cause.message : fallback;
	}

	function selectTag(tag: Tag) {
		if (!selectedIDs.includes(tag.id)) {
			onSelectedChange([...selectedIDs, tag.id]);
		}
		query = '';
		pickerError = null;
		open = true;
		inputEl?.focus();
	}

	function deselectTag(tagID: string) {
		onSelectedChange(selectedIDs.filter((id) => id !== tagID));
		if (editingTagID === tagID) {
			closeEditor();
		}
	}

	async function createFromQuery() {
		const name = trimmedQuery;
		if (name === '' || creating || saving) {
			return;
		}
		if (exactMatch) {
			selectTag(exactMatch);
			return;
		}

		creating = true;
		pickerError = null;
		try {
			const created = await createTag({ name, color: nextTagColor(tags) });
			onTagsChange(sortTags([...tags, created]));
			onSelectedChange([...selectedIDs, created.id]);
			query = '';
			open = true;
			inputEl?.focus();
		} catch (cause) {
			pickerError = errorMessage(cause, 'The tag could not be created.');
		} finally {
			creating = false;
		}
	}

	async function openEditor(tag: Tag) {
		editorOpen = true;
		editingTagID = tag.id;
		draftName = tag.name;
		draftColor = tag.color || nextTagColor(tags);
		editorError = null;
		open = false;
		await tick();
		nameInputEl?.focus();
		nameInputEl?.select();
	}

	function closeEditor() {
		editorOpen = false;
		editingTagID = null;
		draftName = '';
		editorError = null;
	}

	async function saveEditor() {
		if (!editingTag || saving) {
			return;
		}
		const name = draftName.trim();
		if (name === '') {
			editorError = 'Enter a tag name.';
			return;
		}
		if (tagNameExists(tags, name, editingTag.id)) {
			editorError = 'A tag with that name already exists.';
			return;
		}
		if (name === editingTag.name && draftColor === editingTag.color) {
			closeEditor();
			return;
		}

		saving = true;
		editorError = null;
		try {
			const updated = await updateTag(editingTag.id, { name, color: draftColor });
			onTagsChange(sortTags(tags.map((tag) => (tag.id === updated.id ? updated : tag))));
			closeEditor();
		} catch (cause) {
			editorError = errorMessage(cause, 'The tag could not be updated.');
		} finally {
			saving = false;
		}
	}

	function handleInputKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter') {
			event.preventDefault();
			if (!editorOpen) {
				void createFromQuery();
			}
		} else if (event.key === 'Escape') {
			event.preventDefault();
			open = false;
			closeEditor();
		} else if (event.key === 'Backspace' && query === '' && selectedIDs.length > 0) {
			event.preventDefault();
			onSelectedChange(selectedIDs.slice(0, -1));
		} else if (event.key === 'ArrowDown') {
			event.preventDefault();
			open = true;
		}
	}

	function handleEditorKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter') {
			event.preventDefault();
			void saveEditor();
		} else if (event.key === 'Escape') {
			event.preventDefault();
			closeEditor();
		}
	}
</script>

<fieldset>
	<legend class="text-sm font-medium">Tags</legend>
	<p class="mt-1 text-xs text-ink-muted">
		Search existing tags or type a name to add one. Click a selected tag to rename it or change its
		color.
	</p>

	<div class="relative mt-1.5" bind:this={containerEl}>
		<div
			class="flex min-h-11 flex-wrap items-center gap-1.5 rounded-xl border border-line bg-surface px-2 py-1.5 transition-colors focus-within:border-accent"
		>
			{#each selectedTags as tag (tag.id)}
				<span
					class="inline-flex items-center overflow-hidden rounded-lg border bg-surface-muted/60 text-xs"
					style:border-color={tag.color || 'var(--token-line)'}
				>
					<button
						type="button"
						class="inline-flex items-center gap-1.5 py-1 pr-1 pl-2 font-medium text-ink transition-colors hover:bg-surface"
						title={`Edit ${tag.name}`}
						onclick={() => void openEditor(tag)}
					>
						<span
							class="size-2 rounded-full"
							style:background-color={tag.color || 'var(--token-accent)'}
							aria-hidden="true"
						></span>
						{tag.name}
					</button>
					<button
						type="button"
						class="px-1.5 py-1 text-sm leading-none text-ink-muted transition-colors hover:text-negative"
						aria-label={`Remove ${tag.name}`}
						onclick={() => deselectTag(tag.id)}
					>
						×
					</button>
				</span>
			{/each}

			<input
				bind:this={inputEl}
				type="text"
				maxlength="60"
				autocomplete="off"
				class="min-w-24 flex-1 bg-transparent px-1 py-1 text-sm outline-none placeholder:text-ink-muted"
				placeholder={selectedTags.length === 0 ? 'Search or add a tag…' : 'Add another tag…'}
				aria-label="Search or create a tag"
				role="combobox"
				aria-autocomplete="list"
				aria-controls="tag-picker-options"
				aria-expanded={open}
				bind:value={query}
				onfocus={() => {
					open = true;
					if (editorOpen) {
						closeEditor();
					}
				}}
				oninput={() => {
					open = true;
					pickerError = null;
				}}
				onkeydown={handleInputKeydown}
			/>
		</div>

		{#if pickerError}
			<p role="alert" class="mt-1.5 text-xs text-negative">{pickerError}</p>
		{/if}

		{#if open}
			<div
				id="tag-picker-options"
				class="absolute z-20 mt-1 max-h-64 w-full overflow-y-auto rounded-xl border border-line bg-surface p-1 shadow-card"
			>
				{#if matchingTags.length > 0}
					<ul class="space-y-0.5" aria-label="Available tags">
						{#each matchingTags as tag (tag.id)}
							<li>
								<button
									type="button"
									class="flex w-full items-center gap-2 rounded-lg px-2 py-2 text-left text-sm text-ink transition-colors hover:bg-surface-muted"
									onclick={() => selectTag(tag)}
								>
									<span
										class="size-2.5 rounded-full"
										style:background-color={tag.color || 'var(--token-accent)'}
										aria-hidden="true"
									></span>
									<span class="flex-1 truncate">{tag.name}</span>
								</button>
							</li>
						{/each}
					</ul>
				{/if}

				{#if canCreate}
					<button
						type="button"
						class="flex w-full items-center gap-2 rounded-lg px-2 py-2 text-left text-sm font-medium text-accent transition-colors hover:bg-surface-muted disabled:cursor-wait disabled:opacity-60"
						disabled={creating}
						onclick={() => void createFromQuery()}
					>
						<Icon name="plus" class="size-4" />
						{creating ? 'Creating…' : `Create “${trimmedQuery}”`}
					</button>
				{/if}

				{#if matchingTags.length === 0 && !canCreate}
					<p class="px-2 py-2 text-sm text-ink-muted">
						{#if tags.length === 0}
							No tags yet. Type a name to create one.
						{:else if trimmedQuery === ''}
							All tags are selected. Type a name to add another.
						{:else if exactMatch && selectedIDs.includes(exactMatch.id)}
							“{exactMatch.name}” is already selected.
						{:else}
							No tags match. Press Enter to create one.
						{/if}
					</p>
				{/if}
			</div>
		{/if}

		{#if editorOpen && editingTag}
			<div
				class="absolute z-30 mt-1 w-full rounded-xl border border-line bg-surface p-4 shadow-card"
				role="dialog"
				aria-label={`Edit ${editingTag.name}`}
			>
				<label for="tag-name" class="text-sm font-medium">Tag name</label>
				<input
					bind:this={nameInputEl}
					id="tag-name"
					type="text"
					maxlength="60"
					autocomplete="off"
					class="mt-1.5 min-h-10 w-full rounded-lg border border-line bg-surface px-3 text-sm outline-none focus:border-accent"
					bind:value={draftName}
					onkeydown={handleEditorKeydown}
				/>

				<p class="mt-3 text-sm font-medium">Color</p>
				<div class="mt-1.5 flex flex-wrap gap-2" role="group" aria-label="Tag color">
					{#each TAG_COLORS as color (color.value)}
						<button
							type="button"
							class="size-7 rounded-full border-2 transition-transform {draftColor === color.value
								? 'scale-110 border-ink'
								: 'border-transparent hover:scale-105'}"
							style:background-color={color.value}
							aria-label={color.name}
							aria-pressed={draftColor === color.value}
							title={color.name}
							onclick={() => (draftColor = color.value)}
						></button>
					{/each}
				</div>

				{#if editorError}
					<p role="alert" class="mt-2 text-xs text-negative">{editorError}</p>
				{/if}

				<div class="mt-4 flex flex-wrap items-center gap-2">
					<button
						type="button"
						class="inline-flex min-h-9 items-center justify-center rounded-lg bg-accent px-3 text-sm font-semibold text-on-accent transition-colors hover:bg-accent-strong disabled:cursor-wait disabled:opacity-60"
						disabled={saving}
						onclick={() => void saveEditor()}
					>
						{saving ? 'Saving…' : 'Save tag'}
					</button>
					<button
						type="button"
						class="inline-flex min-h-9 items-center justify-center rounded-lg border border-line bg-surface px-3 text-sm font-semibold text-ink transition-colors hover:bg-surface-muted"
						onclick={closeEditor}
					>
						Cancel
					</button>
				</div>
			</div>
		{/if}
	</div>
</fieldset>
