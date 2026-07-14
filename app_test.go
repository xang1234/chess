package main

import (
	"testing"

	appservices "chess-trainer/internal/app"
	"chess-trainer/internal/storage"
)

func TestAppImportBindingsDelegateValidation(t *testing.T) {
	services, err := appservices.Open(storage.PathsAt(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer services.Close()
	app := NewApp(services)

	if _, err := app.StartLichessImport(""); err == nil {
		t.Fatal("StartLichessImport() unexpectedly accepted an empty path")
	}
	if err := app.CancelImport("missing"); err == nil {
		t.Fatal("CancelImport() unexpectedly accepted an unknown job")
	}
	if _, err := app.GetImportResult("missing"); err == nil {
		t.Fatal("GetImportResult() unexpectedly accepted an unknown job")
	}
}
