import type { Dashboard, DateRange } from './types';

export class ApiError extends Error {
	readonly status: number;

	constructor(status: number, message: string) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
	}
}

type Fetcher = (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;

async function getJSON<T>(path: string, fetcher: Fetcher): Promise<T> {
	const response = await fetcher(path, { headers: { Accept: 'application/json' } });
	if (!response.ok) {
		throw new ApiError(response.status, await errorMessage(response));
	}
	return (await response.json()) as T;
}

async function errorMessage(response: Response): Promise<string> {
	try {
		const body = (await response.json()) as { detail?: unknown; title?: unknown };
		if (typeof body.detail === 'string' && body.detail !== '') {
			return body.detail;
		}
		if (typeof body.title === 'string' && body.title !== '') {
			return body.title;
		}
	} catch {
		// The response was not JSON; fall through to the status text.
	}
	return `Request failed with status ${response.status}`;
}

/** Fetch the dashboard read model for an inclusive date range. */
export function getDashboard(range: DateRange, fetcher: Fetcher = fetch): Promise<Dashboard> {
	const params = new URLSearchParams({ from: range.from, to: range.to });
	return getJSON<Dashboard>(`/api/v1/dashboard?${params.toString()}`, fetcher);
}
