package training

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"time"

	"chess-trainer/internal/chessrules"
	"chess-trainer/internal/domain"
)

type Service struct {
	catalog   CatalogPort
	store     *UserStore
	rules     chessrules.Rules
	random    *rand.Rand
	now       func() time.Time
	scheduler Scheduler
}

func NewService(
	catalog CatalogPort,
	store *UserStore,
	rules chessrules.Rules,
	random *rand.Rand,
) *Service {
	return &Service{
		catalog:   catalog,
		store:     store,
		rules:     rules,
		random:    random,
		now:       time.Now,
		scheduler: Scheduler{Catalog: catalog, User: store},
	}
}

func (s *Service) StartGuided(ctx context.Context) (domain.SessionView, error) {
	profile, err := s.store.Profile(ctx)
	if err != nil {
		return domain.SessionView{}, err
	}
	items, err := s.scheduler.BuildGuided(ctx, profile, s.now(), s.random)
	if err != nil {
		return domain.SessionView{}, err
	}
	session, err := s.store.CreateSession(ctx, "guided", items, s.now())
	if err != nil {
		return domain.SessionView{}, err
	}
	return s.view(ctx, session)
}

func (s *Service) Resume(ctx context.Context) (*domain.SessionView, error) {
	session, err := s.store.ResumableSession(ctx)
	if err != nil || session == nil {
		return nil, err
	}
	if session.Status == "paused" {
		if err := s.store.SetSessionStatus(ctx, session.ID, "active", s.now()); err != nil {
			return nil, err
		}
		session.Status = "active"
	}
	prepared, err := s.prepareAvailable(ctx, *session)
	if err != nil {
		return nil, err
	}
	view, err := s.view(ctx, prepared)
	if err != nil {
		return nil, err
	}
	return &view, nil
}

func (s *Service) PlayMove(ctx context.Context, sessionID, uci string) (domain.MoveResult, error) {
	session, err := s.store.LoadSession(ctx, sessionID)
	if err != nil {
		return domain.MoveResult{}, err
	}
	if session.CurrentIndex >= len(session.Items) {
		return domain.MoveResult{}, errors.New("session is complete")
	}
	item := &session.Items[session.CurrentIndex]
	puzzle, err := s.catalog.Get(ctx, item.Fingerprint)
	if err != nil {
		return domain.MoveResult{}, err
	}
	nodes, err := nodesAtPath(puzzle.Solution, item.State.Path)
	if err != nil {
		return domain.MoveResult{}, err
	}
	selected := slices.IndexFunc(nodes, func(node domain.MoveNode) bool {
		return node.UCI == uci
	})
	if selected < 0 {
		if mateInOneNodes(nodes) && s.rules.IsCheckmateMove(item.State.CurrentFEN, uci) {
			item.State.CurrentFEN, err = s.rules.ApplyUCI(item.State.CurrentFEN, uci)
			if err != nil {
				return domain.MoveResult{}, err
			}
			item.State.Completed = true
			return s.completeCurrent(ctx, session, *item, puzzle)
		}
		if _, err := s.rules.ApplyUCI(item.State.CurrentFEN, uci); err != nil {
			return domain.MoveResult{}, fmt.Errorf("illegal move %q: %w", uci, err)
		}
		item.State.IncorrectMoves++
		if err := s.store.SaveItemState(ctx, session.ID, item.Ordinal, item.State, s.now()); err != nil {
			return domain.MoveResult{}, err
		}
		view, err := s.view(ctx, session)
		return domain.MoveResult{Session: view, Correct: false, Message: "Try again"}, err
	}

	item.State.CurrentFEN, err = s.rules.ApplyUCI(item.State.CurrentFEN, uci)
	if err != nil {
		return domain.MoveResult{}, err
	}
	item.State.Path = append(item.State.Path, selected)
	chosen := nodes[selected]
	if len(chosen.Children) == 0 {
		item.State.Completed = true
	} else if len(chosen.Children) == 1 {
		reply := chosen.Children[0]
		item.State.CurrentFEN, err = s.rules.ApplyUCI(item.State.CurrentFEN, reply.UCI)
		if err != nil {
			return domain.MoveResult{}, err
		}
		item.State.Path = append(item.State.Path, 0)
		if len(reply.Children) == 0 {
			item.State.Completed = true
		}
	} else {
		return domain.MoveResult{}, errors.New("solution has multiple automatic replies")
	}
	if item.State.Completed {
		return s.completeCurrent(ctx, session, *item, puzzle)
	}
	if err := s.store.SaveItemState(ctx, session.ID, item.Ordinal, item.State, s.now()); err != nil {
		return domain.MoveResult{}, err
	}
	view, err := s.view(ctx, session)
	return domain.MoveResult{
		Session:         view,
		Correct:         true,
		PuzzleCompleted: item.State.Completed,
	}, err
}

func (s *Service) UseHint(ctx context.Context, sessionID string) (domain.HintResult, error) {
	session, err := s.store.LoadSession(ctx, sessionID)
	if err != nil {
		return domain.HintResult{}, err
	}
	if session.CurrentIndex >= len(session.Items) {
		return domain.HintResult{}, errors.New("session is complete")
	}
	item := &session.Items[session.CurrentIndex]
	puzzle, err := s.catalog.Get(ctx, item.Fingerprint)
	if err != nil {
		return domain.HintResult{}, err
	}
	nodes, err := nodesAtPath(puzzle.Solution, item.State.Path)
	if err != nil || len(nodes) == 0 {
		return domain.HintResult{}, errors.New("puzzle has no remaining hint move")
	}
	if item.State.HintLevel < 3 {
		item.State.HintLevel++
		item.State.HintsUsed++
		if err := s.store.SaveItemState(ctx, session.ID, item.Ordinal, item.State, s.now()); err != nil {
			return domain.HintResult{}, err
		}
	}
	move := nodes[0].UCI
	if len(move) < 4 {
		return domain.HintResult{}, errors.New("hint move is malformed")
	}
	hint := domain.HintResult{Level: item.State.HintLevel, CanReveal: item.State.HintLevel >= 3}
	switch item.State.HintLevel {
	case 1:
		if len(puzzle.Themes) > 0 {
			hint.Text = "Look for: " + puzzle.Themes[0]
		} else {
			hint.Text = "Look for a forcing move."
		}
	case 2:
		hint.Text = "Start with this piece."
		hint.SourceSquare = move[:2]
	case 3:
		hint.Text = "Try this destination."
		hint.SourceSquare = move[:2]
		hint.TargetSquare = move[2:4]
	}
	return hint, nil
}

func (s *Service) Reveal(ctx context.Context, sessionID string) (domain.MoveResult, error) {
	session, err := s.store.LoadSession(ctx, sessionID)
	if err != nil {
		return domain.MoveResult{}, err
	}
	if session.CurrentIndex >= len(session.Items) {
		return domain.MoveResult{}, errors.New("session is complete")
	}
	item := &session.Items[session.CurrentIndex]
	if item.State.HintLevel < 3 {
		return domain.MoveResult{}, errors.New("three hints are required before reveal")
	}
	puzzle, err := s.catalog.Get(ctx, item.Fingerprint)
	if err != nil {
		return domain.MoveResult{}, err
	}
	for {
		nodes, err := nodesAtPath(puzzle.Solution, item.State.Path)
		if err != nil {
			return domain.MoveResult{}, err
		}
		if len(nodes) == 0 {
			break
		}
		item.State.CurrentFEN, err = s.rules.ApplyUCI(item.State.CurrentFEN, nodes[0].UCI)
		if err != nil {
			return domain.MoveResult{}, err
		}
		item.State.Path = append(item.State.Path, 0)
	}
	item.State.Revealed = true
	item.State.Completed = true
	return s.completeCurrent(ctx, session, *item, puzzle)
}

func (s *Service) Pause(ctx context.Context, sessionID string) error {
	return s.store.SetSessionStatus(ctx, sessionID, "paused", s.now())
}

func (s *Service) Summary(ctx context.Context, sessionID string) (domain.SessionSummary, error) {
	session, err := s.store.LoadSession(ctx, sessionID)
	if err != nil {
		return domain.SessionSummary{}, err
	}
	summary := domain.SessionSummary{Total: len(session.Items)}
	for _, item := range session.Items {
		state := item.State
		if state.Unavailable {
			summary.Unavailable++
			continue
		}
		if !state.Completed {
			continue
		}
		if state.IncorrectMoves == 0 && state.HintsUsed == 0 && !state.Revealed {
			summary.FirstTry++
		}
		if state.IncorrectMoves > 0 {
			summary.Retried++
		}
		if state.HintsUsed > 0 {
			summary.UsedHint++
		}
		if state.Revealed {
			summary.Revealed++
		}
	}
	return summary, nil
}

func (s *Service) completeCurrent(
	ctx context.Context,
	session storedSession,
	item storedItem,
	puzzle domain.Puzzle,
) (domain.MoveResult, error) {
	now := s.now()
	effects, err := s.effectsForCompletion(ctx, item, puzzle, now)
	if err != nil {
		return domain.MoveResult{}, err
	}
	session, err = s.store.CompleteItem(ctx, session, item.State, now, effects)
	if err != nil {
		return domain.MoveResult{}, err
	}
	session, err = s.prepareAvailable(ctx, session)
	if err != nil {
		return domain.MoveResult{}, err
	}
	view, err := s.view(ctx, session)
	return domain.MoveResult{Session: view, Correct: true, PuzzleCompleted: true}, err
}

func (s *Service) effectsForCompletion(
	ctx context.Context,
	item storedItem,
	puzzle domain.Puzzle,
	now time.Time,
) (completionEffects, error) {
	var effects completionEffects
	outcome := OutcomeClean
	switch {
	case item.State.Revealed:
		outcome = OutcomeRevealed
	case item.State.HintsUsed > 0:
		outcome = OutcomeHinted
	case item.State.IncorrectMoves > 0:
		outcome = OutcomeMissed
	}
	if item.State.Kind == ScheduledReview || outcome != OutcomeClean {
		current, _, err := s.store.Review(ctx, item.Fingerprint)
		if err != nil {
			return completionEffects{}, err
		}
		next := NextReview(now, current, outcome)
		effects.Review = &next
	}
	if item.State.UpdatesRating && puzzle.Rating != nil {
		seen, err := s.store.HasCompletedAttemptBefore(ctx, item.Fingerprint, item.State.AttemptID)
		if err != nil {
			return completionEffects{}, err
		}
		if !seen {
			profile, err := s.store.Profile(ctx)
			if err != nil {
				return completionEffects{}, err
			}
			score := 1.0
			if item.State.Revealed {
				score = 0
			} else if item.State.IncorrectMoves > 0 || item.State.HintsUsed > 0 {
				score = 0.5
			}
			updated := UpdateRating(profile.LearnerRating, float64(*puzzle.Rating), score, 400, 3000)
			effects.NewRating = &updated
		}
	}
	return effects, nil
}

func (s *Service) prepareAvailable(ctx context.Context, session storedSession) (storedSession, error) {
	for session.Status != "completed" && session.CurrentIndex < len(session.Items) {
		_, err := s.catalog.Get(ctx, session.Items[session.CurrentIndex].Fingerprint)
		if err == nil {
			return session, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return storedSession{}, err
		}
		var skipErr error
		session, skipErr = s.store.SkipUnavailable(ctx, session, s.now())
		if skipErr != nil {
			return storedSession{}, skipErr
		}
	}
	return session, nil
}

func (s *Service) view(ctx context.Context, session storedSession) (domain.SessionView, error) {
	view := domain.SessionView{
		SessionID:    session.ID,
		Mode:         session.Mode,
		Status:       session.Status,
		CurrentIndex: session.CurrentIndex,
		Total:        len(session.Items),
	}
	if session.CurrentIndex >= len(session.Items) {
		summary, err := s.Summary(ctx, session.ID)
		if err != nil {
			return domain.SessionView{}, err
		}
		view.Summary = &summary
		return view, nil
	}
	item := session.Items[session.CurrentIndex]
	puzzle, err := s.catalog.Get(ctx, item.Fingerprint)
	if errors.Is(err, sql.ErrNoRows) {
		return view, nil
	}
	if err != nil {
		return domain.SessionView{}, err
	}
	view.Current = &domain.PuzzleView{
		Fingerprint:    puzzle.Fingerprint,
		SourceFEN:      puzzle.SourceFEN,
		DisplayedFEN:   puzzle.DisplayedFEN,
		CurrentFEN:     item.State.CurrentFEN,
		PreludeUCI:     puzzle.PreludeUCI,
		Solver:         puzzle.Solver,
		CurrentPath:    slices.Clone(item.State.Path),
		PuzzleNumber:   session.CurrentIndex + 1,
		PuzzleTotal:    len(session.Items),
		HintLevel:      item.State.HintLevel,
		IncorrectMoves: item.State.IncorrectMoves,
		CanReveal:      item.State.HintLevel >= 3,
	}
	return view, nil
}

func mateInOneNodes(nodes []domain.MoveNode) bool {
	if len(nodes) == 0 {
		return false
	}
	for _, node := range nodes {
		if len(node.Children) != 0 {
			return false
		}
	}
	return true
}

func nodesAtPath(solution []domain.MoveNode, path []int) ([]domain.MoveNode, error) {
	nodes := solution
	for _, index := range path {
		if index < 0 || index >= len(nodes) {
			return nil, errors.New("stored solution path is invalid")
		}
		nodes = nodes[index].Children
	}
	return nodes, nil
}
