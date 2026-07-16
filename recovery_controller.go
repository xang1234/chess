package main

import (
	"context"

	appservices "chess-trainer/internal/app"
)

type RecoveryState struct {
	Required bool   `json:"required"`
	Path     string `json:"path,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

type RecoveryController struct {
	actions  *controllerActions
	recovery *appservices.RecoveryRequiredError
}

func NewRecoveryController(
	services *appservices.Services,
	recovery *appservices.RecoveryRequiredError,
) *RecoveryController {
	return &RecoveryController{
		actions: newControllerActions(services.Paths, services.Backup), recovery: recovery,
	}
}

func (c *RecoveryController) startup(ctx context.Context) {
	c.actions.startup(ctx)
}

func (c *RecoveryController) GetRecoveryState() RecoveryState {
	return RecoveryState{
		Required: true,
		Path:     c.recovery.Path,
		Detail:   c.recovery.Detail,
	}
}

func (c *RecoveryController) CreateBackup(includeLibrary bool) (string, error) {
	return c.actions.createBackup(includeLibrary)
}

func (c *RecoveryController) RestoreBackup(path string) error {
	return c.actions.restoreBackup(path)
}

func (c *RecoveryController) OpenDataFolder() {
	c.actions.openDataFolder()
}

func (c *RecoveryController) Quit() {
	c.actions.quit()
}
