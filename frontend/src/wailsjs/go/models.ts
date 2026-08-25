export namespace application {
	
	export class AccountInput {
	    name: string;
	    type: string;
	    openingBalanceCents: number;
	    openingDate: string;
	    creditLimitCents: number;
	    closingDay: number;
	    dueDay: number;
	    openingDebtCents: number;
	    openingDebtDueDate: string;
	
	    static createFrom(source: any = {}) {
	        return new AccountInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.openingBalanceCents = source["openingBalanceCents"];
	        this.openingDate = source["openingDate"];
	        this.creditLimitCents = source["creditLimitCents"];
	        this.closingDay = source["closingDay"];
	        this.dueDay = source["dueDay"];
	        this.openingDebtCents = source["openingDebtCents"];
	        this.openingDebtDueDate = source["openingDebtDueDate"];
	    }
	}
	export class BankStatementImportResult {
	    bank: string;
	    importedCount: number;
	    duplicateCount: number;
	    ignoredCount: number;
	
	    static createFrom(source: any = {}) {
	        return new BankStatementImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bank = source["bank"];
	        this.importedCount = source["importedCount"];
	        this.duplicateCount = source["duplicateCount"];
	        this.ignoredCount = source["ignoredCount"];
	    }
	}
	export class BankStatementInput {
	    accountId: string;
	    bank: string;
	    fileName: string;
	    base64Pdf: string;
	
	    static createFrom(source: any = {}) {
	        return new BankStatementInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.accountId = source["accountId"];
	        this.bank = source["bank"];
	        this.fileName = source["fileName"];
	        this.base64Pdf = source["base64Pdf"];
	    }
	}
	export class Bootstrap {
	    profile?: domain.Profile;
	    setup: boolean;
	    accounts: domain.Account[];
	    categories: domain.Category[];
	    dashboard: domain.Dashboard;
	    theme: string;
	
	    static createFrom(source: any = {}) {
	        return new Bootstrap(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profile = this.convertValues(source["profile"], domain.Profile);
	        this.setup = source["setup"];
	        this.accounts = this.convertValues(source["accounts"], domain.Account);
	        this.categories = this.convertValues(source["categories"], domain.Category);
	        this.dashboard = this.convertValues(source["dashboard"], domain.Dashboard);
	        this.theme = source["theme"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ConfirmFixedExpenseOccurrenceInput {
	    amountCents: number;
	    occurrenceDate: string;
	
	    static createFrom(source: any = {}) {
	        return new ConfirmFixedExpenseOccurrenceInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.amountCents = source["amountCents"];
	        this.occurrenceDate = source["occurrenceDate"];
	    }
	}
	export class CreditCardPaymentInput {
	    accountId: string;
	    amountCents: number;
	    occurrenceDate: string;
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.accountId = source["accountId"];
	        this.amountCents = source["amountCents"];
	        this.occurrenceDate = source["occurrenceDate"];
	    }
	}
	export class CreditCardsOverview {
	    cards: domain.CreditCardSummary[];
	    invoices: domain.CreditCardInvoice[];
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cards = (source["cards"] || []).map((item:any)=>new domain.CreditCardSummary(item));
	        this.invoices = (source["invoices"] || []).map((item:any)=>new domain.CreditCardInvoice(item));
	    }
	}
	export class FixedExpenseInput {
	    description: string;
	    amountCents: number;
	    dueDay: number;
	    accountId: string;
	    categoryId: string;
	
	    static createFrom(source: any = {}) {
	        return new FixedExpenseInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.description = source["description"];
	        this.amountCents = source["amountCents"];
	        this.dueDay = source["dueDay"];
	        this.accountId = source["accountId"];
	        this.categoryId = source["categoryId"];
	    }
	}
	export class FixedExpensesOverview {
	    expenses: domain.FixedExpense[];
	    occurrences: domain.FixedExpenseOccurrence[];
	
	    static createFrom(source: any = {}) {
	        return new FixedExpensesOverview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.expenses = this.convertValues(source["expenses"], domain.FixedExpense);
	        this.occurrences = this.convertValues(source["occurrences"], domain.FixedExpenseOccurrence);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OnboardingInput {
	    displayName: string;
	    currency: string;
	    theme: string;
	    firstAccount: AccountInput;
	
	    static createFrom(source: any = {}) {
	        return new OnboardingInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.displayName = source["displayName"];
	        this.currency = source["currency"];
	        this.theme = source["theme"];
	        this.firstAccount = this.convertValues(source["firstAccount"], AccountInput);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TransactionInput {
	    kind: string;
	    amountCents: number;
	    accountId: string;
	    destinationAccountId: string;
	    categoryId: string;
	    description: string;
	    occurrenceDate: string;
	    installmentCount: number;
	
	    static createFrom(source: any = {}) {
	        return new TransactionInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.amountCents = source["amountCents"];
	        this.accountId = source["accountId"];
	        this.destinationAccountId = source["destinationAccountId"];
	        this.categoryId = source["categoryId"];
	        this.description = source["description"];
	        this.occurrenceDate = source["occurrenceDate"];
	        this.installmentCount = source["installmentCount"];
	    }
	}

}

export namespace domain {
	export class CreditCardPayment {
	    id!: string; invoiceId!: string; accountId!: string; accountName!: string; transactionId!: string;
	    amountCents!: number; occurrenceDate!: string; createdAt!: string;
	    constructor(source: any = {}) { Object.assign(this, typeof source === 'string' ? JSON.parse(source) : source); }
	}
	
	export class Account {
	    id: string;
	    name: string;
	    type: string;
	    openingBalanceCents: number;
	    openingDate: string;
	    createdAt: string;
	    currentBalanceCents: number;
	    creditLimitCents: number;
	    closingDay: number;
	    dueDay: number;
	
	    static createFrom(source: any = {}) {
	        return new Account(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.openingBalanceCents = source["openingBalanceCents"];
	        this.openingDate = source["openingDate"];
	        this.createdAt = source["createdAt"];
	        this.currentBalanceCents = source["currentBalanceCents"];
	        this.creditLimitCents = source["creditLimitCents"];
	        this.closingDay = source["closingDay"];
	        this.dueDay = source["dueDay"];
	    }
	}
	export class AccountAllocation {
	    accountId: string;
	    accountName: string;
	    balanceCents: number;
	
	    static createFrom(source: any = {}) {
	        return new AccountAllocation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.accountId = source["accountId"];
	        this.accountName = source["accountName"];
	        this.balanceCents = source["balanceCents"];
	    }
	}
	export class BalanceHistoryPoint {
	    month: string;
	    label: string;
	    balanceCents: number;
	
	    static createFrom(source: any = {}) {
	        return new BalanceHistoryPoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.month = source["month"];
	        this.label = source["label"];
	        this.balanceCents = source["balanceCents"];
	    }
	}
	export class Category {
	    id: string;
	    name: string;
	    kind: string;
	
	    static createFrom(source: any = {}) {
	        return new Category(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	    }
	}
	export class Transaction {
	    id: string;
	    kind: string;
	    amountCents: number;
	    accountId: string;
	    accountName: string;
	    destinationAccountId?: string;
	    destinationAccountName?: string;
	    categoryId?: string;
	    categoryName?: string;
	    description: string;
	    occurrenceDate: string;
	    createdAt: string;
	    updatedAt: string;
	    deletedAt?: string;
	    fixedExpenseOccurrenceId?: string;
	    automaticImport: boolean;
	    importBank?: string;
	    installmentCount: number;
	    invoicePaymentId?: string;
	
	    static createFrom(source: any = {}) {
	        return new Transaction(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.amountCents = source["amountCents"];
	        this.accountId = source["accountId"];
	        this.accountName = source["accountName"];
	        this.destinationAccountId = source["destinationAccountId"];
	        this.destinationAccountName = source["destinationAccountName"];
	        this.categoryId = source["categoryId"];
	        this.categoryName = source["categoryName"];
	        this.description = source["description"];
	        this.occurrenceDate = source["occurrenceDate"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.deletedAt = source["deletedAt"];
	        this.fixedExpenseOccurrenceId = source["fixedExpenseOccurrenceId"];
	        this.automaticImport = source["automaticImport"];
	        this.importBank = source["importBank"];
	        this.installmentCount = source["installmentCount"];
	        this.invoicePaymentId = source["invoicePaymentId"];
	    }
	}
	export class CreditCardInstallment {
	    id!: string; invoiceId!: string; transactionId?: string; description!: string; amountCents!: number;
	    installmentNumber!: number; installmentCount!: number; openingDebt!: boolean;
	    constructor(source:any={}) { Object.assign(this, typeof source==='string'?JSON.parse(source):source); }
	}
	export class CreditCardInvoice {
	    id!: string; accountId!: string; accountName!: string; referenceMonth!: string; closingDate!: string; dueDate!: string;
	    status!: string; chargesCents!: number; carryForwardCents!: number; paidCents!: number; outstandingCents!: number;
	    installments!: CreditCardInstallment[]; payments!: CreditCardPayment[];
	    constructor(source:any={}) { if(typeof source==='string')source=JSON.parse(source);Object.assign(this,source);this.installments=(source.installments||[]).map((item:any)=>new CreditCardInstallment(item));this.payments=(source.payments||[]).map((item:any)=>new CreditCardPayment(item)); }
	}
	export class CreditCardSummary {
	    account!: Account; outstandingCents!: number; availableLimitCents!: number; currentInvoice?: CreditCardInvoice;
	    constructor(source:any={}) { if(typeof source==='string')source=JSON.parse(source);this.account=new Account(source.account);this.outstandingCents=source.outstandingCents;this.availableLimitCents=source.availableLimitCents;this.currentInvoice=source.currentInvoice?new CreditCardInvoice(source.currentInvoice):undefined; }
	}
	export class Dashboard {
	    availableBalanceCents: number;
	    totalBalanceCents: number;
	    pendingFixedExpensesCents: number;
	    pendingFixedExpenseCount: number;
	    monthlyIncomeCents: number;
	    monthlyExpenseCents: number;
	    recentTransactions: Transaction[];
	    balanceHistory: BalanceHistoryPoint[];
	    accountAllocations: AccountAllocation[];
	    hasNegativeBalance: boolean;
	    creditCardDebtCents: number;
	    upcomingInvoices: CreditCardInvoice[];
	
	    static createFrom(source: any = {}) {
	        return new Dashboard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.availableBalanceCents = source["availableBalanceCents"];
	        this.totalBalanceCents = source["totalBalanceCents"];
	        this.pendingFixedExpensesCents = source["pendingFixedExpensesCents"];
	        this.pendingFixedExpenseCount = source["pendingFixedExpenseCount"];
	        this.monthlyIncomeCents = source["monthlyIncomeCents"];
	        this.monthlyExpenseCents = source["monthlyExpenseCents"];
	        this.recentTransactions = this.convertValues(source["recentTransactions"], Transaction);
	        this.balanceHistory = this.convertValues(source["balanceHistory"], BalanceHistoryPoint);
	        this.accountAllocations = this.convertValues(source["accountAllocations"], AccountAllocation);
	        this.hasNegativeBalance = source["hasNegativeBalance"];
	        this.creditCardDebtCents = source["creditCardDebtCents"];
	        this.upcomingInvoices = this.convertValues(source["upcomingInvoices"], CreditCardInvoice);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FixedExpense {
	    id: string;
	    description: string;
	    amountCents: number;
	    dueDay: number;
	    accountId: string;
	    accountName: string;
	    categoryId: string;
	    categoryName: string;
	    archivedAt?: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new FixedExpense(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.description = source["description"];
	        this.amountCents = source["amountCents"];
	        this.dueDay = source["dueDay"];
	        this.accountId = source["accountId"];
	        this.accountName = source["accountName"];
	        this.categoryId = source["categoryId"];
	        this.categoryName = source["categoryName"];
	        this.archivedAt = source["archivedAt"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class FixedExpenseOccurrence {
	    id: string;
	    fixedExpenseId: string;
	    referenceMonth: string;
	    dueDate: string;
	    description: string;
	    expectedAmountCents: number;
	    accountId: string;
	    accountName: string;
	    categoryId: string;
	    categoryName: string;
	    status: string;
	    transactionId?: string;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new FixedExpenseOccurrence(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.fixedExpenseId = source["fixedExpenseId"];
	        this.referenceMonth = source["referenceMonth"];
	        this.dueDate = source["dueDate"];
	        this.description = source["description"];
	        this.expectedAmountCents = source["expectedAmountCents"];
	        this.accountId = source["accountId"];
	        this.accountName = source["accountName"];
	        this.categoryId = source["categoryId"];
	        this.categoryName = source["categoryName"];
	        this.status = source["status"];
	        this.transactionId = source["transactionId"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class Profile {
	    displayName: string;
	    currency: string;
	    theme: string;
	    onboardingStatus: string;
	    balancesHidden: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Profile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.displayName = source["displayName"];
	        this.currency = source["currency"];
	        this.theme = source["theme"];
	        this.onboardingStatus = source["onboardingStatus"];
	        this.balancesHidden = source["balancesHidden"];
	    }
	}

}
