package openings

import (
	"context"
	"reflect"
	"testing"
)

func TestBuildPositionContextShowsRouteChoiceAndNextMoves(t *testing.T) {
	fixture := newOpeningServiceFixture(t)
	note := fixture.compiled.Notes["italian-overview"]
	note.Kind = "plan"
	fixture.compiled.Notes[note.NoteID] = note

	route, err := fixture.service.routeToPosition(
		fixture.compiled, "after-bc5", DepthReference,
	)
	if err != nil {
		t.Fatal(err)
	}
	positionContext, err := buildPositionContext(
		fixture.compiled, "after-bc5", DepthReference, route,
	)
	if err != nil {
		t.Fatal(err)
	}
	if positionContext.Label != "Giuoco Piano" ||
		positionContext.PlanText != note.Text ||
		positionContext.Evaluation.Code != EvaluationEqual {
		t.Fatalf("context = %+v", positionContext)
	}
	if got := contextMoveIDs(positionContext.ArrivedVia); !reflect.DeepEqual(
		got, []string{"black-bc5", "black-nf6"},
	) {
		t.Fatalf("arrivals = %v", got)
	}
	if !positionContext.ArrivedVia[0].Selected || positionContext.ArrivedVia[1].Selected {
		t.Fatalf("selection = %+v", positionContext.ArrivedVia)
	}
	if got := contextMoveIDs(positionContext.NextMoves); !reflect.DeepEqual(
		got, []string{"white-c3", "white-b4"},
	) {
		t.Fatalf("next = %v", got)
	}
}

func TestBuildPositionContextFiltersDepthAndHandlesRoot(t *testing.T) {
	fixture := newOpeningServiceFixture(t)
	route, err := fixture.service.routeToPosition(
		fixture.compiled, "after-bc5", DepthQuick,
	)
	if err != nil {
		t.Fatal(err)
	}
	positionContext, err := buildPositionContext(
		fixture.compiled, "after-bc5", DepthQuick, route,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := contextMoveIDs(positionContext.ArrivedVia); !reflect.DeepEqual(
		got, []string{"black-bc5"},
	) {
		t.Fatalf("quick arrivals = %v", got)
	}
	if got := contextMoveIDs(positionContext.NextMoves); !reflect.DeepEqual(
		got, []string{"white-c3"},
	) {
		t.Fatalf("quick next moves = %v", got)
	}

	rootRoute, err := fixture.service.routeToPosition(
		fixture.compiled, fixture.compiled.Pack.RootPositionID, DepthQuick,
	)
	if err != nil {
		t.Fatal(err)
	}
	root, err := buildPositionContext(
		fixture.compiled, fixture.compiled.Pack.RootPositionID, DepthQuick, rootRoute,
	)
	if err != nil {
		t.Fatal(err)
	}
	if root.ArrivedVia == nil || len(root.ArrivedVia) != 0 {
		t.Fatalf("root arrivals = %#v", root.ArrivedVia)
	}
}

func TestBuildPositionContextUsesTheRibbonRouteAtATransposition(t *testing.T) {
	fixture := newOpeningServiceFixture(t)
	installExplorerTransposition(t, fixture)
	course, err := fixture.catalog.LoadActive(
		context.Background(), fixture.compiled.Pack.CourseID,
	)
	if err != nil {
		t.Fatal(err)
	}
	route, err := fixture.service.routeToPosition(course, "trans-knights", DepthReference)
	if err != nil {
		t.Fatal(err)
	}
	positionContext, err := buildPositionContext(
		course, "trans-knights", DepthReference, route,
	)
	if err != nil {
		t.Fatal(err)
	}
	selectedID := route.MoveIDs[len(route.MoveIDs)-1]
	selectedCount := 0
	for _, move := range positionContext.ArrivedVia {
		if move.Selected {
			selectedCount++
			if move.MoveID != selectedID {
				t.Fatalf("selected arrival = %q, route ends with %q", move.MoveID, selectedID)
			}
		}
	}
	if selectedCount != 1 {
		t.Fatalf("selected arrival count = %d", selectedCount)
	}
}

func contextMoveIDs(moves []OpeningContextMoveView) []string {
	result := make([]string, len(moves))
	for index, move := range moves {
		result[index] = move.MoveID
	}
	return result
}
