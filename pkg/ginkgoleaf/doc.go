// Package ginkgoleaf is a third-party output formatter for Ginkgo v2.
//
// It produces human- and LLM-friendly reports in nine formats: tree, jest,
// markdown, github, gitlab, text, shell, tap, and cucumber. There are two ways
// to use it:
//
// In-suite (this package): call [Register] once at file scope in a *_test.go
// file. It hooks Ginkgo's sanctioned ReportAfterSuite decorator and renders
// the chosen format after the suite runs, alongside Ginkgo's own output.
//
//	var _ = ginkgoleaf.Register(ginkgoleaf.FormatJest)
//
// [Register] takes functional [Option]s — [WithColor] and [WithWriter] — and
// validates its configuration up front via [NewConfig]; a bad format or a
// double registration panics at suite-init, surfacing the programmer error
// immediately.
//
// Standalone CLI: the `ginkgoleaf` command renders a Ginkgo `--json-report`
// into any format without linking the Ginkgo runtime. See cmd/ginkgoleaf.
//
// Structured matcher diffs (clean expected/actual) are available by importing
// the optional ginkgoleaf/gomega sub-package, which is the only path that
// pulls gomega into a consumer's go.sum.
package ginkgoleaf
