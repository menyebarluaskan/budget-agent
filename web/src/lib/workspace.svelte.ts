import { setActiveWorkspaceId } from './activeWorkspace';
import { listWorkspaces } from './api/client';
import type { Workspace } from './api/types';

const STORAGE_KEY = 'budget-agent:active-workspace';

let workspaces = $state<Workspace[]>([]);
let loaded = $state(false);

export function getWorkspaces(): Workspace[] {
	return workspaces;
}

export function isLoaded(): boolean {
	return loaded;
}

export function hasMultiple(): boolean {
	return workspaces.length > 1;
}

export async function loadWorkspaces(): Promise<void> {
	try {
		const list = await listWorkspaces();
		workspaces = list;
		const stored = typeof localStorage !== 'undefined' ? localStorage.getItem(STORAGE_KEY) : null;
		let chosen: string | null = null;
		if (stored && list.some((w) => w.id === stored)) {
			chosen = stored;
		} else if (list.length > 0) {
			chosen = list[0].id;
		}
		setActiveWorkspaceId(chosen);
		if (chosen && typeof localStorage !== 'undefined') {
			localStorage.setItem(STORAGE_KEY, chosen);
		}
	} catch {
		// ignore
	} finally {
		loaded = true;
	}
}

export function setActiveWorkspace(id: string): void {
	setActiveWorkspaceId(id);
	if (typeof localStorage !== 'undefined') {
		localStorage.setItem(STORAGE_KEY, id);
	}
}
