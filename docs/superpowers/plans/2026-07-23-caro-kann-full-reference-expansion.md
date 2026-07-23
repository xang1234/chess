# Caro-Kann Full-Reference Expansion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deepen the private `mco15-caro-kann-black` course from a v1 seed pack into a validated v2 full-reference Caro-Kann pack based on MCO-15 printed pages 171-196.

**Architecture:** Keep the app runtime generic and reuse the existing schema-v2 course-pack compiler, importer, catalogue, lessons, variation explorer, and journey rebase behavior. Add one generic authoring helper to `cmd/coursepack` for SAN-to-UCI line conversion, then perform page-by-page private course authoring outside Git. Commit only generic tooling/docs/tests; never commit MCO-derived private course data.

**Tech Stack:** Go 1.26.4, `github.com/corentings/chess/v2`, JSON `.ctcourse`, private authoring JSON, Poppler `pdftoppm`, `jq`, Wails 2.12.0, Svelte/Vitest/Playwright for final verification only if app code changes or final rebuild is required.

## Global Constraints

- Course ID remains exactly `mco15-caro-kann-black`.
- `contentVersion` must become exactly `2.0.0`.
- Course perspective remains exactly `black`.
- Preserve existing lesson, prompt, position, move, activity, and edge IDs where their meaning remains the same.
- White moves are `opponent`; Black course moves are `repertoire` or `alternative`.
- Decision prompts occur only where Black is to move.
- Printed-page source scope is Caro-Kann Defence pages 171-196, corresponding to PDF pages 188-213 with the current scan offset.
- Keep Quick and Standard concise; put dense source detail into Reference.
- Do not deepen Ruy Lopez in this pass.
- Do not add new openings in this pass.
- Do not special-case Caro-Kann in public app code.
- Do not commit private `.ctcourse` files, private authoring inventories, private checkpoint files, or MCO-derived prose to Git.
- Public reports and commits may contain only counts, file paths, validation results, and high-level summaries.

---

## File Map

Repository files:

- Modify `cmd/coursepack/main.go`: add a generic `sanline` command for private authoring work.
- Modify `cmd/coursepack/main_test.go`: cover `sanline` output and usage errors.
- Create or update no public course data except synthetic tests if a generic app defect is found.
- Create this plan only under `docs/superpowers/plans/`.

Private files outside Git:

- Read/update `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.authoring.json`.
- Read/update `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse`.
- Create `/Users/admin/Documents/Private Chess Courses/checkpoints/caro-kann-black-v1-before-full-reference/`.
- Create `/Users/admin/Documents/Private Chess Courses/checkpoints/caro-kann-black-v2-pages-171-176/`.
- Create `/Users/admin/Documents/Private Chess Courses/checkpoints/caro-kann-black-v2-pages-177-183/`.
- Create `/Users/admin/Documents/Private Chess Courses/checkpoints/caro-kann-black-v2-pages-184-190/`.
- Create `/Users/admin/Documents/Private Chess Courses/checkpoints/caro-kann-black-v2-full-reference/`.
- Create `/private/tmp/mco15-caro-kann-pages/` rendered PNG source pages.
- Create `/private/tmp/caro-kann-validation.json` validation output.

---

### Task 1: Prepare isolated workspace, render pages, and checkpoint v1 seed pack

**Files:**
- Read: `/Users/admin/Downloads/kupdf.net_modern-chess-openings-15th-edition.pdf`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.authoring.json`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse`
- Create: `/private/tmp/mco15-caro-kann-pages/*.png`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/caro-kann-black-v1-before-full-reference/`

**Interfaces:**
- Consumes: current validated v1 Caro-Kann seed pack.
- Produces: recoverable v1 checkpoint and rendered source-page images for visual authoring.

- [ ] **Step 1: Start from a clean branch or worktree**

Run:

```bash
git status --short --branch
git log --oneline --decorate -5
```

Expected: clean checkout on the intended implementation branch. If using a worktree, create it with `superpowers:using-git-worktrees` before editing.

- [ ] **Step 2: Validate the current private v1 pack**

Run:

```bash
go run ./cmd/coursepack validate "/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse" \
  > /private/tmp/caro-kann-v1-before-full-reference-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/caro-kann-v1-before-full-reference-validation.json
```

Expected: `courseId` is `mco15-caro-kann-black`, `contentVersion` is `1.0.0`, and both `missing` and `unexpected` are `0`.

- [ ] **Step 3: Create the v1 private checkpoint**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/caro-kann-black-v1-before-full-reference"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.authoring.json" \
  "$CHECKPOINT/mco15-caro-kann-black.authoring.json"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse" \
  "$CHECKPOINT/mco15-caro-kann-black.ctcourse"
cp -p /private/tmp/caro-kann-v1-before-full-reference-validation.json \
  "$CHECKPOINT/validation.json"
shasum -a 256 "$CHECKPOINT/"* > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
```

Expected: every checksum line prints `OK`.

- [ ] **Step 4: Render source PDF pages 188-213**

Run:

```bash
mkdir -p /private/tmp/mco15-caro-kann-pages
/Users/admin/.cache/codex-runtimes/codex-primary-runtime/dependencies/bin/override/pdftoppm \
  -png -f 188 -l 213 -r 180 \
  "/Users/admin/Downloads/kupdf.net_modern-chess-openings-15th-edition.pdf" \
  /private/tmp/mco15-caro-kann-pages/caro
find /private/tmp/mco15-caro-kann-pages -maxdepth 1 -type f -name 'caro-*.png' | sort | wc -l
```

Expected: `26` PNG files.

- [ ] **Step 5: Visually verify first and last rendered pages**

Inspect:

```bash
open /private/tmp/mco15-caro-kann-pages/caro-188.png
open /private/tmp/mco15-caro-kann-pages/caro-213.png
```

Expected: `caro-188.png` shows printed page `171`, and `caro-213.png` shows printed page `196`. Record this offset in the Task 1 report without quoting source prose.

- [ ] **Step 6: Verify private files remain untracked**

Run:

```bash
git status --short
git ls-files '*.ctcourse'
```

Expected: no private files under `/Users/admin/Documents/Private Chess Courses/` are tracked. Tracked `.ctcourse` files are only synthetic fixtures under `internal/openings/testdata/`.

---

### Task 2: Add generic `coursepack sanline` authoring helper

**Files:**
- Modify: `cmd/coursepack/main.go`
- Modify: `cmd/coursepack/main_test.go`

**Interfaces:**
- Consumes: a start FEN and a sequence of SAN moves entered from private source review.
- Produces: deterministic JSON with each SAN move, derived UCI, resulting FEN, and final FEN.

- [ ] **Step 1: Add failing tests for `sanline`**

Append to `cmd/coursepack/main_test.go`:

```go
func TestRunSANLineConvertsSANMovesFromFEN(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{
		"sanline",
		"--fen", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		"e4", "c6", "d4", "d5",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run() code = %d stderr = %s", code, stderr.String())
	}
	var result struct {
		StartFEN string `json:"startFen"`
		Moves []struct {
			Index int    `json:"index"`
			SAN   string `json:"san"`
			UCI   string `json:"uci"`
			FEN   string `json:"fen"`
		} `json:"moves"`
		FinalFEN string `json:"finalFen"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode stdout %q: %v", stdout.String(), err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	gotUCI := []string{}
	for _, move := range result.Moves {
		gotUCI = append(gotUCI, move.UCI)
		if move.Index == 0 || strings.TrimSpace(move.FEN) == "" {
			t.Fatalf("move metadata not populated: %#v", move)
		}
	}
	wantUCI := []string{"e2e4", "c7c6", "d2d4", "d7d5"}
	if !reflect.DeepEqual(gotUCI, wantUCI) {
		t.Fatalf("uci line = %#v, want %#v", gotUCI, wantUCI)
	}
	if result.StartFEN == "" || result.FinalFEN == "" {
		t.Fatalf("missing FENs: %#v", result)
	}
}

func TestRunSANLineReportsIllegalSAN(t *testing.T) {
	var stderr bytes.Buffer
	code := run([]string{
		"sanline",
		"--fen", "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		"e5",
	}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("run() code = %d stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "move 1") || !strings.Contains(stderr.String(), "e5") {
		t.Fatalf("stderr = %q, want indexed SAN error", stderr.String())
	}
}
```

- [ ] **Step 2: Run tests and verify RED**

Run:

```bash
go test ./cmd/coursepack -run 'TestRunSANLine' -count=1
```

Expected: FAIL because `sanline` is unsupported.

- [ ] **Step 3: Implement `sanline` output types and argument parsing**

In `cmd/coursepack/main.go`, add imports:

```go
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"chess-trainer/internal/chessrules"
	"chess-trainer/internal/openings"

	"github.com/corentings/chess/v2"
)
```

Add these types below `validationFailure`:

```go
type sanlineOutput struct {
	StartFEN string        `json:"startFen"`
	Moves    []sanlineMove `json:"moves"`
	FinalFEN string        `json:"finalFen"`
}

type sanlineMove struct {
	Index int    `json:"index"`
	SAN   string `json:"san"`
	UCI   string `json:"uci"`
	FEN   string `json:"fen"`
}
```

Replace `run` with command dispatch:

```go
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = io.WriteString(stderr, "usage: coursepack validate <path>\n       coursepack sanline --fen <fen> <san>...\n")
		return 2
	}
	switch args[0] {
	case "validate":
		return runValidate(args[1:], stdout, stderr)
	case "sanline":
		return runSANLine(args[1:], stdout, stderr)
	default:
		_, _ = io.WriteString(stderr, "usage: coursepack validate <path>\n       coursepack sanline --fen <fen> <san>...\n")
		return 2
	}
}
```

Move the existing validation body into:

```go
func runValidate(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		_, _ = io.WriteString(stderr, "usage: coursepack validate <path>\n")
		return 2
	}
	compiled, err := openings.ValidateCoursePackFile(
		context.Background(),
		args[0],
		chessrules.Rules{},
	)
	if err != nil {
		var validationErr *openings.ValidationError
		if errors.As(err, &validationErr) {
			if encodeErr := writeIndentedJSON(stderr, validationFailure{
				Diagnostics: validationErr.Diagnostics,
			}); encodeErr != nil {
				_, _ = fmt.Fprintf(stderr, "write diagnostics: %v\n", encodeErr)
			}
		} else {
			_, _ = fmt.Fprintln(stderr, err)
		}
		return 1
	}
	if err := writeIndentedJSON(stdout, validationOutput{
		CourseID:       compiled.Pack.CourseID,
		ContentVersion: compiled.Pack.ContentVersion,
		Counts:         openings.StructuralCounts(compiled),
		Coverage:       compiled.Coverage,
	}); err != nil {
		_, _ = fmt.Fprintf(stderr, "write validation result: %v\n", err)
		return 1
	}
	return 0
}
```

- [ ] **Step 4: Implement SAN conversion**

Add to `cmd/coursepack/main.go`:

```go
func runSANLine(args []string, stdout, stderr io.Writer) int {
	if len(args) < 3 || args[0] != "--fen" || strings.TrimSpace(args[1]) == "" {
		_, _ = io.WriteString(stderr, "usage: coursepack sanline --fen <fen> <san>...\n")
		return 2
	}
	startFEN := args[1]
	sanMoves := args[2:]
	fenOption, err := chess.FEN(startFEN)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "invalid FEN: %v\n", err)
		return 1
	}
	game := chess.NewGame(fenOption)
	uciNotation := chess.UCINotation{}
	algebraic := chess.AlgebraicNotation{}
	result := sanlineOutput{StartFEN: startFEN, Moves: make([]sanlineMove, 0, len(sanMoves))}
	for index, san := range sanMoves {
		san = strings.TrimSpace(san)
		if san == "" {
			_, _ = fmt.Fprintf(stderr, "move %d is empty\n", index+1)
			return 1
		}
		move, err := algebraic.Decode(game.Position(), san)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "move %d %q: %v\n", index+1, san, err)
			return 1
		}
		uci := uciNotation.Encode(game.Position(), move)
		if err := game.PushMove(move, nil); err != nil {
			_, _ = fmt.Fprintf(stderr, "move %d %q: %v\n", index+1, san, err)
			return 1
		}
		result.Moves = append(result.Moves, sanlineMove{
			Index: index + 1,
			SAN:   san,
			UCI:   uci,
			FEN:   game.Position().String(),
		})
	}
	result.FinalFEN = game.Position().String()
	if err := writeIndentedJSON(stdout, result); err != nil {
		_, _ = fmt.Fprintf(stderr, "write sanline result: %v\n", err)
		return 1
	}
	return 0
}
```

- [ ] **Step 5: Run focused tests and format**

Run:

```bash
gofmt -w cmd/coursepack/main.go cmd/coursepack/main_test.go
go test ./cmd/coursepack -run 'TestRunSANLine|TestRunValidate|TestRunRejectsUnsupportedCommand' -count=1
```

Expected: PASS. If `TestRunRejectsUnsupportedCommand` expects the old single-line usage, update it to expect both `validate` and `sanline` usage lines.

- [ ] **Step 6: Commit generic helper**

Run:

```bash
git add cmd/coursepack/main.go cmd/coursepack/main_test.go
git commit -m "feat: add coursepack SAN line helper"
```

Expected: one public commit with only generic authoring-tooling changes.

---

### Task 3: Inventory and author printed pages 171-176

**Files:**
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.authoring.json`
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/caro-kann-black-v2-pages-171-176/`

**Interfaces:**
- Consumes: rendered pages `caro-188.png` through `caro-193.png`, current v1 private Caro pack, and `coursepack sanline`.
- Produces: expanded v2 private pack covering all legible source items on printed pages 171-176.

- [ ] **Step 1: Record page inventory entries**

Open and visually inspect:

```bash
open /private/tmp/mco15-caro-kann-pages/caro-188.png
open /private/tmp/mco15-caro-kann-pages/caro-189.png
open /private/tmp/mco15-caro-kann-pages/caro-190.png
open /private/tmp/mco15-caro-kann-pages/caro-191.png
open /private/tmp/mco15-caro-kann-pages/caro-192.png
open /private/tmp/mco15-caro-kann-pages/caro-193.png
```

Update the private authoring JSON so every legible table column, note label, and overview item on printed pages 171-176 has a private inventory record with these fields:

```json
{
  "id": "caro-p171-overview",
  "printedPage": 171,
  "pdfPage": 188,
  "imagePath": "/private/tmp/mco15-caro-kann-pages/caro-188.png",
  "sourceKind": "overview",
  "label": "overview",
  "family": "foundations",
  "depth": "quick",
  "activityTreatment": "concept",
  "status": "inventoried"
}
```

For each line item, include `san` as an array of SAN tokens if the source item contains moves. Use source-derived SAN only in the private authoring file.

- [ ] **Step 2: Convert SAN lines to UCI as needed**

For each newly captured SAN line, run:

```bash
go run ./cmd/coursepack sanline \
  --fen "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1" \
  e4 c6 d4 d5
```

For branch lines that start from a non-root position, replace `--fen` with the branch start FEN from the compiled course graph or previous `sanline` output. Copy the derived UCI sequence into the private `.ctcourse` graph.

- [ ] **Step 3: Preserve current Quick/Standard IDs while expanding graph coverage**

Keep these existing IDs if their meaning remains unchanged:

```text
caro-foundations
caro-d5-structure
caro-mainline-bishop
caro-advance-c5
```

Add Reference-only activities to these lessons for dense page 171-176 material instead of adding repetitive required decisions.

- [ ] **Step 4: Bump content version and validate**

Ensure `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse` has:

```json
"contentVersion": "2.0.0"
```

Run:

```bash
go run ./cmd/coursepack validate "/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse" \
  > /private/tmp/caro-kann-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/caro-kann-validation.json
```

Expected: `courseId` is `mco15-caro-kann-black`, `contentVersion` is `2.0.0`, and both coverage counts are `0`.

- [ ] **Step 5: Checkpoint pages 171-176**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/caro-kann-black-v2-pages-171-176"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.authoring.json" \
  "$CHECKPOINT/mco15-caro-kann-black.authoring.json"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse" \
  "$CHECKPOINT/mco15-caro-kann-black.ctcourse"
cp -p /private/tmp/caro-kann-validation.json "$CHECKPOINT/validation.json"
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/caro-kann-validation.json > "$CHECKPOINT/summary.json"
shasum -a 256 "$CHECKPOINT/"* > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
```

Expected: validation is clean and checksums print `OK`.

- [ ] **Step 6: Verify no private files are tracked**

Run:

```bash
git status --short
git ls-files '*.ctcourse'
```

Expected: no private course file appears in Git.

---

### Task 4: Inventory and author printed pages 177-183

**Files:**
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.authoring.json`
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/caro-kann-black-v2-pages-177-183/`

**Interfaces:**
- Consumes: rendered pages `caro-194.png` through `caro-200.png`, v2 pages 171-176 checkpoint, and `coursepack sanline`.
- Produces: expanded v2 private pack covering all legible source items on printed pages 177-183.

- [ ] **Step 1: Inventory each source item on pages 177-183**

Open:

```bash
open /private/tmp/mco15-caro-kann-pages/caro-194.png
open /private/tmp/mco15-caro-kann-pages/caro-195.png
open /private/tmp/mco15-caro-kann-pages/caro-196.png
open /private/tmp/mco15-caro-kann-pages/caro-197.png
open /private/tmp/mco15-caro-kann-pages/caro-198.png
open /private/tmp/mco15-caro-kann-pages/caro-199.png
open /private/tmp/mco15-caro-kann-pages/caro-200.png
```

Add private authoring records for every legible table column, note label, and source branch on printed pages 177-183 using the same schema as Task 3. Classify each as Quick, Standard, or Reference according to the spec:

- practical working repertoire lines: Standard;
- dense alternatives, illustrative references, and note-letter branches: Reference;
- only core learner decisions already in the Quick spine: Quick.

- [ ] **Step 2: Extend relevant existing lessons**

Preserve these existing IDs if their meaning remains unchanged:

```text
caro-advance-c5
caro-advance-space
caro-exchange-structure
caro-exchange-minority
caro-panov-targets
caro-panov-iqp
```

Add new Reference-only lessons only for distinct source branches that would make an existing lesson ambiguous.

- [ ] **Step 3: Convert SAN lines and update graph**

For each source line with moves, use:

```bash
go run ./cmd/coursepack sanline --fen "<branch-start-fen>" <san-token>...
```

Add generated UCI moves to the private `.ctcourse`. Assign roles from Black's perspective:

```text
White move at White-to-move position: opponent
Black main course move at Black-to-move position: repertoire
Black playable non-main move at Black-to-move position: alternative
```

- [ ] **Step 4: Validate after the batch**

Run:

```bash
go run ./cmd/coursepack validate "/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse" \
  > /private/tmp/caro-kann-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/caro-kann-validation.json
```

Expected: `contentVersion` is `2.0.0`; both coverage counts are `0`; move and note counts are greater than the pages 171-176 checkpoint.

- [ ] **Step 5: Checkpoint pages 177-183**

Run the checkpoint command from Task 3 with:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/caro-kann-black-v2-pages-177-183"
```

Expected: validation is clean and checksums print `OK`.

---

### Task 5: Inventory and author printed pages 184-190

**Files:**
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.authoring.json`
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/caro-kann-black-v2-pages-184-190/`

**Interfaces:**
- Consumes: rendered pages `caro-201.png` through `caro-207.png`, v2 pages 177-183 checkpoint, and `coursepack sanline`.
- Produces: expanded v2 private pack covering all legible source items on printed pages 184-190.

- [ ] **Step 1: Inventory pages 184-190**

Open:

```bash
open /private/tmp/mco15-caro-kann-pages/caro-201.png
open /private/tmp/mco15-caro-kann-pages/caro-202.png
open /private/tmp/mco15-caro-kann-pages/caro-203.png
open /private/tmp/mco15-caro-kann-pages/caro-204.png
open /private/tmp/mco15-caro-kann-pages/caro-205.png
open /private/tmp/mco15-caro-kann-pages/caro-206.png
open /private/tmp/mco15-caro-kann-pages/caro-207.png
```

Add every legible source item to the private authoring inventory. If a scan line is not legible enough to author safely, add an `illegibleItems` entry in the private authoring JSON:

```json
{
  "printedPage": 184,
  "pdfPage": 201,
  "imagePath": "/private/tmp/mco15-caro-kann-pages/caro-201.png",
  "label": "unreadable-source-item",
  "reason": "visual scan is not legible enough to author safely"
}
```

Do not add illegible items to `sourceCoverage.expectedReferences`.

- [ ] **Step 2: Extend Standard and Reference lessons**

Preserve these existing IDs if their meaning remains unchanged:

```text
caro-classical-main
caro-endgames
caro-sharp-reference
```

Add Reference-only demonstrations/comparisons for dense source branches. Do not add repeated required decisions for lines that teach the same Black choice from the same position.

- [ ] **Step 3: Convert SAN, update graph, and validate**

Run `coursepack sanline` for each newly captured move line, update the private graph, then validate:

```bash
go run ./cmd/coursepack validate "/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse" \
  > /private/tmp/caro-kann-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/caro-kann-validation.json
```

Expected: validation clean, `contentVersion` `2.0.0`, no role/perspective diagnostics.

- [ ] **Step 4: Checkpoint pages 184-190**

Run the checkpoint command from Task 3 with:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/caro-kann-black-v2-pages-184-190"
```

Expected: validation is clean and checksums print `OK`.

---

### Task 6: Inventory and author printed pages 191-196, then finalize source coverage

**Files:**
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.authoring.json`
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/caro-kann-black-v2-full-reference/`

**Interfaces:**
- Consumes: rendered pages `caro-208.png` through `caro-213.png`, v2 pages 184-190 checkpoint, and `coursepack sanline`.
- Produces: final full-reference private Caro-Kann v2 pack.

- [ ] **Step 1: Inventory pages 191-196**

Open:

```bash
open /private/tmp/mco15-caro-kann-pages/caro-208.png
open /private/tmp/mco15-caro-kann-pages/caro-209.png
open /private/tmp/mco15-caro-kann-pages/caro-210.png
open /private/tmp/mco15-caro-kann-pages/caro-211.png
open /private/tmp/mco15-caro-kann-pages/caro-212.png
open /private/tmp/mco15-caro-kann-pages/caro-213.png
```

Add every legible source item to the private authoring inventory. Preserve these existing IDs if their meaning remains unchanged:

```text
caro-sideline-setup
caro-fantasy
caro-two-knights
caro-rare-systems
```

- [ ] **Step 2: Finalize expected source coverage**

In the private `.ctcourse`, ensure:

```json
"sourceCoverage": {
  "printedPages": [171, 172, 173, 174, 175, 176, 177, 178, 179, 180, 181, 182, 183, 184, 185, 186, 187, 188, 189, 190, 191, 192, 193, 194, 195, 196],
  "expectedReferences": []
}
```

Populate `expectedReferences` with exactly the coverage IDs represented by authored moves and notes. Do not include illegible or intentionally omitted items in `expectedReferences`.

- [ ] **Step 3: Verify required stable lesson IDs still exist**

Run:

```bash
jq -e '
  [.lessons[].lessonId] as $ids |
  [
    "caro-foundations",
    "caro-d5-structure",
    "caro-mainline-bishop",
    "caro-advance-c5",
    "caro-exchange-structure",
    "caro-panov-targets",
    "caro-sideline-setup",
    "caro-classical-main",
    "caro-advance-space",
    "caro-exchange-minority",
    "caro-panov-iqp",
    "caro-fantasy",
    "caro-two-knights",
    "caro-endgames",
    "caro-sharp-reference",
    "caro-rare-systems"
  ] | all(. as $id | $ids | index($id))
' "/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse"
```

Expected: command exits `0`.

- [ ] **Step 4: Final validate**

Run:

```bash
go run ./cmd/coursepack validate "/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse" \
  > /private/tmp/caro-kann-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/caro-kann-validation.json
```

Expected:

```json
{
  "courseId": "mco15-caro-kann-black",
  "contentVersion": "2.0.0",
  "missing": 0,
  "unexpected": 0
}
```

Counts must show nonzero `lessons`, `lessonEdges`, `activities`, `prompts`, `moves`, and `notes`; move and note counts should be materially greater than the v1 seed pack.

- [ ] **Step 5: Create final v2 checkpoint**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/caro-kann-black-v2-full-reference"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.authoring.json" \
  "$CHECKPOINT/mco15-caro-kann-black.authoring.json"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse" \
  "$CHECKPOINT/mco15-caro-kann-black.ctcourse"
cp -p /private/tmp/caro-kann-validation.json "$CHECKPOINT/validation.json"
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/caro-kann-validation.json > "$CHECKPOINT/summary.json"
shasum -a 256 "$CHECKPOINT/"* > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
```

Expected: validation is clean and checksums print `OK`.

---

### Task 7: Import, rebase-check, verify, rebuild, and finish

**Files:**
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse`
- Verify/build: repository source tree
- Build: `build/bin/Chess Trainer.app`
- Modify repository files only if a generic app defect is found and fixed with a synthetic failing test first.

**Interfaces:**
- Consumes: final validated private Caro-Kann v2 full-reference pack.
- Produces: real app catalogue updated to Caro-Kann v2, verified app journey, clean branch ready to merge/push.

- [ ] **Step 1: Import the final pack through the app importer service**

Use the same app service path as previous course imports or the UI. Scripted service import is acceptable if it calls `CourseImporter.Inspect` and `CourseImporter.Import`.

Expected import behavior:

```text
sourceId: mco15-caro-kann-black
sourceName: Caro-Kann for Black
replacesExisting: true
```

- [ ] **Step 2: Verify all three courses remain active**

Run a service or UI check confirming:

```text
mco15-italian-white       Italian Game for White     white
mco15-ruy-lopez-white     Ruy Lopez for White        white
mco15-caro-kann-black     Caro-Kann for Black        black
```

Expected: Italian and Ruy active generations are unchanged; Caro-Kann active generation is replaced.

- [ ] **Step 3: Verify Caro-Kann learner journey**

In a disposable app data root or through the UI:

1. Start one Quick Caro-Kann lesson.
2. Complete one passive activity.
3. Complete one Black decision activity.
4. Start one Standard Caro-Kann lesson.
5. Start one Reference Caro-Kann lesson.
6. Open variation explorer from the course root.
7. Open variation explorer from one deeper Reference branch.
8. Pause and resume a Caro-Kann lesson.

Expected:

```text
board orientation: black
decision legal moves are Black-to-move moves where prompted
opening activity result.appliedMoves is always an array
Continue Learning resumes Caro-Kann when the Caro session owns the active session
variation explorer loads without missing-position errors
```

- [ ] **Step 4: Fix generic app defects only if found**

If any generic app defect appears, use `superpowers:systematic-debugging` and add a failing synthetic test before editing app code. Example focused commands:

```bash
go test ./internal/openings -run '<new synthetic regression name>' -count=1
npm --prefix frontend test -- --run src/components/openings/<affected-test>.test.ts
```

Commit any generic app fix with a scoped message:

```bash
git add <generic app files>
git commit -m "fix: <generic opening behavior>"
```

Expected: no private course data is committed.

- [ ] **Step 5: Verify private course files are untracked**

Run:

```bash
git status --short
git ls-files '*.ctcourse'
```

Expected: private files under `/Users/admin/Documents/Private Chess Courses/` are not listed. Tracked `.ctcourse` files are only synthetic fixtures under `internal/openings/testdata/`.

- [ ] **Step 6: Run full verification**

Run:

```bash
npm --prefix frontend run build
go test ./... -count=1
go vet ./...
npm --prefix frontend test -- --run --single-thread
npm --prefix frontend run check
npm --prefix frontend run verify:licenses
npm --prefix frontend run test:e2e -- openings.spec.ts --project=chromium
npm --prefix frontend run test:e2e -- openings.spec.ts --project=webkit
```

Expected: every command exits `0`.

- [ ] **Step 7: Rebuild the macOS app**

Run:

```bash
GOWORK=off GOTOOLCHAIN=local go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 build -clean -trimpath
codesign --verify --deep --strict --verbose=2 "build/bin/Chess Trainer.app"
file "build/bin/Chess Trainer.app/Contents/MacOS/Chess Trainer"
```

Expected: Wails build succeeds, codesign verifies, and the binary is `Mach-O 64-bit executable arm64`.

- [ ] **Step 8: Restore generated binding whitespace if needed**

Run:

```bash
git diff -- frontend/wailsjs/go/models.ts | sed -n '1,120p'
```

If the diff is only whitespace churn from Wails generation, restore it:

```bash
git restore -- frontend/wailsjs/go/models.ts
```

Expected: final branch status is clean except intentional generic commits.

- [ ] **Step 9: Final review and finishing workflow**

Run:

```bash
git status --short --branch
git log --oneline --decorate -8
```

Use `superpowers:requesting-code-review` for any public code diff. Then use `superpowers:finishing-a-development-branch` to choose merge, PR, keep, or discard. Do not push or merge without the user's selected finishing option.
