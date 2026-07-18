package importjob

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"chess-trainer/internal/puzzles"
)

func TestStartPassesCanonicalFormatToImporter(t *testing.T) {
	importer := newBlockingImporter()
	emitter := newRecordingEmitter()
	service := NewService(importer, nil, emitter)
	t.Cleanup(service.Close)

	inspection := puzzles.ImportInspection{
		Format: puzzles.FormatCanonicalJSON, SourceID: "club", Path: "/club.json",
	}
	jobID, err := service.Start(context.Background(), inspection)
	if err != nil {
		t.Fatal(err)
	}
	call := receive(t, importer.started)
	if call.inspection != inspection {
		t.Fatalf("imported inspection = %+v, want %+v", call.inspection, inspection)
	}
	call.finish <- importOutcome{report: puzzles.ImportReport{Accepted: 3}}

	finished := receive(t, emitter.finished)
	if finished.JobID != jobID || finished.Status != Succeeded {
		t.Fatalf("finished = %+v", finished)
	}
}

func TestResultJSONOmitsInspection(t *testing.T) {
	payload, err := json.Marshal(Result{
		JobID:    "job-1",
		Status:   Succeeded,
		Progress: puzzles.Progress{Phase: puzzles.ImportActivating, RowsRead: 3},
		Report:   puzzles.ImportReport{Accepted: 3},
	})
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]json.RawMessage
	if err := json.Unmarshal(payload, &wire); err != nil {
		t.Fatal(err)
	}
	if _, exists := wire["inspection"]; exists {
		t.Fatalf("serialized Result exposes inspection: %s", payload)
	}
	for _, required := range []string{"jobId", "status", "progress", "report"} {
		if _, exists := wire[required]; !exists {
			t.Fatalf("serialized Result lacks %q: %s", required, payload)
		}
	}
}

func TestStartPassesConfirmedInspectionToImporter(t *testing.T) {
	importer := confirmedInspectionImporter{received: make(chan puzzles.ImportInspection, 1)}
	service := NewService(importer, nil, nil)
	t.Cleanup(service.Close)
	inspection := puzzles.ImportInspection{
		Path: "/club.json", Filename: "club.json", Format: puzzles.FormatCanonicalJSON,
		SourceID: "club", SourceIDOrigin: puzzles.SourceIDEmbedded, SourceName: "Club",
	}

	if _, err := service.Start(context.Background(), inspection); err != nil {
		t.Fatal(err)
	}
	if got := receive(t, importer.received); got != inspection {
		t.Fatalf("imported inspection = %+v, want %+v", got, inspection)
	}
}

func TestStartSeedsDetectingProgressBeforePublishingJob(t *testing.T) {
	importer := newBlockingImporter()
	service := NewService(importer, nil, nil)
	t.Cleanup(service.Close)

	jobID, err := service.Start(context.Background(), puzzles.ImportInspection{
		Format: puzzles.FormatLichess, SourceID: "lichess", Path: "/puzzles.csv.zst",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := service.Result(jobID)
	if err != nil {
		t.Fatal(err)
	}
	want := puzzles.Progress{Phase: puzzles.ImportDetecting}
	if result.Status != Running || result.Progress != want {
		t.Fatalf("immediate Result() = %+v, want running progress %+v", result, want)
	}

	call := receive(t, importer.started)
	call.finish <- importOutcome{}
}

func TestStartValidatesInspection(t *testing.T) {
	service := NewService(newBlockingImporter(), nil, nil)
	t.Cleanup(service.Close)

	tests := []struct {
		name       string
		inspection puzzles.ImportInspection
	}{
		{name: "kind", inspection: puzzles.ImportInspection{SourceID: "source", Path: "/puzzles"}},
		{name: "configured kind", inspection: puzzles.ImportInspection{Format: "missing", SourceID: "source", Path: "/puzzles"}},
		{name: "source", inspection: puzzles.ImportInspection{Format: puzzles.FormatLichess, Path: "/puzzles"}},
		{name: "path", inspection: puzzles.ImportInspection{Format: puzzles.FormatLichess, SourceID: "source"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := service.Start(context.Background(), test.inspection); err == nil {
				t.Fatalf("Start(%+v) unexpectedly succeeded", test.inspection)
			}
		})
	}
}

func TestStartRejectsSecondActivePuzzleImport(t *testing.T) {
	importer := newBlockingImporter()
	service := NewService(importer, nil, nil)
	t.Cleanup(service.Close)
	inspection := puzzles.ImportInspection{Format: puzzles.FormatLichess, SourceID: "lichess", Path: "/first.csv.zst"}

	activeJobID, err := service.Start(context.Background(), inspection)
	if err != nil {
		t.Fatal(err)
	}
	call := receive(t, importer.started)
	_, err = service.Start(context.Background(), puzzles.ImportInspection{
		Format: puzzles.FormatLichess, SourceID: "lichess", Path: "/second.csv.zst",
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
	service := NewService(scriptedImporter{
		progress: []puzzles.Progress{
			{RowsRead: 10, BytesRead: 20},
			{RowsRead: 5, BytesRead: 25},
			{RowsRead: 12, BytesRead: 15},
		},
		report: puzzles.ImportReport{Accepted: 9},
	}, nil, emitter)
	t.Cleanup(service.Close)
	inspection := puzzles.ImportInspection{Format: puzzles.FormatLichess, SourceID: "lichess", Path: "/puzzles.csv.zst"}

	jobID, err := service.Start(context.Background(), inspection)
	if err != nil {
		t.Fatal(err)
	}
	want := []puzzles.Progress{
		{Phase: puzzles.ImportDetecting, RowsRead: 10, BytesRead: 20},
		{Phase: puzzles.ImportDetecting, RowsRead: 10, BytesRead: 25},
		{Phase: puzzles.ImportDetecting, RowsRead: 12, BytesRead: 25},
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

func TestImportProgressPhaseAndTotalsRemainMonotonic(t *testing.T) {
	emitter := newRecordingEmitter()
	service := NewService(scriptedImporter{
		progress: []puzzles.Progress{
			{Phase: puzzles.ImportDetecting, RowsRead: 1, BytesRead: 2, TotalBytes: 100},
			{Phase: puzzles.ImportSealing, RowsRead: 10, BytesRead: 90, TotalBytes: 100},
			{Phase: puzzles.ImportParsing, RowsRead: 12, BytesRead: 80, TotalBytes: 50},
			{Phase: puzzles.ImportActivating, RowsRead: 11, BytesRead: 100, TotalBytes: 120},
		},
	}, nil, emitter)
	t.Cleanup(service.Close)

	jobID, err := service.Start(context.Background(), puzzles.ImportInspection{
		Format: puzzles.FormatCanonicalJSON, SourceID: "club", Path: "/club.json",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []puzzles.Progress{
		{Phase: puzzles.ImportDetecting, RowsRead: 1, BytesRead: 2, TotalBytes: 100},
		{Phase: puzzles.ImportSealing, RowsRead: 10, BytesRead: 90, TotalBytes: 100},
		{Phase: puzzles.ImportSealing, RowsRead: 12, BytesRead: 90, TotalBytes: 100},
		{Phase: puzzles.ImportActivating, RowsRead: 12, BytesRead: 100, TotalBytes: 120},
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
	stored, err := service.Result(jobID)
	if err != nil || stored.Progress != want[len(want)-1] {
		t.Fatalf("Result() = %+v, %v", stored, err)
	}

	kinds := map[puzzles.ImportFormat]string{
		puzzles.FormatLichess:       "lichess",
		puzzles.FormatTacticalPGN:   "tactical-pgn",
		puzzles.FormatCanonicalJSON: "canonical-json",
		puzzles.FormatLucasFNS:      "lucas-fns",
		puzzles.FormatLinearFENUCI:  "linear-fen-uci",
	}
	for kind, expected := range kinds {
		if string(kind) != expected {
			t.Fatalf("kind = %q, want %q", kind, expected)
		}
	}
}

func TestCompletedResultRemainsQueryableAfterLaterJob(t *testing.T) {
	importer := newBlockingImporter()
	emitter := newRecordingEmitter()
	service := NewService(importer, nil, emitter)
	t.Cleanup(service.Close)

	firstInspection := puzzles.ImportInspection{Format: puzzles.FormatLichess, SourceID: "first", Path: "/first"}
	firstID, err := service.Start(context.Background(), firstInspection)
	if err != nil {
		t.Fatal(err)
	}
	firstCall := receive(t, importer.started)
	firstCall.progress(puzzles.Progress{RowsRead: 7, BytesRead: 11})
	firstCall.finish <- importOutcome{report: puzzles.ImportReport{Accepted: 1}}
	_ = receive(t, emitter.progress)
	_ = receive(t, emitter.finished)

	secondInspection := puzzles.ImportInspection{Format: puzzles.FormatLichess, SourceID: "second", Path: "/second"}
	secondID, err := service.Start(context.Background(), secondInspection)
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
	if result.Status != Succeeded || result.Progress != (puzzles.Progress{
		Phase: puzzles.ImportDetecting, RowsRead: 7, BytesRead: 11,
	}) || result.Report.Accepted != 1 {
		t.Fatalf("first result after later job = %+v", result)
	}
}

func TestJobReachesExactlyOneTerminalState(t *testing.T) {
	importer := newBlockingImporter()
	emitter := newRecordingEmitter()
	service := NewService(importer, nil, emitter)
	inspection := puzzles.ImportInspection{Format: puzzles.FormatLichess, SourceID: "lichess", Path: "/cancel"}

	jobID, err := service.Start(context.Background(), inspection)
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

func TestSuccessfulImportRemainsSucceededWhenContextCancelledAfterActivation(t *testing.T) {
	importer := activatedThenNilImporter{
		activated: make(chan struct{}),
		release:   make(chan struct{}),
	}
	emitter := newRecordingEmitter()
	service := NewService(importer, nil, emitter)
	t.Cleanup(service.Close)

	ctx, cancel := context.WithCancel(context.Background())
	jobID, err := service.Start(ctx, puzzles.ImportInspection{
		Format: puzzles.FormatLichess, SourceID: "lichess", Path: "/activated",
	})
	if err != nil {
		t.Fatal(err)
	}
	<-importer.activated
	cancel()
	close(importer.release)

	finished := receive(t, emitter.finished)
	if finished.JobID != jobID || finished.Status != Succeeded || finished.Report.Accepted != 1 {
		t.Fatalf("finished after successful activation = %+v", finished)
	}
}

func TestTerminalJobAllowsNextImport(t *testing.T) {
	importer := newBlockingImporter()
	emitter := newRecordingEmitter()
	service := NewService(importer, nil, emitter)
	t.Cleanup(service.Close)

	firstID, err := service.Start(context.Background(), puzzles.ImportInspection{
		Format: puzzles.FormatLichess, SourceID: "first", Path: "/first",
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

	secondID, err := service.Start(context.Background(), puzzles.ImportInspection{
		Format: puzzles.FormatLichess, SourceID: "second", Path: "/second",
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
