package main

import (
	"context"

	appservices "chess-trainer/internal/app"
	"chess-trainer/internal/importjob"
	"chess-trainer/internal/puzzles"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx      context.Context
	services *appservices.Services
}

func NewApp(services *appservices.Services) *App {
	return &App{services: services}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.services.ImportJobs.SetEmitter(wailsEmitter{ctx: ctx})
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
