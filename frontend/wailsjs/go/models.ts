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

export namespace main {

	export class RecoveryState {
	    required: boolean;
	    path?: string;
	    detail?: string;

	    static createFrom(source: any = {}) {
	        return new RecoveryState(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.required = source["required"];
	        this.path = source["path"];
	        this.detail = source["detail"];
	    }
	}

}

export namespace profile {

	export class PracticeSource {
	    id: string;
	    kind: string;
	    minimumRating: number;
	    maximumRating: number;
	    hasRatingRange: boolean;
	    maximumPlies: number;

	    static createFrom(source: any = {}) {
	        return new PracticeSource(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.minimumRating = source["minimumRating"];
	        this.maximumRating = source["maximumRating"];
	        this.hasRatingRange = source["hasRatingRange"];
	        this.maximumPlies = source["maximumPlies"];
	    }
	}
	export class PracticeFilters {
	    sources: PracticeSource[];
	    themes: string[];
	    maximumSolutionPlies: number;

	    static createFrom(source: any = {}) {
	        return new PracticeFilters(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sources = this.convertValues(source["sources"], PracticeSource);
	        this.themes = source["themes"];
	        this.maximumSolutionPlies = source["maximumSolutionPlies"];
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

	export class RatingPoint {
	    rating: number;
	    recordedAt: number;

	    static createFrom(source: any = {}) {
	        return new RatingPoint(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rating = source["rating"];
	        this.recordedAt = source["recordedAt"];
	    }
	}
	export class RecentSession {
	    sessionId: string;
	    mode: string;
	    status: string;
	    updatedAt: number;
	    total: number;
	    completed: number;
	    firstTry: number;
	    usedHint: number;
	    revealed: number;

	    static createFrom(source: any = {}) {
	        return new RecentSession(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.mode = source["mode"];
	        this.status = source["status"];
	        this.updatedAt = source["updatedAt"];
	        this.total = source["total"];
	        this.completed = source["completed"];
	        this.firstTry = source["firstTry"];
	        this.usedHint = source["usedHint"];
	        this.revealed = source["revealed"];
	    }
	}
	export class ThemePerformance {
	    theme: string;
	    attempts: number;
	    accuracy: number;

	    static createFrom(source: any = {}) {
	        return new ThemePerformance(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.attempts = source["attempts"];
	        this.accuracy = source["accuracy"];
	    }
	}
	export class Summary {
	    learnerRating: number;
	    ratingTrend: RatingPoint[];
	    firstAttemptAccuracy: number;
	    hintRate: number;
	    themePerformance: ThemePerformance[];
	    dueReviews: number;
	    recentSessions: RecentSession[];

	    static createFrom(source: any = {}) {
	        return new Summary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.learnerRating = source["learnerRating"];
	        this.ratingTrend = this.convertValues(source["ratingTrend"], RatingPoint);
	        this.firstAttemptAccuracy = source["firstAttemptAccuracy"];
	        this.hintRate = source["hintRate"];
	        this.themePerformance = this.convertValues(source["themePerformance"], ThemePerformance);
	        this.dueReviews = source["dueReviews"];
	        this.recentSessions = this.convertValues(source["recentSessions"], RecentSession);
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

export namespace training {

	export class PracticeRequest {
	    sourceId: string;
	    minimumRating?: number;
	    maximumRating?: number;
	    themes: string[];
	    maximumSolutionPlies?: number;

	    static createFrom(source: any = {}) {
	        return new PracticeRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceId = source["sourceId"];
	        this.minimumRating = source["minimumRating"];
	        this.maximumRating = source["maximumRating"];
	        this.themes = source["themes"];
	        this.maximumSolutionPlies = source["maximumSolutionPlies"];
	    }
	}
	export class Profile {
	    learnerRating: number;
	    sessionSize: number;

	    static createFrom(source: any = {}) {
	        return new Profile(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.learnerRating = source["learnerRating"];
	        this.sessionSize = source["sessionSize"];
	    }
	}

}
