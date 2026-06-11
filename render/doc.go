// Package render is ginkgoleaf's runtime-free rendering core. It defines the
// canonical, Ginkgo-agnostic [Report] model and the nine [Renderer]
// implementations (tree, jest, markdown, github, gitlab, text, shell, tap,
// cucumber), selected via [New].
//
// It is a supported public API for callers building their own pipeline: parse
// or construct a [Report] (or use [Translate] / [TranslateWithParser] to build
// one from a Ginkgo types.Report), pick a renderer with [New], and call
// WriteAll. Unlike the top-level ginkgoleaf package, render does not link the
// Ginkgo runtime, so importing it stays cheap.
//
// Most users want one of the two higher-level entry points instead: the
// in-suite ginkgoleaf package (Register) or the standalone ginkgoleaf CLI.
//
// All untrusted text from a report is neutralized at the [Translate] boundary
// (see sanitizeBlock/sanitizeLine), so renderers never re-escape control
// characters themselves.
package render
