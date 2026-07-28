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
