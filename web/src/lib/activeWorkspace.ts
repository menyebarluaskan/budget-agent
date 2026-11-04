let current: string | null = null;

export function getActiveWorkspaceId(): string | null {
	return current;
}

export function setActiveWorkspaceId(id: string | null): void {
	current = id;
}
