export namespace domain {

	export class HintResult {
	    level: number;
	    text: string;
	    sourceSquare?: string;
	    targetSquare?: string;
	    canReveal: boolean;

	    static createFrom(source: any = {}) {
	        return new HintResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.level = source["level"];
	        this.text = source["text"];
	        this.sourceSquare = source["sourceSquare"];
	        this.targetSquare = source["targetSquare"];
	        this.canReveal = source["canReveal"];
	    }
	}
	export class SessionSummary {
	    total: number;
	    firstTry: number;
	    retried: number;
	    usedHint: number;
	    revealed: number;
	    unavailable: number;

	    static createFrom(source: any = {}) {
	        return new SessionSummary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.firstTry = source["firstTry"];
	        this.retried = source["retried"];
	        this.usedHint = source["usedHint"];
	        this.revealed = source["revealed"];
	        this.unavailable = source["unavailable"];
	    }
	}
	export class PuzzleView {
	    fingerprint: string;
	    sourceFen?: string;
	    displayedFen: string;
	    currentFen: string;
	    preludeUci?: string;
	    solver: string;
	    currentPath: number[];
	    puzzleNumber: number;
	    puzzleTotal: number;
	    hintLevel: number;
	    incorrectMoves: number;
	    canReveal: boolean;

	    static createFrom(source: any = {}) {
	        return new PuzzleView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fingerprint = source["fingerprint"];
	        this.sourceFen = source["sourceFen"];
	        this.displayedFen = source["displayedFen"];
	        this.currentFen = source["currentFen"];
	        this.preludeUci = source["preludeUci"];
	        this.solver = source["solver"];
	        this.currentPath = source["currentPath"];
	        this.puzzleNumber = source["puzzleNumber"];
	        this.puzzleTotal = source["puzzleTotal"];
	        this.hintLevel = source["hintLevel"];
	        this.incorrectMoves = source["incorrectMoves"];
	        this.canReveal = source["canReveal"];
	    }
	}
	export class SessionView {
	    sessionId: string;
	    mode: string;
	    status: string;
	    currentIndex: number;
	    total: number;
	    current?: PuzzleView;
	    summary?: SessionSummary;

	    static createFrom(source: any = {}) {
	        return new SessionView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.mode = source["mode"];
	        this.status = source["status"];
	        this.currentIndex = source["currentIndex"];
	        this.total = source["total"];
	        this.current = this.convertValues(source["current"], PuzzleView);
	        this.summary = this.convertValues(source["summary"], SessionSummary);
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
	export class MoveResult {
	    session: SessionView;
	    correct: boolean;
	    puzzleCompleted: boolean;
	    message?: string;

	    static createFrom(source: any = {}) {
	        return new MoveResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.session = this.convertValues(source["session"], SessionView);
	        this.correct = source["correct"];
	        this.puzzleCompleted = source["puzzleCompleted"];
	        this.message = source["message"];
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

export namespace importjob {

	export class Result {
	    jobId: string;
	    status: string;
	    report: puzzles.ImportReport;
	    error?: string;

	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jobId = source["jobId"];
	        this.status = source["status"];
	        this.report = this.convertValues(source["report"], puzzles.ImportReport);
	        this.error = source["error"];
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

export namespace puzzles {

	export class Rejection {
	    ordinal: number;
	    reason: string;

	    static createFrom(source: any = {}) {
	        return new Rejection(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ordinal = source["ordinal"];
	        this.reason = source["reason"];
	    }
	}
	export class ImportReport {
	    accepted: number;
	    duplicates: number;
	    rejected: number;
	    examples: Rejection[];

	    static createFrom(source: any = {}) {
	        return new ImportReport(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.accepted = source["accepted"];
	        this.duplicates = source["duplicates"];
	        this.rejected = source["rejected"];
	        this.examples = this.convertValues(source["examples"], Rejection);
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
