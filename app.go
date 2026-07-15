package main

import (
	"context"
	"net/url"

	appservices "chess-trainer/internal/app"
	"chess-trainer/internal/backup"
	"chess-trainer/internal/domain"
	"chess-trainer/internal/importjob"
	"chess-trainer/internal/profile"
	"chess-trainer/internal/puzzles"
	"chess-trainer/internal/storage"
	"chess-trainer/internal/training"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx      context.Context
	services *appservices.Services
	paths    storage.Paths
	recovery *storage.IntegrityError
	backup   *backup.Service
}

type RecoveryState struct {
	Required bool   `json:"required"`
	Path     string `json:"path,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

func NewApp(services *appservices.Services) *App {
	return &App{
		ctx: context.Background(), services: services, paths: services.Paths, backup: services.Backup,
	}
}

func NewRecoveryApp(paths storage.Paths, integrity *storage.IntegrityError) *App {
	return &App{
		ctx: context.Background(), paths: paths, recovery: integrity,
		backup: backup.NewService(paths, func() error { return nil }),
	}
}

func (a *App) GetRecoveryState() RecoveryState {
	if a.recovery == nil {
		return RecoveryState{}
	}
	return RecoveryState{Required: true, Path: a.recovery.Path, Detail: a.recovery.Detail}
}

func (a *App) CreateBackup(includeLibrary bool) (string, error) {
	destination, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save Chess Trainer backup",
		DefaultFilename: "Chess Trainer Backup.zip",
		Filters: []runtime.FileFilter{{
			DisplayName: "Zip archive (*.zip)", Pattern: "*.zip",
		}},
		CanCreateDirectories: true,
	})
	if err != nil || destination == "" {
		return destination, err
	}
	if err := a.backup.Create(a.ctx, destination, includeLibrary); err != nil {
		return "", err
	}
	return destination, nil
}

func (a *App) RestoreBackup(path string) error {
	var err error
	if path == "" {
		path, err = runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
			Title: "Restore Chess Trainer backup",
			Filters: []runtime.FileFilter{{
				DisplayName: "Zip archive (*.zip)", Pattern: "*.zip",
			}},
		})
		if err != nil || path == "" {
			return err
		}
	}
	if err := a.backup.Restore(a.ctx, path); err != nil {
		return err
	}
	runtime.Quit(a.ctx)
	return nil
}

func (a *App) OpenDataFolder() {
	runtime.BrowserOpenURL(a.ctx, (&url.URL{Scheme: "file", Path: a.paths.Root}).String())
}

func (a *App) Quit() {
	runtime.Quit(a.ctx)
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if a.services != nil {
		a.services.ImportJobs.SetEmitter(wailsEmitter{ctx: ctx})
	}
}

func (a *App) StartLichessImport(path string) (string, error) {
	return a.services.ImportJobs.Start(path)
}

func (a *App) CancelImport(jobID string) error {
	return a.services.ImportJobs.Cancel(jobID)
}

func (a *App) GetImportResult(jobID string) (importjob.Result, error) {
	return a.services.ImportJobs.Result(jobID)
}

func (a *App) StartGuided() (domain.SessionView, error) {
	return a.services.Training.StartGuided(a.ctx)
}

func (a *App) StartFreePractice(request training.PracticeRequest) (domain.SessionView, error) {
	return a.services.Training.StartFreePractice(a.ctx, request)
}

func (a *App) ResumeSession() (*domain.SessionView, error) {
	return a.services.Training.Resume(a.ctx)
}

func (a *App) PlayMove(sessionID, uci string) (domain.MoveResult, error) {
	return a.services.Training.PlayMove(a.ctx, sessionID, uci)
}

func (a *App) UseHint(sessionID string) (domain.HintResult, error) {
	return a.services.Training.UseHint(a.ctx, sessionID)
}

func (a *App) RevealSolution(sessionID string) (domain.MoveResult, error) {
	return a.services.Training.Reveal(a.ctx, sessionID)
}

func (a *App) PauseSession(sessionID string) error {
	return a.services.Training.Pause(a.ctx, sessionID)
}

func (a *App) GetProfile() (*training.Profile, error) {
	return a.services.Profile.Get(a.ctx)
}

func (a *App) UpdateProfile(value training.Profile) error {
	return a.services.Profile.UpdateSettings(a.ctx, value)
}

func (a *App) GetParentSummary() (profile.Summary, error) {
	return a.services.Profile.Summary(a.ctx)
}

func (a *App) GetPracticeFilters() (profile.PracticeFilters, error) {
	return a.services.Profile.PracticeFilters(a.ctx)
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
