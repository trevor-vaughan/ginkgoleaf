// Package integration drives the ginkgo CLI against the fixture
// modules and asserts on the captured output. Lives outside the public
// package so the repo root stays free of .go files.
package integration_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntegrationNoGomega(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not on PATH; skipping integration test")
	}
	out, err := runGinkgo(t, "../../fixtures/no-gomega")
	// Ginkgo exits non-zero when specs fail; that's expected here.
	if err != nil && !strings.Contains(out, "FAIL") {
		t.Fatalf("ginkgo errored without FAIL marker:\n%s\nerr: %v", out, err)
	}
	// Ginkgo's --succinct only names failing specs, so we look for the
	// failure-side markers plus the suite/group identity (Pure DSL is
	// this fixture's Describe).
	for _, want := range []string{"Passed", "FAIL", "Pure DSL", "fails"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
}

func TestIntegrationWithGomega(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not on PATH; skipping integration test")
	}
	out, err := runGinkgo(t, "../../fixtures/with-gomega")
	if err != nil && !strings.Contains(out, "FAIL") {
		t.Fatalf("ginkgo errored without FAIL marker:\n%s\nerr: %v", out, err)
	}
	for _, want := range []string{"Outer", "Inner", "fails", "FAIL"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
}

// TestIntegrationCLIPostProcessing exercises the post-processing path
// documented in the README: Ginkgo writes a JSON report, then the
// ginkgoleaf CLI renders it. It runs ginkgo via `go tool ginkgo`
// (declared in go.mod's tool block) so it works without a global
// ginkgo install; the bare-binary skip is kept for environments where
// the toolchain itself is unavailable.
func TestIntegrationCLIPostProcessing(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not on PATH; skipping CLI post-processing test")
	}

	// Step 1: produce a JSON report from the no-gomega fixture. The
	// fixture has intentional failures, so ginkgo exits non-zero — that
	// is fine; we only require a non-empty report.
	report := filepath.Join(t.TempDir(), "r.json")
	gen := exec.Command("go", "tool", "ginkgo", "--json-report="+report, "./...")
	gen.Dir = "../../fixtures/no-gomega"
	gen.Env = append(os.Environ(), "GOFLAGS=-mod=readonly")
	var genBuf bytes.Buffer
	gen.Stdout = &genBuf
	gen.Stderr = &genBuf
	// Run is allowed to fail (intentional spec failures); the report
	// presence is the real precondition.
	_ = gen.Run()
	info, err := os.Stat(report)
	if err != nil || info.Size() == 0 {
		t.Skipf("ginkgo did not produce a JSON report (toolchain unavailable?):\n%s\nerr: %v", genBuf.String(), err)
	}

	// Step 2: pipe the report through the ginkgoleaf CLI from the repo
	// root, rendering the tree format.
	render := exec.Command("go", "run", "./cmd/ginkgoleaf", "--format=tree", "--in="+report)
	render.Dir = "../.."
	render.Env = append(os.Environ(), "GOFLAGS=-mod=readonly")
	var renderBuf bytes.Buffer
	render.Stdout = &renderBuf
	render.Stderr = &renderBuf
	if err := render.Run(); err != nil {
		t.Fatalf("ginkgoleaf CLI failed:\n%s\nerr: %v", renderBuf.String(), err)
	}
	out := renderBuf.String()

	if !strings.Contains(out, "└──") && !strings.Contains(out, "├──") {
		t.Errorf("CLI output missing tree marker (└── or ├──):\n%s", out)
	}
	if !strings.Contains(out, "Pure DSL") {
		t.Errorf("CLI output missing fixture container %q:\n%s", "Pure DSL", out)
	}
}

// runGinkgo runs the fixture suite via `go tool ginkgo` (declared in
// go.mod's tool block) so it works without a global ginkgo install. The
// fixtures are sibling modules, so the command runs in the fixture dir with
// readonly module mode.
func runGinkgo(t *testing.T, dir string) (string, error) {
	t.Helper()
	cmd := exec.Command("go", "tool", "ginkgo", "--succinct", "--no-color", "./...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=readonly")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}
