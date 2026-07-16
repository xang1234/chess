//go:build bindings

package main

import appservices "chess-trainer/internal/app"

func newApplication() (*App, *appservices.Services, error) {
	return &App{}, nil, nil
}
