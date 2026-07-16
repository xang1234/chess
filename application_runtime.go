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
	services, err := appservices.Open(paths)
	if err == nil {
		return NewApp(services), services, nil
	}
	var integrityErr *storage.IntegrityError
	if !errors.As(err, &integrityErr) {
		return nil, nil, fmt.Errorf("open application services: %w", err)
	}
	return NewRecoveryApp(paths, integrityErr), nil, nil
}
