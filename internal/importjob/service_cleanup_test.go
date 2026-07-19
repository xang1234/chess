package importjob

import (
	"context"
	"sync"
	"testing"

	"chess-trainer/internal/importing"
	"chess-trainer/internal/puzzles"
)

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
	service := NewService(importer, nil, emitter)
	t.Cleanup(service.Close)
	t.Cleanup(releaseEmitter)

	jobID, err := service.Start(context.Background(), importing.Inspection{
		Format: puzzles.FormatLichess, SourceID: "lichess", Path: "/concurrent-progress",
	})
	if err != nil {
		t.Fatal(err)
	}
	call := receive(t, importer.started)
	firstReturned := make(chan struct{})
	go func() {
		call.progress(importing.Progress{RowsRead: 10, BytesRead: 20})
		close(firstReturned)
	}()
	firstStarted := receive(t, emitter.started)
	if firstStarted.jobID != jobID || firstStarted.snapshot.RowsRead != 10 {
		t.Fatalf("first progress = %+v", firstStarted)
	}

	secondReturned := make(chan struct{})
	go func() {
		call.progress(importing.Progress{RowsRead: 20, BytesRead: 30})
		close(secondReturned)
	}()
	assertNoReceive(t, secondReturned, "later progress overtook a blocked earlier callback")
	releaseEmitter()
	receive(t, firstReturned)
	receive(t, secondReturned)

	first := receive(t, emitter.progress)
	second := receive(t, emitter.progress)
	if first.snapshot != (importing.Progress{
		Phase: puzzles.ImportDetecting, RowsRead: 10, BytesRead: 20,
	}) || second.snapshot != (importing.Progress{
		Phase: puzzles.ImportDetecting, RowsRead: 20, BytesRead: 30,
	}) {
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
	service := NewService(importer, nil, emitter)
	t.Cleanup(service.Close)
	t.Cleanup(releaseEmitter)

	firstID, err := service.Start(context.Background(), importing.Inspection{
		Format: puzzles.FormatLichess, SourceID: "first", Path: "/first",
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

	secondID, err := service.Start(context.Background(), importing.Inspection{
		Format: puzzles.FormatLichess, SourceID: "second", Path: "/second",
	})
	if err != nil {
		t.Fatal(err)
	}
	secondCall := receive(t, importer.started)
	progressReturned := make(chan struct{})
	go func() {
		secondCall.progress(importing.Progress{RowsRead: 1})
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

func TestStaleCleanupWaitsForLaterTerminalEvent(t *testing.T) {
	releaseTerminal := make(chan struct{})
	var releaseOnce sync.Once
	releaseEmitter := func() { releaseOnce.Do(func() { close(releaseTerminal) }) }
	importer := newBlockingImporter()
	maintenance := newBlockingMaintenance()
	emitter := &staleCleanupOrderingEmitter{
		importer: importer,
		nextInspection: importing.Inspection{
			Format: puzzles.FormatLichess, SourceID: "second", Path: "/second",
		},
		nextJob:         make(chan startedNextJob, 1),
		nextImport:      make(chan startedImport, 1),
		terminalStarted: make(chan Result, 1),
		releaseTerminal: releaseTerminal,
		finished:        make(chan Result, 2),
	}
	service := NewService(importer, maintenance, emitter)
	emitter.service = service
	t.Cleanup(service.Close)
	t.Cleanup(releaseEmitter)

	firstID, err := service.Start(context.Background(), importing.Inspection{
		Format: puzzles.FormatLichess, SourceID: "first", Path: "/first",
	})
	if err != nil {
		t.Fatal(err)
	}
	firstCall := receive(t, importer.started)
	firstCall.finish <- importOutcome{}

	nextJob := receive(t, emitter.nextJob)
	if nextJob.err != nil {
		t.Fatalf("start second job from first terminal callback: %v", nextJob.err)
	}
	secondCall := receive(t, emitter.nextImport)
	firstFinished := receive(t, emitter.finished)
	if firstFinished.JobID != firstID {
		t.Fatalf("first terminal job = %q, want %q", firstFinished.JobID, firstID)
	}

	secondCall.finish <- importOutcome{}
	secondTerminal := receive(t, emitter.terminalStarted)
	if secondTerminal.JobID != nextJob.jobID {
		t.Fatalf("blocked terminal job = %q, want %q", secondTerminal.JobID, nextJob.jobID)
	}
	// Ensure a delayed job-one cleanup wake is pending only after job two has
	// cleared active state and entered its still-blocked terminal callback.
	select {
	case service.cleanupRequest <- struct{}{}:
	default:
	}
	assertNoReceive(t, maintenance.started, "stale cleanup began before the later terminal event returned")

	releaseEmitter()
	finished := receive(t, emitter.finished)
	if finished.JobID != nextJob.jobID {
		t.Fatalf("second terminal job = %q, want %q", finished.JobID, nextJob.jobID)
	}
	firstCleanup := receive(t, maintenance.started)
	firstCleanup.finish <- cleanupOutcome{}
}

func TestCleanupStartsAfterTerminalAndNeverOverlapsImport(t *testing.T) {
	importer := newBlockingImporter()
	maintenance := newBlockingMaintenance()
	emitter := newRecordingEmitter()
	service := NewService(importer, maintenance, emitter)
	t.Cleanup(service.Close)

	firstID, err := service.Start(context.Background(), importing.Inspection{
		Format: puzzles.FormatLichess, SourceID: "first", Path: "/first",
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

	_, err = service.Start(context.Background(), importing.Inspection{
		Format: puzzles.FormatLichess, SourceID: "second", Path: "/second",
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
	service := NewService(importer, maintenance, emitter)
	t.Cleanup(service.Close)

	_, err := service.Start(context.Background(), importing.Inspection{
		Format: puzzles.FormatLichess, SourceID: "first", Path: "/first",
	})
	if err != nil {
		t.Fatal(err)
	}
	firstCall := receive(t, importer.started)
	firstCall.finish <- importOutcome{}
	_ = receive(t, emitter.finished)
	firstCleanup := receive(t, maintenance.started)

	_, err = service.Start(context.Background(), importing.Inspection{
		Format: puzzles.FormatLichess, SourceID: "second", Path: "/second",
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
	service := NewService(importer, maintenance, emitter)

	_, err := service.Start(context.Background(), importing.Inspection{
		Format: puzzles.FormatLichess, SourceID: "first", Path: "/first",
	})
	if err != nil {
		t.Fatal(err)
	}
	firstCall := receive(t, importer.started)
	firstCall.finish <- importOutcome{}
	_ = receive(t, emitter.finished)
	cleanup := receive(t, maintenance.started)

	_, err = service.Start(context.Background(), importing.Inspection{
		Format: puzzles.FormatLichess, SourceID: "second", Path: "/second",
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

	if _, err := service.Start(context.Background(), importing.Inspection{
		Format: puzzles.FormatLichess, SourceID: "later", Path: "/later",
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
		service := NewService(importer, nil, nil)
		begin := make(chan struct{})
		started := make(chan startResult, 1)
		closedFirst := make(chan struct{})
		closedSecond := make(chan struct{})

		go func() {
			<-begin
			jobID, err := service.Start(context.Background(), importing.Inspection{
				Format: puzzles.FormatLichess, SourceID: "race", Path: "/race",
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
