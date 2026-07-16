package main

import (
	"context"
	"net/url"

	"chess-trainer/internal/backup"
	"chess-trainer/internal/storage"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type controllerActions struct {
	ctx     context.Context
	paths   storage.Paths
	backup  *backup.Service
	dialogs nativeDialogs
}

type nativeDialogs interface {
	OpenFileDialog(context.Context, runtime.OpenDialogOptions) (string, error)
	SaveFileDialog(context.Context, runtime.SaveDialogOptions) (string, error)
}

type runtimeDialogs struct{}

func (runtimeDialogs) OpenFileDialog(
	ctx context.Context,
	options runtime.OpenDialogOptions,
) (string, error) {
	return runtime.OpenFileDialog(ctx, options)
}

func (runtimeDialogs) SaveFileDialog(
	ctx context.Context,
	options runtime.SaveDialogOptions,
) (string, error) {
	return runtime.SaveFileDialog(ctx, options)
}

func newControllerActions(paths storage.Paths, backupService *backup.Service) *controllerActions {
	return &controllerActions{
		ctx: context.Background(), paths: paths, backup: backupService, dialogs: runtimeDialogs{},
	}
}

func (a *controllerActions) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *controllerActions) createBackup(includeLibrary bool) (string, error) {
	destination, err := a.dialogs.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
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

func (a *controllerActions) restoreBackup(path string) error {
	var err error
	if path == "" {
		path, err = a.dialogs.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
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

func (a *controllerActions) openDataFolder() {
	runtime.BrowserOpenURL(a.ctx, (&url.URL{Scheme: "file", Path: a.paths.Root}).String())
}

func (a *controllerActions) quit() {
	runtime.Quit(a.ctx)
}
