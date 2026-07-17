package main

import (
	"context"

	appservices "chess-trainer/internal/app"
	"chess-trainer/internal/buildinfo"
)

type ApplicationMode string

const (
	ApplicationModeNormal   ApplicationMode = "normal"
	ApplicationModeRecovery ApplicationMode = "recovery"
)

type ModeController struct {
	mode ApplicationMode
}

func (c *ModeController) GetApplicationMode() string {
	return string(c.mode)
}

func (*ModeController) GetBuildInfo() buildinfo.Info {
	return buildinfo.Current()
}

type startupController interface {
	startup(context.Context)
}

type ApplicationRuntime struct {
	lifecycle  *appservices.Services
	controller startupController
	bindings   []interface{}
}

func newNormalRuntime(services *appservices.Services) *ApplicationRuntime {
	mode := &ModeController{mode: ApplicationModeNormal}
	controller := NewNormalController(services)
	return &ApplicationRuntime{
		lifecycle: services, controller: controller,
		bindings: []interface{}{mode, controller},
	}
}

func newRecoveryRuntime(
	services *appservices.Services,
	recovery *appservices.RecoveryRequiredError,
) *ApplicationRuntime {
	mode := &ModeController{mode: ApplicationModeRecovery}
	controller := NewRecoveryController(services, recovery)
	return &ApplicationRuntime{
		lifecycle: services, controller: controller,
		bindings: []interface{}{mode, controller},
	}
}

func (a *ApplicationRuntime) Bindings() []interface{} {
	return append([]interface{}(nil), a.bindings...)
}

func (a *ApplicationRuntime) startup(ctx context.Context) {
	if a.controller != nil {
		a.controller.startup(ctx)
	}
}

func (a *ApplicationRuntime) Close() error {
	if a.lifecycle == nil {
		return nil
	}
	return a.lifecycle.Close()
}
