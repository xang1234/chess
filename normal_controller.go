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
		actions: newControllerActions(services.Paths), services: services,
	}
}

func (c *NormalController) startup(ctx context.Context) {
	c.actions.startup(ctx)
	c.services.ImportJobs.SetEmitter(wailsEmitter{ctx: ctx})
}

func (c *NormalController) CreateBackup(includeLibrary bool) (string, error) {
	destination, err := c.actions.chooseBackupDestination()
	if err != nil || destination == "" {
		return destination, err
	}
	if err := c.services.CreateBackup(c.actions.ctx, destination, includeLibrary); err != nil {
		return "", err
	}
	return destination, nil
}

func (c *NormalController) RestoreBackup(path string) error {
	if path == "" {
		var err error
		path, err = c.actions.chooseRestoreSource()
		if err != nil || path == "" {
			return err
		}
	}
	if err := c.services.RestoreBackup(c.actions.ctx, path); err != nil {
		return err
	}
	c.actions.finishRestore()
	return nil
}

func (c *NormalController) OpenDataFolder() {
	c.actions.openDataFolder()
}

func (c *NormalController) Quit() {
	c.actions.quit()
}

func (c *NormalController) StartLichessImport(path string) (string, error) {
	return c.StartPuzzleImport(path)
}

func (c *NormalController) ChoosePuzzleImportFile() (string, error) {
	return c.actions.dialogs.OpenFileDialog(c.actions.ctx, runtime.OpenDialogOptions{
		Title: "Choose a puzzle collection",
		Filters: []runtime.FileFilter{
			{DisplayName: "Zstandard archive (*.zst)", Pattern: "*.zst"},
			{DisplayName: "PGN collection (*.pgn)", Pattern: "*.pgn"},
			{DisplayName: "JSON collection (*.json)", Pattern: "*.json"},
			{DisplayName: "Lucas collection (*.fns)", Pattern: "*.fns"},
			{DisplayName: "FEN/UCI collection (*.txt)", Pattern: "*.txt"},
		},
	})
}

func (c *NormalController) InspectPuzzleImport(path string) (puzzles.ImportInspection, error) {
	return runNormalOperation(c, func() (puzzles.ImportInspection, error) {
		return c.services.Importer.Inspect(c.actions.ctx, path)
	})
}

func (c *NormalController) StartPuzzleImport(path string) (string, error) {
	return runNormalOperation(c, func() (string, error) {
		inspection, err := c.services.Importer.Inspect(c.actions.ctx, path)
		if err != nil {
			return "", err
		}
		return c.services.ImportJobs.Start(c.actions.ctx, importjob.ImportRequest{
			Kind:     inspection.Format,
			SourceID: inspection.SourceID,
			Path:     inspection.Path,
		})
	})
}

func (c *NormalController) CancelImport(jobID string) error {
	return runNormalAction(c, func() error {
		return c.services.ImportJobs.Cancel(jobID)
	})
}

func (c *NormalController) GetImportResult(jobID string) (importjob.Result, error) {
	return runNormalOperation(c, func() (importjob.Result, error) {
		return c.services.ImportJobs.Result(jobID)
	})
}

func (c *NormalController) StartGuided() (domain.SessionView, error) {
	return runNormalOperation(c, func() (domain.SessionView, error) {
		return c.services.Training.StartGuided(c.actions.ctx)
	})
}

func (c *NormalController) StartFreePractice(request training.PracticeRequest) (domain.SessionView, error) {
	return runNormalOperation(c, func() (domain.SessionView, error) {
		return c.services.Training.StartFreePractice(c.actions.ctx, request)
	})
}

func (c *NormalController) ResumeSession() (*domain.SessionView, error) {
	return runNormalOperation(c, func() (*domain.SessionView, error) {
		return c.services.Training.Resume(c.actions.ctx)
	})
}

func (c *NormalController) PlayMove(sessionID, uci string) (domain.MoveResult, error) {
	return runNormalOperation(c, func() (domain.MoveResult, error) {
		return c.services.Training.PlayMove(c.actions.ctx, sessionID, uci)
	})
}

func (c *NormalController) UseHint(sessionID string) (domain.HintResult, error) {
	return runNormalOperation(c, func() (domain.HintResult, error) {
		return c.services.Training.UseHint(c.actions.ctx, sessionID)
	})
}

func (c *NormalController) RevealSolution(sessionID string) (domain.MoveResult, error) {
	return runNormalOperation(c, func() (domain.MoveResult, error) {
		return c.services.Training.Reveal(c.actions.ctx, sessionID)
	})
}

func (c *NormalController) PauseSession(sessionID string) error {
	return runNormalAction(c, func() error {
		return c.services.Training.Pause(c.actions.ctx, sessionID)
	})
}

func (c *NormalController) GetProfile() (*training.Profile, error) {
	return runNormalOperation(c, func() (*training.Profile, error) {
		return c.services.Profile.Get(c.actions.ctx)
	})
}

func (c *NormalController) UpdateProfile(value training.Profile) error {
	return runNormalAction(c, func() error {
		return c.services.Profile.UpdateSettings(c.actions.ctx, value)
	})
}

func (c *NormalController) GetParentSummary() (profile.Summary, error) {
	return runNormalOperation(c, func() (profile.Summary, error) {
		return c.services.Profile.Summary(c.actions.ctx)
	})
}

func (c *NormalController) GetPracticeFilters() (profile.PracticeFilters, error) {
	return runNormalOperation(c, func() (profile.PracticeFilters, error) {
		return c.services.Profile.PracticeFilters(c.actions.ctx)
	})
}

func runNormalOperation[T any](
	controller *NormalController,
	operation func() (T, error),
) (T, error) {
	var zero T
	release, err := controller.services.AcquireOperation()
	if err != nil {
		return zero, err
	}
	defer release()
	return operation()
}

func runNormalAction(controller *NormalController, operation func() error) error {
	_, err := runNormalOperation(controller, func() (struct{}, error) {
		return struct{}{}, operation()
	})
	return err
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
