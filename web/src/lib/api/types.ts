/** Shapes returned by the API. Amounts are integer minor units. */

export interface Account {
	id: string;
	workspace_id: string;
	name: string;
	type: 'wallet' | 'savings' | 'credit';
	currency: string;
	opening_balance_minor: number;
	archived: boolean;
	created_at: string;
	updated_at: string;
	balance_minor: number;
}

export interface Tag {
	id: string;
	workspace_id: string;
	name: string;
	color: string;
	created_at: string;
}

export interface TagInput {
	name: string;
	color?: string;
}

export type TagUpdate = Partial<TagInput>;

export type TransactionSource = 'manual' | 'whatsapp' | 'mcp' | 'receipt' | 'import';

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

export type TransactionSort = 'occurred_at' | '-occurred_at' | 'amount_minor' | '-amount_minor';

export interface TransactionFilters {
	accountId?: string;
	tagIds?: string[];
	from?: string;
	to?: string;
	minAmountMinor?: number;
	maxAmountMinor?: number;
	q?: string;
	source?: string;
	includeTransfers?: boolean;
	sort?: TransactionSort;
	limit?: number;
	cursor?: string;
}

export interface TransactionPage {
	transactions: Transaction[];
	next_cursor?: string;
}

export interface TransactionInput {
	account_id: string;
	amount_minor: number;
	occurred_at: string;
	description: string;
	notes: string;
	source?: TransactionSource;
	tag_ids: string[];
}

export interface TransactionUpdate {
	account_id?: string;
	amount_minor?: number;
	occurred_at?: string;
	description?: string;
	notes?: string;
	tag_ids?: string[];
}

export type AccountType = 'wallet' | 'savings' | 'credit';

export interface AccountInput {
	name: string;
	type: AccountType;
	currency: string;
	opening_balance_minor: number;
}

export interface AccountUpdate {
	name?: string;
	archived?: boolean;
	opening_balance_minor?: number;
}

export interface TransferInput {
	from_account_id: string;
	to_account_id: string;
	amount_minor: number;
	occurred_at: string;
	description: string;
	notes: string;
}

export interface Transfer {
	transfer_id: string;
	from: Transaction;
	to: Transaction;
}

export type BudgetPeriod = 'weekly' | 'monthly';

export interface Budget {
	id: string;
	workspace_id: string;
	name: string;
	amount_minor: number;
	period: BudgetPeriod;
	start_date: string;
	end_date?: string | null;
	rollover: boolean;
	account_id?: string | null;
	tag_ids: string[];
	created_at: string;
	updated_at: string;
}

export interface BudgetInput {
	name: string;
	amount_minor: number;
	period: BudgetPeriod;
	start_date: string;
	end_date?: string | null;
	rollover: boolean;
	account_id?: string | null;
	tag_ids: string[];
}

export type BudgetUpdate = Partial<BudgetInput>;

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

export interface BudgetSummary {
	tag_id: string;
	tag_name: string;
	color: string;
	spent_minor: number;
}

export interface DateRange {
	from: string;
	to: string;
}
