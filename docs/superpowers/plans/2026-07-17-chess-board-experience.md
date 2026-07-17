# Chess Board Experience Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the Unicode-button board with a polished, offline Chessground puzzle experience that exposes all legal moves without revealing the answer, keeps a solved position visible until explicit acknowledgement, and distributes the public application compliantly under GPL-3.0-or-later.

**Architecture:** Go remains authoritative for legality, puzzle correctness, persistence, and every resulting FEN. A small typed Chessground adapter owns only rendering and input; pure TypeScript modules own UCI parsing, request identity, animation sequencing, and sound classification. `PuzzleScreen` coordinates those ports through a discriminated interaction state and stores the already-persisted next session as presentation-only pending state. Build identity comes from Go linker metadata and the same legal/source information is available in normal and recovery modes.

**Tech Stack:** Go module language 1.25 with exact release toolchain Go 1.26.4, `github.com/corentings/chess/v2`, Wails v2.12, Svelte 3, TypeScript 4.6, Vite 3, `@lichess-org/chessground` 10.1.1, Vitest, Testing Library, Playwright Chromium/WebKit, Web Audio, macOS `codesign`.

**Design source:** `docs/superpowers/specs/2026-07-17-chess-board-experience-design.md`

## Global Constraints

- Work only in `/Users/admin/Documents/Work/chess/.worktrees/chess-trainer` on `feature/chess-trainer`.
- Preserve the pre-existing unstaged database-picker edits in `app_test.go` and `normal_controller.go`. Do not stage, reformat, or include them in any task commit.
- Use RED, GREEN, refactor for every behavior change. Record the focused failing command before editing production code.
- At every task checkpoint, run the focused tests plus `go test ./... -count=1` or the relevant complete frontend suite so filtered tests cannot hide compile failures.
- Do not implement an engine, best-move search, evaluation, or analysis. The existing chess rules library may enumerate legal moves; the stored puzzle line remains the correctness authority.
- Do not add or migrate any database schema. New legal-move and animation fields are response projections over existing state.
- Treat every FEN returned by Go as authoritative. TypeScript must not recreate castling, en-passant, promotion, clocks, or other chess move semantics.
- A fresh puzzle prelude runs only when `currentPath` is empty and both `sourceFen` and `preludeUci` exist. A resumed partial puzzle starts at `currentFen`. Reduced motion skips directly to the applicable authoritative FEN.
- Persist completion immediately, but never render the returned next session until **Next puzzle** or **See results**. Navigating home from the transient solved view must still adopt the persisted next/completed session in parent state.
- Pin Chessground exactly to `10.1.1`; load JavaScript, CSS, pieces, sounds, and legal documents only from bundled application assets. No CDN or runtime network dependency is permitted.
- Do not set Chessground `viewOnly` after construction; it is not a reconfigurable lock. Lock input with empty destinations, no movable color, and disabled selection/drag.
- Never set `trustAllEvents` in production. Exercise real pointer behavior in Playwright.
- Generate Wails bindings with `/Users/admin/go/bin/wails generate module -tags bindings`; do not hand-maintain generated bindings.
- Use selective `git add` paths for every commit, then inspect `git diff --cached --name-only` and `git diff --cached --check` before committing.
- The public release wrapper must refuse a dirty, untagged, unpublished, or source-mismatched build. During implementation, verify it with unit fixtures and an ordinary local Wails build; do not weaken it merely because the development commit is not yet publicly tagged.
- Public source/build files must contain no absolute machine-local module replacement. Every binary release is accompanied by the verified generated corresponding-source artifact, not only lockfiles or download recipes.
- Build distributable binaries only with `go1.26.4`; record and verify that exact toolchain plus its Go runtime/standard-library `LICENSE` and `PATENTS` bytes. A different installed Go version requires an explicit reviewed lock/notices/source-manifest update before release.

## File Structure

### Backend and shared contracts

- Modify `internal/chessrules/rules.go` and `internal/chessrules/rules_test.go` for deterministic legal UCI enumeration.
- Modify `internal/domain/training.go`; create `internal/domain/training_test.go` for the JSON contract.
- Modify `internal/training/service.go`, `internal/training/service_test.go`, and `internal/training/task5_cutover_test.go`; create `internal/training/service_response_test.go` for animation projections and completion FEN behavior.
- Modify `internal/buildinfo/buildinfo.go` and `internal/buildinfo/buildinfo_test.go` for executable source identity.
- Modify `application.go` and `controllers_test.go` to expose build identity through `ModeController` in both modes.
- Regenerate `frontend/wailsjs/go/models.ts` and `frontend/wailsjs/go/main/ModeController.{js,d.ts}`.

### Board, puzzle orchestration, and feedback

- Create `frontend/src/lib/uci.ts` and `frontend/src/lib/uci.test.ts` for strict UCI parsing/grouping.
- Create `frontend/src/components/puzzle/puzzle-state.ts` and `.test.ts` for explicit phases and stale-response guards.
- Create `frontend/src/components/puzzle/move-animation.ts` and `.test.ts` for abortable authoritative-FEN sequences.
- Create `frontend/src/lib/move-feedback.ts` and `.test.ts` for move/capture classification by FEN occupancy.
- Create `frontend/src/components/chess/chessground-adapter.ts` and `.test.ts` for the imperative dependency boundary.
- Replace `frontend/src/components/chess/ChessBoard.svelte` and its tests with the controlled wrapper, keyboard cursor, legal destinations, and promotion chooser.
- Create `frontend/src/styles/chessground-theme.css`; modify `frontend/src/main.ts`, `frontend/src/styles/app.css`, and `frontend/src/styles/tokens.css` for local Chessground assets and the Classic Green layout.
- Create `frontend/scripts/generate-sounds.mjs`, commit generated WAV files under `frontend/src/assets/sounds/`, and create `frontend/src/lib/sound.ts` plus tests.
- Refactor `frontend/src/components/puzzle/PuzzleScreen.svelte` and tests; modify `frontend/src/components/app/NormalShell.svelte` and `frontend/src/App.test.ts` for deferred persisted session state.
- Modify `frontend/src/lib/api.ts`, `frontend/src/lib/api.test.ts`, and `frontend/src/test-fakes.ts` for exact frontend contracts and preview data.

### GPL, About, release, and browser coverage

- Create root `LICENSE` and `THIRD_PARTY_NOTICES.md`, matching Chessground/Nunito `frontend/public/legal/*` assets, and exact Go 1.26.4 `LICENSE`/`PATENTS` copies under `third_party/legal/go1.26.4/`.
- Commit `third_party/source/chessground-v10.1.1.tar.gz`, the complete upstream `v10.1.1` tag archive containing Chessground's preferred TypeScript source, tests, lockfile, and build configuration; verify its source/assets against the locked installed dependency.
- Commit the complete Svelte `v3.59.2` tag source archive, generate a locked Darwin runtime dependency inventory, and bundle every actual Go/Svelte/Wails/Chessground/Nunito license/NOTICE text in `THIRD_PARTY_NOTICES.md`.
- Create `frontend/src/lib/legal-assets.ts`, `frontend/src/lib/external-links.ts`, `frontend/src/components/legal/AboutLegal.svelte`, and their tests.
- Modify `frontend/src/lib/navigation.ts`, both application shells, `frontend/src/App.svelte`, `frontend/src/App.test.ts`, and `frontend/src/vite-env.d.ts` so legal information works in normal and recovery modes.
- Create `scripts/verify-legal-assets.mjs`, `scripts/verify-release.mjs`, `scripts/verify-release.test.mjs`, and `scripts/build-release.sh`.
- Create notice-generation and corresponding-source builders/verifiers; the release source artifact contains the tagged app source, production Go vendor tree, full Wails build-tool source, and committed frontend runtime source archives.
- Replace the template `README.md`; update `docs/operations/local-build.md`; create `docs/operations/release.md`.
- Create `frontend/tests/board-driver.ts` and `frontend/tests/board-interactions.spec.ts`; update `frontend/tests/trainer.spec.ts` and `frontend/playwright.config.ts`.

---

## Task 1: Adopt GPL-3.0-or-later and Pin Chessground

**Files:**

- Create: `LICENSE`
- Create: `THIRD_PARTY_NOTICES.md`
- Create: `frontend/public/legal/LICENSE.txt`
- Create: `frontend/public/legal/CHESSGROUND_LICENSE.txt`
- Create: `frontend/public/legal/NUNITO_OFL.txt`
- Create: `frontend/public/legal/THIRD_PARTY_NOTICES.md`
- Create: `scripts/verify-legal-assets.mjs`
- Create: `scripts/verify-legal-assets.test.mjs`
- Create: `scripts/generate-third-party-notices.mjs`
- Create: `scripts/generate-third-party-notices.test.mjs`
- Create: `third_party/source/chessground-v10.1.1.tar.gz`
- Create: `third_party/source/svelte-v3.59.2.tar.gz`
- Create: `third_party/legal/go1.26.4/LICENSE`
- Create: `third_party/legal/go1.26.4/PATENTS`
- Create: `third_party/runtime-dependencies.lock.json`
- Modify: `go.mod`
- Modify: `go.sum` only if `go mod tidy` changes it
- Modify: `frontend/package.json`
- Modify: `frontend/package-lock.json`

- [ ] **Step 1: Add the failing legal-asset verifier**

Implement `scripts/verify-legal-assets.mjs` with exported pure helpers and a CLI entry point. It must read the repository root relative to `import.meta.url` and assert all of these exact invariants:

```js
assert.equal(packageJSON.dependencies['@lichess-org/chessground'], '10.1.1')
assert.equal(
  packageLock.packages['node_modules/@lichess-org/chessground'].version,
  '10.1.1'
)
assert.equal(frontendAppLicense, rootLicense)
assert.equal(frontendNotice, rootNotice)
assert.equal(frontendChessgroundLicense, installedChessgroundLicense)
assert.equal(frontendNunitoLicense, repositoryNunitoLicense)
assert.match(rootNotice, /@lichess-org\/chessground 10\.1\.1/)
assert.match(rootNotice, /GPL-3\.0-or-later/)
assert.match(rootNotice, /Lichess Team <contact@lichess\.org>/)
assert.match(rootNotice, /Nunito/)
assert.equal(sha256(rootLicense), '8ceb4b9ee5adedde47b31e975c1d90c73ad27b6b165a1dcd80c7c545eb65b903')
assert.equal(sha256(chessgroundSourceArchive), 'a926875d49a5a3302bc17051480577ddbc221f879f990cda5c5f6cea38bfecd5')
assert.equal(sha256(svelteSourceArchive), '2360bdebd06141a2f0566364c7b42e87140bf2ed9494df29dac0bd43dbf99bad')
assert.equal(lockedRuntime.goToolchain.version, 'go1.26.4')
assert.equal(sha256(goToolchainLicense), '911f8f5782931320f5b8d1160a76365b83aea6447ee6c04fa6d5591467db9dad')
assert.equal(sha256(goToolchainPatents), '96f408bfae65bf137fc2525d3ecb030271c50c1e90799f87abf8846d8dd505cc')
assert.equal(
  packageLock.packages['node_modules/@lichess-org/chessground'].integrity,
  'sha512-IBEs8+J64/zE8QB4NXxsvpjm/tHRjfQAdWwUh4xzqqN+RValgthWHemLnxsmtHFwuxvO4lHd+crp1ecgZxKVoQ=='
)
assert.equal(
  packageLock.packages['node_modules/svelte'].integrity,
  'sha512-vzSyuGr3eEoAtT/A6bmajosJZIUWySzY2CzB3w2pgPvnkUjGqlDnsNnA0PMO+mMAhuyMul6C2uuZzY6ELSkzyA=='
)
```

The verifier must extract the committed archive into a temporary directory, require full upstream build inputs such as `tsconfig.json`, `pnpm-lock.yaml`, tests, and `src/*.ts`, then recursively compare its `src/`, `assets/`, and `LICENSE` content with `frontend/node_modules/@lichess-org/chessground`. The npm package's generated `dist/` remains authenticated by the exact lockfile integrity. Together these checks prove the clean installed build input maps to the complete corresponding source archive, not merely its version string or license.

In `scripts/verify-legal-assets.test.mjs`, use temporary fixture roots to prove a truncated-but-identical root/public GPL copy is rejected by the canonical digest, changed Chessground/Svelte source archives are rejected, an installed package differing from its locked source/version is rejected, and a missing Nunito, Go toolchain license/patent grant, or generated runtime notice is rejected.

Run:

```bash
node scripts/verify-legal-assets.mjs
node --test scripts/verify-legal-assets.test.mjs
```

Expected: FAIL because the exact dependency and legal files do not exist.

- [ ] **Step 2: Remove the machine-local Go module replacement**

The existing dormant commented replacement still publishes a machine-local path and makes the source misleading. Delete this exact commented line with an intentional patch; `go mod edit -dropreplace` does not remove comments:

```text
// replace github.com/wailsapp/wails/v2 v2.12.0 => /Users/admin/go/pkg/mod
```

Run:

```bash
go mod tidy
go test ./... -count=1
if rg -n '/Users/|^\s*//\s*replace\s+' go.mod; then exit 1; fi
```

Expected: the module resolves `github.com/wailsapp/wails/v2 v2.12.0` through `go.sum`; no active or commented absolute `/Users/...` replacement remains. Retain `go.sum` only if tidy changes it.

- [ ] **Step 3: Install the exact dependency**

Run:

```bash
npm --prefix frontend install --save-exact @lichess-org/chessground@10.1.1
```

Verify that `frontend/package.json` contains a `dependencies` entry of exactly `"10.1.1"`, not a caret or tilde range, and that the lockfile package entry agrees. Add:

```json
"verify:licenses": "node ../scripts/verify-legal-assets.mjs"
```

to `frontend/package.json` scripts. `verify-legal-assets.mjs` also invokes the notice generator in `--check` mode, so one command verifies both legal copies and the runtime dependency closure.

- [ ] **Step 4: Commit exact Chessground and Svelte preferred-source archives**

Fetch `https://github.com/lichess-org/chessground/archive/refs/tags/v10.1.1.tar.gz`, the official upstream tag archive for the peeled `v10.1.1` commit `4d7e91bb02bd7ed2796aac2c1956c9552b323e7c`. Inspect that it contains the full source tree, tests, `tsconfig.json`, `pnpm-lock.yaml`, package/build scripts, assets, README, and license, then copy the unchanged archive to `third_party/source/chessground-v10.1.1.tar.gz`. Its required SHA-256 is:

```text
a926875d49a5a3302bc17051480577ddbc221f879f990cda5c5f6cea38bfecd5
```

The corresponding npm integrity is the full SHA-512 value asserted above. The verifier compares the installed npm package's preferred source/assets/license to this upstream tag archive. The archive must remain byte-for-byte unchanged and publicly available in every matching repository tag/release source tree.

Also fetch `https://github.com/sveltejs/svelte/archive/refs/tags/v3.59.2.tar.gz`, the complete Svelte tag archive for peeled commit `06553d9b0927bcd9016842abef749a226b86dd9e`. Store it unchanged as `third_party/source/svelte-v3.59.2.tar.gz` and require SHA-256:

```text
2360bdebd06141a2f0566364c7b42e87140bf2ed9494df29dac0bd43dbf99bad
```

The verifier checks the installed `svelte@3.59.2` against its exact lockfile integrity and requires this full source/build/test archive.

- [ ] **Step 5: Write failing runtime-inventory and notice tests**

`scripts/generate-third-party-notices.mjs` supports `--write` and `--check`. Its production closure is the union of:

```text
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go list -deps -json .
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go list -deps -json .
```

excluding the main module. Package-level standard-library entries are not duplicated, but the generator adds one explicit `goToolchain` runtime record for the statically linked Go runtime/standard library. That record contains `go env GOVERSION`, requires exact release version `go1.26.4`, and records the SHA-256 and full text source paths for the committed Go `LICENSE` and `PATENTS`. It also adds bundled frontend runtime entries for `@lichess-org/chessground@10.1.1`, `svelte@3.59.2`, and the Nunito font. Wails' generated browser runtime is covered by the Wails module entry.

The current Go closure must include the exact versions of `github.com/corentings/chess/v2`, `github.com/dustin/go-humanize`, `github.com/google/uuid`, `github.com/klauspost/compress`, `github.com/leaanthony/go-ansi-parser`, `github.com/leaanthony/slicer`, `github.com/leaanthony/u`, `github.com/mattn/go-isatty`, `github.com/ncruces/go-strftime`, `github.com/pkg/errors`, `github.com/remyoudompheng/bigfft`, `github.com/rivo/uniseg`, `github.com/wailsapp/wails/v2`, `golang.org/x/sys`, `modernc.org/libc`, `modernc.org/mathutil`, `modernc.org/memory`, and `modernc.org/sqlite`. The generated lock records the exact version plus SHA-256 of every copied `LICENSE*`, `COPYING*`, and `NOTICE*` file. A missing license/notice or absolute/local module replacement is a hard failure.

Tests use injected `go list`, module-cache, and frontend-package fixtures to prove:

- arm64/amd64 closures are unioned and deduplicated;
- an added/removed/version-changed production module fails `--check`;
- an absolute replace is rejected;
- all exact license and upstream NOTICE texts appear in the generated document;
- a missing/different Go toolchain version, Go license, or Go patent grant fails `--check`;
- identical license texts may be grouped only when every dependency/version remains named;
- Svelte, Chessground, Wails, and Nunito cannot be omitted;
- the public notice copy must byte-match the root document.

Run:

```bash
node --test scripts/generate-third-party-notices.test.mjs
```

Expected: FAIL because the generator and lock do not exist.

- [ ] **Step 6: Generate the complete runtime notices and offline legal assets**

Add the unabridged GNU General Public License version 3 text dated 29 June 2007 to root `LICENSE`. Copy the exact Go 1.26.4 distribution files into `third_party/legal/go1.26.4/LICENSE` and `third_party/legal/go1.26.4/PATENTS`. The installed Homebrew layout places `LICENSE` one directory above `go env GOROOT` and `PATENTS` inside it, so the generator/verifier must locate both explicitly rather than assuming `$GOROOT/LICENSE`; require these exact SHA-256 values:

```text
LICENSE  911f8f5782931320f5b8d1160a76365b83aea6447ee6c04fa6d5591467db9dad
PATENTS  96f408bfae65bf137fc2525d3ecb030271c50c1e90799f87abf8846d8dd505cc
```

Implement the generator so `--write` creates `third_party/runtime-dependencies.lock.json`, root `THIRD_PARTY_NOTICES.md`, and its byte-identical `frontend/public/legal/THIRD_PARTY_NOTICES.md` copy containing:

- application license: `GPL-3.0-or-later`;
- package: `@lichess-org/chessground 10.1.1`;
- package author metadata: `Lichess Team <contact@lichess.org>`;
- upstream repository: `https://github.com/lichess-org/chessground`;
- corresponding preferred source: `third_party/source/chessground-v10.1.1.tar.gz`;
- a statement that the complete upstream license follows in the bundled `CHESSGROUND_LICENSE.txt`;
- Svelte 3.59.2, its MIT license/copyright, and preferred-source archive path;
- Nunito font copyright `Copyright 2016 The Nunito Project Authors (contact@sansoxygen.com)` and SIL Open Font License 1.1;
- Go runtime/standard library `go1.26.4`, the complete Go BSD license text, and the complete Go patent grant from the committed exact copies;
- every exact Darwin production Go module/version, complete license text, and upstream NOTICE text found by the locked closure.

Run:

```bash
go mod download all
node scripts/generate-third-party-notices.mjs --write
node scripts/generate-third-party-notices.mjs --check
```

Copy root `LICENSE` byte-for-byte to `frontend/public/legal/LICENSE.txt`, the installed `frontend/node_modules/@lichess-org/chessground/LICENSE` byte-for-byte to `frontend/public/legal/CHESSGROUND_LICENSE.txt`, and existing `frontend/src/assets/fonts/OFL.txt` byte-for-byte to `frontend/public/legal/NUNITO_OFL.txt`. Do not invent an upstream Chessground copyright line not present in the package.

- [ ] **Step 7: Make the verifier pass**

Run:

```bash
npm --prefix frontend run verify:licenses
node --test scripts/verify-legal-assets.test.mjs
node --test scripts/generate-third-party-notices.test.mjs
npm --prefix frontend run check
npm --prefix frontend run build
```

Expected: all pass; `frontend/dist/legal/` contains all four bundled documents, the exact installed Chessground/Svelte inputs have their matching preferred-source archives, and the locked Darwin runtime closure—including the Go runtime/standard-library license and patent grant—has complete bundled notices.

- [ ] **Step 8: Commit only licensing, source, and dependency files**

Run:

```bash
git add LICENSE THIRD_PARTY_NOTICES.md go.mod go.sum frontend/package.json frontend/package-lock.json frontend/public/legal scripts/verify-legal-assets.mjs scripts/verify-legal-assets.test.mjs scripts/generate-third-party-notices.mjs scripts/generate-third-party-notices.test.mjs third_party/legal/go1.26.4/LICENSE third_party/legal/go1.26.4/PATENTS third_party/runtime-dependencies.lock.json third_party/source/chessground-v10.1.1.tar.gz third_party/source/svelte-v3.59.2.tar.gz
git diff --cached --name-only
git diff --cached --check
git commit -m "chore: adopt GPL and pin Chessground"
```

Expected staged paths are limited to this task; `app_test.go` and `normal_controller.go` remain unstaged.

---

## Task 2: Enumerate Every Legal UCI Move in Go

**Files:**

- Modify: `internal/chessrules/rules.go`
- Modify: `internal/chessrules/rules_test.go`

- [ ] **Step 1: Write failing ordinary and special-move tests**

Add:

```go
func TestLegalMovesReturnsSortedStartingPosition(t *testing.T)
func TestLegalMovesIncludesCastlingEnPassantAndPromotions(t *testing.T)
func TestLegalMovesRejectsInvalidFEN(t *testing.T)
```

Use the exact starting-position expectation:

```go
want := []string{
	"a2a3", "a2a4", "b1a3", "b1c3", "b2b3", "b2b4",
	"c2c3", "c2c4", "d2d3", "d2d4", "e2e3", "e2e4",
	"f2f3", "f2f4", "g1f3", "g1h3", "g2g3", "g2g4",
	"h2h3", "h2h4",
}
```

Use these special fixtures and assert the listed moves are present:

```go
"r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1" // e1c1, e1g1
"8/8/8/3pP3/8/8/8/K6k w - d6 0 1"       // e5d6
"7k/P7/8/8/8/8/8/K7 w - - 0 1"          // a7a8b/n/q/r
```

Run:

```bash
go test ./internal/chessrules -run '^TestLegalMoves' -count=1
```

Expected: FAIL because `Rules.LegalMoves` does not exist.

- [ ] **Step 2: Implement deterministic legal move enumeration**

Add `sort` to the imports and implement:

```go
func (Rules) LegalMoves(fen string) ([]string, error) {
	game, err := gameAt(fen)
	if err != nil {
		return nil, fmt.Errorf("list legal moves: %w", err)
	}

	validMoves := game.ValidMoves()
	result := make([]string, len(validMoves))
	notation := chess.UCINotation{}
	for index := range validMoves {
		result[index] = notation.Encode(game.Position(), &validMoves[index])
	}
	sort.Strings(result)
	return result, nil
}
```

Do not filter by the stored puzzle solution.

- [ ] **Step 3: Verify and commit**

Run:

```bash
go test ./internal/chessrules -count=1
go test ./... -count=1
git add internal/chessrules/rules.go internal/chessrules/rules_test.go
git diff --cached --check
git commit -m "feat: expose legal puzzle moves"
```

---

## Task 3: Return Authoritative Applied FENs and the Completed Position

**Files:**

- Modify: `internal/domain/training.go`
- Create: `internal/domain/training_test.go`
- Modify: `internal/training/service.go`
- Create: `internal/training/service_response_test.go`
- Modify: `internal/training/service_test.go`
- Modify: `internal/training/task5_cutover_test.go`
- Regenerate: `frontend/wailsjs/go/models.ts`

- [ ] **Step 1: Lock the JSON contract with failing tests**

In `internal/domain/training_test.go`, marshal representative values and assert exact field names and omission behavior:

```go
type AppliedMove struct {
	UCI          string `json:"uci"`
	ResultingFEN string `json:"resultingFen"`
}
```

The encoded puzzle view must contain `"legalMoves"`. A populated result must contain:

```json
"appliedMoves":[{"uci":"e2e4","resultingFen":"after"}],"finalFen":"after"
```

An empty result must omit both `appliedMoves` and `finalFen`.

Run:

```bash
go test ./internal/domain -run 'JSON' -count=1
```

Expected: FAIL because the fields and type do not exist.

- [ ] **Step 2: Add the domain projection fields**

Add:

```go
type AppliedMove struct {
	UCI          string `json:"uci"`
	ResultingFEN string `json:"resultingFen"`
}
```

Add this field to `PuzzleView`:

```go
LegalMoves []string `json:"legalMoves"`
```

Add these fields to `MoveResult`:

```go
AppliedMoves []AppliedMove `json:"appliedMoves,omitempty"`
FinalFEN     string        `json:"finalFen,omitempty"`
```

Run the domain test and confirm GREEN.

- [ ] **Step 3: Write failing service response tests**

Create focused tests using the existing `openTrainingStores`, `importTrainingPuzzles`, `trainingPuzzle`, and `testMoveLine` helpers:

```go
func TestServiceViewIncludesEveryLegalMove(t *testing.T)
func TestServiceCorrectNonFinalMoveRefreshesLegalMoves(t *testing.T)
func TestServicePlayMoveReturnsSubmittedMoveAndAutomaticReply(t *testing.T)
func TestServiceWrongMoveReturnsNoAppliedMovesOrFinalFEN(t *testing.T)
func TestServiceCompletionKeepsFinalFENWhilePreparingNextPuzzle(t *testing.T)
func TestServiceLastPuzzleCompletionReturnsFinalFENWithSummary(t *testing.T)
func TestServiceRevealReturnsCompleteRemainingAppliedLine(t *testing.T)
```

Add a test-only helper that applies each expected UCI through `chessrules.Rules.ApplyUCI` and returns `[]domain.AppliedMove` plus the final FEN. Assert:

- the initial start position exposes all 20 legal moves, including moves unrelated to the solution;
- for solution `e2e4, e7e5, g1f3`, the first response carries `e2e4` and automatic `e7e5`, each with its intermediate FEN;
- legal-but-wrong `e2e3` carries no applied moves or final FEN and leaves `currentFen` unchanged;
- a completed first puzzle returns its terminal `finalFen` while `session.current` already identifies puzzle two;
- a completed last puzzle retains `finalFen` while returning `status == "completed"`, `current == nil`, and a summary;
- reveal returns every remaining move and intermediate FEN in order.

Extend `TestServiceAcceptsAlternativeMateAndPauseResume` to require the actual accepted `f7g7` and its FEN, not the stored alternative `f7f8`. Extend `TestEverySolvePathRevalidatesExactOccurrenceAtCompletion` to require no applied/final payload when the occurrence disappears.

Run:

```bash
go test ./internal/training -run '^(TestServiceViewIncludesEveryLegalMove|TestServiceCorrectNonFinalMoveRefreshesLegalMoves|TestServicePlayMoveReturnsSubmittedMoveAndAutomaticReply|TestServiceWrongMoveReturnsNoAppliedMovesOrFinalFEN|TestServiceCompletionKeepsFinalFENWhilePreparingNextPuzzle|TestServiceLastPuzzleCompletionReturnsFinalFENWithSummary|TestServiceRevealReturnsCompleteRemainingAppliedLine|TestServiceAcceptsAlternativeMateAndPauseResume|TestEverySolvePathRevalidatesExactOccurrenceAtCompletion)$' -count=1
```

Expected: FAIL on missing legal-move and response data.

- [ ] **Step 4: Record every accepted move after Go applies it**

Add this service helper:

```go
func (s *Service) applyRecordedMove(fen, uci string) (domain.AppliedMove, error) {
	resultingFEN, err := s.rules.ApplyUCI(fen, uci)
	if err != nil {
		return domain.AppliedMove{}, err
	}
	return domain.AppliedMove{UCI: uci, ResultingFEN: resultingFEN}, nil
}
```

Use it only after the existing validation has accepted a submitted/revealed move. Stage applied moves locally and include them in a response only after all persistence and catalogue revalidation succeeds. Illegal moves, incorrect puzzle moves, persistence errors, and disappeared occurrences must never leak optimistic animation data.

At each accepted move site, update state from the recorded result rather than applying the move twice:

```go
applied, err := s.applyRecordedMove(item.State.CurrentFEN, uci)
if err != nil {
	return domain.MoveResult{}, err
}
item.State.CurrentFEN = applied.ResultingFEN
appliedMoves = append(appliedMoves, applied)
```

Repeat the same pattern for the single automatic reply and each revealed move. The alternative-mate branch records the submitted UCI that actually mated.

- [ ] **Step 5: Populate legal moves and preserve the completed FEN**

In `view`, call `s.rules.LegalMoves(item.State.CurrentFEN)` and wrap failure with the puzzle fingerprint. Populate `PuzzleView.LegalMoves` for every active puzzle.

Change the completion helper to accept `appliedMoves []domain.AppliedMove`. Capture:

```go
finalFEN := item.State.CurrentFEN
```

immediately before durable `CompleteItem` advancement. Return `FinalFEN: finalFEN` and a cloned applied slice after the next available session view is prepared. Pass recorded moves through normal completion, alternative mate, and reveal. A correct non-final move returns applied moves but an empty final FEN.

- [ ] **Step 6: Regenerate and inspect bindings**

Run:

```bash
/Users/admin/go/bin/wails generate module -tags bindings
```

Confirm `frontend/wailsjs/go/models.ts` contains `AppliedMove`, `PuzzleView.legalMoves`, optional `MoveResult.appliedMoves`, and optional `MoveResult.finalFen`. Confirm no `NormalController` signature changed.

- [ ] **Step 7: Verify and commit**

Run:

```bash
go test ./internal/domain ./internal/training -count=1
go test ./... -count=1
go test -race ./... -count=1
git add internal/domain/training.go internal/domain/training_test.go internal/training/service.go internal/training/service_response_test.go internal/training/service_test.go internal/training/task5_cutover_test.go frontend/wailsjs/go/models.ts
git diff --cached --check
git commit -m "feat: return authoritative puzzle move frames"
```

---

## Task 4: Add Typed Frontend Contracts and Strict UCI Utilities

**Files:**

- Modify: `frontend/src/lib/api.ts`
- Modify: `frontend/src/lib/api.test.ts`
- Modify: `frontend/src/test-fakes.ts`
- Create: `frontend/src/lib/uci.ts`
- Create: `frontend/src/lib/uci.test.ts`

- [ ] **Step 1: Write failing UCI tests**

Cover:

```ts
parseUCI('e2e4')
parseUCI('a7a8q')
groupLegalMoves(['e2e4', 'e2e3', 'a7a8q', 'a7a8r'])
promotionChoices(['a7a8q', 'a7a8r'], 'a7', 'a8')
```

Require invalid ranks/files, uppercase, trailing text, unsupported promotion suffixes, and malformed entries to throw an error that includes the offending value. Structurally valid four-character UCI remains valid because only the backend knows whether a route requires promotion. Grouping must deduplicate the visual `a8` destination while retaining both legal promotion suffixes.

Run:

```bash
npm --prefix frontend test -- run src/lib/uci.test.ts --single-thread
```

Expected: FAIL because the module does not exist.

- [ ] **Step 2: Implement the pure UCI module**

Use these public contracts:

```ts
export type Square = `${'a'|'b'|'c'|'d'|'e'|'f'|'g'|'h'}${1|2|3|4|5|6|7|8}`
export type Promotion = 'q' | 'r' | 'b' | 'n'
export type ParsedUCI = { from: Square; to: Square; promotion?: Promotion }

export function parseUCI(value: string): ParsedUCI
export function groupLegalMoves(values: string[]): Map<Square, Square[]>
export function promotionChoices(
  values: string[], from: Square, to: Square
): Promotion[]
export function moveSquares(value: string): [Square, Square]
```

Validate with `/^[a-h][1-8][a-h][1-8][qrbn]?$/`. Sort/deduplicate destinations and promotion choices deterministically, but preserve the complete legal UCI set.

- [ ] **Step 3: Extend the frontend transport contract**

Add:

```ts
export type AppliedMove = { uci: string; resultingFen: string }
```

Add `legalMoves: string[]` to `PuzzleView`, and optional `appliedMoves`/`finalFen` to `MoveResult`. Update preview sessions and `test-fakes.ts` so every current puzzle has a complete legal move list and every successful move/reveal fixture has authoritative applied FENs. Make `previewNormalAPI` a deterministic two-puzzle in-memory flow: it must return **Try again** for its configured wrong legal move, return authoritative completion frames for the configured correct move, retain the persisted next session for resume, and produce a final summary. This gives the real-browser/manual touch check a functional board without Wails or network data.

In `api.test.ts`, assert production adaptation preserves `legalMoves`, `appliedMoves`, and `finalFen` exactly; no frontend inference or field renaming is allowed.

- [ ] **Step 4: Verify and commit**

Run:

```bash
npm --prefix frontend test -- run src/lib/uci.test.ts src/lib/api.test.ts --single-thread
npm --prefix frontend run check
git add frontend/src/lib/api.ts frontend/src/lib/api.test.ts frontend/src/lib/uci.ts frontend/src/lib/uci.test.ts frontend/src/test-fakes.ts
git diff --cached --check
git commit -m "feat: add typed puzzle board contracts"
```

---

## Task 5: Model Puzzle Phases and Authoritative FEN Animation

**Files:**

- Create: `frontend/src/components/puzzle/puzzle-state.ts`
- Create: `frontend/src/components/puzzle/puzzle-state.test.ts`
- Create: `frontend/src/components/puzzle/move-animation.ts`
- Create: `frontend/src/components/puzzle/move-animation.test.ts`
- Create: `frontend/src/lib/move-feedback.ts`
- Create: `frontend/src/lib/move-feedback.test.ts`

- [ ] **Step 1: Write failing phase-transition tests**

Define fixtures for a fresh puzzle, a partially solved/resumed puzzle, a returned next puzzle, and a final summary. Test all of these transitions:

- fresh `currentPath: []` plus `sourceFen`/`preludeUci` starts in `prelude` at `sourceFen`;
- non-empty `currentPath` starts in `ready` at `currentFen` and does not show a prelude;
- reduced motion starts directly at `currentFen`;
- `beginRequest` records operation, monotonic request ID, authoritative starting FEN, and optional submitted UCI;
- only the matching request ID is accepted;
- a successful same-puzzle hint request adopts returned counters/legal moves and hint data directly into `ready` without replaying the prelude;
- incorrect recovery adopts the returned session counters/legal moves, stores the UCI pair, and reopens input only after reconciliation;
- solved state retains the old displayed session and `finalFen` while storing the returned session as `pendingSession`;
- acknowledging non-final solved state produces a fresh state for the pending next puzzle;
- acknowledging final solved state exposes the pending summary;
- recoverable request failure retains an actionable message and authoritative FEN while allowing retry;
- fatal malformed-contract/legal-move failure retains its message and keeps input locked.

Run:

```bash
npm --prefix frontend test -- run src/components/puzzle/puzzle-state.test.ts --single-thread
```

Expected: FAIL because the state module does not exist.

- [ ] **Step 2: Implement a discriminated state union**

Use these public shapes:

```ts
export type Operation = 'move' | 'hint' | 'reveal' | 'pause'
export type SolvedOutcome = 'correct' | 'revealed'

type Common = {
  displaySession: SessionView
  fen: string
  hint: HintResult | null
  lastMove?: [Square, Square]
  notice?: string
}

export type PuzzleState =
  | (Common & { phase: 'prelude' })
  | (Common & { phase: 'ready' })
  | (Common & {
      phase: 'requesting'
      operation: Operation
      requestId: number
      authoritativeFen: string
      submittedUci?: string
    })
  | (Common & {
      phase: 'animating'
      operation: 'move' | 'reveal'
      requestId: number
    })
  | (Common & {
      phase: 'incorrect'
      wrongMove: [Square, Square]
      message: string
    })
  | (Common & {
      phase: 'solved'
      outcome: SolvedOutcome
      finalFen: string
      pendingSession: SessionView
    })
  | (Common & {
      phase: 'failed'
      message: string
      recoverable: boolean
    })
```

Export pure transition functions `initialisePuzzle`, `beginRequest`, `acceptsResponse`, `beginAnimation`, `finishReadyRequest`, `markIncorrect`, `markSolved`, `acknowledgeSolved`, and `markFailed`. `finishReadyRequest(state, requestId, returnedSession, hint)` transitions a matching hint or non-final move request to `ready` using `returnedSession.current.currentFen` and returned counters/legal moves; it never calls `initialisePuzzle`, so a same-puzzle hint cannot replay the prelude. `markIncorrect(state, requestId, returnedSession, uci, message)` likewise adopts returned counters/legal moves while retaining the reconciled authoritative FEN. `markFailed` requires an explicit `recoverable` argument. `beginRequest` accepts `ready`, `incorrect`, and recoverable `failed` states, but rejects fatal `failed`, locked, animating, and solved states. Reject all other impossible transitions with descriptive errors in development/tests instead of silently mutating unrelated phases.

The only fresh-prelude predicate is:

```ts
current.currentPath.length === 0 &&
Boolean(current.sourceFen && current.preludeUci)
```

- [ ] **Step 3: Write failing animation and feedback tests**

For `move-animation.test.ts`, use a fake position sink and fake delay. Cover:

- each `AppliedMove.resultingFen` is applied in order;
- an optimistic first UCI is reconciled to its first resulting FEN immediately but its route is not replayed;
- later automatic/reveal moves receive the configured delay and last-move pair;
- the promise remains pending for the final 180ms visual animation and resolves only after that frame finishes;
- reduced motion applies only the last authoritative FEN with no delay;
- abort stops further frames;
- a position sink that throws on every call invokes the separate caller-owned recovery/remount boundary with the authoritative final FEN and resolves as a fallback rather than retrying the broken sink or trapping the session;
- a recovery boundary that also throws yields a warning result so the caller can keep Next usable and show an actionable board notice.

For `move-feedback.test.ts`, compare FEN occupancy and require `capture` for ordinary capture and en passant, `move` for a quiet move/castling/promotion without capture, and an error for malformed FEN. This classifies feedback only; it must not decide legality.

Run:

```bash
npm --prefix frontend test -- run src/components/puzzle/move-animation.test.ts src/lib/move-feedback.test.ts --single-thread
```

Expected: FAIL because both modules are absent.

- [ ] **Step 4: Implement abortable FEN-frame sequencing**

Use an injected port rather than importing Chessground:

```ts
export type PositionFrame = {
  fen: string
  lastMove?: [Square, Square]
  animate: boolean
}

export type AnimationPort = {
  setPosition(frame: PositionFrame): void
  delay(milliseconds: number, signal: AbortSignal): Promise<void>
  recover(finalFen: string): void
}

export type AnimationResult = {
  status: 'completed' | 'aborted' | 'reconciled'
  warning?: string
}

export async function animateAppliedMoves(options: {
  port: AnimationPort
  startingFen: string
  appliedMoves: readonly AppliedMove[]
  optimisticUci?: string
  finalFen: string
  reducedMotion: boolean
  signal: AbortSignal
  onStep?: (kind: 'move' | 'capture', move: AppliedMove) => void
}): Promise<AnimationResult>
```

Use a consistent 180ms board animation and 220ms pause before later automatic/reveal frames. After every animated `setPosition`, including the final frame, await the abortable 180ms animation interval before resolving or enabling input. The matching optimistic first frame is reconciled without replay or animation and is the only no-wait exception. Always use `resultingFen`; never compute the next position in TypeScript.

On a sink/delay failure, call `port.recover(finalFen)` instead of retrying `setPosition`. In `PuzzleScreen`, that recovery updates controlled state to `finalFen`, increments a keyed board generation, and constructs a fresh adapter at that FEN; it never calls the failed adapter. Contain even a recovery/remount exception and return `{ status: 'reconciled', warning }` so the caller records the warning in `PuzzleState.notice` while the solved/next action remains usable. On lifecycle abort, return `{ status: 'aborted' }` without recovering or updating a subsequently mounted puzzle.

- [ ] **Step 5: Verify and commit**

Run:

```bash
npm --prefix frontend test -- run src/components/puzzle/puzzle-state.test.ts src/components/puzzle/move-animation.test.ts src/lib/move-feedback.test.ts --single-thread
npm --prefix frontend run check
git add frontend/src/components/puzzle/puzzle-state.ts frontend/src/components/puzzle/puzzle-state.test.ts frontend/src/components/puzzle/move-animation.ts frontend/src/components/puzzle/move-animation.test.ts frontend/src/lib/move-feedback.ts frontend/src/lib/move-feedback.test.ts
git diff --cached --check
git commit -m "feat: model puzzle interaction phases"
```

---

## Task 6: Wrap Chessground with Controlled Input and Accessible Keyboard Use

**Files:**

- Create: `frontend/src/components/chess/chessground-adapter.ts`
- Create: `frontend/src/components/chess/chessground-adapter.test.ts`
- Replace: `frontend/src/components/chess/ChessBoard.svelte`
- Replace: `frontend/src/components/chess/ChessBoard.test.ts`
- Create: `frontend/src/components/chess/PromotionDialog.svelte`
- Create: `frontend/src/components/chess/PromotionDialog.test.ts`
- Create: `frontend/src/styles/chessground-theme.css`
- Modify: `frontend/src/main.ts`
- Modify: `frontend/src/lib/uci.ts`
- Modify: `frontend/src/lib/uci.test.ts`
- Modify: `frontend/src/lib/fen.ts`
- Modify: `frontend/src/lib/fen.test.ts`

- [ ] **Step 1: Add failing adapter tests around a fake Chessground API**

Inject the dependency factory:

```ts
export type GroundFactory = (
  element: HTMLElement,
  config?: Config
) => Api

export type BoardCallbacks = {
  onRoute(from: Key, to: Key): void
  onSelect(key: Key): void
}

export type ChessgroundAdapterFactory = (
  element: HTMLElement,
  initialFen: string,
  interaction: BoardInteraction,
  callbacks: BoardCallbacks
) => ChessBoardAdapter

export function createChessgroundAdapter(
  element: HTMLElement,
  initialFen: string,
  interaction: BoardInteraction,
  callbacks: BoardCallbacks,
  factory?: GroundFactory
): ChessBoardAdapter
```

Capture every initial/update config and test:

- exact FEN and orientation;
- both `turnColor` and `movable.color` are white for a white puzzle and black for a black puzzle;
- `movable.free === false`, premoves/predrops are disabled, and `trustAllEvents` is not enabled;
- destinations come only from the validated legal move map;
- disabled state uses `movable.color: undefined`, empty destinations, `draggable.enabled: false`, and `selectable.enabled: false` without toggling `viewOnly`;
- selection of a movable-color piece with no destination entry immediately calls `api.selectSquare(null)`;
- last/wrong/hint/keyboard marker classes merge without erasing each other;
- `setPosition` is the only operation that includes a FEN, so an interaction-only update does not snap an optimistic move back prematurely;
- deferred `events.select`/`movable.events.after` callbacks do nothing after destroy;
- `destroy()` delegates exactly once.

Run:

```bash
npm --prefix frontend test -- run src/components/chess/chessground-adapter.test.ts --single-thread
```

Expected: FAIL because the adapter does not exist.

- [ ] **Step 2: Implement the typed Chessground adapter**

Use direct supported imports:

```ts
import { Chessground } from '@lichess-org/chessground'
import type { Api } from '@lichess-org/chessground/api'
import type { Config } from '@lichess-org/chessground/config'
import type { Dests, Key, KeyPair, SquareClasses } from '@lichess-org/chessground/types'
```

Expose:

```ts
export type BoardInteraction = {
  orientation: 'white' | 'black'
  legalMoves: readonly string[]
  inputEnabled: boolean
  lastMove?: KeyPair
  wrongMove?: KeyPair
  hintSource?: Key
  hintTarget?: Key
  keyboardCursor?: Key
  reducedMotion: boolean
}

export interface ChessBoardAdapter {
  configure(interaction: BoardInteraction): void
  setPosition(fen: string, lastMove?: KeyPair, animate?: boolean): void
  selectSquare(key: Key | null): void
  destroy(): void
}
```

The constructor's initial Chessground config includes `fen: initialFen` plus the supplied interaction. Create it with `coordinates: true`, `coordinatesOnSquares: false`, `autoCastle: true`, `disableContextMenu: true`, `blockTouchScroll: true`, `jsHover: true`, 180ms animation, `movable.showDests: true`, `movable.rookCastle: true`, `draggable.showGhost: true`, `draggable.deleteOnDropOff: false`, and drawing disabled. `events.select` and `movable.events.after` call `BoardCallbacks` and must consult the latest validated move index and a destroyed flag because Chessground defers callbacks with `setTimeout`.

Do not include `fen` in `configure`. `setPosition` is the explicit authoritative reconciliation boundary.

- [ ] **Step 3: Add failing keyboard, promotion, and wrapper lifecycle tests**

Extend `uci.ts` with:

```ts
export function moveKeyboardCursor(
  square: Square,
  arrow: 'ArrowUp' | 'ArrowDown' | 'ArrowLeft' | 'ArrowRight',
  orientation: 'white' | 'black'
): Square
```

Test board-edge clamping and reversed movement under black orientation. Add focused promotion dialog tests for restricted legal choices, focus placement, suffix dispatch, Escape cancellation, and focus restoration.

Rewrite `ChessBoard.test.ts` against an injected adapter factory. Cover:

- creation after mount, controlled configure updates, explicit position reconciliation, and destruction;
- click/drag route resolves only a legal full UCI;
- a promotion route opens a dialog and emits only after a legal suffix is chosen;
- cancellation restores the authoritative FEN;
- invalid legal-move data emits an actionable error and keeps input locked;
- Arrow keys update the visual cursor by orientation;
- Enter/Space selects a legal source, changes source, or submits a destination;
- Escape clears board selection or promotion;
- immobile pieces cannot retain selection;
- semantic square descriptions still use `parseFEN` and identify pieces/squares.

Run:

```bash
npm --prefix frontend test -- run src/lib/uci.test.ts src/components/chess/PromotionDialog.test.ts src/components/chess/ChessBoard.test.ts --single-thread
```

Expected: FAIL against the Unicode-button board.

- [ ] **Step 4: Replace the board with one focusable controlled wrapper**

Use props:

```ts
export let fen: string
export let orientation: 'white' | 'black' = 'white'
export let legalMoves: string[] = []
export let inputEnabled = true
export let lastMove: [Square, Square] | undefined
export let wrongMove: [Square, Square] | undefined
export let hintSource: Square | undefined
export let hintTarget: Square | undefined
export let reducedMotion = false
export let adapterFactory: ChessgroundAdapterFactory = createChessgroundAdapter
```

Emit `move`, `error`, and `announce`. Export imperative `setPosition(fen, lastMove, animate)` so `PuzzleScreen` can reconcile the same FEN after an optimistic wrong move; a reactive FEN assignment alone cannot do that.

Render one visible Chessground host as `aria-hidden="true"` inside a focusable `role="grid"`. Add an offscreen 8×8 semantic grid with stable IDs and `aria-activedescendant` for the keyboard cursor. Associate concise instructions with the board. Keep selection and keyboard cursor visually distinct. `PromotionDialog` offers Queen, Rook, Bishop, and Knight only when their suffixes exist in the legal UCI set.

- [ ] **Step 5: Import local Chessground assets and add the Classic Green theme**

Import in `frontend/src/main.ts` in this order:

```ts
import '@lichess-org/chessground/assets/chessground.base.css'
import '@lichess-org/chessground/assets/chessground.cburnett.css'
import './styles/tokens.css'
import './styles/app.css'
import './styles/chessground-theme.css'
```

Do not import Chessground's brown board CSS. In the global theme, use `#eeeed2` and `#769656`, a high-contrast selected square, a softer last-move overlay, red wrong source/target, gold hints, and a separate keyboard cursor. Preserve Chessground's `.move-dest` dot and `.move-dest.oc` capture-ring semantics. Put these selectors in global CSS because Chessground creates the elements dynamically.

Start from these exact board/destination rules and add the named wrong/hint/cursor classes without replacing them:

```css
.cg-wrap cg-board {
  background-color: #eeeed2;
  background-image: conic-gradient(
    from 90deg,
    #769656 25%,
    #eeeed2 0 50%,
    #769656 0 75%,
    #eeeed2 0
  );
  background-size: 25% 25%;
  touch-action: none;
}

.cg-wrap cg-board square.selected {
  background-color: rgba(246, 246, 84, 0.62);
}

.cg-wrap cg-board square.last-move {
  background-color: rgba(246, 246, 84, 0.34);
}

.cg-wrap cg-board square.move-dest {
  background: radial-gradient(rgba(30, 30, 30, 0.34) 0 22%, transparent 24%);
}

.cg-wrap cg-board square.move-dest.oc {
  background: radial-gradient(transparent 0 72%, rgba(30, 30, 30, 0.34) 74% 100%);
}
```

- [ ] **Step 6: Verify and commit**

Run:

```bash
npm --prefix frontend test -- run src/lib/uci.test.ts src/lib/fen.test.ts src/components/chess/chessground-adapter.test.ts src/components/chess/PromotionDialog.test.ts src/components/chess/ChessBoard.test.ts --single-thread
npm --prefix frontend run check
npm --prefix frontend run build
git add frontend/src/components/chess frontend/src/lib/uci.ts frontend/src/lib/uci.test.ts frontend/src/lib/fen.ts frontend/src/lib/fen.test.ts frontend/src/styles/chessground-theme.css frontend/src/main.ts
git diff --cached --check
git commit -m "feat: replace puzzle board with Chessground"
```

---

## Task 7: Add Reproducible Local Puzzle Sounds and Persistent Mute

**Files:**

- Create: `frontend/scripts/generate-sounds.mjs`
- Create: `frontend/src/assets/sounds/move.wav`
- Create: `frontend/src/assets/sounds/capture.wav`
- Create: `frontend/src/assets/sounds/correct.wav`
- Create: `frontend/src/assets/sounds/incorrect.wav`
- Create: `frontend/src/lib/sound.ts`
- Create: `frontend/src/lib/sound.test.ts`

- [ ] **Step 1: Write failing service tests with injected storage/audio**

Use these contracts:

```ts
export type SoundName = 'move' | 'capture' | 'correct' | 'incorrect'

export interface SoundBackend {
  unlock(): Promise<void>
  play(url: string, volume: number): void
  destroy(): void
}

export interface SoundService {
  readonly muted: boolean
  unlock(): Promise<void>
  play(name: SoundName): void
  setMuted(muted: boolean): void
  toggleMuted(): boolean
  destroy(): void
}
```

Test default unmuted state, restoration from `chess-trainer:sound-muted:v1`, persisted toggles, suppression while muted, correct local URL/volume selection, unlock from a user gesture, swallowed backend/audio failures, and one destroy call.

Run:

```bash
npm --prefix frontend test -- run src/lib/sound.test.ts --single-thread
```

Expected: FAIL because the service is absent.

- [ ] **Step 2: Implement a resilient Web Audio backend**

`createSoundService` accepts optional `backend`, `storage`, and `volume` dependencies for tests. The production backend lazily creates/resumes `AudioContext`, loads the four imported local asset URLs, and plays decoded buffers at a restrained default gain of `0.22`. `unlock()` must construct/resume the context synchronously before its first `await`; deferred fetching/decoding may continue afterward. A rejected fetch/decode/resume/play operation must be contained; audio can never block or fail puzzle state.

- [ ] **Step 3: Generate original deterministic WAV assets**

Implement a dependency-free Node generator for 44.1kHz mono 16-bit PCM with a short attack/release envelope. Use distinct, restrained project-original tone patterns:

- move: one 55ms wooden tap;
- capture: two descending taps totalling at most 95ms;
- correct: a three-note rising cue totalling at most 240ms;
- incorrect: two low descending notes totalling at most 180ms.

Support `--check` by regenerating in memory and byte-comparing all committed outputs. Run:

```bash
node frontend/scripts/generate-sounds.mjs
node frontend/scripts/generate-sounds.mjs --check
```

Expected: four non-empty reproducible WAV files and a passing check.

- [ ] **Step 4: Verify and commit**

Run:

```bash
npm --prefix frontend test -- run src/lib/sound.test.ts --single-thread
npm --prefix frontend run check
npm --prefix frontend run build
git add frontend/scripts/generate-sounds.mjs frontend/src/assets/sounds frontend/src/lib/sound.ts frontend/src/lib/sound.test.ts
git diff --cached --check
git commit -m "feat: add local puzzle sound feedback"
```

---

## Task 8: Orchestrate Wrong, Reply, Reveal, Solved, and Explicit Next States

**Files:**

- Replace: `frontend/src/components/puzzle/PuzzleScreen.svelte`
- Replace: `frontend/src/components/puzzle/PuzzleScreen.test.ts`
- Modify: `frontend/src/components/app/NormalShell.svelte`
- Modify: `frontend/src/App.test.ts`
- Modify: `frontend/src/styles/tokens.css`
- Modify: `frontend/src/styles/app.css`

- [ ] **Step 1: Write failing component tests for the complete puzzle lifecycle**

Use a fake API, injected effects port, and injected board adapter factory. Every puzzle fixture must include complete `legalMoves`; every successful response must include authoritative `appliedMoves`, and completion must include `finalFen`.

Cover:

- a fresh puzzle renders `sourceFen`, announces **Watch the last move…**, animates `preludeUci` to `displayedFen`, then enables input;
- a resumed puzzle with non-empty `currentPath` starts immediately at `currentFen` and skips the prelude;
- a legal wrong move disables input, calls the backend once, explicitly reconciles to the recorded authoritative FEN, waits for snapback, then shows red markers and **Try again** with input enabled;
- an unavailable-puzzle move or reveal response that advances to another session index/fingerprint (or no current puzzle) is adopted immediately with **Puzzle unavailable**, no red wrong marker, and no incorrect sound;
- a rejected move API call reconciles the optimistic board to the saved FEN, reports the error as recoverable, and permits a retry; malformed response/legal-move contracts remain fatal and locked;
- a correct non-final user move reconciles its optimistic first frame, animates the automatic reply frames in order, updates legal destinations, and reopens input;
- reveal animates the complete returned line and ends at `finalFen`;
- an animation exception jumps to the authoritative final FEN and still exposes the appropriate next action;
- a persistently throwing board adapter is replaced through the keyed recovery boundary at the final FEN instead of being called again;
- a stale API response after a session change or destroy cannot update the current board;
- completion keeps the old puzzle number and final board visible, shows **Correct!**, and does not render the returned next puzzle;
- **Next puzzle** applies the pending session locally without calling another backend mutation;
- a completed final response shows **See results**, then the existing summary only after acknowledgement;
- reveal completion says **Solution shown** and does not play the rewarding correct cue;
- quiet/capture frames play the correct sound, incorrect plays incorrect, and correct completion plays correct;
- sound unlock begins synchronously on the first pointerdown or Enter/Space activation, including a Hint/Reveal button used before any board move;
- the mute button has an accurate accessible label and `aria-pressed`, persists its state, and never removes textual feedback;
- malformed legal UCI data locks the board and shows an actionable puzzle error;
- an unavailable-puzzle hint response that advances to a different session index/fingerprint (or no current puzzle) is adopted immediately and preserves its **Puzzle unavailable** feedback rather than showing a false solved state;
- pause aborts owned work before dispatching home;
- unmount aborts prelude, request, animation, reconciliation, and sound work.

Run:

```bash
npm --prefix frontend test -- run src/components/puzzle/PuzzleScreen.test.ts --single-thread
```

Expected: FAIL against immediate session replacement and overlapping booleans.

- [ ] **Step 2: Refactor `PuzzleScreen` around the pure state and effect ports**

Declare the component's testable effect boundary explicitly:

```ts
export type PuzzleEffects = {
  createSound(): SoundService
  delay(milliseconds: number, signal: AbortSignal): Promise<void>
  prefersReducedMotion(): boolean
}

export let effects: PuzzleEffects = browserPuzzleEffects
export let boardAdapterFactory: ChessgroundAdapterFactory = createChessgroundAdapter
```

Forward `boardAdapterFactory` to `ChessBoard.adapterFactory`. Unit tests inject an in-memory adapter and sound backend; they must not instantiate Web Audio or real Chessground geometry in JSDOM.

Replace `shownFen`, `inputDisabled`, and independent status booleans with one `PuzzleState`. Keep a monotonically increasing request sequence and one current `AbortController`. On every displayed-session change and on destroy:

```ts
abortController?.abort()
requestSequence += 1
```

Check `acceptsResponse(state, requestId)` before applying every API result. Derive board `inputEnabled` from `ready`, reconciled `incorrect`, or `failed` with `recoverable: true`; a fatal failed state stays locked. Use the board's explicit `setPosition` method for same-FEN snapback and authoritative frames.

Keep `boardGeneration` alongside state and render the board in `{#key boardGeneration}`. The animation port's `recover(finalFen)` first updates controlled `state.fen`, then increments `boardGeneration`; this destroys the failed adapter and creates a new one with `initialFen: finalFen`. Recovery must not call the failed adapter again. Copy any `AnimationResult.warning` into `PuzzleState.notice` and the live/text feedback area without replacing a solved state or hiding its Next action.

For the prelude, synthesize one display frame `{ uci: preludeUci, resultingFen: displayedFen }`; do not call the backend. Skip it for resumed paths or reduced motion.

- [ ] **Step 3: Implement exact move/reveal ordering**

For a move:

1. save `authoritativeFen = state.fen`;
2. rely on the earlier synchronous capture-phase gesture hook to have begun audio unlock; do not defer first unlock to this Chessground callback;
3. enter `requesting` and clear stale wrong/hint feedback;
4. await `api.playMove`;
5. ignore a stale response;
6. if a non-correct response has changed `currentIndex`, changed fingerprint, or has no current puzzle, adopt and dispatch that returned session immediately with its **Puzzle unavailable** message and no wrong-move treatment;
7. otherwise on incorrect, call `board.setPosition(authoritativeFen, undefined, !reducedMotion)`; wait one abortable 180ms reconciliation interval only when motion is enabled, then adopt/dispatch the returned same-puzzle session, enter `incorrect`, play incorrect, and announce **Try again**;
8. on correct, choose the target as `result.puzzleCompleted ? result.finalFen : result.session.current?.currentFen`; require that target and every applied frame before entering animation, otherwise enter fatal failed state;
9. enter `animating`, run `animateAppliedMoves` with the optimistic submitted UCI, and reconcile to that validated target—never to the returned next puzzle's FEN when `puzzleCompleted` is true;
10. when incomplete, adopt and dispatch the returned session and enter `ready`;
11. when complete, require non-empty `finalFen`, retain `displaySession`, store `pendingSession`, clear destinations, enter `solved`, and dispatch the pending durable session through the event described below.

Reveal uses the same animation path without an optimistic UCI. It enters `solved` with `outcome: 'revealed'`. A backend success missing required authoritative frames is an actionable failed contract, not an inferred position.

A hint response for the same `currentIndex` and fingerprint updates hint state and the returned session's counters/legal moves. If catalogue revalidation changes the session index/fingerprint or removes the current puzzle, adopt that returned session immediately and show its **Puzzle unavailable** message; this is not puzzle completion and has no final-board acknowledgement step.

On a rejected move request, explicitly reconcile to the recorded authoritative FEN using the reduced-motion rule, then call `markFailed(..., recoverable: true)` so the learner can retry. API failures for hint/reveal are likewise recoverable when the current authoritative board is intact. Missing/invalid authoritative frames and malformed legal UCI call `markFailed(..., recoverable: false)`.

Attach capture-phase `pointerdown` and Enter/Space handlers to the puzzle root and call `sound.unlock()` before any asynchronous work. Hint, Reveal, Pause, sound toggle, board keyboard, click, and drag all pass through this boundary. Because Chessground defers its route callback, the later `move` event must not be the first unlock attempt. The Web Audio backend must create/resume its context before its first internal `await` so WebKit associates it with the trusted gesture.

- [ ] **Step 4: Keep parent durable state current without changing the solved presentation**

Extend the dispatcher:

```ts
const dispatch = createEventDispatcher<{
  home: { completed: boolean }
  change: SessionView
  persisted: SessionView
}>()
```

`change` means the child is now visibly rendering that session. `persisted` means the backend has durably advanced to that session but the child is intentionally retaining the solved board.

In `NormalShell`, add `deferredSession`. On `persisted`, store it without replacing `activeSession`. On **Next puzzle**/**See results**, `change` adopts the session and clears `deferredSession`. Route every header-home action through one `goHome()` helper that first promotes `deferredSession`: retain it as the active next session when non-final, or clear `activeSession` when it is completed. This ensures:

- home after non-final success shows **Continue** for the already-persisted next puzzle;
- home after final success does not offer a dead completed session;
- the solved board remains mounted until acknowledgement.

Add App-level tests for both header-home scenarios and explicit Next/See results.

- [ ] **Step 5: Build the board-dominant Classic Green puzzle layout**

Give the normal shell a puzzle-active class. Use a maximum content width around 1240px, a flexible square board plus a 300–340px charcoal side panel, and:

```css
width: min(100%, calc(100dvh - 120px), 760px);
aspect-ratio: 1;
```

for the board wrapper. At `max-width: 820px`, stack board and panel without clipping. Keep non-board controls at least 44px tall. The solved panel replaces ordinary Hint/Reveal/Pause actions with a single prominent Next/See results action. Do not use a fixed `72vh` board.

Add a visible sound toggle, text feedback for all sound/color states, and a polite live region for selection, retry, reveal, completion, pause, and errors. Under `prefers-reduced-motion: reduce`, disable nonessential CSS transitions as well as JS animation delays.

- [ ] **Step 6: Verify and commit**

Run:

```bash
npm --prefix frontend test -- run src/components/puzzle/PuzzleScreen.test.ts src/App.test.ts --single-thread
npm --prefix frontend test -- run --single-thread
npm --prefix frontend run check
npm --prefix frontend run build
git add frontend/src/components/puzzle/PuzzleScreen.svelte frontend/src/components/puzzle/PuzzleScreen.test.ts frontend/src/components/app/NormalShell.svelte frontend/src/App.test.ts frontend/src/styles/tokens.css frontend/src/styles/app.css
git diff --cached --check
git commit -m "feat: keep solved puzzles visible until next"
```

---

## Task 9: Expose Matching Source and Complete Legal Information In-App

**Files:**

- Modify: `internal/buildinfo/buildinfo.go`
- Modify: `internal/buildinfo/buildinfo_test.go`
- Modify: `application.go`
- Modify: `controllers_test.go`
- Regenerate: `frontend/wailsjs/go/main/ModeController.js`
- Regenerate: `frontend/wailsjs/go/main/ModeController.d.ts`
- Regenerate: `frontend/wailsjs/go/models.ts`
- Modify: `frontend/src/lib/api.ts`
- Modify: `frontend/src/lib/api.test.ts`
- Modify: `frontend/src/test-fakes.ts`
- Create: `frontend/src/lib/legal-assets.ts`
- Create: `frontend/src/lib/legal-assets.test.ts`
- Create: `frontend/src/lib/external-links.ts`
- Create: `frontend/src/lib/external-links.test.ts`
- Create: `frontend/src/components/legal/AboutLegal.svelte`
- Create: `frontend/src/components/legal/AboutLegal.test.ts`
- Modify: `frontend/src/lib/navigation.ts`
- Modify: `frontend/src/components/app/NormalShell.svelte`
- Modify: `frontend/src/components/app/RecoveryShell.svelte`
- Modify: `frontend/src/App.svelte`
- Modify: `frontend/src/App.test.ts`
- Modify: `frontend/src/vite-env.d.ts`
- Modify: `frontend/src/styles/app.css`

- [ ] **Step 1: Write failing build identity tests**

Add cases for a full lowercase 40-character commit, `development`, uppercase, abbreviated, and malformed values. Only a full lowercase hash may claim an exact tree URL. Use this contract:

```go
const (
	Name          = "Chess Trainer"
	RepositoryURL = "https://github.com/xang1234/chess"
)

var Commit = "development"

type Info struct {
	Name      string `json:"name"`
	Commit    string `json:"commit"`
	SourceURL string `json:"sourceUrl"`
}
```

For a valid commit, `SourceURL` must be `RepositoryURL + "/tree/" + Commit`. Invalid values normalize to commit `development` and repository root, never a falsely precise URL.

Add a controller test proving `ModeController.GetBuildInfo()` is available in both normal and recovery runtimes while the recovery controller still exposes no training capabilities.

Run:

```bash
go test ./internal/buildinfo -count=1
go test . -run 'TestModeController|Test.*RuntimeBinds' -count=1
```

Expected: FAIL on the missing identity API.

- [ ] **Step 2: Implement and bind build identity**

Use an anchored lowercase SHA regexp in `buildinfo.Current()`. Add:

```go
func (*ModeController) GetBuildInfo() buildinfo.Info {
	return buildinfo.Current()
}
```

Do not add a third controller binding. This common controller is deliberately safe in recovery mode.

Regenerate:

```bash
/Users/admin/go/bin/wails generate module -tags bindings
```

Inspect generated changes and retain only expected build-info model/method updates.

- [ ] **Step 3: Write failing frontend bootstrap and legal-loader tests**

Add:

```ts
export type BuildInfo = {
  name: string
  commit: string
  sourceUrl: string
}
```

Require both `ApplicationAPI` alternatives to carry `buildInfo`. Test that production bootstrap loads application mode and build info together and preserves exact fields. Browser preview uses:

```ts
{
  name: 'Chess Trainer',
  commit: 'development',
  sourceUrl: 'https://github.com/xang1234/chess'
}
```

For `legal-assets.ts`, inject `fetch` and require exactly these bundled paths:

```ts
const legalPaths = {
  application: '/legal/LICENSE.txt',
  notices: '/legal/THIRD_PARTY_NOTICES.md',
  chessground: '/legal/CHESSGROUND_LICENSE.txt',
  nunito: '/legal/NUNITO_OFL.txt'
} as const
```

All four must load; a non-OK response must identify the missing asset. No remote URL may be fetched.

For `external-links.ts`, test Wails `BrowserOpenURL` and browser `window.open(url, '_blank', 'noopener,noreferrer')` fallback. Reject any URL whose origin is not `https://github.com` so the legal view cannot become a general arbitrary-link bridge.

Run:

```bash
npm --prefix frontend test -- run src/lib/api.test.ts src/lib/legal-assets.test.ts src/lib/external-links.test.ts --single-thread
```

Expected: FAIL on missing build info/loaders.

- [ ] **Step 4: Bootstrap both application modes with build info**

Import generated `GetBuildInfo` beside `GetApplicationMode` and resolve both via `Promise.all`. Pass `buildInfo` from `App.svelte` into `NormalShell` and `RecoveryShell`. Update all fake application constructors and Playwright Wails mocks. Extend `vite-env.d.ts` with the narrow runtime shape used for `BrowserOpenURL`; do not declare the runtime as `any`.

- [ ] **Step 5: Write and implement the About & Legal component**

Before any asynchronous document load, render:

- `Copyright © 2026 David Ten and Chess Trainer contributors`;
- GPL-3.0-or-later modification and redistribution rights;
- an uppercase `WITHOUT ANY WARRANTY` statement;
- `@lichess-org/chessground 10.1.1` and its upstream attribution;
- Nunito's copyright and SIL Open Font License 1.1 attribution;
- the repository path to the committed Chessground preferred-source archive;
- the complete build commit;
- the exact source URL.

Inject `loadDocuments` and `openExternal` props. Render application license, third-party notices, Chessground license, and Nunito OFL text in four labelled `<details><pre>` regions. The source control invokes only the validated exact `buildInfo.sourceUrl`. A loader failure keeps the always-available rights/source text visible and reports the local asset error.

Component tests must assert all exact identity/rights text, successful local document display, loader failure behavior, external-link invocation, and Back.

- [ ] **Step 6: Make About & Legal available in normal and recovery shells**

Add `'legal'` to the normal `Screen` union and a small **About & Legal** header button. Back returns home. In recovery mode, use local `showLegal` state to replace `RecoveryPanel` temporarily; Back returns to recovery. Do not expose `NormalAPI` from recovery mode.

Add App tests for opening, source link, and returning in both modes, including when recovery database startup failed.

- [ ] **Step 7: Verify and commit**

Run:

```bash
go test ./internal/buildinfo -count=1
go test . -run 'TestModeController|Test.*RuntimeBinds' -count=1
go test ./... -count=1
npm --prefix frontend test -- run src/lib/api.test.ts src/lib/legal-assets.test.ts src/lib/external-links.test.ts src/components/legal/AboutLegal.test.ts src/App.test.ts --single-thread
npm --prefix frontend run check
npm --prefix frontend run build
npm --prefix frontend run verify:licenses
git add internal/buildinfo application.go controllers_test.go frontend/wailsjs/go/main/ModeController.js frontend/wailsjs/go/main/ModeController.d.ts frontend/wailsjs/go/models.ts frontend/src/lib/api.ts frontend/src/lib/api.test.ts frontend/src/test-fakes.ts frontend/src/lib/legal-assets.ts frontend/src/lib/legal-assets.test.ts frontend/src/lib/external-links.ts frontend/src/lib/external-links.test.ts frontend/src/components/legal frontend/src/lib/navigation.ts frontend/src/components/app/NormalShell.svelte frontend/src/components/app/RecoveryShell.svelte frontend/src/App.svelte frontend/src/App.test.ts frontend/src/vite-env.d.ts frontend/src/styles/app.css
git diff --cached --check
git commit -m "feat: show license and matching public source"
```

---

## Task 10: Enforce Reproducible Public Release Compliance

**Files:**

- Create: `scripts/verify-release.mjs`
- Create: `scripts/verify-release.test.mjs`
- Create: `scripts/build-corresponding-source.mjs`
- Create: `scripts/build-corresponding-source.test.mjs`
- Create: `scripts/build-release.sh`
- Create: `scripts/build-release.test.mjs`
- Modify: `README.md`
- Modify: `docs/operations/local-build.md`
- Create: `docs/operations/release.md`

- [ ] **Step 1: Write failing pure release-verifier tests**

Factor command execution and filesystem reads behind injected ports. Cover at minimum:

- reject `^10.1.1`, a missing legal asset, a dirty tree, an untracked required file, and an executable lacking the embedded commit;
- reject a missing/changed `third_party/source/chessground-v10.1.1.tar.gz`, a truncated GPL text, a missing Nunito OFL asset, and installed Chessground source/assets that differ from the committed source archive;
- reject a changed Darwin runtime closure/notice lock, wrong Go release toolchain or changed Go `LICENSE`/`PATENTS`, an absolute Go module replacement, and a corresponding-source archive missing a production Go vendor module, license/NOTICE, Go legal files, complete Wails source, Svelte source, Chessground source, build instructions, or tracked app file;
- accept a lightweight remote tag that resolves to HEAD;
- accept an annotated tag only when its peeled `^{}` record resolves to HEAD;
- reject a tag resolving to another commit;
- reject the release when an authenticated `origin` would resolve but the same fixed GitHub HTTPS tag is not reachable through the verifier's credential-free runner;
- reject a short commit and an app whose bundled legal bytes differ;
- reject a tag outside `^v[0-9]+\.[0-9]+\.[0-9]+$`, including slashes or path traversal;
- reject failed `codesign --verify --deep --strict`.

Run:

```bash
node --test scripts/verify-release.test.mjs
node --test scripts/build-corresponding-source.test.mjs
node --test scripts/build-release.test.mjs
```

Expected: FAIL because the verifier does not exist.

- [ ] **Step 2: Implement strict pre- and post-build verification**

The CLI accepts `--phase pre|post --tag <tag>` and, for post, `--app <path>`. Both phases require:

1. `git status --porcelain=v1 --untracked-files=all` is empty;
2. the requested tag matches `^v[0-9]+\.[0-9]+\.[0-9]+$` and HEAD is a full 40-character lowercase SHA;
3. HEAD has that requested exact local tag;
4. a credential-free query of the fixed URL `https://github.com/xang1234/chess.git` proves the public lightweight or peeled annotated tag resolves to HEAD;
5. `LICENSE`, generated runtime notices/lock, exact Go 1.26.4 `LICENSE`/`PATENTS`, dependency lockfiles, all four public legal assets, Chessground/Svelte preferred-source archives, source builder, build script, and verifiers are tracked;
6. there is no active or commented local/absolute Go module replacement; `go env GOVERSION` is exactly `go1.26.4`; `go mod verify` succeeds; the dependency/lockfile integrity is exact; the clean installed frontend inputs match their source records; the GPL text has the canonical complete digest; the Go legal bytes and Darwin production closure match `third_party/runtime-dependencies.lock.json`; and `verify-legal-assets.mjs` passes;
7. `CHESS_TRAINER_RELEASE_ROOT`, `GOMODCACHE`, `GOCACHE`, and `npm_config_cache` are set, each cache resolves beneath the release root, `GOWORK=off`, and `GOTOOLCHAIN=local`. This proves all Go verification/build/source commands are using the fresh release-scoped caches created by the wrapper rather than a mutable shared module extraction.

Post additionally requires:

8. all four `frontend/dist/legal/*` documents byte-match the committed source legal files;
9. the macOS executable contains the complete GPL, notices, Chessground license, Nunito OFL, and the exact Go `LICENSE`/`PATENTS` texts embedded through the notices;
10. the executable contains the exact linker-injected commit;
11. `codesign --verify --deep --strict` succeeds;
12. the required `--source` archive contains every tracked app source file, a production Go vendor tree and copied module license/NOTICE files matching the locked Darwin closure, exact Go 1.26.4 `LICENSE`/`PATENTS`, full `github.com/wailsapp/wails/v2@v2.12.0` source including `cmd/wails`, the committed Chessground and Svelte preferred-source archives, exact build instructions, and a manifest naming this tag/commit/toolchain.

Return a non-zero exit with a single actionable cause for every failure. Never fall back to a local-only source URL.

For the public check, never query configurable `origin`. Spawn:

```text
git -c credential.helper= -c http.extraHeader= ls-remote --tags \
  https://github.com/xang1234/chess.git \
  refs/tags/<tag> refs/tags/<tag>^{}
```

with a sanitized environment containing only the required `PATH` plus `GIT_CONFIG_NOSYSTEM=1`, `GIT_CONFIG_GLOBAL=/dev/null`, an empty temporary `HOME`, `GIT_TERMINAL_PROMPT=0`, and `GIT_ASKPASS=/usr/bin/false`. Do not pass inherited GitHub tokens, credential headers, SSH configuration, or credential helpers. A private repository must fail this check even if the developer's normal `origin` succeeds.

- [ ] **Step 3: Build and test a complete corresponding-source artifact**

`scripts/build-corresponding-source.mjs` accepts `--tag`, `--commit`, and `--output`. It must work in a temporary directory and never dirty the checkout:

1. extract `git archive <commit>` under `app/`;
2. write `app/vendor/` with `go mod vendor -o` from that extracted app;
3. read both Darwin closures from the committed runtime lock and copy every module-root `LICENSE*`, `COPYING*`, and `NOTICE*` file into `app/vendor/_licenses/<escaped-module>@<version>/`, even when `go mod vendor` also retained it;
4. copy committed `third_party/legal/go1.26.4/LICENSE` and `PATENTS` into `toolchain/go1.26.4/` and verify their lockfile digests;
5. use `go mod download -json github.com/wailsapp/wails/v2@v2.12.0` and copy its complete module directory, including `cmd/wails`, under `build-tools/wails-v2.12.0/`;
6. retain the committed Chessground and Svelte tag archives under `app/third_party/source/`;
7. generate `BUILDING.md`, `TRACKED_FILES.sha256`, and `SOURCE_MANIFEST.json` containing tag, commit, exact `go1.26.4` release toolchain and Go legal-file digests, Go vendor/module digests, Wails tree digest, frontend source-archive digests, and runtime notice-lock digest;
8. create `build/release/Chess-Trainer-<tag>-corresponding-source.tar.gz`.

`BUILDING.md` requires Go 1.26.4, explains how to build the Wails CLI from the included Wails source, then build the extracted app with the included vendor tree and locked frontend install. The source verifier must compare every tracked-file digest, not just a short required-file list.

Unit tests inject git/go/filesystem runners and prove missing app files, missing vendor modules/licenses, missing or altered Go legal files/toolchain identity, altered manifests, absent Wails CLI source, wrong frontend archives, and an absolute replace all fail; a complete fixture passes. The builder must reject release execution unless its Go/npm caches resolve beneath `CHESS_TRAINER_RELEASE_ROOT`.

- [ ] **Step 4: Add the only supported distributable build wrapper**

Implement `scripts/build-release.sh` with `set -euo pipefail`:

```bash
release_tmp=$(mktemp -d "${TMPDIR:-/tmp}/chess-trainer-release.XXXXXX")
trap 'rm -rf "$release_tmp"' EXIT

export CHESS_TRAINER_RELEASE_ROOT="$release_tmp"
export GOMODCACHE="$release_tmp/go-mod-cache"
export GOCACHE="$release_tmp/go-build-cache"
export npm_config_cache="$release_tmp/npm-cache"
export GOWORK=off
export GOTOOLCHAIN=local
mkdir -p "$GOMODCACHE" "$GOCACHE" "$npm_config_cache"

tag=${1:?usage: scripts/build-release.sh <public-tag>}
commit=$(git rev-parse HEAD)

test "$(go env GOVERSION)" = go1.26.4

npm --prefix frontend ci
go mod download all
go mod verify
npm --prefix frontend run verify:licenses
node scripts/verify-release.mjs --phase pre --tag "$tag"

go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 build \
  -clean \
  -trimpath \
  -ldflags "-X chess-trainer/internal/buildinfo.Commit=${commit}"

source_archive="build/release/Chess-Trainer-${tag}-corresponding-source.tar.gz"
node scripts/build-corresponding-source.mjs \
  --tag "$tag" \
  --commit "$commit" \
  --output "$source_archive"

node scripts/verify-release.mjs \
  --phase post \
  --tag "$tag" \
  --app "build/bin/Chess Trainer.app" \
  --source "$source_archive"
```

`scripts/build-release.test.mjs` runs the wrapper with command shims and temporary fixture state. It proves every npm/go/build/source subprocess sees unique release-scoped caches, `GOWORK=off`, and `GOTOOLCHAIN=local`; the caches begin empty; a pre-seeded shared `GOMODCACHE` is ignored; and the trap removes `CHESS_TRAINER_RELEASE_ROOT` on both success and an injected failure.

`npm ci` and `go mod download all` are mandatory before preflight and run inside the newly created caches, so a modified extraction in a shared `$GOMODCACHE` can never select release inputs. The legal verifier compares the clean install, exact Go toolchain/legal files, and runtime closure with committed locks/source archives. If Wails rewrites tracked generated files, the post-build clean-tree check must stop the release so they can be reviewed, committed, retagged, and pushed. Do not automatically commit generated changes inside the release script.

- [ ] **Step 5: Document GPL rights, local builds, and exact release procedure**

Replace the Wails template README with the actual product purpose, offline architecture, local development/build commands, public source link, GPL-3.0-or-later modification/redistribution rights, uppercase no-warranty statement, third-party notice link, Nunito OFL link, and Chessground preferred-source archive link.

Update local build documentation for the exact dependency install, frontend verification, Go tests, and Wails build. Document the module's Go 1.25 language level separately from the mandatory Go 1.26.4 distributable-build toolchain, the fresh-cache wrapper, and the Go license/patent copies. In `docs/operations/release.md`, require:

1. complete verification;
2. commit all source/assets/scripts/lockfile changes;
3. create and push the matching tag;
4. run `scripts/build-release.sh <tag>`;
5. archive the `.app` with `ditto`;
6. attach the binary archive to the matching GitHub release/tag;
7. attach the generated `Chess-Trainer-<tag>-corresponding-source.tar.gz` beside the binary and verify it is downloadable without authentication;
8. retain GitHub's tag-matched source ZIP/tarball beside it;
9. optionally attach the individual committed Chessground/Svelte archives too; they are already present inside the required corresponding-source artifact.

- [ ] **Step 6: Verify and commit without attempting an unpublished release**

Run:

```bash
node --test scripts/verify-release.test.mjs
node --test scripts/verify-legal-assets.test.mjs
node --test scripts/generate-third-party-notices.test.mjs
node --test scripts/build-corresponding-source.test.mjs
node --test scripts/build-release.test.mjs
bash -n scripts/build-release.sh
npm --prefix frontend run verify:licenses
npm --prefix frontend run build
go test ./... -count=1
git add scripts/verify-release.mjs scripts/verify-release.test.mjs scripts/build-corresponding-source.mjs scripts/build-corresponding-source.test.mjs scripts/build-release.sh scripts/build-release.test.mjs README.md docs/operations/local-build.md docs/operations/release.md
git diff --cached --check
git commit -m "build: verify GPL release source identity"
```

Do not run `scripts/build-release.sh` until the implementation commit is pushed and deliberately tagged; preflight rejection before then is correct behavior.

---

## Task 11: Prove Real Pointer, Touch, Keyboard, Completion, and Layout Behavior

**Files:**

- Create: `frontend/tests/board-driver.ts`
- Create: `frontend/tests/board-interactions.spec.ts`
- Create: `docs/operations/board-acceptance.md`
- Modify: `frontend/tests/trainer.spec.ts`
- Modify: `frontend/playwright.config.ts`
- Modify: `frontend/src/test-setup.ts` only if deterministic DOM/browser polyfills are required by component tests

- [ ] **Step 1: Create a board-coordinate driver before changing end-to-end assertions**

Implement:

```ts
export async function squarePoint(
  board: Locator,
  square: string,
  orientation: 'white' | 'black'
): Promise<{ x: number; y: number }>
```

For white orientation, file `a` is column 0 and rank 8 is row 0; for black, reverse both axes. Add helpers for trusted mouse click-to-move, trusted mouse drag/cancel, trusted touchscreen tap-to-move, and marker lookup. Inspect Chessground marker keys using the element's `cgKey`/`data-key` assigned under `jsHover`, not fragile DOM order.

- [ ] **Step 2: Update the browser fake backend to the exact new contract**

Add `ModeController.GetBuildInfo`. Every puzzle contains all legal moves, for example:

```ts
legalMoves: ['e2e3', 'e2e4']
```

Every correct response carries frames and a final FEN when completed:

```ts
{
  session: nextSession,
  correct: true,
  puzzleCompleted: true,
  appliedMoves: [
    { uci: 'e2e4', resultingFen: finalFen }
  ],
  finalFen
}
```

Change the existing trainer flow: after `e2e4`, assert **Correct!**, the final position, and **Next puzzle** while the panel still says puzzle 1. Click **Next puzzle** before expecting puzzle 2. After final reveal, assert **Solution shown**, click **See results**, and only then assert the summary.

- [ ] **Step 3: Write real-browser interaction tests**

Cover, with the actual Chessground dependency rather than a mock:

- trusted click-to-move and drag-to-move;
- trusted touchscreen tap-to-move in WebKit touch mode;
- trusted mouse/pointer drag, cancellation, and wrong-move snapback;
- white and black orientation coordinate mapping;
- strong selected marker;
- legal empty destination dot and occupied destination capture ring;
- an immobile same-color piece does not remain selected;
- promotion chooser and complete UCI suffix;
- Arrow keys, Enter/Space, Escape, focus outline, and live announcements;
- accepted automatic reply order;
- reveal order and **Solution shown**;
- solved final board retained before explicit Next;
- final **See results** transition;
- persistent mute across reload;
- reduced motion skips prelude/reply delay but reaches the same authoritative FEN;
- desktop board/panel side-by-side geometry and narrow stacked geometry with no clipped controls.

The marker/capture fixture must use a position with at least one quiet and one occupied legal destination so visual semantics are independently asserted.

- [ ] **Step 4: Add a WebKit touch project without dropping desktop WebKit**

Retain Chromium and Desktop Safari for the full suite. Add an iPad/WebKit project scoped to `board-interactions.spec.ts`, for example:

```ts
{
  name: 'webkit-touch',
  testMatch: /board-interactions\.spec\.ts/,
  use: { ...devices['iPad Mini'] }
}
```

Do not replace desktop WebKit: Wails on macOS remains a desktop WebKit target.

Playwright's WebKit touchscreen API supports trusted taps but not a trusted touchmove drag stream. Do not claim a synthetic `TouchEvent` proves drag behavior, and do not enable Chessground's trusted-event bypass.

- [ ] **Step 5: Run each browser target and fix behavior, not assertions**

Run:

```bash
npm --prefix frontend run test:e2e -- --project=chromium
npm --prefix frontend run test:e2e -- --project=webkit
npm --prefix frontend run test:e2e -- --project=webkit-touch
```

Expected: all pass with real trusted input. Do not enable `trustAllEvents` to make tests pass.

- [ ] **Step 6: Perform and record manual trusted touch-drag acceptance**

Make the browser preview API from Task 4 expose a deterministic two-puzzle session so it is usable without Wails. In `docs/operations/board-acceptance.md`, document how to run Vite temporarily on the Mac's tailnet interface, open it from a touch-capable Safari/Chromium device over Tailscale, and verify:

1. tap-to-move;
2. press-drag-release to a legal destination;
3. drag cancellation outside the board;
4. legal-but-wrong snapback;
5. dragging on the board does not scroll the page, while ordinary scrolling that begins outside the board still works;
6. explicit solved/Next behavior.

Record device/browser/date and pass/fail in the implementation handoff. This is the trusted touch-drag evidence that Playwright WebKit cannot synthesize. Stop the temporary dev server afterward; it is not a hosting requirement.

- [ ] **Step 7: Run the complete production verification matrix**

Run:

```bash
go test ./... -count=1
go test -race ./... -count=1
go mod verify
node frontend/scripts/generate-sounds.mjs --check
node scripts/generate-third-party-notices.mjs --check
npm --prefix frontend run verify:licenses
node --test scripts/verify-legal-assets.test.mjs
node --test scripts/generate-third-party-notices.test.mjs
npm --prefix frontend test -- run --single-thread
npm --prefix frontend run check
npm --prefix frontend run build
npm --prefix frontend run test:e2e
node --test scripts/verify-release.test.mjs
node --test scripts/build-corresponding-source.test.mjs
node --test scripts/build-release.test.mjs
bash -n scripts/build-release.sh
go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 build -clean -trimpath
git diff --check
git status --short
```

Expected:

- all Go, race, frontend, browser, legal, sound, and release-fixture tests pass;
- Wails produces `build/bin/Chess Trainer.app`;
- the ordinary local build contains the bundled legal assets and reports `development` source identity;
- the installed Chessground/Svelte inputs match committed preferred-source archives and the complete locked Go-runtime/standard-library/module, Svelte, Wails, Chessground, and Nunito notices are bundled;
- release fixture coverage proves all Go/npm inputs use a disposable release-scoped cache and that cleanup runs after both success and failure;
- the only unstaged pre-existing paths remain `app_test.go` and `normal_controller.go`;
- no generated Wails drift is left after the build.

If Wails regenerates tracked bindings, review and commit only the expected generated changes, rerun the full matrix, and do not hide drift with cleanup commands.

- [ ] **Step 8: Thermonuclear review and focused remediation**

Invoke `thermo-nuclear-code-quality-review` against the complete diff from the pre-feature commit. Review especially:

- whether `PuzzleScreen.svelte` has accumulated API, animation, state, audio, and rendering responsibilities that belong in the pure modules;
- whether Chessground-specific types leak into puzzle/domain code;
- whether branch-heavy state conditions duplicate the discriminated state transitions;
- whether legal/source release checks have bypasses;
- whether tests rely on timing or internal DOM order rather than ports and observable behavior.

Address every actionable P1/P2 correctness or maintainability finding with a failing regression test first, rerun the complete verification matrix, and keep any unrelated picker edits unstaged.

- [ ] **Step 9: Commit browser coverage and final integration fixes**

Run:

```bash
git add frontend/tests/board-driver.ts frontend/tests/board-interactions.spec.ts frontend/tests/trainer.spec.ts frontend/playwright.config.ts frontend/src/test-setup.ts docs/operations/board-acceptance.md
git add --update frontend internal application.go controllers_test.go README.md docs scripts LICENSE THIRD_PARTY_NOTICES.md
git diff --cached --name-only
git diff --cached --check
git commit -m "test: verify polished puzzle board experience"
```

Before committing, remove `frontend/src/test-setup.ts` from the command if it did not actually change. Confirm `app_test.go` and `normal_controller.go` were never included in the staged pathspec and remain untouched in the working tree.

## Completion Evidence

The implementation is complete only when the handoff includes:

- the exact passing Go, frontend, Playwright Chromium, Playwright WebKit, WebKit touch, Wails build, legal verifier, sound reproducibility, and release-verifier outputs;
- the recorded trusted touch-drag device/browser/date acceptance result;
- the absolute path to `build/bin/Chess Trainer.app` for local testing;
- the final commit list and `git status --short` showing the preserved picker edits separately;
- confirmation that no engine or runtime network dependency was added;
- confirmation that the exact Chessground/Svelte preferred sources, production Go vendor/Wails sources, exact Go 1.26.4 legal files, and complete runtime license notices accompany every public binary through the generated corresponding-source artifact;
- confirmation that an unpublished local build identifies itself as `development`, while the documented release path requires a matching public commit/tag.
