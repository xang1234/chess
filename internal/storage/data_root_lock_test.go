package storage

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestAcquireDataRootLockRejectsSecondProcessOwner(t *testing.T) {
	root := t.TempDir()
	command := exec.Command(os.Args[0], "-test.run=^TestDataRootLockProcessHelper$")
	command.Env = append(os.Environ(), "CHESS_TRAINER_LOCK_HELPER_ROOT="+root)
	stdin, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	t.Cleanup(func() {
		stdin.Close()
		if !waited {
			command.Process.Kill()
			_ = command.Wait()
		}
	})

	ready := make(chan error, 1)
	go func() {
		line, readErr := bufio.NewReader(stdout).ReadString('\n')
		if readErr != nil {
			ready <- fmt.Errorf("read helper readiness: %w", readErr)
			return
		}
		if strings.TrimSpace(line) != "locked" {
			ready <- fmt.Errorf("unexpected helper readiness %q", line)
			return
		}
		ready <- nil
	}()
	select {
	case err := <-ready:
		if err != nil {
			t.Fatalf("%v; helper stderr: %s", err, stderr.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("lock helper did not become ready; stderr: %s", stderr.String())
	}

	started := time.Now()
	second, err := AcquireDataRootLock(root)
	if second != nil {
		second.Close()
		t.Fatal("AcquireDataRootLock() returned a lock while another process owned the root")
	}
	var lockErr *DataRootLockError
	if !errors.As(err, &lockErr) || !errors.Is(err, ErrDataRootLocked) {
		t.Fatalf("AcquireDataRootLock() err = %v, want typed already-running error", err)
	}
	if lockErr.Root != root || !strings.Contains(err.Error(), "already running") {
		t.Fatalf("DataRootLockError = %+v (%v)", lockErr, err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("second lock rejection took %v, want prompt failure", elapsed)
	}

	if err := stdin.Close(); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err != nil {
		t.Fatalf("lock helper failed: %v; stderr: %s", err, stderr.String())
	}
	waited = true
}

func TestDataRootLockIsReleasedOnClose(t *testing.T) {
	root := t.TempDir()
	first, err := AcquireDataRootLock(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("second Close() = %v, want idempotent close", err)
	}

	second, err := AcquireDataRootLock(root)
	if err != nil {
		t.Fatalf("AcquireDataRootLock() after Close() = %v", err)
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestDataRootLockProcessHelper(t *testing.T) {
	root := os.Getenv("CHESS_TRAINER_LOCK_HELPER_ROOT")
	if root == "" {
		t.Skip("subprocess helper")
	}
	lock, err := AcquireDataRootLock(root)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if _, err := fmt.Fprintln(os.Stdout, "locked"); err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(io.Discard, os.Stdin); err != nil {
		t.Fatal(err)
	}
}
