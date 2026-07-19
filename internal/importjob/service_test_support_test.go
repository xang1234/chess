package importjob

import (
	"context"
	"sync"
	"testing"
	"time"

	"chess-trainer/internal/importing"
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
	inspection importing.Inspection
	ctx        context.Context
	progress   importing.ProgressSink
	finish     chan importOutcome
}

type blockingImporter struct {
	started     chan startedImport
	afterCancel <-chan struct{}
}

func newBlockingImporter() *blockingImporter {
	return &blockingImporter{started: make(chan startedImport, 8)}
}

func supportsTestFormat(format puzzles.ImportFormat) bool {
	switch format {
	case puzzles.FormatLichess,
		puzzles.FormatTacticalPGN,
		puzzles.FormatCanonicalJSON,
		puzzles.FormatLucasFNS,
		puzzles.FormatLinearFENUCI:
		return true
	default:
		return false
	}
}

func (*blockingImporter) Supports(format puzzles.ImportFormat) bool {
	return supportsTestFormat(format)
}

func (i *blockingImporter) Import(
	ctx context.Context,
	inspection importing.Inspection,
	progress importing.ProgressSink,
) (puzzles.ImportReport, error) {
	call := startedImport{
		inspection: inspection,
		ctx:        ctx,
		progress:   progress,
		finish:     make(chan importOutcome, 1),
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
	progress []importing.Progress
	report   puzzles.ImportReport
}

type confirmedInspectionImporter struct {
	received chan importing.Inspection
}

func (confirmedInspectionImporter) Supports(format puzzles.ImportFormat) bool {
	return supportsTestFormat(format)
}

func (i confirmedInspectionImporter) Import(
	_ context.Context,
	inspection importing.Inspection,
	_ importing.ProgressSink,
) (puzzles.ImportReport, error) {
	i.received <- inspection
	return puzzles.ImportReport{}, nil
}

func (scriptedImporter) Supports(format puzzles.ImportFormat) bool {
	return supportsTestFormat(format)
}

func (i scriptedImporter) Import(
	_ context.Context,
	_ importing.Inspection,
	progress importing.ProgressSink,
) (puzzles.ImportReport, error) {
	for _, snapshot := range i.progress {
		progress(snapshot)
	}
	return i.report, nil
}

type activatedThenNilImporter struct {
	activated chan struct{}
	release   chan struct{}
}

func (activatedThenNilImporter) Supports(format puzzles.ImportFormat) bool {
	return supportsTestFormat(format)
}

func (i activatedThenNilImporter) Import(
	context.Context,
	importing.Inspection,
	importing.ProgressSink,
) (puzzles.ImportReport, error) {
	close(i.activated)
	<-i.release
	return puzzles.ImportReport{Accepted: 1}, nil
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
	snapshot importing.Progress
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

func (e *blockingFirstProgressEmitter) Progress(jobID string, progress importing.Progress) {
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

type startedNextJob struct {
	jobID string
	err   error
}

type staleCleanupOrderingEmitter struct {
	mu              sync.Mutex
	terminalCount   int
	service         *Service
	importer        *blockingImporter
	nextInspection  importing.Inspection
	nextJob         chan startedNextJob
	nextImport      chan startedImport
	terminalStarted chan Result
	releaseTerminal <-chan struct{}
	finished        chan Result
}

func (e *staleCleanupOrderingEmitter) Progress(string, importing.Progress) {}

func (e *staleCleanupOrderingEmitter) Finished(result Result) {
	e.mu.Lock()
	e.terminalCount++
	terminalNumber := e.terminalCount
	e.mu.Unlock()

	switch terminalNumber {
	case 1:
		jobID, err := e.service.Start(context.Background(), e.nextInspection)
		e.nextJob <- startedNextJob{jobID: jobID, err: err}
		if err == nil {
			e.nextImport <- <-e.importer.started
		}
	case 2:
		e.terminalStarted <- result
		<-e.releaseTerminal
	}
	e.finished <- result
}

func (e *blockingFirstTerminalEmitter) Progress(jobID string, progress importing.Progress) {
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

func (e *recordingEmitter) Progress(jobID string, progress importing.Progress) {
	e.progress <- emittedProgress{jobID: jobID, snapshot: progress}
}

func (e *recordingEmitter) Finished(result Result) {
	e.finished <- result
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
