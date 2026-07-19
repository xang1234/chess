package main

import (
	"context"
	"fmt"

	appservices "chess-trainer/internal/app"
	"chess-trainer/internal/domain"
	"chess-trainer/internal/importing"
	"chess-trainer/internal/importjob"
	"chess-trainer/internal/openings"
	"chess-trainer/internal/profile"
	"chess-trainer/internal/training"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type NormalController struct {
	actions  *controllerActions
	services *appservices.Services
}

func NewNormalController(services *appservices.Services) *NormalController {
	return &NormalController{
		actions: newControllerActions(services.Paths), services: services,
	}
}

func (c *NormalController) startup(ctx context.Context) {
	c.actions.startup(ctx)
	c.services.ImportJobs.SetEmitter(wailsEmitter{ctx: ctx})
}

func (c *NormalController) CreateBackup(includeLibrary bool) (string, error) {
	destination, err := c.actions.chooseBackupDestination()
	if err != nil || destination == "" {
		return destination, err
	}
	if err := c.services.CreateBackup(c.actions.ctx, destination, includeLibrary); err != nil {
		return "", err
	}
	return destination, nil
}

func (c *NormalController) RestoreBackup(path string) error {
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

func (c *NormalController) OpenDataFolder() {
	c.actions.openDataFolder()
}

func (c *NormalController) Quit() {
	c.actions.quit()
}

func (c *NormalController) ChoosePuzzleImportFile() (string, error) {
	descriptors, err := c.services.Importer.FormatDescriptors()
	if err != nil {
		return "", err
	}
	filters := make([]runtime.FileFilter, 0, len(descriptors))
	for _, descriptor := range descriptors {
		filters = append(filters, runtime.FileFilter{
			DisplayName: fmt.Sprintf(
				"%s (*%s)",
				descriptor.FileFilterDescription,
				descriptor.CanonicalExtension,
			),
			Pattern: "*" + descriptor.CanonicalExtension,
		})
	}
	return c.actions.dialogs.OpenFileDialog(c.actions.ctx, runtime.OpenDialogOptions{
		Title: "Choose a puzzle collection", Filters: filters,
	})
}

func (c *NormalController) ChooseOpeningCourseFile() (string, error) {
	return c.actions.dialogs.OpenFileDialog(c.actions.ctx, runtime.OpenDialogOptions{
		Title: "Choose an opening course",
		Filters: []runtime.FileFilter{{
			DisplayName: "Opening course (*.ctcourse)", Pattern: "*.ctcourse",
		}},
	})
}

func (c *NormalController) InspectPuzzleImport(path string) (importing.Inspection, error) {
	return runNormalOperation(c, func() (importing.Inspection, error) {
		return c.services.Importer.Inspect(c.actions.ctx, path)
	})
}

func (c *NormalController) StartPuzzleImport(inspection importing.Inspection) (string, error) {
	return c.startImport(inspection)
}

func (c *NormalController) InspectOpeningCourseImport(path string) (importing.Inspection, error) {
	return runNormalOperation(c, func() (importing.Inspection, error) {
		if c.services.CourseImporter == nil {
			return importing.Inspection{}, fmt.Errorf(
				"opening course imports are unavailable: %s",
				c.services.CourseNotice.Detail,
			)
		}
		return c.services.CourseImporter.Inspect(c.actions.ctx, path)
	})
}

func (c *NormalController) StartOpeningCourseImport(
	inspection importing.Inspection,
) (string, error) {
	return c.startImport(inspection)
}

func (c *NormalController) startImport(inspection importing.Inspection) (string, error) {
	return runNormalOperation(c, func() (string, error) {
		return c.services.ImportJobs.Start(c.actions.ctx, inspection)
	})
}

func (c *NormalController) CancelImport(jobID string) error {
	return runNormalAction(c, func() error {
		return c.services.ImportJobs.Cancel(jobID)
	})
}

func (c *NormalController) GetImportResult(jobID string) (importjob.Result, error) {
	return runNormalOperation(c, func() (importjob.Result, error) {
		return c.services.ImportJobs.Result(jobID)
	})
}

func (c *NormalController) StartGuided() (domain.SessionView, error) {
	return runNormalOperation(c, func() (domain.SessionView, error) {
		return c.services.Training.StartGuided(c.actions.ctx)
	})
}

func (c *NormalController) StartFreePractice(request training.PracticeRequest) (domain.SessionView, error) {
	return runNormalOperation(c, func() (domain.SessionView, error) {
		return c.services.Training.StartFreePractice(c.actions.ctx, request)
	})
}

func (c *NormalController) ResumeSession() (*domain.SessionView, error) {
	return runNormalOperation(c, func() (*domain.SessionView, error) {
		return c.services.Training.Resume(c.actions.ctx)
	})
}

func (c *NormalController) PlayMove(sessionID, uci string) (domain.MoveResult, error) {
	return runNormalOperation(c, func() (domain.MoveResult, error) {
		return c.services.Training.PlayMove(c.actions.ctx, sessionID, uci)
	})
}

func (c *NormalController) UseHint(sessionID string) (domain.HintResult, error) {
	return runNormalOperation(c, func() (domain.HintResult, error) {
		return c.services.Training.UseHint(c.actions.ctx, sessionID)
	})
}

func (c *NormalController) RevealSolution(sessionID string) (domain.MoveResult, error) {
	return runNormalOperation(c, func() (domain.MoveResult, error) {
		return c.services.Training.Reveal(c.actions.ctx, sessionID)
	})
}

func (c *NormalController) PauseSession(sessionID string) error {
	return runNormalAction(c, func() error {
		return c.services.Training.Pause(c.actions.ctx, sessionID)
	})
}

func (c *NormalController) GetOpeningHome() (openings.OpeningHomeView, error) {
	return runNormalOperation(c, func() (openings.OpeningHomeView, error) {
		service, err := c.openingService()
		if err != nil {
			return openings.OpeningHomeView{}, err
		}
		return service.Home(c.actions.ctx)
	})
}

func (c *NormalController) GetOpeningPosition(
	courseID string,
	positionID string,
	depth openings.Depth,
) (openings.ExplorerPositionView, error) {
	return runNormalOperation(c, func() (openings.ExplorerPositionView, error) {
		service, err := c.openingService()
		if err != nil {
			return openings.ExplorerPositionView{}, err
		}
		return service.Explore(c.actions.ctx, courseID, positionID, depth)
	})
}

func (c *NormalController) SetOpeningDepth(courseID string, depth openings.Depth) error {
	return runNormalAction(c, func() error {
		service, err := c.openingService()
		if err != nil {
			return err
		}
		return service.SetDepth(c.actions.ctx, courseID, depth)
	})
}

func (c *NormalController) StartOpeningLesson(
	courseID string,
	lessonID string,
) (openings.OpeningSessionView, error) {
	return runNormalOperation(c, func() (openings.OpeningSessionView, error) {
		service, err := c.openingService()
		if err != nil {
			return openings.OpeningSessionView{}, err
		}
		return service.StartLesson(c.actions.ctx, courseID, lessonID)
	})
}

func (c *NormalController) ResumeOpeningSession() (*openings.OpeningSessionView, error) {
	return runNormalOperation(c, func() (*openings.OpeningSessionView, error) {
		service, err := c.openingService()
		if err != nil {
			return nil, err
		}
		return service.Resume(c.actions.ctx)
	})
}

func (c *NormalController) RestartOpeningSession(
	sessionID string,
) (openings.OpeningSessionView, error) {
	return runNormalOperation(c, func() (openings.OpeningSessionView, error) {
		service, err := c.openingService()
		if err != nil {
			return openings.OpeningSessionView{}, err
		}
		return service.Restart(c.actions.ctx, sessionID)
	})
}

func (c *NormalController) AdvanceOpeningStep(
	sessionID string,
) (openings.OpeningStepResult, error) {
	return runNormalOperation(c, func() (openings.OpeningStepResult, error) {
		service, err := c.openingService()
		if err != nil {
			return openings.OpeningStepResult{}, err
		}
		return service.Advance(c.actions.ctx, sessionID)
	})
}

func (c *NormalController) PlayOpeningMove(
	sessionID string,
	uci string,
) (openings.OpeningStepResult, error) {
	return runNormalOperation(c, func() (openings.OpeningStepResult, error) {
		service, err := c.openingService()
		if err != nil {
			return openings.OpeningStepResult{}, err
		}
		return service.PlayMove(c.actions.ctx, sessionID, uci)
	})
}

func (c *NormalController) UseOpeningHint(
	sessionID string,
) (openings.OpeningHintResult, error) {
	return runNormalOperation(c, func() (openings.OpeningHintResult, error) {
		service, err := c.openingService()
		if err != nil {
			return openings.OpeningHintResult{}, err
		}
		return service.UseHint(c.actions.ctx, sessionID)
	})
}

func (c *NormalController) RevealOpeningMove(
	sessionID string,
) (openings.OpeningStepResult, error) {
	return runNormalOperation(c, func() (openings.OpeningStepResult, error) {
		service, err := c.openingService()
		if err != nil {
			return openings.OpeningStepResult{}, err
		}
		return service.Reveal(c.actions.ctx, sessionID)
	})
}

func (c *NormalController) PauseOpeningSession(sessionID string) error {
	return runNormalAction(c, func() error {
		service, err := c.openingService()
		if err != nil {
			return err
		}
		return service.Pause(c.actions.ctx, sessionID)
	})
}

func (c *NormalController) StartOpeningReview(
	courseID string,
) (openings.OpeningSessionView, error) {
	return runNormalOperation(c, func() (openings.OpeningSessionView, error) {
		service, err := c.openingService()
		if err != nil {
			return openings.OpeningSessionView{}, err
		}
		return service.StartReview(c.actions.ctx, courseID)
	})
}

func (c *NormalController) openingService() (*openings.Service, error) {
	if c.services.Openings == nil {
		return nil, fmt.Errorf("Opening courses are unavailable. Reimport the private course pack.")
	}
	return c.services.Openings, nil
}

func (c *NormalController) GetProfile() (*training.Profile, error) {
	return runNormalOperation(c, func() (*training.Profile, error) {
		return c.services.Profile.Get(c.actions.ctx)
	})
}

func (c *NormalController) UpdateProfile(value training.Profile) error {
	return runNormalAction(c, func() error {
		return c.services.Profile.UpdateSettings(c.actions.ctx, value)
	})
}

func (c *NormalController) GetParentSummary() (profile.Summary, error) {
	return runNormalOperation(c, func() (profile.Summary, error) {
		return c.services.Profile.Summary(c.actions.ctx)
	})
}

func (c *NormalController) GetPracticeFilters() (profile.PracticeFilters, error) {
	return runNormalOperation(c, func() (profile.PracticeFilters, error) {
		return c.services.Profile.PracticeFilters(c.actions.ctx)
	})
}

func runNormalOperation[T any](
	controller *NormalController,
	operation func() (T, error),
) (T, error) {
	var zero T
	release, err := controller.services.AcquireOperation()
	if err != nil {
		return zero, err
	}
	defer release()
	return operation()
}

func runNormalAction(controller *NormalController, operation func() error) error {
	_, err := runNormalOperation(controller, func() (struct{}, error) {
		return struct{}{}, operation()
	})
	return err
}

type wailsEmitter struct {
	ctx context.Context
}

func (e wailsEmitter) Progress(jobID string, progress importing.Progress) {
	runtime.EventsEmit(e.ctx, "import:progress", struct {
		JobID string `json:"jobId"`
		importing.Progress
	}{JobID: jobID, Progress: progress})
}

func (e wailsEmitter) Finished(result importjob.Result) {
	runtime.EventsEmit(e.ctx, "import:finished", result)
}
