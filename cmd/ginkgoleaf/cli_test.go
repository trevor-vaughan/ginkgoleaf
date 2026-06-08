package main

import (
	"bytes"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CLI dependency hygiene", func() {
	// Guards the structural win: the CLI must depend only on leaf
	// packages, never the ginkgoleaf library package that links the
	// Ginkgo runtime. If this regresses, `--help` regrows ~40 framework
	// flags and the binary balloons. We inspect the production import
	// graph (go list -deps excludes test-only imports, so the Ginkgo
	// imported by these very test files does not count).
	It("does not link the Ginkgo runtime into the binary", func() {
		cmd := exec.Command("go", "list", "-deps", ".")
		// Drop GOFLAGS: this repo uses go.work, and a stray GOFLAGS=-mod=mod
		// (set in some dev containers) is rejected in workspace mode.
		cmd.Env = withoutEnv(os.Environ(), "GOFLAGS")
		out, err := cmd.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), "go list failed:\n%s", out)

		for _, dep := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			Expect(dep).NotTo(Equal("github.com/onsi/ginkgo/v2"),
				"cmd/ginkgoleaf must not import the Ginkgo runtime package")
		}
	})
})

// withoutEnv returns env with any KEY=... entry for key removed.
func withoutEnv(env []string, key string) []string {
	out := env[:0:0]
	for _, kv := range env {
		if !strings.HasPrefix(kv, key+"=") {
			out = append(out, kv)
		}
	}
	return out
}

var _ = Describe("writeUsage", func() {
	It("lists only the real flags and the exit-code contract", func() {
		var buf bytes.Buffer
		writeUsage(&buf)
		got := buf.String()

		for _, want := range []string{"-in", "-out", "-format", "-color", "-version", "tree", "Exit codes"} {
			Expect(got).To(ContainSubstring(want),
				"usage text must mention %q", want)
		}
	})

	It("never leaks the Ginkgo test-framework flags", func() {
		var buf bytes.Buffer
		writeUsage(&buf)
		Expect(buf.String()).NotTo(ContainSubstring("ginkgo."),
			"the post-processor must not surface Ginkgo runtime flags")
	})
})

var _ = Describe("noInputOnTTY", func() {
	It("is false when an explicit -in path is given (we read the file)", func() {
		Expect(noInputOnTTY("report.json", os.Stdin)).To(BeFalse())
	})

	It("is false when stdin is a regular file (a redirect)", func() {
		f, err := os.CreateTemp("", "ginkgoleaf-in-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { _ = f.Close(); _ = os.Remove(f.Name()) })
		Expect(noInputOnTTY("-", f)).To(BeFalse())
	})

	It("is false when stdin is a pipe (piped input)", func() {
		pr, pw, err := os.Pipe()
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { _ = pr.Close(); _ = pw.Close() })
		Expect(noInputOnTTY("-", pr)).To(BeFalse())
	})
})

var _ = Describe("resolveVersion", func() {
	It("returns the ldflag values verbatim when version was injected", func() {
		v, c, d := resolveVersion("1.2.3", "abc123", "2026-05-29", nil)
		Expect(v).To(Equal("1.2.3"))
		Expect(c).To(Equal("abc123"))
		Expect(d).To(Equal("2026-05-29"))
	})

	It("falls back to build info for go install / go build binaries", func() {
		info := &debug.BuildInfo{
			Main: debug.Module{Version: "v0.4.0"},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "deadbeef"},
				{Key: "vcs.time", Value: "2026-05-29T00:00:00Z"},
			},
		}
		v, c, d := resolveVersion("dev", "unknown", "unknown", info)
		Expect(v).To(Equal("v0.4.0"))
		Expect(c).To(Equal("deadbeef"))
		Expect(d).To(Equal("2026-05-29T00:00:00Z"))
	})

	It("keeps the dev label for a local (devel) build but adopts the VCS revision", func() {
		info := &debug.BuildInfo{
			Main: debug.Module{Version: "(devel)"},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "cafef00d"},
			},
		}
		v, c, _ := resolveVersion("dev", "unknown", "unknown", info)
		Expect(v).To(Equal("dev"))
		Expect(c).To(Equal("cafef00d"))
	})

	It("returns the defaults when no build info is available", func() {
		v, c, d := resolveVersion("dev", "unknown", "unknown", nil)
		Expect(v).To(Equal("dev"))
		Expect(c).To(Equal("unknown"))
		Expect(d).To(Equal("unknown"))
	})
})
