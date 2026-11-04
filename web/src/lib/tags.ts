import type { Tag } from './api/types';

export interface TagColor {
	name: string;
	value: string;
}

/** Fixed palette used by the tag picker so colors stay consistent across the UI. */
export const TAG_COLORS: readonly TagColor[] = [
	{ name: 'Red', value: '#ef4444' },
	{ name: 'Orange', value: '#f97316' },
	{ name: 'Amber', value: '#f59e0b' },
	{ name: 'Lime', value: '#84cc16' },
	{ name: 'Green', value: '#22c55e' },
	{ name: 'Teal', value: '#14b8a6' },
	{ name: 'Cyan', value: '#06b6d4' },
	{ name: 'Blue', value: '#3b82f6' },
	{ name: 'Indigo', value: '#6366f1' },
	{ name: 'Violet', value: '#8b5cf6' },
	{ name: 'Pink', value: '#ec4899' },
	{ name: 'Slate', value: '#64748b' }
];

/** Case-insensitive duplicate check, optionally ignoring the tag being edited. */
export function tagNameExists(tags: Tag[], name: string, excludeID?: string): boolean {
	const normalized = name.trim().toLocaleLowerCase();
	if (normalized === '') {
		return false;
	}
	return tags.some((tag) => tag.id !== excludeID && tag.name.toLocaleLowerCase() === normalized);
}

/** Pick the next unused palette color, cycling when every color is in use. */
export function nextTagColor(tags: Tag[]): string {
	const used = new Set(tags.map((tag) => tag.color));
	const available = TAG_COLORS.find((color) => !used.has(color.value));
	return available?.value ?? TAG_COLORS[tags.length % TAG_COLORS.length].value;
}

/** Sort tags alphabetically, ignoring case. */
export function sortTags(tags: Tag[]): Tag[] {
	return [...tags].sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: 'base' }));
}
