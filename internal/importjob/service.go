package importjob

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"chess-trainer/internal/puzzles"

	"github.com/google/uuid"
)

const cleanupBatchSize = 1_000

type BusyError struct {
	ActiveJobID string `json:"activeJobId"`
}

func (e BusyError) Error() string {
	return fmt.Sprintf("puzzle import %q is already running", e.ActiveJobID)
}

type Status string

const (
	Running   Status = "running"
	Succeeded Status = "succeeded"
	Failed    Status = "failed"
	Cancelled Status = "cancelled"
)

type Result struct {
	JobID    string               `json:"jobId"`
	Status   Status               `json:"status"`
	Progress puzzles.Progress     `json:"progress"`
	Report   puzzles.ImportReport `json:"report"`
	Error    string               `json:"error,omitempty"`
}

type Emitter interface {
	Progress(string, puzzles.Progress)
	Finished(Result)
}

type Importer interface {
	Supports(puzzles.ImportFormat) bool
	Import(
		context.Context,
		puzzles.ImportInspection,
		puzzles.ProgressSink,
	) (puzzles.ImportReport, error)
}

type Maintenance interface {
	CleanupBatch(context.Context, int) (bool, error)
}

type jobState struct {
	cancel context.CancelFunc
	result Result
}

type Service struct {
	// When locks nest: writer precedes eventMu or mu, and eventMu precedes mu.
	// No path holds mu while waiting for either outer gate.
	mu             sync.Mutex
	eventMu        sync.Mutex
	writer         sync.Mutex
	importer       Importer
	maintenance    Maintenance
	emitter        Emitter
	jobs           map[string]*jobState
	activeJobID    string
	closing        bool
	cleanupCtx     context.Context
	cancelCleanup  context.CancelFunc
	cleanupRequest chan struct{}
	wg             sync.WaitGroup
}

func NewService(
	importer Importer,
	maintenance Maintenance,
	emitter Emitter,
) *Service {
	cleanupCtx, cancelCleanup := context.WithCancel(context.Background())
	service := &Service{
		importer:       importer,
		maintenance:    maintenance,
		emitter:        emitter,
		jobs:           make(map[string]*jobState),
		cleanupCtx:     cleanupCtx,
		cancelCleanup:  cancelCleanup,
		cleanupRequest: make(chan struct{}, 1),
	}
	service.wg.Add(1)
	go service.cleanupWorker()
	return service
}

func (s *Service) SetEmitter(emitter Emitter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.emitter = emitter
}

func (s *Service) Start(
	ctx context.Context,
	inspection puzzles.ImportInspection,
) (string, error) {
	if strings.TrimSpace(string(inspection.Format)) == "" {
		return "", errors.New("import kind is required")
	}
	if strings.TrimSpace(inspection.SourceID) == "" {
		return "", errors.New("import source ID is required")
	}
	if strings.TrimSpace(inspection.Path) == "" {
		return "", errors.New("import path is required")
	}
	if s.importer == nil || !s.importer.Supports(inspection.Format) {
		return "", fmt.Errorf("importer for kind %q is not configured", inspection.Format)
	}

	jobID := uuid.NewString()
	s.mu.Lock()
	if s.closing {
		s.mu.Unlock()
		return "", errors.New("import job service is closed")
	}
	if s.activeJobID != "" {
		activeJobID := s.activeJobID
		s.mu.Unlock()
		return "", &BusyError{ActiveJobID: activeJobID}
	}
	jobCtx, cancel := context.WithCancel(ctx)
	s.activeJobID = jobID
	s.jobs[jobID] = &jobState{
		cancel: cancel,
		result: Result{
			JobID: jobID, Status: Running,
			Progress: puzzles.Progress{Phase: puzzles.ImportDetecting},
		},
	}
	// Add and the closing check share mu so Close cannot begin Wait concurrently.
	s.wg.Add(1)
	s.mu.Unlock()

	go func() {
		defer s.wg.Done()
		defer cancel()
		s.run(jobCtx, jobID, inspection)
	}()
	return jobID, nil
}

func (s *Service) run(
	ctx context.Context,
	jobID string,
	inspection puzzles.ImportInspection,
) {
	// Import and cleanup writes share this gate. Never wait for it while holding mu.
	s.writer.Lock()
	report, err := s.importer.Import(
		ctx,
		inspection,
		func(progress puzzles.Progress) {
			s.recordProgress(jobID, progress)
		},
	)
	s.writer.Unlock()

	s.finish(jobID, report, err)
}

func (s *Service) recordProgress(jobID string, progress puzzles.Progress) {
	// Sequence state mutation with delivery so concurrent callbacks cannot expose
	// a newer snapshot before an older one or overtake a terminal event.
	s.eventMu.Lock()
	defer s.eventMu.Unlock()
	s.mu.Lock()
	state, exists := s.jobs[jobID]
	if !exists || state.result.Status != Running {
		s.mu.Unlock()
		return
	}
	state.result.Progress.RowsRead = max(state.result.Progress.RowsRead, progress.RowsRead)
	state.result.Progress.BytesRead = max(state.result.Progress.BytesRead, progress.BytesRead)
	state.result.Progress.TotalBytes = max(state.result.Progress.TotalBytes, progress.TotalBytes)
	if importPhaseRank(progress.Phase) >= importPhaseRank(state.result.Progress.Phase) {
		state.result.Progress.Phase = progress.Phase
	}
	snapshot := state.result.Progress
	emitter := s.emitter
	s.mu.Unlock()

	if emitter != nil {
		emitter.Progress(jobID, snapshot)
	}
}

func importPhaseRank(phase puzzles.ImportPhase) int {
	switch phase {
	case puzzles.ImportDetecting:
		return 0
	case puzzles.ImportParsing:
		return 1
	case puzzles.ImportSealing:
		return 2
	case puzzles.ImportActivating:
		return 3
	default:
		return -1
	}
}

func (s *Service) finish(
	jobID string,
	report puzzles.ImportReport,
	err error,
) {
	status := Succeeded
	errorText := ""
	switch {
	case errors.Is(err, context.Canceled):
		status = Cancelled
	case err != nil:
		status = Failed
		errorText = err.Error()
	}

	s.eventMu.Lock()
	s.mu.Lock()
	state, exists := s.jobs[jobID]
	if !exists || state.result.Status != Running {
		s.mu.Unlock()
		s.eventMu.Unlock()
		return
	}
	state.result.Status = status
	state.result.Report = cloneReport(report)
	state.result.Error = errorText
	state.cancel = nil
	if s.activeJobID == jobID {
		s.activeJobID = ""
	}
	result := cloneResult(state.result)
	emitter := s.emitter
	s.mu.Unlock()

	if emitter != nil {
		emitter.Finished(result)
	}
	s.eventMu.Unlock()
	s.requestCleanup()
}

func (s *Service) requestCleanup() {
	if s.maintenance == nil {
		return
	}
	select {
	case s.cleanupRequest <- struct{}{}:
	default:
	}
}

// RequestCleanup schedules one maintenance pass. The worker continues with
// further bounded batches only while no import is reserved.
func (s *Service) RequestCleanup() {
	s.requestCleanup()
}

func (s *Service) cleanupWorker() {
	defer s.wg.Done()
	for {
		select {
		case <-s.cleanupCtx.Done():
			return
		case <-s.cleanupRequest:
			s.cleanupUntilDone()
		}
	}
}

func (s *Service) cleanupUntilDone() {
	for {
		if s.cleanupCtx.Err() != nil {
			return
		}

		// Cleanup admission joins the event sequence before rechecking state. This
		// prevents a stale request from crossing a later job's terminal callback.
		// Start only takes mu, so it can still reserve the active slot during a batch.
		s.writer.Lock()
		s.eventMu.Lock()
		s.mu.Lock()
		if s.closing || s.activeJobID != "" {
			s.mu.Unlock()
			s.eventMu.Unlock()
			s.writer.Unlock()
			return
		}
		s.mu.Unlock()
		s.eventMu.Unlock()

		more, err := s.maintenance.CleanupBatch(s.cleanupCtx, cleanupBatchSize)
		s.writer.Unlock()
		if err != nil || !more {
			return
		}
	}
}

func (s *Service) Cancel(jobID string) error {
	s.mu.Lock()
	state, exists := s.jobs[jobID]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("unknown import job %q", jobID)
	}
	cancel := state.cancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

func (s *Service) Result(jobID string) (Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, exists := s.jobs[jobID]
	if !exists {
		return Result{}, fmt.Errorf("unknown import job %q", jobID)
	}
	return cloneResult(state.result), nil
}

func (s *Service) Close() {
	s.mu.Lock()
	s.closing = true
	var cancelActive context.CancelFunc
	if state, exists := s.jobs[s.activeJobID]; exists {
		cancelActive = state.cancel
	}
	s.mu.Unlock()

	if cancelActive != nil {
		cancelActive()
	}
	s.cancelCleanup()
	s.wg.Wait()
}

func cloneResult(result Result) Result {
	result.Report = cloneReport(result.Report)
	return result
}

func cloneReport(report puzzles.ImportReport) puzzles.ImportReport {
	report.Examples = append([]puzzles.Rejection(nil), report.Examples...)
	return report
}
