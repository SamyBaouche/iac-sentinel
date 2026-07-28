package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunVersion(t *testing.T) {
	out := captureStdout(t, func() {
		code := run([]string{"version"})
		if code != 0 {
			t.Fatalf("exit = %d", code)
		}
	})
	if !strings.Contains(out, "iac-sentinel") {
		t.Fatalf("version output = %q", out)
	}
}

func TestRunScanMissingPlan(t *testing.T) {
	code := run([]string{"scan"})
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
}

func TestRunScanFailOn(t *testing.T) {
	plan := filepath.Join("..", "..", "testdata", "plan_mixed.json")
	code := run([]string{
		"scan",
		"-plan", plan,
		"-fail-on", "DANGER",
		"-skip-checkov",
		"-skip-tfsec",
	})
	// mixed plan deletes aws_db_instance → CRITICAL risk → must fail
	if code != 1 {
		t.Fatalf("exit = %d, want 1 (fail-on triggered)", code)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	code := run([]string{"nope"})
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	_ = w.Close()
	return <-done
}
