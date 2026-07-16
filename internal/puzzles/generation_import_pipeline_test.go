package puzzles

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"
)

const pipelineTestTimeout = 2 * time.Second

type pipelineRecordingImport struct {
	mu sync.Mutex

	events []string

	addEntered chan struct{}
	releaseAdd chan struct{}
	addErr     error
	waitForCtx bool

	sealEntered    chan struct{}
	abandonEntered chan struct{}
}

func (i *pipelineRecordingImport) Add(ctx context.Context, puzzle TrainingPuzzle) error {
	i.record("add:" + puzzle.Occurrence.ExternalID)
	if i.addEntered != nil {
		select {
		case i.addEntered <- struct{}{}:
		default:
		}
	}
	if i.waitForCtx {
		<-ctx.Done()
		i.record("add-cancelled")
		return ctx.Err()
	}
	if i.releaseAdd != nil {
		select {
		case <-i.releaseAdd:
		case <-ctx.Done():
			i.record("add-cancelled")
			return ctx.Err()
		}
	}
	return i.addErr
}

func (i *pipelineRecordingImport) Reject(rejection Rejection) {
	i.record(fmt.Sprintf("reject:%d", rejection.Ordinal))
}

func (i *pipelineRecordingImport) Seal(context.Context, string) (ImportReport, error) {
	i.record("seal")
	if i.sealEntered != nil {
		close(i.sealEntered)
	}
	return ImportReport{}, nil
}

func (i *pipelineRecordingImport) Activate(context.Context) error {
	i.record("activate")
	return nil
}

func (i *pipelineRecordingImport) Abandon(context.Context) error {
	i.record("abandon")
	if i.abandonEntered != nil {
		close(i.abandonEntered)
	}
	return nil
}

func (i *pipelineRecordingImport) record(event string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.events = append(i.events, event)
}

func (i *pipelineRecordingImport) recordedEvents() []string {
	i.mu.Lock()
	defer i.mu.Unlock()
	return append([]string(nil), i.events...)
}

func pipelinePuzzle(id string) TrainingPuzzle {
	return TrainingPuzzle{Occurrence: PuzzleOccurrence{ExternalID: id}}
}

func waitPipelineTest(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(pipelineTestTimeout):
		t.Fatalf("timed out waiting for %s", description)
	}
}

func assertPipelineTestBlocked(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-signal:
		t.Fatalf("%s completed before it was safe", description)
	case <-time.After(25 * time.Millisecond):
	}
}

func TestOrderedGenerationImportOverlapsProducerWithWriter(t *testing.T) {
	underlying := &pipelineRecordingImport{
		addEntered: make(chan struct{}, 1),
		releaseAdd: make(chan struct{}),
	}
	queued := newOrderedGenerationImport(context.Background(), underlying)

	if err := queued.Add(context.Background(), pipelinePuzzle("first")); err != nil {
		t.Fatal(err)
	}
	waitPipelineTest(t, underlying.addEntered, "the writer to enter Add")

	producerReturned := make(chan struct{})
	go func() {
		defer close(producerReturned)
		if err := queued.Add(context.Background(), pipelinePuzzle("second")); err != nil {
			t.Errorf("enqueue second puzzle: %v", err)
		}
	}()
	waitPipelineTest(t, producerReturned, "the producer to enqueue while the writer is busy")

	close(underlying.releaseAdd)
	if _, err := queued.Seal(context.Background(), "checksum"); err != nil {
		t.Fatal(err)
	}
	if got, want := underlying.recordedEvents(), []string{"add:first", "add:second", "seal"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %q, want %q", got, want)
	}
}

func TestOrderedGenerationImportPreservesAddRejectOrder(t *testing.T) {
	underlying := &pipelineRecordingImport{}
	queued := newOrderedGenerationImport(context.Background(), underlying)

	if err := queued.Add(context.Background(), pipelinePuzzle("first")); err != nil {
		t.Fatal(err)
	}
	queued.Reject(Rejection{Ordinal: 7})
	if err := queued.Add(context.Background(), pipelinePuzzle("second")); err != nil {
		t.Fatal(err)
	}
	if _, err := queued.Seal(context.Background(), "checksum"); err != nil {
		t.Fatal(err)
	}

	if got, want := underlying.recordedEvents(), []string{"add:first", "reject:7", "add:second", "seal"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %q, want %q", got, want)
	}
}

func TestOrderedGenerationImportWriterErrorUnblocksProducer(t *testing.T) {
	writeErr := errors.New("write failed")
	underlying := &pipelineRecordingImport{
		addEntered: make(chan struct{}, 1),
		releaseAdd: make(chan struct{}),
		addErr:     writeErr,
	}
	queued := newOrderedGenerationImport(context.Background(), underlying)

	if err := queued.Add(context.Background(), pipelinePuzzle("failing")); err != nil {
		t.Fatal(err)
	}
	waitPipelineTest(t, underlying.addEntered, "the failing writer call")
	for index := 0; index < orderedGenerationImportQueueCapacity; index++ {
		if err := queued.Add(context.Background(), pipelinePuzzle(fmt.Sprintf("queued-%d", index))); err != nil {
			t.Fatalf("fill queue at %d: %v", index, err)
		}
	}

	producerResult := make(chan error, 1)
	go func() {
		producerResult <- queued.Add(context.Background(), pipelinePuzzle("blocked"))
	}()
	select {
	case err := <-producerResult:
		t.Fatalf("producer on a full queue returned early with %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	close(underlying.releaseAdd)
	select {
	case err := <-producerResult:
		if !errors.Is(err, writeErr) {
			t.Fatalf("blocked Add error = %v, want %v", err, writeErr)
		}
	case <-time.After(pipelineTestTimeout):
		t.Fatal("writer error did not unblock producer")
	}
	if _, err := queued.Seal(context.Background(), "checksum"); !errors.Is(err, writeErr) {
		t.Fatalf("Seal error = %v, want %v", err, writeErr)
	}
}

func TestOrderedGenerationImportCancellationUnblocksWriterAndProducer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	underlying := &pipelineRecordingImport{
		addEntered: make(chan struct{}, 1),
		waitForCtx: true,
	}
	queued := newOrderedGenerationImport(ctx, underlying)

	if err := queued.Add(ctx, pipelinePuzzle("first")); err != nil {
		t.Fatal(err)
	}
	waitPipelineTest(t, underlying.addEntered, "the cancellable writer call")
	for index := 0; index < orderedGenerationImportQueueCapacity; index++ {
		if err := queued.Add(ctx, pipelinePuzzle(fmt.Sprintf("queued-%d", index))); err != nil {
			t.Fatalf("fill queue at %d: %v", index, err)
		}
	}
	producerResult := make(chan error, 1)
	go func() {
		producerResult <- queued.Add(ctx, pipelinePuzzle("blocked"))
	}()
	cancel()
	select {
	case err := <-producerResult:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("blocked Add error = %v, want context cancellation", err)
		}
	case <-time.After(pipelineTestTimeout):
		t.Fatal("cancellation did not unblock producer")
	}

	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), pipelineTestTimeout)
	defer cleanupCancel()
	if err := queued.Abandon(cleanupCtx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Abandon error = %v, want joined writer cancellation", err)
	}
	waitPipelineTest(t, queued.done, "the writer goroutine to stop")
	if got := underlying.recordedEvents(); len(got) < 3 || got[0] != "add:first" || got[1] != "add-cancelled" || got[len(got)-1] != "abandon" {
		t.Fatalf("events = %q, want writer cancellation before abandon", got)
	}
}

func TestOrderedGenerationImportDrainsBeforeSeal(t *testing.T) {
	underlying := &pipelineRecordingImport{
		addEntered:  make(chan struct{}, 1),
		releaseAdd:  make(chan struct{}),
		sealEntered: make(chan struct{}),
	}
	queued := newOrderedGenerationImport(context.Background(), underlying)
	if err := queued.Add(context.Background(), pipelinePuzzle("first")); err != nil {
		t.Fatal(err)
	}
	waitPipelineTest(t, underlying.addEntered, "the writer call")

	sealed := make(chan error, 1)
	go func() {
		_, err := queued.Seal(context.Background(), "checksum")
		sealed <- err
	}()
	assertPipelineTestBlocked(t, underlying.sealEntered, "Seal")
	close(underlying.releaseAdd)
	if err := <-sealed; err != nil {
		t.Fatal(err)
	}
	waitPipelineTest(t, underlying.sealEntered, "underlying Seal")
}

func TestOrderedGenerationImportAbandonWaitsForWriterExit(t *testing.T) {
	underlying := &pipelineRecordingImport{
		addEntered:     make(chan struct{}, 1),
		waitForCtx:     true,
		abandonEntered: make(chan struct{}),
	}
	queued := newOrderedGenerationImport(context.Background(), underlying)
	if err := queued.Add(context.Background(), pipelinePuzzle("first")); err != nil {
		t.Fatal(err)
	}
	waitPipelineTest(t, underlying.addEntered, "the writer call")

	cleanupCtx, cancel := context.WithTimeout(context.Background(), pipelineTestTimeout)
	defer cancel()
	if err := queued.Abandon(cleanupCtx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Abandon error = %v, want cancelled writer error", err)
	}
	waitPipelineTest(t, underlying.abandonEntered, "underlying Abandon")
	waitPipelineTest(t, queued.done, "writer goroutine exit")
	if got, want := underlying.recordedEvents(), []string{"add:first", "add-cancelled", "abandon"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %q, want %q", got, want)
	}
}

func TestOrderedGenerationImportSealDeadlineCancelsBlockedWriterAndSender(t *testing.T) {
	underlying := &pipelineRecordingImport{
		addEntered: make(chan struct{}, 1),
		waitForCtx: true,
	}
	queued := newOrderedGenerationImport(context.Background(), underlying)
	if err := queued.Add(context.Background(), pipelinePuzzle("stalled")); err != nil {
		t.Fatal(err)
	}
	waitPipelineTest(t, underlying.addEntered, "the stalled writer call")
	for index := 0; index < orderedGenerationImportQueueCapacity; index++ {
		if err := queued.Add(context.Background(), pipelinePuzzle(fmt.Sprintf("queued-%d", index))); err != nil {
			t.Fatalf("fill queue at %d: %v", index, err)
		}
	}
	blockedAdd := make(chan error, 1)
	go func() {
		blockedAdd <- queued.Add(context.Background(), pipelinePuzzle("blocked-sender"))
	}()
	select {
	case err := <-blockedAdd:
		t.Fatalf("full-queue sender returned early with %v", err)
	case <-time.After(25 * time.Millisecond):
	}

	sealContext, cancelSeal := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancelSeal()
	sealResult := make(chan error, 1)
	go func() {
		_, err := queued.Seal(sealContext, "checksum")
		sealResult <- err
	}()
	select {
	case err := <-sealResult:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Seal error = %v, want deadline exceeded", err)
		}
	case <-time.After(250 * time.Millisecond):
		// Unblock the old implementation before failing so the test itself
		// never leaves a writer or producer goroutine behind.
		queued.cancelWorker()
		select {
		case <-sealResult:
		case <-time.After(pipelineTestTimeout):
			t.Fatal("Seal and writer remained leaked after emergency cancellation")
		}
		t.Fatal("Seal ignored its deadline while waiting for a blocked sender")
	}

	select {
	case err := <-blockedAdd:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("blocked Add error = %v, want writer cancellation", err)
		}
	case <-time.After(pipelineTestTimeout):
		t.Fatal("deadline cancellation did not unblock the producer")
	}
	waitPipelineTest(t, queued.done, "deadline-cancelled writer goroutine exit")
	cleanupContext, cancelCleanup := context.WithTimeout(context.Background(), pipelineTestTimeout)
	defer cancelCleanup()
	if err := queued.Abandon(cleanupContext); !errors.Is(err, context.Canceled) {
		t.Fatalf("Abandon error = %v, want joined writer cancellation", err)
	}
	for _, event := range underlying.recordedEvents() {
		if event == "seal" {
			t.Fatalf("underlying Seal ran before the deadline barrier: %q", underlying.recordedEvents())
		}
	}
}
