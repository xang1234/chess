package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestBindingDiscoveryDoesNotInitializeApplicationData(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("the release build and application-data path are macOS-specific")
	}

	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	workspace := t.TempDir()
	home := filepath.Join(workspace, "home")
	if err := os.Mkdir(home, 0o700); err != nil {
		t.Fatal(err)
	}
	project := `{
  "name": "Binding Safety Test",
  "frontend:dir": "frontend",
  "wailsjsdir": "frontend"
}`
	if err := os.WriteFile(filepath.Join(workspace, "wails.json"), []byte(project), 0o600); err != nil {
		t.Fatal(err)
	}

	binary := filepath.Join(workspace, "bindings")
	build := exec.Command("go", "build", "-buildvcs=false", "-tags", "bindings", "-o", binary, ".")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build bindings binary: %v\n%s", err, output)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	discover := exec.CommandContext(ctx, binary)
	discover.Dir = workspace
	discover.Env = bindingTestEnvironment(home)
	if output, err := discover.CombinedOutput(); err != nil {
		t.Fatalf("run bindings discovery: %v\n%s", err, output)
	}

	applicationData := filepath.Join(home, "Library", "Application Support", "Chess Trainer")
	if _, err := os.Stat(applicationData); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("bindings discovery initialized application data at %s: %v", applicationData, err)
	}
}

func bindingTestEnvironment(home string) []string {
	environment := make([]string, 0, len(os.Environ())+1)
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "HOME=") {
			environment = append(environment, value)
		}
	}
	return append(environment, "HOME="+home)
}
