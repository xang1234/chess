# macOS Native Full-Screen Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enable Chess Trainer's native green macOS title-bar control so the normal application window can enter native full-screen mode.

**Architecture:** Extract the static Wails desktop window configuration into a pure `desktopWindowOptions` helper and make its macOS options non-nil. Keep lifecycle callbacks and bindings in `main`, allowing a focused Go regression test to exercise the real window configuration without starting the application.

**Tech Stack:** Go 1.26.4, Wails v2.12.0, Cocoa window options, macOS codesign.

## Global Constraints

- Continue to open as a normal 1024×768 window.
- Use the native macOS green title-bar control and native full-screen Space.
- Keep resizing enabled and leave maximum width and height unset.
- Do not add an in-app full-screen control or start full-screen automatically.
- Preserve current non-macOS behavior.

---

### Task 1: Enable and test the native macOS window control

**Files:**
- Create: `main_window_test.go`
- Modify: `main.go:3-40`

**Interfaces:**
- Produces: `desktopWindowOptions() *options.App`, the static Wails window configuration consumed by `main`.

- [ ] **Step 1: Write the failing configuration test**

```go
package main

import "testing"

func TestDesktopWindowOptionsEnableMacNativeFullScreen(t *testing.T) {
	configured := desktopWindowOptions()
	if configured.Mac == nil {
		t.Fatal("macOS options must be present so Wails enables the native zoom button")
	}
	if configured.Mac.DisableZoom {
		t.Fatal("macOS native zoom and full-screen control must remain enabled")
	}
	if configured.DisableResize {
		t.Fatal("desktop window must remain resizable")
	}
	if configured.MaxWidth != 0 || configured.MaxHeight != 0 {
		t.Fatalf("desktop window must remain unconstrained, got max %dx%d",
			configured.MaxWidth, configured.MaxHeight)
	}
	if configured.Width != 1024 || configured.Height != 768 {
		t.Fatalf("initial desktop size changed to %dx%d", configured.Width, configured.Height)
	}
}
```

- [ ] **Step 2: Run the focused test and verify RED**

Run:

```bash
go test . -run TestDesktopWindowOptionsEnableMacNativeFullScreen -count=1
```

Expected: compilation fails because `desktopWindowOptions` is undefined.

- [ ] **Step 3: Extract the static options and enable macOS zoom**

Import `github.com/wailsapp/wails/v2/pkg/options/mac`, then add:

```go
func desktopWindowOptions() *options.App {
	return &options.App{
		Title:  "Chess Trainer",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		Mac:              &mac.Options{},
	}
}
```

In `main`, obtain `appOptions := desktopWindowOptions()`, attach
`OnStartup`, `OnShutdown`, and `Bind`, then call `wails.Run(appOptions)`.

- [ ] **Step 4: Verify GREEN and the root test suite**

```bash
go test . -run TestDesktopWindowOptionsEnableMacNativeFullScreen -count=1
go test ./... -count=1
go vet ./...
git diff --check
```

Expected: all commands exit successfully.

- [ ] **Step 5: Commit the behavior change**

```bash
git add main.go main_window_test.go
git commit -m "fix: enable macOS native full-screen"
```

---

### Task 2: Rebuild and verify the macOS application

**Files:**
- Normalize if regenerated: `frontend/wailsjs/go/models.ts`
- Generate ignored: `build/bin/Chess Trainer.app`

**Interfaces:**
- Consumes: `desktopWindowOptions() *options.App` from Task 1.
- Produces: a verified local arm64 macOS application bundle.

- [ ] **Step 1: Run the complete project verification**

```bash
go test -race ./... -count=1
go mod verify
npm --prefix frontend test -- --run --single-thread
npm --prefix frontend run check
npm --prefix frontend run build
npm --prefix frontend run test:e2e
git diff --check
```

Expected: all applicable tests and validation commands pass.

- [ ] **Step 2: Build the app with the pinned Wails release**

```bash
GOWORK=off GOTOOLCHAIN=local \
  go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 build -clean -trimpath
perl -pi -e 's/[ \t]+$//' frontend/wailsjs/go/models.ts
perl -0pi -e 's/\s+\z/\n/' frontend/wailsjs/go/models.ts
chmod 0644 frontend/wailsjs/runtime/package.json \
  frontend/wailsjs/runtime/runtime.d.ts frontend/wailsjs/runtime/runtime.js
```

- [ ] **Step 3: Verify the bundle and repository state**

```bash
git diff --check
codesign --verify --deep --strict --verbose=2 "build/bin/Chess Trainer.app"
file "build/bin/Chess Trainer.app/Contents/MacOS/Chess Trainer"
git status --short
```

Expected: strict signature verification passes, the executable is Mach-O arm64,
and no generated tracked change remains.

- [ ] **Step 4: Perform native macOS acceptance**

Launch the rebuilt app. Confirm the green title-bar control is enabled, enters
native full-screen with the menu bar and Dock hidden, and returns to a normal
resizable window when full-screen is exited.
