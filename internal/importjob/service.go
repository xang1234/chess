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

type Status string

const (
	Running   Status = "running"
	Succeeded Status = "succeeded"
	Failed    Status = "failed"
	Cancelled Status = "cancelled"
)

type Result struct {
	JobID  string               `json:"jobId"`
	Status Status               `json:"status"`
	Report puzzles.ImportReport `json:"report"`
	Error  string               `json:"error,omitempty"`
}

type Emitter interface {
	Progress(string, puzzles.Progress)
	Finished(Result)
}

type Importer interface {
	Import(context.Context, string, string, puzzles.ProgressSink) (puzzles.ImportReport, error)
}

type jobState struct {
	cancel context.CancelFunc
	result Result
}

type Service struct {
	mu       sync.Mutex
	importer Importer
	emitter  Emitter
	jobs     map[string]*jobState
}

func NewService(importer Importer, emitter Emitter) *Service {
	return &Service{
		importer: importer,
		emitter:  emitter,
		jobs:     make(map[string]*jobState),
	}
}

func (s *Service) SetEmitter(emitter Emitter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.emitter = emitter
}

func (s *Service) Start(path string) (string, error) {
	if s.importer == nil {
		return "", errors.New("importer is not configured")
	}
	if strings.TrimSpace(path) == "" {
		return "", errors.New("import path is required")
	}
	jobID := uuid.NewString()
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.jobs[jobID] = &jobState{
		cancel: cancel,
		result: Result{JobID: jobID, Status: Running},
	}
	s.mu.Unlock()

	go s.run(ctx, jobID, path)
	return jobID, nil
}

func (s *Service) run(ctx context.Context, jobID string, path string) {
	report, err := s.importer.Import(ctx, "lichess", path, func(progress puzzles.Progress) {
		s.mu.Lock()
		emitter := s.emitter
		s.mu.Unlock()
		if emitter != nil {
			emitter.Progress(jobID, progress)
		}
	})
	result := Result{JobID: jobID, Report: report}
	switch {
	case errors.Is(err, context.Canceled):
		result.Status = Cancelled
	case err != nil:
		result.Status = Failed
		result.Error = err.Error()
	default:
		result.Status = Succeeded
	}

	s.mu.Lock()
	state, exists := s.jobs[jobID]
	if exists {
		state.result = result
		state.cancel = nil
	}
	emitter := s.emitter
	s.mu.Unlock()
	if emitter != nil {
		emitter.Finished(result)
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
	return state.result, nil
}
