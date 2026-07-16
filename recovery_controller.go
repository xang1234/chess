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
	services *appservices.Services
}

func NewRecoveryController(
	services *appservices.Services,
	recovery *appservices.RecoveryRequiredError,
) *RecoveryController {
	return &RecoveryController{
		actions:  newControllerActions(services.Paths),
		recovery: recovery,
		services: services,
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
	destination, err := c.actions.chooseBackupDestination()
	if err != nil || destination == "" {
		return destination, err
	}
	if err := c.services.CreateBackup(c.actions.ctx, destination, includeLibrary); err != nil {
		return "", err
	}
	return destination, nil
}

func (c *RecoveryController) RestoreBackup(path string) error {
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

func (c *RecoveryController) OpenDataFolder() {
	c.actions.openDataFolder()
}

func (c *RecoveryController) Quit() {
	c.actions.quit()
}
