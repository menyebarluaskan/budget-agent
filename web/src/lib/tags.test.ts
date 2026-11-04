import { describe, expect, it } from 'vitest';
import type { Tag } from './api/types';
import { nextTagColor, sortTags, TAG_COLORS, tagNameExists } from './tags';

function tag(id: string, name: string, color = TAG_COLORS[0].value): Tag {
	return {
		id,
		workspace_id: 'default',
		name,
		color,
		created_at: '2026-01-01T00:00:00Z'
	};
}

describe('tagNameExists', () => {
	it('matches names case-insensitively after trimming', () => {
		const tags = [tag('food', 'Food')];

		expect(tagNameExists(tags, ' food ')).toBe(true);
		expect(tagNameExists(tags, 'FOOD')).toBe(true);
		expect(tagNameExists(tags, 'Transport')).toBe(false);
	});

	it('ignores the tag being edited', () => {
		const tags = [tag('food', 'Food'), tag('transport', 'Transport')];

		expect(tagNameExists(tags, 'food', 'food')).toBe(false);
		expect(tagNameExists(tags, 'transport', 'food')).toBe(true);
	});

	it('does not report blank names', () => {
		expect(tagNameExists([tag('food', 'Food')], '   ')).toBe(false);
	});
});

describe('nextTagColor', () => {
	it('returns the first unused palette color', () => {
		expect(nextTagColor([tag('food', 'Food', TAG_COLORS[0].value)])).toBe(TAG_COLORS[1].value);
	});

	it('cycles when every palette color is used', () => {
		const tags = TAG_COLORS.map((color, index) => tag(String(index), `Tag ${index}`, color.value));

		expect(nextTagColor(tags)).toBe(TAG_COLORS[tags.length % TAG_COLORS.length].value);
	});
});

describe('sortTags', () => {
	it('sorts alphabetically without case sensitivity', () => {
		const sorted = sortTags([tag('b', 'banana'), tag('a', 'Apple'), tag('c', 'cherry')]);

		expect(sorted.map((item) => item.name)).toEqual(['Apple', 'banana', 'cherry']);
	});
});
