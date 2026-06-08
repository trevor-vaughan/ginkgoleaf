package ginkgoleaf

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"

	"github.com/trevor-vaughan/ginkgoleaf/internal/parse"
	"github.com/trevor-vaughan/ginkgoleaf/render"
)

// Note: the render package's matcher-entry decoder defaults to a no-op via
// a package-var initializer (not an init func). The optional ginkgoleaf/gomega
// sub-package overrides it with the real decoder from its own init. Go
// guarantees render's var init completes before gomega's init runs (gomega
// imports render), so there is a single, ordered writer and no init race.
// This package never sets the decoder itself.

// Register wires the formatter into the active Ginkgo suite.
//
// Call once at top-level in a *_test.go file:
//
//	var _ = ginkgoleaf.Register(ginkgoleaf.FormatJest)
//
// Returns bool (always true) to match Ginkgo's DSL idiom so it can sit
// at file scope. Misconfiguration (unknown format, double registration)
// panics — Ginkgo surfaces the panic at suite-init.
func Register(f Format, opts ...Option) bool {
	cfg, err := NewConfig(f, opts...)
	if err != nil {
		panic(err)
	}
	registry.attach(cfg)
	return true
}

type registryT struct {
	mu       sync.Mutex
	attached bool
}

var registry = &registryT{}

func (r *registryT) attach(cfg *Config) {
	r.mu.Lock()
	if r.attached {
		r.mu.Unlock()
		panic(ErrAlreadyRegistered)
	}
	r.attached = true
	r.mu.Unlock()

	renderer := buildRenderer(cfg)

	// Always render via WriteAll at end-of-suite. We deliberately do NOT
	// hook ReportAfterEach: Ginkgo's default reporter runs its own
	// per-spec output (the green dots and inline failures users expect)
	// during the run, and our pretty rendering follows below — cleanly
	// separated, never interleaved. The leading newline + separator
	// pulls our output off of Ginkgo's trailing dot line so there is a
	// visible break between the two reporters' work.
	ginkgo.ReportAfterSuite("ginkgoleaf render", func(report types.Report) {
		w := cfg.Writer()
		_, _ = io.WriteString(w, "\n\n─── ginkgoleaf ───\n\n")
		// The rendered report is load-bearing, but ReportAfterSuite's callback
		// cannot return an error — so a write failure (broken pipe, full disk)
		// is logged rather than silently dropped (single-handling rule).
		if err := renderer.WriteAll(w, render.TranslateWithParser(report, parse.ParseGomega)); err != nil {
			fmt.Fprintln(os.Stderr, "ginkgoleaf: render failed:", err)
		}
	})
}

func buildRenderer(cfg *Config) render.Renderer {
	// Format was validated in NewConfig, so New cannot fail here.
	r, _ := render.New(cfg.Format(), cfgColor(cfg))
	return r
}

// cfgColor resolves whether the renderer should emit ANSI for cfg's
// writer and color mode.
func cfgColor(cfg *Config) bool {
	return render.ColorEnabled(render.ColorMode(cfg.Color()), cfg.Writer())
}

// Compile-time guarantee that ginkgoleaf.ColorMode and render.ColorMode stay
// numerically aligned — cfgColor casts between them. If either enum is
// reordered, one of these array indices goes out of range and the build
// fails, instead of silently miscoloring output.
var (
	_ = [1]struct{}{}[ColorAuto-ColorMode(render.ColorAuto)]
	_ = [1]struct{}{}[ColorAlways-ColorMode(render.ColorAlways)]
	_ = [1]struct{}{}[ColorNever-ColorMode(render.ColorNever)]
)
