//go:build !bindings

package main

import (
	"errors"
	"fmt"

	appservices "chess-trainer/internal/app"
	"chess-trainer/internal/storage"
)

func newApplication() (*ApplicationRuntime, error) {
	paths, err := storage.DefaultPaths()
	if err != nil {
		return nil, fmt.Errorf("resolve application data paths: %w", err)
	}
	return newApplicationAt(paths)
}

func newApplicationAt(paths storage.Paths) (*ApplicationRuntime, error) {
	services, err := appservices.OpenApplication(paths)
	if err == nil {
		return newNormalRuntime(services), nil
	}
	var recoveryErr *appservices.RecoveryRequiredError
	if !errors.As(err, &recoveryErr) {
		return nil, fmt.Errorf("open application services: %w", err)
	}
	return newRecoveryRuntime(services, recoveryErr), nil
}
