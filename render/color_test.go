package render_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/trevor-vaughan/ginkgoleaf/render"
)

var _ = Describe("ParseColorMode", func() {
	DescribeTable("maps the three documented modes",
		func(s string, want render.ColorMode) {
			got, err := render.ParseColorMode(s)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(want))
		},
		Entry("auto", "auto", render.ColorAuto),
		Entry("always", "always", render.ColorAlways),
		Entry("never", "never", render.ColorNever),
	)

	It("rejects an unknown mode with ErrUnknownColor and names the value", func() {
		_, err := render.ParseColorMode("purple")
		Expect(errors.Is(err, render.ErrUnknownColor)).To(BeTrue(),
			"expected ErrUnknownColor, got %v", err)
		Expect(err.Error()).To(ContainSubstring(`"purple"`))
	})
})

var _ = Describe("Color", func() {
	// always/never force the decision regardless of writer or env.
	// auto colors a terminal OR a pipe (so `… | less -R` is colored) but
	// stays plain for file redirects, with NO_COLOR/CLICOLOR overriding.
	Describe("ColorEnabled", func() {
		It("always emits color regardless of the writer", func() {
			var buf bytes.Buffer
			Expect(render.ColorEnabled(render.ColorAlways, &buf)).To(BeTrue())
		})

		It("never emits color regardless of the writer", func() {
			var buf bytes.Buffer
			Expect(render.ColorEnabled(render.ColorNever, &buf)).To(BeFalse())
		})

		Describe("auto", func() {
			// Neutralise any ambient NO_COLOR/CLICOLOR so we exercise the
			// terminal-vs-pipe-vs-file decision itself, not the env.
			BeforeEach(func() {
				for _, k := range []string{"NO_COLOR", "CLICOLOR", "CLICOLOR_FORCE"} {
					if orig, had := os.LookupEnv(k); had {
						Expect(os.Unsetenv(k)).To(Succeed())
						DeferCleanup(os.Setenv, k, orig)
					}
				}
			})

			It("colors a pipe — what `ginkgoleaf … | less -R` produces", func() {
				pr, pw, err := os.Pipe()
				Expect(err).NotTo(HaveOccurred())
				DeferCleanup(func() { _ = pr.Close(); _ = pw.Close() })
				Expect(render.ColorEnabled(render.ColorAuto, pw)).To(BeTrue(),
					"a pipe (FIFO) destined for a pager should be colored")
			})

			It("stays plain for a regular file — `ginkgoleaf … > out.txt`", func() {
				f, err := os.CreateTemp("", "ginkgoleaf-color-*")
				Expect(err).NotTo(HaveOccurred())
				DeferCleanup(func() { _ = f.Close(); _ = os.Remove(f.Name()) })
				Expect(render.ColorEnabled(render.ColorAuto, f)).To(BeFalse(),
					"a file redirect must not embed escape codes")
			})

			It("stays plain for a non-file writer", func() {
				var buf bytes.Buffer
				Expect(render.ColorEnabled(render.ColorAuto, &buf)).To(BeFalse())
			})

			It("still honors NO_COLOR even for a pipe", func() {
				Expect(os.Setenv("NO_COLOR", "1")).To(Succeed())
				DeferCleanup(os.Unsetenv, "NO_COLOR")
				pr, pw, err := os.Pipe()
				Expect(err).NotTo(HaveOccurred())
				DeferCleanup(func() { _ = pr.Close(); _ = pw.Close() })
				Expect(render.ColorEnabled(render.ColorAuto, pw)).To(BeFalse(),
					"NO_COLOR must override the pipe-coloring rule")
			})
		})
	})

	Describe("ANSI writer", func() {
		It("emits plain text when color is disabled", func() {
			var buf bytes.Buffer
			a := render.NewANSI(&buf, false)
			a.WriteRed("error")
			Expect(buf.String()).To(Equal("error"))
		})

		It("wraps text in red ANSI escape codes when color is enabled", func() {
			var buf bytes.Buffer
			a := render.NewANSI(&buf, true)
			a.WriteRed("error")
			Expect(buf.String()).To(Equal("\x1b[31merror\x1b[0m"))
		})
	})
})

// failWriter always fails — used to assert write errors are surfaced.
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

// failOnWriter forwards to w but fails on any write whose bytes contain
// marker — letting a test target one specific colored write.
type failOnWriter struct {
	w      io.Writer
	marker string
}

func (f *failOnWriter) Write(p []byte) (int, error) {
	if strings.Contains(string(p), f.marker) {
		return 0, errors.New("write failed")
	}
	return f.w.Write(p)
}

func TestANSIRecordsColoredWriteError(t *testing.T) {
	a := render.NewANSI(failWriter{}, true)
	a.WriteRed("x")
	if a.Err() == nil {
		t.Fatal("expected Err() to report the failed colored write")
	}
}

func TestTreePropagatesColoredWriteError(t *testing.T) {
	r := render.Report{Suite: render.SuiteRow{Name: "S", SuiteSucceeded: true, SpecialFailure: "SPECIAL"}}
	w := &failOnWriter{w: io.Discard, marker: "SPECIAL"}
	if err := render.NewTree(true).WriteAll(w, r); err == nil {
		t.Fatal("tree: expected WriteAll to surface the failed SpecialFailure write")
	}
}

func TestJestPropagatesColoredWriteError(t *testing.T) {
	r := render.Report{Suite: render.SuiteRow{Name: "S", SuiteSucceeded: true, SpecialFailure: "SPECIAL"}}
	w := &failOnWriter{w: io.Discard, marker: "SPECIAL"}
	if err := render.NewJest(true).WriteAll(w, r); err == nil {
		t.Fatal("jest: expected WriteAll to surface the failed SpecialFailure write")
	}
}

// Every renderer must surface a write failure from WriteAll rather than
// swallow it — the errWriter/ANSI sticky-error contract.
func TestRenderersSurfaceWriteErrors(t *testing.T) {
	r := render.Report{
		Suite: render.SuiteRow{Name: "S", NumSpecs: 1, NumFailed: 1},
		Specs: []render.SpecRow{{
			State:         render.StateFailed,
			FullText:      []string{"Group", "fails"},
			ContainerHier: []string{"Group"},
			LeafText:      "fails",
			Failure:       &render.FailureRow{Message: "boom", Location: render.CodeLocation{FileName: "f.go", LineNumber: 1}},
		}},
	}
	renderers := map[string]render.Renderer{
		"tree":     render.NewTree(true),
		"jest":     render.NewJest(true),
		"text":     render.NewText(),
		"shell":    render.NewShell(),
		"tap":      render.NewTAP(),
		"markdown": render.NewMarkdown(),
		"github":   render.NewGitHub(),
		"gitlab":   render.NewGitLab(func() int64 { return 1 }, true),
		"cucumber": render.NewCucumber(true),
	}
	for name, rdr := range renderers {
		if err := rdr.WriteAll(failWriter{}, r); err == nil {
			t.Errorf("%s: WriteAll returned nil on a failing writer", name)
		}
	}
}
