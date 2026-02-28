export namespace main {
	
	export class DailyJournalEntry {
	    id: string;
	    date: string;
	    ruleExecuted: string;
	    moodStable: string;
	    didRecord: string;
	    prepared: string;
	    noFOMO: string;
	    totalPnL: number;
	    note: string;
	    createdAt: number;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new DailyJournalEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.date = source["date"];
	        this.ruleExecuted = source["ruleExecuted"];
	        this.moodStable = source["moodStable"];
	        this.didRecord = source["didRecord"];
	        this.prepared = source["prepared"];
	        this.noFOMO = source["noFOMO"];
	        this.totalPnL = source["totalPnL"];
	        this.note = source["note"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class ErrorTypeEntry {
	    id: string;
	    reason: string;
	    count: number;
	    exitReason: string;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new ErrorTypeEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.reason = source["reason"];
	        this.count = source["count"];
	        this.exitReason = source["exitReason"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class TradeSummary {
	    totalTrades: number;
	    wins: number;
	    losses: number;
	    breakeven: number;
	    winRate: number;
	    totalPnL: number;
	    avgWin: number;
	    avgLoss: number;
	    profitLossRatio: number;
	    averagePosition: number;
	    averageHoldRange: number;
	
	    static createFrom(source: any = {}) {
	        return new TradeSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalTrades = source["totalTrades"];
	        this.wins = source["wins"];
	        this.losses = source["losses"];
	        this.breakeven = source["breakeven"];
	        this.winRate = source["winRate"];
	        this.totalPnL = source["totalPnL"];
	        this.avgWin = source["avgWin"];
	        this.avgLoss = source["avgLoss"];
	        this.profitLossRatio = source["profitLossRatio"];
	        this.averagePosition = source["averagePosition"];
	        this.averageHoldRange = source["averageHoldRange"];
	    }
	}
	export class TradeEntry {
	    id: string;
	    date: string;
	    note: string;
	    entryReason: string;
	    tradeType: string;
	    exitReason: string;
	    supplement: string;
	    positionSize: number;
	    direction: string;
	    entryPrice: number;
	    exitPrice1: number;
	    exitPrice2: number;
	    pnl: number;
	    errorReason: string;
	    createdAt: number;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new TradeEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.date = source["date"];
	        this.note = source["note"];
	        this.entryReason = source["entryReason"];
	        this.tradeType = source["tradeType"];
	        this.exitReason = source["exitReason"];
	        this.supplement = source["supplement"];
	        this.positionSize = source["positionSize"];
	        this.direction = source["direction"];
	        this.entryPrice = source["entryPrice"];
	        this.exitPrice1 = source["exitPrice1"];
	        this.exitPrice2 = source["exitPrice2"];
	        this.pnl = source["pnl"];
	        this.errorReason = source["errorReason"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class TradeDashboard {
	    trades: TradeEntry[];
	    errorTypes: ErrorTypeEntry[];
	    journals: DailyJournalEntry[];
	    summary: TradeSummary;
	    dataFile: string;
	    defaultImportPath: string;
	    defaultExportDir: string;
	
	    static createFrom(source: any = {}) {
	        return new TradeDashboard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.trades = this.convertValues(source["trades"], TradeEntry);
	        this.errorTypes = this.convertValues(source["errorTypes"], ErrorTypeEntry);
	        this.journals = this.convertValues(source["journals"], DailyJournalEntry);
	        this.summary = this.convertValues(source["summary"], TradeSummary);
	        this.dataFile = source["dataFile"];
	        this.defaultImportPath = source["defaultImportPath"];
	        this.defaultExportDir = source["defaultExportDir"];
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
	

}

