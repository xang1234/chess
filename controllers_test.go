//go:build !bindings

package main

import (
	"errors"
	"os"
	"reflect"
	"slices"
	"testing"

	appservices "chess-trainer/internal/app"
	"chess-trainer/internal/storage"
)

func TestRecoveryRuntimeBindsOnlyModeAndRecoveryCapabilities(t *testing.T) {
	paths := storage.PathsAt(t.TempDir())
	if err := paths.Ensure(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.UserDB, []byte("corrupt user database"), 0o600); err != nil {
		t.Fatal(err)
	}
	application, err := newApplicationAt(paths)
	if err != nil {
		t.Fatal(err)
	}
	defer application.Close()

	bindings := application.Bindings()
	if len(bindings) != 2 ||
		reflect.TypeOf(bindings[0]) != reflect.TypeOf((*ModeController)(nil)) ||
		reflect.TypeOf(bindings[1]) != reflect.TypeOf((*RecoveryController)(nil)) {
		t.Fatalf("recovery bindings = %#v, want ModeController + RecoveryController", bindingTypes(bindings))
	}
	wantMethods := []string{
		"CreateBackup",
		"GetRecoveryState",
		"OpenDataFolder",
		"Quit",
		"RestoreBackup",
	}
	if got := exportedMethodNames(reflect.TypeOf((*RecoveryController)(nil))); !slices.Equal(got, wantMethods) {
		t.Fatalf("RecoveryController methods = %v, want %v", got, wantMethods)
	}
}

func TestNormalRuntimeBindsOnlyModeAndNormalCapabilities(t *testing.T) {
	application, err := newApplicationAt(storage.PathsAt(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer application.Close()

	bindings := application.Bindings()
	if len(bindings) != 2 ||
		reflect.TypeOf(bindings[0]) != reflect.TypeOf((*ModeController)(nil)) ||
		reflect.TypeOf(bindings[1]) != reflect.TypeOf((*NormalController)(nil)) {
		t.Fatalf("normal bindings = %#v, want ModeController + NormalController", bindingTypes(bindings))
	}
	if slices.Contains(exportedMethodNames(reflect.TypeOf((*RecoveryController)(nil))), "StartGuided") {
		t.Fatal("recovery controller exposes normal training methods")
	}
}

func TestNormalControllerRejectsOperationsAfterServicesClose(t *testing.T) {
	services, err := appservices.Open(storage.PathsAt(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	controller := NewNormalController(services)
	if err := services.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := controller.GetProfile(); !errors.Is(err, appservices.ErrRuntimeUnavailable) {
		t.Fatalf("GetProfile() after close = %v, want runtime unavailable", err)
	}
	if _, err := controller.StartGuided(); !errors.Is(err, appservices.ErrRuntimeUnavailable) {
		t.Fatalf("StartGuided() after close = %v, want runtime unavailable", err)
	}
	if err := controller.RestoreBackup("/missing.zip"); !errors.Is(err, appservices.ErrRuntimeUnavailable) {
		t.Fatalf("RestoreBackup() after close = %v, want runtime unavailable", err)
	}
}

func bindingTypes(bindings []interface{}) []string {
	result := make([]string, len(bindings))
	for index, binding := range bindings {
		result[index] = reflect.TypeOf(binding).String()
	}
	return result
}

func exportedMethodNames(controller reflect.Type) []string {
	methods := make([]string, 0, controller.NumMethod())
	for index := 0; index < controller.NumMethod(); index++ {
		methods = append(methods, controller.Method(index).Name)
	}
	return methods
}
