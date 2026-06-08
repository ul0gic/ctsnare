package main

import (
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// buildBinary compiles the ctsnare entry point into a temp dir and returns the
// path. This exercises that cmd/ctsnare actually builds and wires up to
// internal/cmd — the one thing a unit test on main() cannot reach, since main
// only calls cmd.Execute() and os.Exit on error.
func buildBinary(t *testing.T) string {
	t.Helper()

	bin := filepath.Join(t.TempDir(), "ctsnare")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}

	cmd := exec.Command("go", "build", "-o", bin, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building ctsnare: %v\n%s", err, out)
	}
	return bin
}

// TestMainHelp runs the built binary with --help and asserts it exits 0 and
// prints recognizable usage. This is the smoke test for the entry point.
func TestMainHelp(t *testing.T) {
	bin := buildBinary(t)

	out, err := exec.Command(bin, "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("ctsnare --help exited with error: %v\n%s", err, out)
	}

	got := string(out)
	for _, want := range []string{"ctsnare", "Certificate Transparency", "Usage:"} {
		if !strings.Contains(got, want) {
			t.Errorf("--help output missing %q\nfull output:\n%s", want, got)
		}
	}
}

// TestMainUnknownCommandFails asserts the entry point surfaces a non-zero exit
// code for an unknown command — confirming main() propagates cmd.Execute errors
// to os.Exit(1).
func TestMainUnknownCommandFails(t *testing.T) {
	bin := buildBinary(t)

	cmd := exec.Command(bin, "definitely-not-a-real-command")
	err := cmd.Run()

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *exec.ExitError for unknown command, got %T: %v", err, err)
	}
	if exitErr.ExitCode() == 0 {
		t.Error("expected non-zero exit code for unknown command")
	}
}
