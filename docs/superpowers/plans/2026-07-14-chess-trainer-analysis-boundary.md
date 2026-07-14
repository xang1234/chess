# Chess Trainer External Analysis Boundary Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Define and verify the version-one external-analysis contract without bundling, launching, calling, or implementing any chess engine.

**Architecture:** Add transport-neutral analysis request, capability, update, and session interfaces to the Go core; compose a disabled provider in the application; and use a deterministic fake provider for contract tests. UCI executable and remote HTTP adapters are named extension points only and are explicitly absent from this plan. This is plan 3 of 3 and begins after plans 1 and 2 pass their completion gates.

**Tech Stack:** Existing Go/Wails project, Go contexts and channels, standard-library tests, and no new runtime dependency.

## Global Constraints

- Do not add a chess engine binary, evaluation algorithm, UCI process launcher, HTTP client, engine settings screen, or analysis panel.
- Puzzle-source solutions remain authoritative and never depend on analysis availability.
- The interface must support local UCI and remote HTTP adapters without importing either transport into domain packages.
- Scores are normalized from the side-to-move perspective.
- Cancellation must be observable and idempotent.
- Provider failures must not mutate puzzle, library, session, attempt, rating, or review data.

## Planned File Map

- `internal/analysis/provider.go` — transport-neutral contract and value types.
- `internal/analysis/disabled.go` — safe version-one provider.
- `internal/analysis/registry.go` — explicit provider selection without transport knowledge.
- `internal/analysis/contract_test.go` — reusable behavioral contract exercised by a fake.
- `internal/analysis/architecture_test.go` — guards the version-one no-engine/no-network boundary.
- `internal/app/services.go` — composes the disabled provider.
- `docs/architecture/analysis-provider.md` — adapter obligations for later work.

---

### Task 1: Define analysis value types and streaming-session contract

**Files:**
- Create: `internal/analysis/provider.go`
- Test: `internal/analysis/provider_test.go`

**Interfaces:**
- Produces: `analysis.Provider`, `analysis.Session`, `analysis.Request`, `analysis.Limits`, `analysis.Capabilities`, `analysis.Update`, and `analysis.Score`.

- [ ] **Step 1: Write request-validation tests**

Create `internal/analysis/provider_test.go`:

```go
package analysis

import "testing"

func TestRequestValidate(t *testing.T) {
	valid := Request{
		FEN: "7k/5Q2/6K1/8/8/8/8/8 w - - 0 1",
		Limits: Limits{Depth: 16},
		MultiPV: 1,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid request: %v", err)
	}

	invalid := []Request{
		{},
		{FEN: valid.FEN, MultiPV: 1},
		{FEN: valid.FEN, Limits: Limits{Depth: 1, Nodes: 10}, MultiPV: 1},
		{FEN: valid.FEN, Limits: Limits{Depth: 16}, MultiPV: 0},
	}
	for _, request := range invalid {
		if err := request.Validate(); err == nil {
			t.Fatalf("expected invalid request: %#v", request)
		}
	}
}
```

Run: `go test ./internal/analysis -run TestRequestValidate -v`

Expected: FAIL because the types are undefined.

- [ ] **Step 2: Implement exact value types and validation**

Create `internal/analysis/provider.go`:

```go
package analysis

import (
	"context"
	"errors"
	"time"
)

var ErrUnavailable = errors.New("analysis provider unavailable")

type Limits struct {
	Depth int `json:"depth,omitempty"`
	Nodes int64 `json:"nodes,omitempty"`
	MoveTime time.Duration `json:"moveTime,omitempty"`
}

type Request struct {
	FEN string `json:"fen"`
	Limits Limits `json:"limits"`
	MultiPV int `json:"multiPv"`
}

func (r Request) Validate() error {
	if r.FEN == "" {
		return errors.New("fen is required")
	}
	limitCount := 0
	if r.Limits.Depth > 0 { limitCount++ }
	if r.Limits.Nodes > 0 { limitCount++ }
	if r.Limits.MoveTime > 0 { limitCount++ }
	if limitCount != 1 {
		return errors.New("exactly one analysis limit is required")
	}
	if r.MultiPV < 1 {
		return errors.New("multiPv must be at least 1")
	}
	return nil
}

type Capabilities struct {
	ProviderID string `json:"providerId"`
	SupportsDepth bool `json:"supportsDepth"`
	SupportsNodes bool `json:"supportsNodes"`
	SupportsMoveTime bool `json:"supportsMoveTime"`
	MaxMultiPV int `json:"maxMultiPv"`
}

type Score struct {
	Centipawns *int `json:"centipawns,omitempty"`
	MateIn *int `json:"mateIn,omitempty"`
}

type Line struct {
	Rank int `json:"rank"`
	Score Score `json:"score"`
	MovesUCI []string `json:"movesUci"`
}

type Update struct {
	Depth int `json:"depth"`
	Nodes int64 `json:"nodes"`
	Lines []Line `json:"lines"`
	Final bool `json:"final"`
}

type Session interface {
	ID() string
	Updates() <-chan Update
	Cancel() error
	Wait() error
}

type Provider interface {
	Capabilities(context.Context) (Capabilities, error)
	HealthCheck(context.Context) error
	Start(context.Context, Request) (Session, error)
}
```

Run: `go test ./internal/analysis -run TestRequestValidate -v`

Expected: PASS.

- [ ] **Step 3: Add score-invariant tests**

Add `Score.Validate()` and tests requiring exactly one of centipawns or mate-in. Reject a line rank below 1, empty principal variation, duplicate ranks, updates with decreasing depth, and a terminal update followed by more updates in the contract helper introduced in Task 2.

- [ ] **Step 4: Commit the domain contract**

```bash
git add internal/analysis/provider.go internal/analysis/provider_test.go
git commit -m "feat: define external analysis contract"
```

### Task 2: Add disabled/registry implementations and reusable contract tests

**Files:**
- Create: `internal/analysis/disabled.go`
- Create: `internal/analysis/registry.go`
- Test: `internal/analysis/contract_test.go`
- Test: `internal/analysis/registry_test.go`
- Test: `internal/analysis/architecture_test.go`
- Modify: `internal/app/services.go`
- Create: `docs/architecture/analysis-provider.md`

**Interfaces:**
- Consumes: `analysis.Provider` from Task 1.
- Produces: `analysis.NewRegistry(defaultProvider)` and `analysis.DisabledProvider`.
- Produces: `RunProviderContract(t, factory)` for future UCI and HTTP adapter plans.

- [ ] **Step 1: Write the provider contract around a deterministic fake**

Create `internal/analysis/contract_test.go` with an in-test fake whose `Start` emits depth 8, depth 12, and a final depth 16 update. `RunProviderContract` must assert:

- capabilities are nonempty and permit the request;
- health check succeeds before start;
- session ID is nonempty;
- update depths never decrease;
- each update validates scores, ranks, and nonempty UCI lines;
- exactly one final update occurs;
- `Wait` returns nil after final update;
- cancelling a second session twice returns nil both times;
- the cancelled session closes updates and `Wait` returns `context.Canceled`.

Run: `go test ./internal/analysis -run TestFakeProviderContract -v`

Expected: PASS using only the in-test fake.

- [ ] **Step 2: Write disabled-provider and registry tests**

Assert `DisabledProvider.Capabilities`, `HealthCheck`, and `Start` all return `ErrUnavailable`. Assert registry rejects empty IDs, duplicate IDs, and unknown selection; selecting a registered fake returns the same provider instance.

- [ ] **Step 3: Implement safe version-one providers**

Create `internal/analysis/disabled.go`:

```go
package analysis

import "context"

type DisabledProvider struct{}

func (DisabledProvider) Capabilities(context.Context) (Capabilities, error) {
	return Capabilities{}, ErrUnavailable
}

func (DisabledProvider) HealthCheck(context.Context) error {
	return ErrUnavailable
}

func (DisabledProvider) Start(context.Context, Request) (Session, error) {
	return nil, ErrUnavailable
}
```

Create a mutex-protected `Registry` with `Register(id string, provider Provider) error`, `Select(id string) error`, and `Current() Provider`. Initialise it with ID `disabled` selected.

Run: `go test -race ./internal/analysis -v`

Expected: PASS.

- [ ] **Step 4: Compose the disabled provider without adding UI or bindings**

Add `Analysis analysis.Provider` to `internal/app.Services` and set it to `analysis.DisabledProvider{}`. Do not expose an App method, frontend binding, settings entry, or screen in version one.

- [ ] **Step 5: Add an architecture guard**

Create `internal/analysis/architecture_test.go` that parses non-test Go files under `internal/analysis` with `go/parser` and fails if an import path is `os/exec`, `net/http`, or any path ending `/uci`. It must also walk the repository and fail if an executable file is stored under `internal/analysis`, `assets/engine`, or `build/engine`.

Run: `go test ./internal/analysis -run TestVersionOneHasNoEngineTransport -v`

Expected: PASS.

- [ ] **Step 6: Document future adapter obligations without implementation steps**

Create `docs/architecture/analysis-provider.md` documenting:

- normalized side-to-move score perspective;
- capabilities and health-check behavior;
- one-limit request rule;
- monotonically increasing streamed depth;
- exactly one final update;
- idempotent cancellation;
- no mutation of training data;
- UCI adapter ownership of process lifecycle and handshake;
- HTTP adapter ownership of HTTPS/Tailscale transport and Keychain token retrieval;
- the requirement that both future adapters pass `RunProviderContract`.

- [ ] **Step 7: Run the complete version-one verification**

Run:

```bash
go test -race ./...
cd frontend
npm test -- --run
npm run test:e2e
npm run build
cd ..
wails build -clean
```

Expected: PASS with no new frontend analysis surface and no network listener.

- [ ] **Step 8: Commit the verified extension boundary**

```bash
git add internal/analysis internal/app/services.go docs/architecture/analysis-provider.md
git commit -m "feat: define external analysis boundary"
```

## Plan 3 Completion Gate

Run:

```bash
git status --short
go test -race ./...
go test ./internal/analysis -run TestVersionOneHasNoEngineTransport -v
wails build -clean
```

Expected: clean worktree, all tests PASS, package builds, the dependency guard produces no engine transport match, and the application behavior is unchanged when analysis is unavailable.
