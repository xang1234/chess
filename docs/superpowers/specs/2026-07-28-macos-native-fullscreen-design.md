# macOS Native Full-Screen Design

## Problem

Chess Trainer's macOS window is resizable, but its green title-bar control is
disabled. The learner therefore cannot enter native macOS full-screen mode.

The application currently passes no `Mac` value in its Wails `options.App`.
In Wails v2.12.0, the Darwin window adapter initializes its `zoomable` flag to
false and only changes it when `options.App.Mac` is non-nil. Cocoa then
explicitly disables the standard zoom button when the window is resizable but
not zoomable. This is the root cause; the frontend layout and app bundle plist
do not impose the restriction.

## Desired Behavior

- Chess Trainer continues to open as a normal 1024×768 window.
- The native green macOS title-bar control is enabled.
- Activating that control enters native macOS full-screen mode, using a
  separate Space and hiding the menu bar and Dock.
- The learner can leave full-screen using normal macOS controls.
- The app remains freely resizable and has no maximum width or height.
- Other platforms retain their current window behavior.

## Design

Introduce a small pure helper that returns the static desktop window options.
It owns the title, initial dimensions, embedded assets, background color, and
platform window configuration. `main` adds the live application lifecycle
callbacks and bindings before calling `wails.Run`.

The helper supplies an empty `mac.Options` value:

```go
Mac: &mac.Options{}
```

For Wails v2.12.0, a non-nil macOS options value with the default
`DisableZoom: false` makes the Darwin window zoomable. No custom frontend
control, runtime full-screen call, startup full-screen state, or plist change
is needed.

## Regression Testing

A Go unit test exercises the real window-options helper and requires:

- a non-nil `Mac` configuration;
- `DisableZoom` to remain false;
- resizing to remain enabled;
- `MaxWidth` and `MaxHeight` to remain unset; and
- the existing 1024×768 initial size to remain unchanged.

The test must fail against the current configuration before the fix is added.
The normal Go and frontend suites remain the regression boundary for unrelated
application behavior.

## Build and Acceptance

After the tests pass, rebuild `build/bin/Chess Trainer.app` with the pinned
Wails v2.12.0 command. Normalize regenerated bindings, verify the worktree,
check the bundle with strict `codesign`, and confirm the executable is arm64.

Manual acceptance on macOS is:

1. launch the rebuilt app in its normal window;
2. confirm the green title-bar control is enabled;
3. enter native full-screen and confirm the app covers the display while the
   menu bar and Dock are hidden; and
4. exit full-screen and confirm the window returns to normal operation.

## Non-Goals

- Starting the app in full-screen automatically.
- Adding an in-app full-screen button or keyboard shortcut.
- Persisting the previous window size or full-screen state.
- Changing chess lesson layout or responsive breakpoints.
