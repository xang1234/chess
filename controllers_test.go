//go:build !bindings

package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	appservices "chess-trainer/internal/app"
	"chess-trainer/internal/buildinfo"
	"chess-trainer/internal/chessrules"
	"chess-trainer/internal/openings"
	"chess-trainer/internal/storage"
)

func TestModeControllerProvidesBuildInfoInBothRuntimes(t *testing.T) {
	paths := storage.PathsAt(t.TempDir())
	normal, err := newApplicationAt(paths)
	if err != nil {
		t.Fatal(err)
	}

	want := buildinfo.Current()
	mode := normal.Bindings()[0].(*ModeController)
	if got := mode.GetBuildInfo(); got != want {
		t.Fatalf("normal GetBuildInfo() = %#v, want %#v", got, want)
	}

	if err := normal.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.UserDB, []byte("corrupt user database"), 0o600); err != nil {
		t.Fatal(err)
	}
	recovery, err := newApplicationAt(paths)
	if err != nil {
		t.Fatal(err)
	}
	defer recovery.Close()

	mode = recovery.Bindings()[0].(*ModeController)
	if got := mode.GetBuildInfo(); got != want {
		t.Fatalf("recovery GetBuildInfo() = %#v, want %#v", got, want)
	}
	if slices.Contains(exportedMethodNames(reflect.TypeOf((*RecoveryController)(nil))), "StartGuided") {
		t.Fatal("recovery controller exposes normal training methods")
	}
}

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
	if _, err := controller.GetOpeningHome(); !errors.Is(err, appservices.ErrRuntimeUnavailable) {
		t.Fatalf("GetOpeningHome() after close = %v, want runtime unavailable", err)
	}
	if _, err := controller.GetOpeningPosition("course", "position", openings.DepthReference); !errors.Is(err, appservices.ErrRuntimeUnavailable) {
		t.Fatalf("GetOpeningPosition() after close = %v, want runtime unavailable", err)
	}
	if err := controller.RestoreBackup("/missing.zip"); !errors.Is(err, appservices.ErrRuntimeUnavailable) {
		t.Fatalf("RestoreBackup() after close = %v, want runtime unavailable", err)
	}
}

func TestNormalControllerOpeningBindingsDelegateThroughOpeningService(t *testing.T) {
	ctx := context.Background()
	services, err := appservices.Open(storage.PathsAt(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer services.Close()
	contents, err := os.ReadFile("internal/openings/testdata/mini.ctcourse")
	if err != nil {
		t.Fatal(err)
	}
	pack, err := openings.DecodeCoursePack(bytes.NewReader(contents))
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := openings.Compile(pack, chessrules.Rules{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := services.OpeningCatalog.Replace(
		ctx, compiled, "/private/mini.ctcourse", "sha-mini",
	); err != nil {
		t.Fatal(err)
	}
	controller := NewNormalController(services)
	controller.actions.ctx = ctx

	home, err := controller.GetOpeningHome()
	if err != nil || len(home.Courses) != 1 {
		t.Fatalf("GetOpeningHome() = %+v err=%v", home, err)
	}
	if err := controller.SetOpeningDepth(pack.CourseID, openings.DepthReference); err != nil {
		t.Fatal(err)
	}
	position, err := controller.GetOpeningPosition(
		pack.CourseID, pack.RootPositionID, openings.DepthReference,
	)
	if err != nil || position.PositionID != pack.RootPositionID || len(position.Moves) == 0 {
		t.Fatalf("GetOpeningPosition() = %+v err=%v", position, err)
	}
	session, err := controller.StartOpeningLesson(pack.CourseID, "giuoco-c3")
	if err != nil || session.Current == nil || session.Current.Kind != openings.ActivityConcept {
		t.Fatalf("StartOpeningLesson() = %+v err=%v", session, err)
	}
	if err := services.OpeningStore.SetSessionStatus(
		ctx, session.SessionID, openings.OpeningStatusRestartRequired, time.Now().UTC(),
	); err != nil {
		t.Fatal(err)
	}
	session, err = controller.RestartOpeningSession(session.SessionID)
	if err != nil || session.Status != openings.OpeningStatusActive {
		t.Fatalf("RestartOpeningSession() = %+v err=%v", session, err)
	}
	if _, err := controller.AdvanceOpeningActivity(session.SessionID); err != nil {
		t.Fatal(err)
	}
	advanced, err := controller.AdvanceOpeningStep(session.SessionID)
	if err != nil || advanced.Session.Current == nil || advanced.Session.Current.Kind != openings.ActivityDecision {
		t.Fatalf("AdvanceOpeningActivity()/compatibility alias = %+v err=%v", advanced, err)
	}
	hint, err := controller.UseOpeningHint(session.SessionID)
	if err != nil || hint.Level != 1 {
		t.Fatalf("UseOpeningHint() = %+v err=%v", hint, err)
	}
	if _, err := controller.PlayOpeningMove(session.SessionID, "c2c3"); err != nil {
		t.Fatal(err)
	}
	if _, err := controller.PlayOpeningMove(session.SessionID, "c2c3"); err != nil {
		t.Fatal(err)
	}
	if err := controller.PauseOpeningSession(session.SessionID); err != nil {
		t.Fatal(err)
	}
	resumed, err := controller.ResumeOpeningSession()
	if err != nil || resumed == nil || resumed.Status != openings.OpeningStatusActive {
		t.Fatalf("ResumeOpeningSession() = %+v err=%v", resumed, err)
	}
	for level := 1; level <= 4; level++ {
		if _, err := controller.UseOpeningHint(session.SessionID); err != nil {
			t.Fatal(err)
		}
	}
	revealed, err := controller.RevealOpeningMove(session.SessionID)
	if err != nil || revealed.Session.Status != openings.OpeningStatusCompleted {
		t.Fatalf("RevealOpeningMove() = %+v err=%v", revealed, err)
	}
	if _, err := services.UserDB.Exec(
		`UPDATE opening_review_state SET due_at = 0 WHERE course_id = ?`, pack.CourseID,
	); err != nil {
		t.Fatal(err)
	}
	review, err := controller.StartOpeningReview(pack.CourseID)
	if err != nil || review.Mode != openings.OpeningModeReview {
		t.Fatalf("StartOpeningReview() = %+v err=%v", review, err)
	}
}

func TestNormalControllerOpeningBindingsReturnStableUnavailableError(t *testing.T) {
	services, err := appservices.Open(storage.PathsAt(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	defer services.Close()
	services.Openings = nil
	controller := NewNormalController(services)
	controller.actions.ctx = context.Background()

	_, err = controller.GetOpeningHome()
	if err == nil || err.Error() != "Opening courses are unavailable. Reimport the private course pack." {
		t.Fatalf("GetOpeningHome() error = %v", err)
	}
	if !strings.Contains(err.Error(), "Reimport") {
		t.Fatalf("unavailable error = %v", err)
	}
	_, err = controller.GetOpeningPosition("course", "position", openings.DepthReference)
	if err == nil || err.Error() != "Opening courses are unavailable. Reimport the private course pack." {
		t.Fatalf("GetOpeningPosition() error = %v", err)
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
