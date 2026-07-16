//go:build !bindings

package main

import (
	"errors"
	"fmt"

	appservices "chess-trainer/internal/app"
	"chess-trainer/internal/storage"
)

func newApplication() (*App, *appservices.Services, error) {
	paths, err := storage.DefaultPaths()
	if err != nil {
		return nil, nil, fmt.Errorf("resolve application data paths: %w", err)
	}
	return newApplicationAt(paths)
}

func newApplicationAt(paths storage.Paths) (*App, *appservices.Services, error) {
	services, err := appservices.OpenApplication(paths)
	if err == nil {
		return NewApp(services), services, nil
	}
	var recoveryErr *appservices.RecoveryRequiredError
	if !errors.As(err, &recoveryErr) {
		return nil, nil, fmt.Errorf("open application services: %w", err)
	}
	return NewRecoveryApp(services, recoveryErr), services, nil
}
