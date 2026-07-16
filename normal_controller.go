package main

import (
	"context"

	appservices "chess-trainer/internal/app"
	"chess-trainer/internal/domain"
	"chess-trainer/internal/importjob"
	"chess-trainer/internal/profile"
	"chess-trainer/internal/puzzles"
	"chess-trainer/internal/training"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type NormalController struct {
	actions  *controllerActions
	services *appservices.Services
}

func NewNormalController(services *appservices.Services) *NormalController {
	return &NormalController{
		actions: newControllerActions(services.Paths, services.Backup), services: services,
	}
}

func (c *NormalController) startup(ctx context.Context) {
	c.actions.startup(ctx)
	c.services.ImportJobs.SetEmitter(wailsEmitter{ctx: ctx})
}

func (c *NormalController) CreateBackup(includeLibrary bool) (string, error) {
	if err := c.services.EnsureRunning(); err != nil {
		return "", err
	}
	return c.actions.createBackup(includeLibrary)
}

func (c *NormalController) RestoreBackup(path string) error {
	if err := c.services.EnsureRunning(); err != nil {
		return err
	}
	return c.actions.restoreBackup(path)
}

func (c *NormalController) OpenDataFolder() {
	c.actions.openDataFolder()
}

func (c *NormalController) Quit() {
	c.actions.quit()
}

func (c *NormalController) StartLichessImport(path string) (string, error) {
	return c.StartPuzzleImport(importjob.ImportRequest{
		Kind:     importjob.KindLichess,
		SourceID: "lichess",
		Path:     path,
	})
}

func (c *NormalController) ChoosePuzzleImportFile() (string, error) {
	return c.actions.dialogs.OpenFileDialog(c.actions.ctx, runtime.OpenDialogOptions{
		Title: "Choose a Lichess puzzle database",
		Filters: []runtime.FileFilter{{
			DisplayName: "Compressed CSV (*.csv.zst)", Pattern: "*.csv.zst",
		}},
	})
}

func (c *NormalController) StartPuzzleImport(request importjob.ImportRequest) (string, error) {
	if err := c.services.EnsureRunning(); err != nil {
		return "", err
	}
	return c.services.ImportJobs.Start(c.actions.ctx, request)
}

func (c *NormalController) CancelImport(jobID string) error {
	if err := c.services.EnsureRunning(); err != nil {
		return err
	}
	return c.services.ImportJobs.Cancel(jobID)
}

func (c *NormalController) GetImportResult(jobID string) (importjob.Result, error) {
	if err := c.services.EnsureRunning(); err != nil {
		return importjob.Result{}, err
	}
	return c.services.ImportJobs.Result(jobID)
}

func (c *NormalController) StartGuided() (domain.SessionView, error) {
	if err := c.services.EnsureRunning(); err != nil {
		return domain.SessionView{}, err
	}
	return c.services.Training.StartGuided(c.actions.ctx)
}

func (c *NormalController) StartFreePractice(request training.PracticeRequest) (domain.SessionView, error) {
	if err := c.services.EnsureRunning(); err != nil {
		return domain.SessionView{}, err
	}
	return c.services.Training.StartFreePractice(c.actions.ctx, request)
}

func (c *NormalController) ResumeSession() (*domain.SessionView, error) {
	if err := c.services.EnsureRunning(); err != nil {
		return nil, err
	}
	return c.services.Training.Resume(c.actions.ctx)
}

func (c *NormalController) PlayMove(sessionID, uci string) (domain.MoveResult, error) {
	if err := c.services.EnsureRunning(); err != nil {
		return domain.MoveResult{}, err
	}
	return c.services.Training.PlayMove(c.actions.ctx, sessionID, uci)
}

func (c *NormalController) UseHint(sessionID string) (domain.HintResult, error) {
	if err := c.services.EnsureRunning(); err != nil {
		return domain.HintResult{}, err
	}
	return c.services.Training.UseHint(c.actions.ctx, sessionID)
}

func (c *NormalController) RevealSolution(sessionID string) (domain.MoveResult, error) {
	if err := c.services.EnsureRunning(); err != nil {
		return domain.MoveResult{}, err
	}
	return c.services.Training.Reveal(c.actions.ctx, sessionID)
}

func (c *NormalController) PauseSession(sessionID string) error {
	if err := c.services.EnsureRunning(); err != nil {
		return err
	}
	return c.services.Training.Pause(c.actions.ctx, sessionID)
}

func (c *NormalController) GetProfile() (*training.Profile, error) {
	if err := c.services.EnsureRunning(); err != nil {
		return nil, err
	}
	return c.services.Profile.Get(c.actions.ctx)
}

func (c *NormalController) UpdateProfile(value training.Profile) error {
	if err := c.services.EnsureRunning(); err != nil {
		return err
	}
	return c.services.Profile.UpdateSettings(c.actions.ctx, value)
}

func (c *NormalController) GetParentSummary() (profile.Summary, error) {
	if err := c.services.EnsureRunning(); err != nil {
		return profile.Summary{}, err
	}
	return c.services.Profile.Summary(c.actions.ctx)
}

func (c *NormalController) GetPracticeFilters() (profile.PracticeFilters, error) {
	if err := c.services.EnsureRunning(); err != nil {
		return profile.PracticeFilters{}, err
	}
	return c.services.Profile.PracticeFilters(c.actions.ctx)
}

type wailsEmitter struct {
	ctx context.Context
}

func (e wailsEmitter) Progress(jobID string, progress puzzles.Progress) {
	runtime.EventsEmit(e.ctx, "import:progress", struct {
		JobID string `json:"jobId"`
		puzzles.Progress
	}{JobID: jobID, Progress: progress})
}

func (e wailsEmitter) Finished(result importjob.Result) {
	runtime.EventsEmit(e.ctx, "import:finished", result)
}
