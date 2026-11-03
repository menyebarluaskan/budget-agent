/** Shapes returned by the dashboard endpoint. Amounts are integer minor units. */

export interface Transaction {
	id: string;
	workspace_id: string;
	account_id: string;
	amount_minor: number;
	occurred_at: string;
	description: string;
	notes: string;
	source: string;
	tag_ids: string[];
	transfer_id?: string | null;
	created_at: string;
	updated_at: string;
}

export interface BudgetProgress {
	id: string;
	name: string;
	period: 'weekly' | 'monthly';
	amount_minor: number;
	spent_minor: number;
	remaining_minor: number;
	rollover: boolean;
	period_start: string;
	period_end: string;
}

export interface Dashboard {
	from: string;
	to: string;
	currency: string;
	total_balance_minor: number;
	account_count: number;
	income_minor: number;
	expense_minor: number;
	net_minor: number;
	budgets: BudgetProgress[];
	recent_transactions: Transaction[];
}

export interface DateRange {
	from: string;
	to: string;
}
