package training

import (
	"context"
	"database/sql"
	"encoding/json"
	"math/rand"
	"path/filepath"
	"slices"
	"sort"
	"testing"
	"time"

	"chess-trainer/internal/chessrules"
	"chess-trainer/internal/domain"
	"chess-trainer/internal/puzzles"
	"chess-trainer/internal/storage"
)

type task5ResolveCall struct {
	fingerprint     string
	preferredSource string
}

type task5CatalogFake struct {
	puzzles       map[puzzles.PuzzleKey]puzzles.TrainingPuzzle
	rated         []puzzles.TrainingPuzzle
	practice      []puzzles.TrainingPuzzle
	getCalls      []puzzles.PuzzleKey
	resolveCalls  []task5ResolveCall
	ratedCalls    int
	practiceCalls int
	ratingBounds  puzzles.RatingBounds
	boundsCalls   int
}

func (f *task5CatalogFake) LearnerRatingBounds(context.Context) (puzzles.RatingBounds, error) {
	f.boundsCalls++
	if f.ratingBounds == (puzzles.RatingBounds{}) {
		return puzzles.DefaultLearnerRatingBounds(), nil
	}
	return f.ratingBounds, nil
}

func (f *task5CatalogFake) Get(_ context.Context, key puzzles.PuzzleKey) (puzzles.TrainingPuzzle, error) {
	f.getCalls = append(f.getCalls, key)
	puzzle, ok := f.puzzles[key]
	if !ok {
		return puzzles.TrainingPuzzle{}, sql.ErrNoRows
	}
	return puzzle, nil
}

func (f *task5CatalogFake) Resolve(
	_ context.Context,
	fingerprint string,
	preferredSource string,
) (puzzles.TrainingPuzzle, error) {
	f.resolveCalls = append(f.resolveCalls, task5ResolveCall{
		fingerprint: fingerprint, preferredSource: preferredSource,
	})
	if preferredSource != "" {
		if puzzle, ok := f.puzzles[puzzles.PuzzleKey{
			Fingerprint: fingerprint,
			SourceID:    preferredSource,
		}]; ok {
			return puzzle, nil
		}
	}
	sources := make([]string, 0)
	for key := range f.puzzles {
		if key.Fingerprint == fingerprint {
			sources = append(sources, key.SourceID)
		}
	}
	if len(sources) == 0 {
		return puzzles.TrainingPuzzle{}, sql.ErrNoRows
	}
	sort.Strings(sources)
	return f.puzzles[puzzles.PuzzleKey{Fingerprint: fingerprint, SourceID: sources[0]}], nil
}

func (f *task5CatalogFake) RatedCandidates(
	context.Context,
	int,
	int,
	[]string,
	int,
) ([]puzzles.TrainingPuzzle, error) {
	f.ratedCalls++
	return slices.Clone(f.rated), nil
}

func (f *task5CatalogFake) FreePracticeCandidates(
	context.Context,
	string,
	*int,
	*int,
	[]string,
	*int,
	int,
) ([]puzzles.TrainingPuzzle, error) {
	f.practiceCalls++
	return slices.Clone(f.practice), nil
}

type task5UserFake struct {
	due []ReviewState
}

func (f task5UserFake) DueReviews(
	_ context.Context,
	_ time.Time,
	after *ReviewCursor,
	limit int,
) ([]ReviewState, error) {
	return task5DueReviewPage(f.due, after, limit), nil
}

func (task5UserFake) RecentFingerprints(context.Context, int) ([]string, error) {
	return nil, nil
}

type task5LimitingUserFake struct {
	due []ReviewState
}

func (f task5LimitingUserFake) DueReviews(
	_ context.Context,
	_ time.Time,
	after *ReviewCursor,
	limit int,
) ([]ReviewState, error) {
	return task5DueReviewPage(f.due, after, limit), nil
}

func (task5LimitingUserFake) RecentFingerprints(context.Context, int) ([]string, error) {
	return nil, nil
}

func task5DueReviewPage(
	due []ReviewState,
	after *ReviewCursor,
	limit int,
) []ReviewState {
	due = slices.Clone(due)
	sort.SliceStable(due, func(i, j int) bool {
		return reviewCursorLess(reviewCursor(due[i]), reviewCursor(due[j]))
	})
	start := 0
	if after != nil {
		start = len(due)
		for index, review := range due {
			if reviewCursorLess(*after, reviewCursor(review)) {
				start = index
				break
			}
		}
	}
	return slices.Clone(due[start:min(start+limit, len(due))])
}

func task5Puzzle(
	fingerprint string,
	sourceID string,
	sourceKind string,
	rating int,
	themes []string,
	sourceFEN string,
	prelude string,
) puzzles.TrainingPuzzle {
	return puzzles.TrainingPuzzle{
		Core: puzzles.PuzzleCore{
			Fingerprint:  fingerprint,
			DisplayedFEN: "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			Solver:       domain.White,
			Solution:     []domain.MoveNode{{UCI: "e2e4"}},
		},
		Occurrence: puzzles.PuzzleOccurrence{
			SourceID:   sourceID,
			SourceKind: sourceKind,
			Rating:     &rating,
			Themes:     slices.Clone(themes),
			SourceFEN:  sourceFEN,
			PreludeUCI: prelude,
		},
	}
}

func task5Scheduled(puzzle puzzles.TrainingPuzzle, kind ScheduledKind, updatesRating bool) ScheduledPuzzle {
	return ScheduledPuzzle{Puzzle: puzzle, Kind: kind, UpdatesRating: updatesRating}
}

func openTask5UserStore(t *testing.T) (*sql.DB, *UserStore) {
	t.Helper()
	db, err := storage.Open(filepath.Join(t.TempDir(), "user.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := storage.Migrate(db, "user"); err != nil {
		t.Fatal(err)
	}
	return db, NewUserStore(db)
}

func TestGuidedReviewResolvesPreferredSource(t *testing.T) {
	wanted := task5Puzzle("review", "preferred", "lichess", 1500, nil, "", "")
	catalog := &task5CatalogFake{puzzles: map[puzzles.PuzzleKey]puzzles.TrainingPuzzle{
		wanted.Key(): wanted,
	}}
	scheduler := Scheduler{Catalog: catalog, User: task5UserFake{due: []ReviewState{{
		Fingerprint:       "review",
		PreferredSourceID: "preferred",
		DueAt:             time.Unix(1, 0),
	}}}}

	items, err := scheduler.BuildGuided(
		context.Background(),
		Profile{LearnerRating: 1500, SessionSize: 1},
		time.Unix(10, 0),
		rand.New(rand.NewSource(1)),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Puzzle.Key() != wanted.Key() {
		t.Fatalf("items=%+v", items)
	}
	wantCall := task5ResolveCall{fingerprint: "review", preferredSource: "preferred"}
	if !slices.Equal(catalog.resolveCalls, []task5ResolveCall{wantCall}) {
		t.Fatalf("resolve calls=%+v", catalog.resolveCalls)
	}
}

func TestGuidedReviewFallsBackLexicographically(t *testing.T) {
	a := task5Puzzle("review", "alpha", "lichess", 1500, nil, "", "")
	z := task5Puzzle("review", "zulu", "other", 1500, nil, "", "")
	catalog := &task5CatalogFake{puzzles: map[puzzles.PuzzleKey]puzzles.TrainingPuzzle{
		a.Key(): a,
		z.Key(): z,
	}}
	scheduler := Scheduler{Catalog: catalog, User: task5UserFake{due: []ReviewState{{
		Fingerprint:       "review",
		PreferredSourceID: "missing",
		DueAt:             time.Unix(1, 0),
	}}}}

	items, err := scheduler.BuildGuided(
		context.Background(),
		Profile{LearnerRating: 1500, SessionSize: 1},
		time.Unix(10, 0),
		rand.New(rand.NewSource(1)),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Puzzle.Occurrence.SourceID != "alpha" {
		t.Fatalf("items=%+v", items)
	}
}

func TestGuidedReviewRemainsDormantWithoutActiveOccurrence(t *testing.T) {
	catalog := &task5CatalogFake{puzzles: map[puzzles.PuzzleKey]puzzles.TrainingPuzzle{}}
	scheduler := Scheduler{Catalog: catalog, User: task5UserFake{due: []ReviewState{{
		Fingerprint: "missing",
		DueAt:       time.Unix(1, 0),
	}}}}

	items, err := scheduler.BuildGuided(
		context.Background(),
		Profile{LearnerRating: 1500, SessionSize: 1},
		time.Unix(10, 0),
		rand.New(rand.NewSource(1)),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("items=%+v", items)
	}
	if len(catalog.resolveCalls) != 1 || catalog.resolveCalls[0].fingerprint != "missing" {
		t.Fatalf("resolve calls=%+v", catalog.resolveCalls)
	}
}

func TestGuidedReviewFindsFifthActiveAfterFourDormant(t *testing.T) {
	active := task5Puzzle("active-fifth", "source", "lichess", 1500, nil, "", "")
	due := make([]ReviewState, 0, 5)
	for index := 1; index <= 4; index++ {
		due = append(due, ReviewState{
			Fingerprint: "dormant-" + string(rune('0'+index)),
			DueAt:       time.Unix(int64(index), 0),
		})
	}
	due = append(due, ReviewState{
		Fingerprint: active.Core.Fingerprint,
		DueAt:       time.Unix(5, 0),
	})
	scheduler := Scheduler{
		Catalog: &task5CatalogFake{puzzles: map[puzzles.PuzzleKey]puzzles.TrainingPuzzle{
			active.Key(): active,
		}},
		User: task5LimitingUserFake{due: due},
	}

	items, err := scheduler.BuildGuided(
		context.Background(),
		Profile{LearnerRating: 1500, SessionSize: 4},
		time.Unix(10, 0),
		rand.New(rand.NewSource(1)),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Kind != ScheduledReview ||
		items[0].Puzzle.Key() != active.Key() {
		t.Fatalf("items after dormant first page = %+v", items)
	}
}

func TestNewAndPracticeCandidatesKeepSelectedOccurrence(t *testing.T) {
	t.Run("guided new", func(t *testing.T) {
		selected := task5Puzzle("new", "source-b", "custom", 1500, []string{"fork"}, "old-fen", "a2a3")
		catalog := &task5CatalogFake{
			puzzles: map[puzzles.PuzzleKey]puzzles.TrainingPuzzle{selected.Key(): selected},
			rated:   []puzzles.TrainingPuzzle{selected},
		}
		scheduler := Scheduler{Catalog: catalog, User: task5UserFake{}}
		items, err := scheduler.BuildGuided(
			context.Background(),
			Profile{LearnerRating: 1500, SessionSize: 1},
			time.Unix(10, 0),
			rand.New(rand.NewSource(1)),
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].Puzzle.Occurrence.SourceID != "source-b" {
			t.Fatalf("items=%+v", items)
		}
	})

	t.Run("free practice", func(t *testing.T) {
		_, store := openTask5UserStore(t)
		if err := store.UpdateProfile(context.Background(), Profile{LearnerRating: 1500, SessionSize: 5}); err != nil {
			t.Fatal(err)
		}
		selected := task5Puzzle("practice", "source-b", "custom", 1500, []string{"fork"}, "old-fen", "a2a3")
		catalog := &task5CatalogFake{
			puzzles:  map[puzzles.PuzzleKey]puzzles.TrainingPuzzle{selected.Key(): selected},
			practice: []puzzles.TrainingPuzzle{selected},
		}
		service := NewService(catalog, store, chessrules.Rules{}, rand.New(rand.NewSource(1)))
		view, err := service.StartFreePractice(context.Background(), PracticeRequest{SourceID: "source-b"})
		if err != nil {
			t.Fatal(err)
		}
		loaded, err := store.LoadSession(context.Background(), view.SessionID)
		if err != nil {
			t.Fatal(err)
		}
		if len(loaded.Items) != 1 || loaded.Items[0].SourceID != "source-b" {
			t.Fatalf("loaded=%+v", loaded)
		}
	})
}

func TestEverySolvePathLooksUpFingerprintAndSource(t *testing.T) {
	actions := map[string]func(context.Context, *Service, storedSession) error{
		"resume": func(ctx context.Context, service *Service, _ storedSession) error {
			_, err := service.Resume(ctx)
			return err
		},
		"view": func(ctx context.Context, service *Service, session storedSession) error {
			_, err := service.view(ctx, session)
			return err
		},
		"move and completion": func(ctx context.Context, service *Service, session storedSession) error {
			_, err := service.PlayMove(ctx, session.ID, "e2e4")
			return err
		},
		"hint": func(ctx context.Context, service *Service, session storedSession) error {
			_, err := service.UseHint(ctx, session.ID)
			return err
		},
		"reveal": func(ctx context.Context, service *Service, session storedSession) error {
			item := session.Items[0]
			item.State.HintLevel = 3
			if err := service.store.SaveItemState(ctx, session.ID, item.Ordinal, item.State, time.Unix(11, 0)); err != nil {
				return err
			}
			_, err := service.Reveal(ctx, session.ID)
			return err
		},
	}

	for name, action := range actions {
		t.Run(name, func(t *testing.T) {
			_, store := openTask5UserStore(t)
			puzzle := task5Puzzle("exact", "source-b", "custom", 1500, []string{"fork"}, "", "")
			session, err := store.CreateSession(
				context.Background(),
				"guided",
				[]ScheduledPuzzle{task5Scheduled(puzzle, ScheduledNew, false)},
				time.Unix(10, 0),
			)
			if err != nil {
				t.Fatal(err)
			}
			catalog := &task5CatalogFake{puzzles: map[puzzles.PuzzleKey]puzzles.TrainingPuzzle{
				puzzle.Key(): puzzle,
			}}
			service := NewService(catalog, store, chessrules.Rules{}, rand.New(rand.NewSource(1)))
			service.now = func() time.Time { return time.Unix(20, 0) }
			if err := action(context.Background(), service, session); err != nil {
				t.Fatal(err)
			}
			if len(catalog.getCalls) == 0 {
				t.Fatal("solve path made no exact Get call")
			}
			for _, key := range catalog.getCalls {
				if key != puzzle.Key() {
					t.Fatalf("Get key=%+v, want %+v", key, puzzle.Key())
				}
			}
		})
	}
}

func TestEverySolvePathSkipsUnavailableExactOccurrence(t *testing.T) {
	actions := map[string]func(context.Context, *Service, storedSession) error{
		"resume": func(ctx context.Context, service *Service, _ storedSession) error {
			_, err := service.Resume(ctx)
			return err
		},
		"view": func(ctx context.Context, service *Service, session storedSession) error {
			_, err := service.view(ctx, session)
			return err
		},
		"move": func(ctx context.Context, service *Service, session storedSession) error {
			_, err := service.PlayMove(ctx, session.ID, "e2e4")
			return err
		},
		"hint": func(ctx context.Context, service *Service, session storedSession) error {
			_, err := service.UseHint(ctx, session.ID)
			return err
		},
		"reveal": func(ctx context.Context, service *Service, session storedSession) error {
			item := session.Items[0]
			item.State.HintLevel = 3
			if err := service.store.SaveItemState(ctx, session.ID, item.Ordinal, item.State, time.Unix(11, 0)); err != nil {
				return err
			}
			_, err := service.Reveal(ctx, session.ID)
			return err
		},
	}

	for name, action := range actions {
		t.Run(name, func(t *testing.T) {
			db, store := openTask5UserStore(t)
			puzzle := task5Puzzle("missing", "removed-source", "custom", 1500, nil, "", "")
			session, err := store.CreateSession(
				context.Background(),
				"guided",
				[]ScheduledPuzzle{task5Scheduled(puzzle, ScheduledNew, false)},
				time.Unix(10, 0),
			)
			if err != nil {
				t.Fatal(err)
			}
			catalog := &task5CatalogFake{puzzles: map[puzzles.PuzzleKey]puzzles.TrainingPuzzle{}}
			service := NewService(catalog, store, chessrules.Rules{}, rand.New(rand.NewSource(1)))
			service.now = func() time.Time { return time.Unix(20, 0) }

			_ = action(context.Background(), service, session)

			loaded, err := store.LoadSession(context.Background(), session.ID)
			if err != nil {
				t.Fatal(err)
			}
			if loaded.Status != "completed" || loaded.CurrentIndex != 1 ||
				!loaded.Items[0].State.Unavailable {
				t.Fatalf("session did not skip unavailable item: %+v", loaded)
			}
			var attempts int
			if err := db.QueryRow(`SELECT COUNT(*) FROM attempts`).Scan(&attempts); err != nil {
				t.Fatal(err)
			}
			if attempts != 0 {
				t.Fatalf("attempts=%d, want unavailable attempt removed", attempts)
			}
		})
	}
}

func TestUseHintReturnsAdvancedSessionWhenExactOccurrenceUnavailable(t *testing.T) {
	_, store := openTask5UserStore(t)
	missing := task5Puzzle("missing-hint", "removed-source", "custom", 1500, nil, "", "")
	next := task5Puzzle("next-hint", "active-source", "custom", 1500, []string{"fork"}, "", "")
	session, err := store.CreateSession(
		context.Background(),
		"guided",
		[]ScheduledPuzzle{
			task5Scheduled(missing, ScheduledNew, false),
			task5Scheduled(next, ScheduledNew, false),
		},
		time.Unix(10, 0),
	)
	if err != nil {
		t.Fatal(err)
	}
	catalog := &task5CatalogFake{puzzles: map[puzzles.PuzzleKey]puzzles.TrainingPuzzle{
		next.Key(): next,
	}}
	service := NewService(catalog, store, chessrules.Rules{}, rand.New(rand.NewSource(1)))
	service.now = func() time.Time { return time.Unix(20, 0) }

	result, err := service.UseHint(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("UseHint() after unavailable occurrence: %v", err)
	}
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		Session domain.SessionView `json:"session"`
		Text    string             `json:"text"`
	}
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatal(err)
	}
	if response.Text != "Puzzle unavailable" || response.Session.CurrentIndex != 1 ||
		response.Session.Current == nil || response.Session.Current.Fingerprint != next.Core.Fingerprint {
		t.Fatalf("observable hint response = %+v", response)
	}

	loaded, err := store.LoadSession(context.Background(), session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.CurrentIndex != 1 || !loaded.Items[0].State.Unavailable {
		t.Fatalf("persisted session = %+v", loaded)
	}
}

type task5VanishingCatalogFake struct {
	*task5CatalogFake
}

func (f *task5VanishingCatalogFake) Get(
	_ context.Context,
	key puzzles.PuzzleKey,
) (puzzles.TrainingPuzzle, error) {
	f.getCalls = append(f.getCalls, key)
	if len(f.getCalls) > 1 {
		return puzzles.TrainingPuzzle{}, sql.ErrNoRows
	}
	puzzle, ok := f.puzzles[key]
	if !ok {
		return puzzles.TrainingPuzzle{}, sql.ErrNoRows
	}
	return puzzle, nil
}

func TestEverySolvePathRevalidatesExactOccurrenceAtCompletion(t *testing.T) {
	db, store := openTask5UserStore(t)
	puzzle := task5Puzzle("vanishing", "source", "custom", 1500, nil, "", "")
	session, err := store.CreateSession(
		context.Background(),
		"guided",
		[]ScheduledPuzzle{task5Scheduled(puzzle, ScheduledNew, false)},
		time.Unix(10, 0),
	)
	if err != nil {
		t.Fatal(err)
	}
	catalog := &task5VanishingCatalogFake{task5CatalogFake: &task5CatalogFake{
		puzzles: map[puzzles.PuzzleKey]puzzles.TrainingPuzzle{puzzle.Key(): puzzle},
	}}
	service := NewService(catalog, store, chessrules.Rules{}, rand.New(rand.NewSource(1)))
	service.now = func() time.Time { return time.Unix(20, 0) }

	result, err := service.PlayMove(context.Background(), session.ID, "e2e4")
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.getCalls) != 2 {
		t.Fatalf("Get calls=%v, want solve and completion validation", catalog.getCalls)
	}
	if result.Session.Status != "completed" {
		t.Fatalf("result=%+v", result)
	}
	if len(result.AppliedMoves) != 0 || result.FinalFEN != "" {
		t.Fatalf("vanished occurrence leaked animation payload: %+v", result)
	}
	loaded, err := store.LoadSession(context.Background(), session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Items[0].State.Unavailable {
		t.Fatalf("vanished occurrence was completed as a learner attempt: %+v", loaded.Items[0])
	}
	var attempts int
	if err := db.QueryRow(`SELECT COUNT(*) FROM attempts`).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if attempts != 0 {
		t.Fatalf("attempts=%d, want vanished attempt removed", attempts)
	}
}

func TestCreateSessionWritesCompleteOccurrenceSnapshot(t *testing.T) {
	db, store := openTask5UserStore(t)
	puzzle := task5Puzzle(
		"snapshot", "source-b", "custom", 1625,
		[]string{"fork", "pin"}, "source-fen", "a2a3 b7b6",
	)
	session, err := store.CreateSession(
		context.Background(), "guided",
		[]ScheduledPuzzle{task5Scheduled(puzzle, ScheduledNew, true)},
		time.Unix(10, 0),
	)
	if err != nil {
		t.Fatal(err)
	}

	var sourceKind, themesJSON, sourceFEN, prelude string
	var rating int
	err = db.QueryRow(`
		SELECT source_kind, rating_snapshot, themes_json,
		       source_fen_snapshot, prelude_uci_snapshot
		FROM session_items WHERE session_id = ? AND ordinal = 0`, session.ID,
	).Scan(&sourceKind, &rating, &themesJSON, &sourceFEN, &prelude)
	if err != nil {
		t.Fatal(err)
	}
	var themes []string
	if err := json.Unmarshal([]byte(themesJSON), &themes); err != nil {
		t.Fatal(err)
	}
	if sourceKind != "custom" || rating != 1625 ||
		!slices.Equal(themes, []string{"fork", "pin"}) ||
		sourceFEN != "source-fen" || prelude != "a2a3 b7b6" {
		t.Fatalf("snapshot kind=%q rating=%d themes=%v fen=%q prelude=%q",
			sourceKind, rating, themes, sourceFEN, prelude)
	}
}

func TestQueuedAttemptCopiesStoredSnapshot(t *testing.T) {
	db, store := openTask5UserStore(t)
	first := task5Puzzle("first", "one", "custom", 1400, []string{"fork"}, "", "")
	queued := task5Puzzle("queued", "two", "lichess", 1777, []string{"pin", "skewer"}, "", "")
	session, err := store.CreateSession(
		context.Background(), "guided",
		[]ScheduledPuzzle{
			task5Scheduled(first, ScheduledNew, false),
			task5Scheduled(queued, ScheduledNew, false),
		},
		time.Unix(10, 0),
	)
	if err != nil {
		t.Fatal(err)
	}
	state := session.Items[0].State
	state.Completed = true
	if _, err := store.CompleteItem(
		context.Background(), session, state, time.Unix(20, 0), completionEffects{},
	); err != nil {
		t.Fatal(err)
	}

	var sourceKind, themesJSON string
	var rating int
	if err := db.QueryRow(`
		SELECT source_kind, rating_snapshot, themes_json
		FROM attempts WHERE fingerprint = 'queued'`,
	).Scan(&sourceKind, &rating, &themesJSON); err != nil {
		t.Fatal(err)
	}
	var themes []string
	if err := json.Unmarshal([]byte(themesJSON), &themes); err != nil {
		t.Fatal(err)
	}
	if sourceKind != "lichess" || rating != 1777 || !slices.Equal(themes, []string{"pin", "skewer"}) {
		t.Fatalf("attempt kind=%q rating=%d themes=%v", sourceKind, rating, themes)
	}
}

func TestQueuedSessionPresentationSurvivesSameSourceReimport(t *testing.T) {
	_, store := openTask5UserStore(t)
	old := task5Puzzle("same-core", "source", "lichess", 1500, []string{"fork"}, "old-source-fen", "a2a3")
	session, err := store.CreateSession(
		context.Background(), "guided",
		[]ScheduledPuzzle{task5Scheduled(old, ScheduledNew, false)},
		time.Unix(10, 0),
	)
	if err != nil {
		t.Fatal(err)
	}
	newOccurrence := task5Puzzle("same-core", "source", "lichess", 2400, []string{"skewer"}, "new-source-fen", "h2h3")
	catalog := &task5CatalogFake{puzzles: map[puzzles.PuzzleKey]puzzles.TrainingPuzzle{
		newOccurrence.Key(): newOccurrence,
	}}
	service := NewService(catalog, store, chessrules.Rules{}, rand.New(rand.NewSource(1)))
	resumed, err := service.Resume(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if resumed == nil || resumed.Current == nil {
		t.Fatalf("resumed=%+v", resumed)
	}
	if resumed.SessionID != session.ID || resumed.Current.SourceFEN != "old-source-fen" ||
		resumed.Current.PreludeUCI != "a2a3" {
		t.Fatalf("resumed=%+v", resumed)
	}
}

func TestHintAndRatingUseStoredSnapshotAfterReimport(t *testing.T) {
	_, store := openTask5UserStore(t)
	if err := store.UpdateProfile(context.Background(), Profile{LearnerRating: 1500, SessionSize: 5}); err != nil {
		t.Fatal(err)
	}
	old := task5Puzzle("same-core", "source", "lichess", 1000, []string{"fork"}, "", "")
	session, err := store.CreateSession(
		context.Background(), "guided",
		[]ScheduledPuzzle{task5Scheduled(old, ScheduledNew, true)},
		time.Unix(10, 0),
	)
	if err != nil {
		t.Fatal(err)
	}
	newOccurrence := task5Puzzle("same-core", "source", "lichess", 2500, []string{"skewer"}, "", "")
	catalog := &task5CatalogFake{puzzles: map[puzzles.PuzzleKey]puzzles.TrainingPuzzle{
		newOccurrence.Key(): newOccurrence,
	}}
	service := NewService(catalog, store, chessrules.Rules{}, rand.New(rand.NewSource(1)))
	service.now = func() time.Time { return time.Unix(20, 0) }
	hint, err := service.UseHint(context.Background(), session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if hint.Text != "Look for: fork" {
		t.Fatalf("hint=%+v", hint)
	}
	if _, err := service.PlayMove(context.Background(), session.ID, "e2e4"); err != nil {
		t.Fatal(err)
	}
	profile, err := store.Profile(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := UpdateRating(1500, 1000, 0.5, 400, 3000)
	if profile.LearnerRating != want {
		t.Fatalf("rating=%v, want %v from stored snapshot", profile.LearnerRating, want)
	}
}

func TestCompletionClampsRatingToCurrentCatalogBounds(t *testing.T) {
	_, store := openTask5UserStore(t)
	if err := store.UpdateProfile(context.Background(), Profile{LearnerRating: 1500, SessionSize: 5}); err != nil {
		t.Fatal(err)
	}
	puzzle := task5Puzzle("bounded-core", "lichess", "lichess", 2500, nil, "", "")
	session, err := store.CreateSession(
		context.Background(),
		"guided",
		[]ScheduledPuzzle{task5Scheduled(puzzle, ScheduledNew, true)},
		time.Unix(10, 0),
	)
	if err != nil {
		t.Fatal(err)
	}
	catalog := &task5CatalogFake{
		puzzles:      map[puzzles.PuzzleKey]puzzles.TrainingPuzzle{puzzle.Key(): puzzle},
		ratingBounds: puzzles.RatingBounds{Minimum: 800, Maximum: 1505},
	}
	service := NewService(catalog, store, chessrules.Rules{}, rand.New(rand.NewSource(1)))
	if _, err := service.PlayMove(context.Background(), session.ID, "e2e4"); err != nil {
		t.Fatal(err)
	}
	profile, err := store.Profile(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if profile.LearnerRating != 1505 {
		t.Fatalf("learner rating = %v, want active catalogue maximum 1505", profile.LearnerRating)
	}
	if catalog.boundsCalls != 1 {
		t.Fatalf("rating-bound calls = %d, want 1 at completion", catalog.boundsCalls)
	}
}

func TestNewAttemptsWriteSourceKindRatingAndThemes(t *testing.T) {
	db, store := openTask5UserStore(t)
	puzzle := task5Puzzle("attempt", "source", "custom", 1888, []string{"fork", "pin"}, "", "")
	if _, err := store.CreateSession(
		context.Background(), "guided",
		[]ScheduledPuzzle{task5Scheduled(puzzle, ScheduledNew, false)},
		time.Unix(10, 0),
	); err != nil {
		t.Fatal(err)
	}
	var kind, themesJSON string
	var rating int
	if err := db.QueryRow(`
		SELECT source_kind, rating_snapshot, themes_json
		FROM attempts WHERE fingerprint = 'attempt'`,
	).Scan(&kind, &rating, &themesJSON); err != nil {
		t.Fatal(err)
	}
	var themes []string
	if err := json.Unmarshal([]byte(themesJSON), &themes); err != nil {
		t.Fatal(err)
	}
	if kind != "custom" || rating != 1888 || !slices.Equal(themes, []string{"fork", "pin"}) {
		t.Fatalf("attempt kind=%q rating=%d themes=%v", kind, rating, themes)
	}
}

func TestReviewCompletionStoresActualPreferredSource(t *testing.T) {
	db, store := openTask5UserStore(t)
	puzzle := task5Puzzle("review", "actual-source", "custom", 1500, nil, "", "")
	session, err := store.CreateSession(
		context.Background(), "guided",
		[]ScheduledPuzzle{task5Scheduled(puzzle, ScheduledReview, false)},
		time.Unix(10, 0),
	)
	if err != nil {
		t.Fatal(err)
	}
	catalog := &task5CatalogFake{puzzles: map[puzzles.PuzzleKey]puzzles.TrainingPuzzle{
		puzzle.Key(): puzzle,
	}}
	service := NewService(catalog, store, chessrules.Rules{}, rand.New(rand.NewSource(1)))
	service.now = func() time.Time { return time.Unix(20, 0) }
	if _, err := service.PlayMove(context.Background(), session.ID, "e2e4"); err != nil {
		t.Fatal(err)
	}
	var preferred string
	if err := db.QueryRow(`
		SELECT preferred_source_id FROM review_state WHERE fingerprint = 'review'`,
	).Scan(&preferred); err != nil {
		t.Fatal(err)
	}
	if preferred != "actual-source" {
		t.Fatalf("preferred source=%q", preferred)
	}
	loaded, ok, err := store.Review(context.Background(), "review")
	if err != nil || !ok || loaded.PreferredSourceID != "actual-source" {
		t.Fatalf("loaded=%+v ok=%v err=%v", loaded, ok, err)
	}
	next := NextReview(time.Unix(30, 0), loaded, OutcomeClean)
	if next.PreferredSourceID != "actual-source" {
		t.Fatalf("NextReview lost preferred source: %+v", next)
	}
}
