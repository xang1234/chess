package importjob

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"chess-trainer/internal/puzzles"
)

const (
	testWait        = 2 * time.Second
	testQuietPeriod = 50 * time.Millisecond
)

type importOutcome struct {
	report puzzles.ImportReport
	err    error
}

type startedImport struct {
	sourceID string
	path     string
	ctx      context.Context
	progress puzzles.ProgressSink
	finish   chan importOutcome
}

type blockingImporter struct {
	started     chan startedImport
	afterCancel <-chan struct{}
}

func newBlockingImporter() *blockingImporter {
	return &blockingImporter{started: make(chan startedImport, 8)}
}

func (i *blockingImporter) Import(
	ctx context.Context,
	sourceID string,
	path string,
	progress puzzles.ProgressSink,
) (puzzles.ImportReport, error) {
	call := startedImport{
		sourceID: sourceID,
		path:     path,
		ctx:      ctx,
		progress: progress,
		finish:   make(chan importOutcome, 1),
	}
	i.started <- call
	select {
	case outcome := <-call.finish:
		return outcome.report, outcome.err
	case <-ctx.Done():
		if i.afterCancel != nil {
			<-i.afterCancel
		}
		return puzzles.ImportReport{}, ctx.Err()
	}
}

type scriptedImporter struct {
	progress []puzzles.Progress
	report   puzzles.ImportReport
}

func (i scriptedImporter) Import(
	_ context.Context,
	_ string,
	_ string,
	progress puzzles.ProgressSink,
) (puzzles.ImportReport, error) {
	for _, snapshot := range i.progress {
		progress(snapshot)
	}
	return i.report, nil
}

type cleanupOutcome struct {
	more bool
	err  error
}

type startedCleanup struct {
	ctx    context.Context
	limit  int
	finish chan cleanupOutcome
}

type blockingMaintenance struct {
	started     chan startedCleanup
	afterCancel <-chan struct{}
}

func newBlockingMaintenance() *blockingMaintenance {
	return &blockingMaintenance{started: make(chan startedCleanup, 8)}
}

func (m *blockingMaintenance) CleanupBatch(ctx context.Context, limit int) (bool, error) {
	call := startedCleanup{ctx: ctx, limit: limit, finish: make(chan cleanupOutcome, 1)}
	m.started <- call
	select {
	case outcome := <-call.finish:
		return outcome.more, outcome.err
	case <-ctx.Done():
		if m.afterCancel != nil {
			<-m.afterCancel
		}
		return false, ctx.Err()
	}
}

type emittedProgress struct {
	jobID    string
	snapshot puzzles.Progress
}

type recordingEmitter struct {
	progress chan emittedProgress
	finished chan Result
}

type blockingFirstProgressEmitter struct {
	blockOnce sync.Once
	started   chan emittedProgress
	release   <-chan struct{}
	progress  chan emittedProgress
	finished  chan Result
}

func (e *blockingFirstProgressEmitter) Progress(jobID string, progress puzzles.Progress) {
	event := emittedProgress{jobID: jobID, snapshot: progress}
	blocked := false
	e.blockOnce.Do(func() { blocked = true })
	if blocked {
		e.started <- event
		<-e.release
	}
	e.progress <- event
}

func (e *blockingFirstProgressEmitter) Finished(result Result) {
	e.finished <- result
}

type blockingFirstTerminalEmitter struct {
	blockOnce sync.Once
	started   chan Result
	release   <-chan struct{}
	progress  chan emittedProgress
	finished  chan Result
}

func (e *blockingFirstTerminalEmitter) Progress(jobID string, progress puzzles.Progress) {
	e.progress <- emittedProgress{jobID: jobID, snapshot: progress}
}

func (e *blockingFirstTerminalEmitter) Finished(result Result) {
	blocked := false
	e.blockOnce.Do(func() { blocked = true })
	if blocked {
		e.started <- result
		<-e.release
	}
	e.finished <- result
}

func newRecordingEmitter() *recordingEmitter {
	return &recordingEmitter{
		progress: make(chan emittedProgress, 16),
		finished: make(chan Result, 16),
	}
}

func (e *recordingEmitter) Progress(jobID string, progress puzzles.Progress) {
	e.progress <- emittedProgress{jobID: jobID, snapshot: progress}
}

func (e *recordingEmitter) Finished(result Result) {
	e.finished <- result
}

func TestStartRoutesRequestByKind(t *testing.T) {
	lichess := newBlockingImporter()
	alternate := newBlockingImporter()
	const alternateKind Kind = "alternate"
	importers := map[Kind]Importer{
		KindLichess:   lichess,
		alternateKind: alternate,
	}
	emitter := newRecordingEmitter()
	service := NewService(importers, nil, emitter)
	t.Cleanup(service.Close)

	// Constructor ownership must isolate routing from later caller mutations.
	importers[alternateKind] = lichess
	delete(importers, KindLichess)

	request := ImportRequest{Kind: alternateKind, SourceID: "club", Path: "/club.csv"}
	jobID, err := service.Start(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	call := receive(t, alternate.started)
	if call.sourceID != request.SourceID || call.path != request.Path {
		t.Fatalf("routed source/path = %q/%q, want %q/%q", call.sourceID, call.path, request.SourceID, request.Path)
	}
	assertNoReceive(t, lichess.started, "wrong importer started")
	call.finish <- importOutcome{report: puzzles.ImportReport{Accepted: 3}}

	finished := receive(t, emitter.finished)
	if finished.JobID != jobID || finished.Request != request || finished.Status != Succeeded {
		t.Fatalf("finished = %+v", finished)
	}
}

func TestStartValidatesRequest(t *testing.T) {
	service := NewService(map[Kind]Importer{KindLichess: newBlockingImporter()}, nil, nil)
	t.Cleanup(service.Close)

	tests := []struct {
		name    string
		request ImportRequest
	}{
		{name: "kind", request: ImportRequest{SourceID: "source", Path: "/puzzles"}},
		{name: "configured kind", request: ImportRequest{Kind: "missing", SourceID: "source", Path: "/puzzles"}},
		{name: "source", request: ImportRequest{Kind: KindLichess, Path: "/puzzles"}},
		{name: "path", request: ImportRequest{Kind: KindLichess, SourceID: "source"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := service.Start(context.Background(), test.request); err == nil {
				t.Fatalf("Start(%+v) unexpectedly succeeded", test.request)
			}
		})
	}
}

func TestStartRejectsSecondActivePuzzleImport(t *testing.T) {
	importer := newBlockingImporter()
	service := NewService(map[Kind]Importer{KindLichess: importer}, nil, nil)
	t.Cleanup(service.Close)
	request := ImportRequest{Kind: KindLichess, SourceID: "lichess", Path: "/first.csv.zst"}

	activeJobID, err := service.Start(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	call := receive(t, importer.started)
	_, err = service.Start(context.Background(), ImportRequest{
		Kind: KindLichess, SourceID: "lichess", Path: "/second.csv.zst",
	})
	var busy *BusyError
	if !errors.As(err, &busy) {
		t.Fatalf("Start() error = %T %v, want *BusyError", err, err)
	}
	if busy.ActiveJobID != activeJobID {
		t.Fatalf("busy active job = %q, want %q", busy.ActiveJobID, activeJobID)
	}

	call.finish <- importOutcome{}
}

func TestResultKeepsMonotonicProgress(t *testing.T) {
	emitter := newRecordingEmitter()
	service := NewService(map[Kind]Importer{
		KindLichess: scriptedImporter{
			progress: []puzzles.Progress{
				{RowsRead: 10, BytesRead: 20},
				{RowsRead: 5, BytesRead: 25},
				{RowsRead: 12, BytesRead: 15},
			},
			report: puzzles.ImportReport{Accepted: 9},
		},
	}, nil, emitter)
	t.Cleanup(service.Close)
	request := ImportRequest{Kind: KindLichess, SourceID: "lichess", Path: "/puzzles.csv.zst"}

	jobID, err := service.Start(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	want := []puzzles.Progress{
		{RowsRead: 10, BytesRead: 20},
		{RowsRead: 10, BytesRead: 25},
		{RowsRead: 12, BytesRead: 25},
	}
	for index, expected := range want {
		event := receive(t, emitter.progress)
		if event.jobID != jobID || event.snapshot != expected {
			t.Fatalf("progress[%d] = %+v, want job %q snapshot %+v", index, event, jobID, expected)
		}
	}
	finished := receive(t, emitter.finished)
	if finished.Progress != want[len(want)-1] {
		t.Fatalf("finished progress = %+v, want %+v", finished.Progress, want[len(want)-1])
	}
	result, err := service.Result(jobID)
	if err != nil || result.Progress != want[len(want)-1] || result.Report.Accepted != 9 {
		t.Fatalf("Result() = %+v, %v", result, err)
	}
}

func TestConcurrentProgressEmitsMonotonicSnapshots(t *testing.T) {
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseEmitter := func() { releaseOnce.Do(func() { close(release) }) }
	importer := newBlockingImporter()
	emitter := &blockingFirstProgressEmitter{
		started:  make(chan emittedProgress, 1),
		release:  release,
		progress: make(chan emittedProgress, 2),
		finished: make(chan Result, 1),
	}
	service := NewService(map[Kind]Importer{KindLichess: importer}, nil, emitter)
	t.Cleanup(service.Close)
	t.Cleanup(releaseEmitter)

	jobID, err := service.Start(context.Background(), ImportRequest{
		Kind: KindLichess, SourceID: "lichess", Path: "/concurrent-progress",
	})
	if err != nil {
		t.Fatal(err)
	}
	call := receive(t, importer.started)
	firstReturned := make(chan struct{})
	go func() {
		call.progress(puzzles.Progress{RowsRead: 10, BytesRead: 20})
		close(firstReturned)
	}()
	firstStarted := receive(t, emitter.started)
	if firstStarted.jobID != jobID || firstStarted.snapshot.RowsRead != 10 {
		t.Fatalf("first progress = %+v", firstStarted)
	}

	secondReturned := make(chan struct{})
	go func() {
		call.progress(puzzles.Progress{RowsRead: 20, BytesRead: 30})
		close(secondReturned)
	}()
	assertNoReceive(t, secondReturned, "later progress overtook a blocked earlier callback")
	releaseEmitter()
	receive(t, firstReturned)
	receive(t, secondReturned)

	first := receive(t, emitter.progress)
	second := receive(t, emitter.progress)
	if first.snapshot != (puzzles.Progress{RowsRead: 10, BytesRead: 20}) ||
		second.snapshot != (puzzles.Progress{RowsRead: 20, BytesRead: 30}) {
		t.Fatalf("emitted progress = %+v then %+v", first.snapshot, second.snapshot)
	}
	call.finish <- importOutcome{}
	_ = receive(t, emitter.finished)
}

func TestTerminalEventPrecedesLaterJobProgress(t *testing.T) {
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseEmitter := func() { releaseOnce.Do(func() { close(release) }) }
	importer := newBlockingImporter()
	emitter := &blockingFirstTerminalEmitter{
		started:  make(chan Result, 1),
		release:  release,
		progress: make(chan emittedProgress, 1),
		finished: make(chan Result, 2),
	}
	service := NewService(map[Kind]Importer{KindLichess: importer}, nil, emitter)
	t.Cleanup(service.Close)
	t.Cleanup(releaseEmitter)

	firstID, err := service.Start(context.Background(), ImportRequest{
		Kind: KindLichess, SourceID: "first", Path: "/first",
	})
	if err != nil {
		t.Fatal(err)
	}
	firstCall := receive(t, importer.started)
	firstCall.finish <- importOutcome{}
	terminalStarted := receive(t, emitter.started)
	if terminalStarted.JobID != firstID {
		t.Fatalf("blocked terminal job = %q, want %q", terminalStarted.JobID, firstID)
	}

	secondID, err := service.Start(context.Background(), ImportRequest{
		Kind: KindLichess, SourceID: "second", Path: "/second",
	})
	if err != nil {
		t.Fatal(err)
	}
	secondCall := receive(t, importer.started)
	progressReturned := make(chan struct{})
	go func() {
		secondCall.progress(puzzles.Progress{RowsRead: 1})
		close(progressReturned)
	}()
	assertNoReceive(t, progressReturned, "later-job progress overtook the earlier terminal event")

	releaseEmitter()
	firstFinished := receive(t, emitter.finished)
	if firstFinished.JobID != firstID {
		t.Fatalf("first emitted terminal job = %q, want %q", firstFinished.JobID, firstID)
	}
	receive(t, progressReturned)
	progress := receive(t, emitter.progress)
	if progress.jobID != secondID {
		t.Fatalf("progress job = %q, want %q", progress.jobID, secondID)
	}
	secondCall.finish <- importOutcome{}
	secondFinished := receive(t, emitter.finished)
	if secondFinished.JobID != secondID {
		t.Fatalf("second emitted terminal job = %q, want %q", secondFinished.JobID, secondID)
	}
}

func TestCompletedResultRemainsQueryableAfterLaterJob(t *testing.T) {
	importer := newBlockingImporter()
	emitter := newRecordingEmitter()
	service := NewService(map[Kind]Importer{KindLichess: importer}, nil, emitter)
	t.Cleanup(service.Close)

	firstRequest := ImportRequest{Kind: KindLichess, SourceID: "first", Path: "/first"}
	firstID, err := service.Start(context.Background(), firstRequest)
	if err != nil {
		t.Fatal(err)
	}
	firstCall := receive(t, importer.started)
	firstCall.progress(puzzles.Progress{RowsRead: 7, BytesRead: 11})
	firstCall.finish <- importOutcome{report: puzzles.ImportReport{Accepted: 1}}
	_ = receive(t, emitter.progress)
	_ = receive(t, emitter.finished)

	secondRequest := ImportRequest{Kind: KindLichess, SourceID: "second", Path: "/second"}
	secondID, err := service.Start(context.Background(), secondRequest)
	if err != nil {
		t.Fatal(err)
	}
	if secondID == firstID {
		t.Fatal("later import reused the completed job ID")
	}
	secondCall := receive(t, importer.started)
	secondCall.finish <- importOutcome{report: puzzles.ImportReport{Accepted: 2}}
	_ = receive(t, emitter.finished)

	result, err := service.Result(firstID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Request != firstRequest || result.Status != Succeeded ||
		result.Progress != (puzzles.Progress{RowsRead: 7, BytesRead: 11}) || result.Report.Accepted != 1 {
		t.Fatalf("first result after later job = %+v", result)
	}
}

func TestJobReachesExactlyOneTerminalState(t *testing.T) {
	importer := newBlockingImporter()
	emitter := newRecordingEmitter()
	service := NewService(map[Kind]Importer{KindLichess: importer}, nil, emitter)
	request := ImportRequest{Kind: KindLichess, SourceID: "lichess", Path: "/cancel"}

	jobID, err := service.Start(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	_ = receive(t, importer.started)
	if err := service.Cancel(jobID); err != nil {
		t.Fatal(err)
	}
	if err := service.Cancel(jobID); err != nil {
		t.Fatal(err)
	}
	finished := receive(t, emitter.finished)
	service.Close()

	if finished.JobID != jobID || finished.Status != Cancelled {
		t.Fatalf("finished = %+v", finished)
	}
	assertNoReceive(t, emitter.finished, "job emitted a second terminal state")
	result, err := service.Result(jobID)
	if err != nil || result.Status != Cancelled {
		t.Fatalf("Result() = %+v, %v", result, err)
	}
}

func TestTerminalJobAllowsNextImport(t *testing.T) {
	importer := newBlockingImporter()
	emitter := newRecordingEmitter()
	service := NewService(map[Kind]Importer{KindLichess: importer}, nil, emitter)
	t.Cleanup(service.Close)

	firstID, err := service.Start(context.Background(), ImportRequest{
		Kind: KindLichess, SourceID: "first", Path: "/first",
	})
	if err != nil {
		t.Fatal(err)
	}
	firstCall := receive(t, importer.started)
	firstCall.finish <- importOutcome{err: errors.New("broken source")}
	firstResult := receive(t, emitter.finished)
	if firstResult.JobID != firstID || firstResult.Status != Failed {
		t.Fatalf("first result = %+v", firstResult)
	}

	secondID, err := service.Start(context.Background(), ImportRequest{
		Kind: KindLichess, SourceID: "second", Path: "/second",
	})
	if err != nil {
		t.Fatalf("Start() after terminal job: %v", err)
	}
	if secondID == firstID {
		t.Fatal("second job reused first job ID")
	}
	secondCall := receive(t, importer.started)
	secondCall.finish <- importOutcome{}
}

func TestCleanupStartsAfterTerminalAndNeverOverlapsImport(t *testing.T) {
	importer := newBlockingImporter()
	maintenance := newBlockingMaintenance()
	emitter := newRecordingEmitter()
	service := NewService(map[Kind]Importer{KindLichess: importer}, maintenance, emitter)
	t.Cleanup(service.Close)

	firstID, err := service.Start(context.Background(), ImportRequest{
		Kind: KindLichess, SourceID: "first", Path: "/first",
	})
	if err != nil {
		t.Fatal(err)
	}
	firstCall := receive(t, importer.started)
	assertNoReceive(t, maintenance.started, "cleanup began before import reached a terminal state")
	firstCall.finish <- importOutcome{}
	finished := receive(t, emitter.finished)
	if finished.JobID != firstID || finished.Status != Succeeded {
		t.Fatalf("first terminal result = %+v", finished)
	}
	cleanup := receive(t, maintenance.started)
	if cleanup.limit <= 0 {
		t.Fatalf("cleanup limit = %d", cleanup.limit)
	}
	stored, err := service.Result(firstID)
	if err != nil || stored.Status != Succeeded {
		t.Fatalf("Result() when cleanup started = %+v, %v", stored, err)
	}

	_, err = service.Start(context.Background(), ImportRequest{
		Kind: KindLichess, SourceID: "second", Path: "/second",
	})
	if err != nil {
		t.Fatal(err)
	}
	assertNoReceive(t, importer.started, "import overlapped cleanup batch")
	cleanup.finish <- cleanupOutcome{}
	secondCall := receive(t, importer.started)
	secondCall.finish <- importOutcome{}
	_ = receive(t, emitter.finished)
	lastCleanup := receive(t, maintenance.started)
	lastCleanup.finish <- cleanupOutcome{}
}

func TestNewImportPreemptsFurtherCleanupBatches(t *testing.T) {
	importer := newBlockingImporter()
	maintenance := newBlockingMaintenance()
	emitter := newRecordingEmitter()
	service := NewService(map[Kind]Importer{KindLichess: importer}, maintenance, emitter)
	t.Cleanup(service.Close)

	_, err := service.Start(context.Background(), ImportRequest{
		Kind: KindLichess, SourceID: "first", Path: "/first",
	})
	if err != nil {
		t.Fatal(err)
	}
	firstCall := receive(t, importer.started)
	firstCall.finish <- importOutcome{}
	_ = receive(t, emitter.finished)
	firstCleanup := receive(t, maintenance.started)

	_, err = service.Start(context.Background(), ImportRequest{
		Kind: KindLichess, SourceID: "second", Path: "/second",
	})
	if err != nil {
		t.Fatal(err)
	}
	firstCleanup.finish <- cleanupOutcome{more: true}
	secondCall := receive(t, importer.started)
	assertNoReceive(t, maintenance.started, "cleanup started another batch while an import was active")

	secondCall.finish <- importOutcome{}
	_ = receive(t, emitter.finished)
	postImportCleanup := receive(t, maintenance.started)
	postImportCleanup.finish <- cleanupOutcome{}
}

func TestCloseCancelsAndWaitsForImporterAndCleanup(t *testing.T) {
	importerGate := make(chan struct{})
	cleanupGate := make(chan struct{})
	var releaseImporter sync.Once
	var releaseCleanup sync.Once
	releaseImporterCall := func() { releaseImporter.Do(func() { close(importerGate) }) }
	releaseCleanupCall := func() { releaseCleanup.Do(func() { close(cleanupGate) }) }
	t.Cleanup(releaseImporterCall)
	t.Cleanup(releaseCleanupCall)

	importer := newBlockingImporter()
	importer.afterCancel = importerGate
	maintenance := newBlockingMaintenance()
	maintenance.afterCancel = cleanupGate
	emitter := newRecordingEmitter()
	service := NewService(map[Kind]Importer{KindLichess: importer}, maintenance, emitter)

	_, err := service.Start(context.Background(), ImportRequest{
		Kind: KindLichess, SourceID: "first", Path: "/first",
	})
	if err != nil {
		t.Fatal(err)
	}
	firstCall := receive(t, importer.started)
	firstCall.finish <- importOutcome{}
	_ = receive(t, emitter.finished)
	cleanup := receive(t, maintenance.started)

	_, err = service.Start(context.Background(), ImportRequest{
		Kind: KindLichess, SourceID: "second", Path: "/second",
	})
	if err != nil {
		t.Fatal(err)
	}
	closedFirst := make(chan struct{})
	closedSecond := make(chan struct{})
	go func() {
		service.Close()
		close(closedFirst)
	}()
	go func() {
		service.Close()
		close(closedSecond)
	}()

	receive(t, cleanup.ctx.Done())
	assertNoReceive(t, closedFirst, "first Close returned while cleanup was still running")
	assertNoReceive(t, closedSecond, "second Close returned while cleanup was still running")
	releaseCleanupCall()
	secondCall := receive(t, importer.started)
	receive(t, secondCall.ctx.Done())
	assertNoReceive(t, closedFirst, "first Close returned while importer was still running")
	assertNoReceive(t, closedSecond, "second Close returned while importer was still running")
	releaseImporterCall()
	finished := receive(t, emitter.finished)
	if finished.Status != Cancelled {
		t.Fatalf("cancelled import finished as %q", finished.Status)
	}
	receive(t, closedFirst)
	receive(t, closedSecond)

	if _, err := service.Start(context.Background(), ImportRequest{
		Kind: KindLichess, SourceID: "later", Path: "/later",
	}); err == nil {
		t.Fatal("Start() unexpectedly succeeded after Close()")
	}
	service.Close()
}

func TestConcurrentStartAndCloseWaitsForEveryRegisteredJob(t *testing.T) {
	type startResult struct {
		jobID string
		err   error
	}
	for iteration := range 25 {
		importerGate := make(chan struct{})
		importer := newBlockingImporter()
		importer.afterCancel = importerGate
		service := NewService(map[Kind]Importer{KindLichess: importer}, nil, nil)
		begin := make(chan struct{})
		started := make(chan startResult, 1)
		closedFirst := make(chan struct{})
		closedSecond := make(chan struct{})

		go func() {
			<-begin
			jobID, err := service.Start(context.Background(), ImportRequest{
				Kind: KindLichess, SourceID: "race", Path: "/race",
			})
			started <- startResult{jobID: jobID, err: err}
		}()
		go func() {
			<-begin
			service.Close()
			close(closedFirst)
		}()
		go func() {
			<-begin
			service.Close()
			close(closedSecond)
		}()
		close(begin)

		result := receive(t, started)
		if result.err == nil {
			if result.jobID == "" {
				t.Fatalf("iteration %d successful Start returned an empty job ID", iteration)
			}
			call := receive(t, importer.started)
			receive(t, call.ctx.Done())
			assertNoReceive(t, closedFirst, "first concurrent Close missed a registered job")
			assertNoReceive(t, closedSecond, "second concurrent Close missed a registered job")
			close(importerGate)
		} else {
			close(importerGate)
		}
		receive(t, closedFirst)
		receive(t, closedSecond)
		service.Close()
	}
}

func receive[T any](t *testing.T, values <-chan T) T {
	t.Helper()
	select {
	case value := <-values:
		return value
	case <-time.After(testWait):
		t.Fatal("timed out waiting for channel value")
		var zero T
		return zero
	}
}

func assertNoReceive[T any](t *testing.T, values <-chan T, message string) {
	t.Helper()
	select {
	case value := <-values:
		t.Fatalf("%s: %+v", message, value)
	case <-time.After(testQuietPeriod):
	}
}
