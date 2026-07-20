export namespace buildinfo {

	export class Info {
	    name: string;
	    commit: string;
	    sourceUrl: string;

	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.commit = source["commit"];
	        this.sourceUrl = source["sourceUrl"];
	    }
	}

}

export namespace domain {

	export class AppliedMove {
	    uci: string;
	    resultingFen: string;

	    static createFrom(source: any = {}) {
	        return new AppliedMove(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.uci = source["uci"];
	        this.resultingFen = source["resultingFen"];
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
	    legalMoves: string[];

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
	        this.legalMoves = source["legalMoves"];
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
	export class HintResult {
	    session: SessionView;
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
	        this.session = this.convertValues(source["session"], SessionView);
	        this.level = source["level"];
	        this.text = source["text"];
	        this.sourceSquare = source["sourceSquare"];
	        this.targetSquare = source["targetSquare"];
	        this.canReveal = source["canReveal"];
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
	    appliedMoves?: AppliedMove[];
	    finalFen?: string;

	    static createFrom(source: any = {}) {
	        return new MoveResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.session = this.convertValues(source["session"], SessionView);
	        this.correct = source["correct"];
	        this.puzzleCompleted = source["puzzleCompleted"];
	        this.message = source["message"];
	        this.appliedMoves = this.convertValues(source["appliedMoves"], AppliedMove);
	        this.finalFen = source["finalFen"];
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

export namespace importing {

	export class Inspection {
	    path: string;
	    filename: string;
	    format: string;
	    formatLabel: string;
	    sourceId: string;
	    sourceIdOrigin: string;
	    sourceName?: string;
	    url?: string;
	    attribution?: string;
	    replacesExisting: boolean;

	    static createFrom(source: any = {}) {
	        return new Inspection(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.filename = source["filename"];
	        this.format = source["format"];
	        this.formatLabel = source["formatLabel"];
	        this.sourceId = source["sourceId"];
	        this.sourceIdOrigin = source["sourceIdOrigin"];
	        this.sourceName = source["sourceName"];
	        this.url = source["url"];
	        this.attribution = source["attribution"];
	        this.replacesExisting = source["replacesExisting"];
	    }
	}
	export class Progress {
	    phase: string;
	    rowsRead: number;
	    bytesRead: number;
	    totalBytes: number;

	    static createFrom(source: any = {}) {
	        return new Progress(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.phase = source["phase"];
	        this.rowsRead = source["rowsRead"];
	        this.bytesRead = source["bytesRead"];
	        this.totalBytes = source["totalBytes"];
	    }
	}
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
	export class Report {
	    accepted: number;
	    duplicates: number;
	    rejected: number;
	    examples: Rejection[];
	    counts?: Record<string, number>;

	    static createFrom(source: any = {}) {
	        return new Report(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.accepted = source["accepted"];
	        this.duplicates = source["duplicates"];
	        this.rejected = source["rejected"];
	        this.examples = this.convertValues(source["examples"], Rejection);
	        this.counts = source["counts"];
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
	    progress: importing.Progress;
	    report: importing.Report;
	    error?: string;

	    static createFrom(source: any = {}) {
	        return new Result(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jobId = source["jobId"];
	        this.status = source["status"];
	        this.progress = this.convertValues(source["progress"], importing.Progress);
	        this.report = this.convertValues(source["report"], importing.Report);
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

export namespace openings {

	export class BoardAnnotation {
	    kind: string;
	    from: string;
	    to?: string;
	    label?: string;

	    static createFrom(source: any = {}) {
	        return new BoardAnnotation(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.from = source["from"];
	        this.to = source["to"];
	        this.label = source["label"];
	    }
	}
	export class Evaluation {
	    code: string;
	    sourceSymbol?: string;

	    static createFrom(source: any = {}) {
	        return new Evaluation(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.sourceSymbol = source["sourceSymbol"];
	    }
	}
	export class SourceRef {
	    printedPage: number;
	    tableColumn?: string;
	    noteLabel?: string;
	    coverageId: string;

	    static createFrom(source: any = {}) {
	        return new SourceRef(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.printedPage = source["printedPage"];
	        this.tableColumn = source["tableColumn"];
	        this.noteLabel = source["noteLabel"];
	        this.coverageId = source["coverageId"];
	    }
	}
	export class ExplorerMove {
	    moveId: string;
	    uci: string;
	    san: string;
	    toPositionId: string;
	    role: string;
	    variationName?: string;
	    evaluation: Evaluation;
	    sourceRef: SourceRef;

	    static createFrom(source: any = {}) {
	        return new ExplorerMove(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.moveId = source["moveId"];
	        this.uci = source["uci"];
	        this.san = source["san"];
	        this.toPositionId = source["toPositionId"];
	        this.role = source["role"];
	        this.variationName = source["variationName"];
	        this.evaluation = this.convertValues(source["evaluation"], Evaluation);
	        this.sourceRef = this.convertValues(source["sourceRef"], SourceRef);
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
	export class NoteView {
	    kind: string;
	    text: string;
	    sourceRef: SourceRef;

	    static createFrom(source: any = {}) {
	        return new NoteView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.text = source["text"];
	        this.sourceRef = this.convertValues(source["sourceRef"], SourceRef);
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
	export class ExplorerPositionView {
	    courseId: string;
	    positionId: string;
	    fen: string;
	    label: string;
	    evaluation: Evaluation;
	    notes: NoteView[];
	    moves: ExplorerMove[];
	    incomingPaths: number;

	    static createFrom(source: any = {}) {
	        return new ExplorerPositionView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.courseId = source["courseId"];
	        this.positionId = source["positionId"];
	        this.fen = source["fen"];
	        this.label = source["label"];
	        this.evaluation = this.convertValues(source["evaluation"], Evaluation);
	        this.notes = this.convertValues(source["notes"], NoteView);
	        this.moves = this.convertValues(source["moves"], ExplorerMove);
	        this.incomingPaths = source["incomingPaths"];
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

	export class OpeningActivityLine {
	    label: string;
	    moves: string[];

	    static createFrom(source: any = {}) {
	        return new OpeningActivityLine(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.moves = source["moves"];
	    }
	}
	export class OpeningPathItem {
	    lessonId: string;
	    title: string;

	    static createFrom(source: any = {}) {
	        return new OpeningPathItem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lessonId = source["lessonId"];
	        this.title = source["title"];
	    }
	}
	export class OpeningRoadmapCheckpoint {
	    completedLessonId: string;
	    path: OpeningPathItem[];
	    availableLessonIds: string[];
	    recommendedLessonId?: string;
	    recommendedLessonTitle?: string;
	    completedLessons: number;
	    totalLessons: number;

	    static createFrom(source: any = {}) {
	        return new OpeningRoadmapCheckpoint(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.completedLessonId = source["completedLessonId"];
	        this.path = this.convertValues(source["path"], OpeningPathItem);
	        this.availableLessonIds = source["availableLessonIds"];
	        this.recommendedLessonId = source["recommendedLessonId"];
	        this.recommendedLessonTitle = source["recommendedLessonTitle"];
	        this.completedLessons = source["completedLessons"];
	        this.totalLessons = source["totalLessons"];
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
	export class OpeningSummary {
	    totalPrompts: number;
	    positionsRecalled: number;
	    branchesRecognized: number;
	    retried: number;
	    usedHint: number;
	    revealed: number;

	    static createFrom(source: any = {}) {
	        return new OpeningSummary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalPrompts = source["totalPrompts"];
	        this.positionsRecalled = source["positionsRecalled"];
	        this.branchesRecognized = source["branchesRecognized"];
	        this.retried = source["retried"];
	        this.usedHint = source["usedHint"];
	        this.revealed = source["revealed"];
	    }
	}
	export class OpeningReferenceSection {
	    activityId: string;
	    title: string;
	    instruction: string;
	    positionId?: string;
	    noteTexts: string[];
	    annotations: BoardAnnotation[];

	    static createFrom(source: any = {}) {
	        return new OpeningReferenceSection(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.activityId = source["activityId"];
	        this.title = source["title"];
	        this.instruction = source["instruction"];
	        this.positionId = source["positionId"];
	        this.noteTexts = source["noteTexts"];
	        this.annotations = this.convertValues(source["annotations"], BoardAnnotation);
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
	export class OpeningActivityView {
	    activityId: string;
	    kind: string;
	    title: string;
	    instruction: string;
	    required: boolean;
	    variationName?: string;
	    positionId?: string;
	    currentFen: string;
	    orientation: string;
	    legalMoves: string[];
	    teachingNoteTexts: string[];
	    referenceNoteTexts: string[];
	    comparison: OpeningActivityLine[];
	    annotations: BoardAnnotation[];
	    movesToHere: domain.AppliedMove[];
	    activityNumber: number;
	    activityTotal: number;
	    completedIdeas: number;
	    requiredIdeas: number;
	    hintLevel: number;
	    canReveal: boolean;
	    referenceSections: OpeningReferenceSection[];

	    static createFrom(source: any = {}) {
	        return new OpeningActivityView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.activityId = source["activityId"];
	        this.kind = source["kind"];
	        this.title = source["title"];
	        this.instruction = source["instruction"];
	        this.required = source["required"];
	        this.variationName = source["variationName"];
	        this.positionId = source["positionId"];
	        this.currentFen = source["currentFen"];
	        this.orientation = source["orientation"];
	        this.legalMoves = source["legalMoves"];
	        this.teachingNoteTexts = source["teachingNoteTexts"];
	        this.referenceNoteTexts = source["referenceNoteTexts"];
	        this.comparison = this.convertValues(source["comparison"], OpeningActivityLine);
	        this.annotations = this.convertValues(source["annotations"], BoardAnnotation);
	        this.movesToHere = this.convertValues(source["movesToHere"], domain.AppliedMove);
	        this.activityNumber = source["activityNumber"];
	        this.activityTotal = source["activityTotal"];
	        this.completedIdeas = source["completedIdeas"];
	        this.requiredIdeas = source["requiredIdeas"];
	        this.hintLevel = source["hintLevel"];
	        this.canReveal = source["canReveal"];
	        this.referenceSections = this.convertValues(source["referenceSections"], OpeningReferenceSection);
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
	export class OpeningSessionView {
	    sessionId: string;
	    mode: string;
	    status: string;
	    courseId: string;
	    generationId: string;
	    lessonId: string;
	    depth: string;
	    current?: OpeningActivityView;
	    summary?: OpeningSummary;
	    notice?: string;

	    static createFrom(source: any = {}) {
	        return new OpeningSessionView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.mode = source["mode"];
	        this.status = source["status"];
	        this.courseId = source["courseId"];
	        this.generationId = source["generationId"];
	        this.lessonId = source["lessonId"];
	        this.depth = source["depth"];
	        this.current = this.convertValues(source["current"], OpeningActivityView);
	        this.summary = this.convertValues(source["summary"], OpeningSummary);
	        this.notice = source["notice"];
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
	export class OpeningActivityResult {
	    session: OpeningSessionView;
	    activityCompleted: boolean;
	    stepCompleted?: boolean;
	    feedback?: string;
	    message?: string;
	    appliedMoves?: domain.AppliedMove[];
	    finalFen?: string;
	    checkpoint?: OpeningRoadmapCheckpoint;

	    static createFrom(source: any = {}) {
	        return new OpeningActivityResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.session = this.convertValues(source["session"], OpeningSessionView);
	        this.activityCompleted = source["activityCompleted"];
	        this.stepCompleted = source["stepCompleted"];
	        this.feedback = source["feedback"];
	        this.message = source["message"];
	        this.appliedMoves = this.convertValues(source["appliedMoves"], domain.AppliedMove);
	        this.finalFen = source["finalFen"];
	        this.checkpoint = this.convertValues(source["checkpoint"], OpeningRoadmapCheckpoint);
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

	export class OpeningLessonSummary {
	    lessonId: string;
	    title: string;
	    completedSteps: number;
	    totalSteps: number;
	    completed: boolean;

	    static createFrom(source: any = {}) {
	        return new OpeningLessonSummary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lessonId = source["lessonId"];
	        this.title = source["title"];
	        this.completedSteps = source["completedSteps"];
	        this.totalSteps = source["totalSteps"];
	        this.completed = source["completed"];
	    }
	}
	export class OpeningChapterSummary {
	    chapterId: string;
	    title: string;
	    lessons: OpeningLessonSummary[];

	    static createFrom(source: any = {}) {
	        return new OpeningChapterSummary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.chapterId = source["chapterId"];
	        this.title = source["title"];
	        this.lessons = this.convertValues(source["lessons"], OpeningLessonSummary);
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
	export class OpeningTeachingEdgeView {
	    edgeId: string;
	    fromLessonId: string;
	    toLessonId: string;
	    ordinal: number;
	    kind: string;
	    label?: string;
	    minimumDepth: string;

	    static createFrom(source: any = {}) {
	        return new OpeningTeachingEdgeView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.edgeId = source["edgeId"];
	        this.fromLessonId = source["fromLessonId"];
	        this.toLessonId = source["toLessonId"];
	        this.ordinal = source["ordinal"];
	        this.kind = source["kind"];
	        this.label = source["label"];
	        this.minimumDepth = source["minimumDepth"];
	    }
	}
	export class OpeningTeachingNodeView {
	    lessonId: string;
	    chapterId: string;
	    title: string;
	    objective: string;
	    minimumDepth: string;
	    progress: string;
	    completedActivities: number;
	    requiredActivities: number;
	    recommended: boolean;
	    reviewDue: boolean;
	    visible: boolean;

	    static createFrom(source: any = {}) {
	        return new OpeningTeachingNodeView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lessonId = source["lessonId"];
	        this.chapterId = source["chapterId"];
	        this.title = source["title"];
	        this.objective = source["objective"];
	        this.minimumDepth = source["minimumDepth"];
	        this.progress = source["progress"];
	        this.completedActivities = source["completedActivities"];
	        this.requiredActivities = source["requiredActivities"];
	        this.recommended = source["recommended"];
	        this.reviewDue = source["reviewDue"];
	        this.visible = source["visible"];
	    }
	}
	export class OpeningTeachingTreeView {
	    rootLessonId: string;
	    nodes: OpeningTeachingNodeView[];
	    edges: OpeningTeachingEdgeView[];

	    static createFrom(source: any = {}) {
	        return new OpeningTeachingTreeView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootLessonId = source["rootLessonId"];
	        this.nodes = this.convertValues(source["nodes"], OpeningTeachingNodeView);
	        this.edges = this.convertValues(source["edges"], OpeningTeachingEdgeView);
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
	export class OpeningCourseSummary {
	    courseId: string;
	    title: string;
	    perspective: string;
	    depth: string;
	    rootPositionId: string;
	    completedLessons: number;
	    totalLessons: number;
	    dueReviews: number;
	    nextLessonId?: string;
	    nextLessonTitle?: string;
	    currentLessonId?: string;
	    currentActivityId?: string;
	    currentPath: OpeningPathItem[];
	    recommendedLessonId?: string;
	    recommendedLessonTitle?: string;
	    hasResumable: boolean;
	    hasResumableReview: boolean;
	    tree: OpeningTeachingTreeView;
	    chapters: OpeningChapterSummary[];

	    static createFrom(source: any = {}) {
	        return new OpeningCourseSummary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.courseId = source["courseId"];
	        this.title = source["title"];
	        this.perspective = source["perspective"];
	        this.depth = source["depth"];
	        this.rootPositionId = source["rootPositionId"];
	        this.completedLessons = source["completedLessons"];
	        this.totalLessons = source["totalLessons"];
	        this.dueReviews = source["dueReviews"];
	        this.nextLessonId = source["nextLessonId"];
	        this.nextLessonTitle = source["nextLessonTitle"];
	        this.currentLessonId = source["currentLessonId"];
	        this.currentActivityId = source["currentActivityId"];
	        this.currentPath = this.convertValues(source["currentPath"], OpeningPathItem);
	        this.recommendedLessonId = source["recommendedLessonId"];
	        this.recommendedLessonTitle = source["recommendedLessonTitle"];
	        this.hasResumable = source["hasResumable"];
	        this.hasResumableReview = source["hasResumableReview"];
	        this.tree = this.convertValues(source["tree"], OpeningTeachingTreeView);
	        this.chapters = this.convertValues(source["chapters"], OpeningChapterSummary);
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
	export class OpeningHintResult {
	    session: OpeningSessionView;
	    level: number;
	    text: string;
	    sourceSquare?: string;
	    targetSquare?: string;
	    canReveal: boolean;

	    static createFrom(source: any = {}) {
	        return new OpeningHintResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.session = this.convertValues(source["session"], OpeningSessionView);
	        this.level = source["level"];
	        this.text = source["text"];
	        this.sourceSquare = source["sourceSquare"];
	        this.targetSquare = source["targetSquare"];
	        this.canReveal = source["canReveal"];
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
	export class OpeningHomeView {
	    notice?: string;
	    courses: OpeningCourseSummary[];

	    static createFrom(source: any = {}) {
	        return new OpeningHomeView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.notice = source["notice"];
	        this.courses = this.convertValues(source["courses"], OpeningCourseSummary);
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
	    learnerRatingBounds: puzzles.RatingBounds;

	    static createFrom(source: any = {}) {
	        return new PracticeFilters(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sources = this.convertValues(source["sources"], PracticeSource);
	        this.themes = source["themes"];
	        this.maximumSolutionPlies = source["maximumSolutionPlies"];
	        this.learnerRatingBounds = this.convertValues(source["learnerRatingBounds"], puzzles.RatingBounds);
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

	export class RatingBounds {
	    minimum: number;
	    maximum: number;

	    static createFrom(source: any = {}) {
	        return new RatingBounds(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.minimum = source["minimum"];
	        this.maximum = source["maximum"];
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
