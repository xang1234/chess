package importjob

import (
	"context"
	"testing"
	"time"

	"chess-trainer/internal/puzzles"
)

type blockingImporter struct {
	started chan context.Context
}

func (i *blockingImporter) Import(
	ctx context.Context,
	_ string,
	_ string,
	progress puzzles.ProgressSink,
) (puzzles.ImportReport, error) {
	i.started <- ctx
	progress(puzzles.Progress{RowsRead: 10, BytesRead: 20})
	<-ctx.Done()
	return puzzles.ImportReport{}, ctx.Err()
}

type emittedProgress struct {
	jobID    string
	progress puzzles.Progress
}

type captureEmitter struct {
	progress chan emittedProgress
	finished chan Result
}

func (e *captureEmitter) Progress(jobID string, progress puzzles.Progress) {
	e.progress <- emittedProgress{jobID: jobID, progress: progress}
}

func (e *captureEmitter) Finished(result Result) {
	e.finished <- result
}

func TestServiceRunsAndCancelsIndependentJobs(t *testing.T) {
	importer := &blockingImporter{started: make(chan context.Context, 2)}
	emitter := &captureEmitter{
		progress: make(chan emittedProgress, 2),
		finished: make(chan Result, 2),
	}
	service := NewService(importer, emitter)

	firstID, err := service.Start("/first.csv.zst")
	if err != nil {
		t.Fatal(err)
	}
	secondID, err := service.Start("/second.csv.zst")
	if err != nil {
		t.Fatal(err)
	}
	if firstID == secondID {
		t.Fatal("job IDs are not unique")
	}
	firstContext := <-importer.started
	secondContext := <-importer.started
	if firstContext == secondContext {
		t.Fatal("jobs share a context")
	}

	for range 2 {
		event := <-emitter.progress
		if event.jobID != firstID && event.jobID != secondID {
			t.Fatalf("progress jobID=%q", event.jobID)
		}
		if event.progress.RowsRead != 10 {
			t.Fatalf("progress=%+v", event.progress)
		}
	}
	result, err := service.Result(firstID)
	if err != nil || result.Status != Running {
		t.Fatalf("Result()=%+v err=%v", result, err)
	}

	if err := service.Cancel(firstID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-firstContext.Done():
	case <-time.After(time.Second):
		t.Fatal("first context was not cancelled")
	}
	select {
	case <-secondContext.Done():
		t.Fatal("cancelling first job cancelled second job")
	default:
	}

	finished := <-emitter.finished
	if finished.JobID != firstID || finished.Status != Cancelled {
		t.Fatalf("finished=%+v", finished)
	}
	result, err = service.Result(firstID)
	if err != nil || result.Status != Cancelled {
		t.Fatalf("Result()=%+v err=%v", result, err)
	}
	if secondContext.Err() != nil {
		t.Fatalf("second context err=%v", secondContext.Err())
	}
	if err := service.Cancel(secondID); err != nil {
		t.Fatal(err)
	}
	<-emitter.finished
}
