package ginkgoleaf_test

import "github.com/trevor-vaughan/ginkgoleaf/pkg/ginkgoleaf"

// ExampleRegister shows the in-suite registration call. Place it once at file
// scope in a *_test.go file; ginkgoleaf renders at end-of-suite via Ginkgo's
// ReportAfterSuite. (Compiled as documentation; not run, so it does not
// register a real reporter.)
func ExampleRegister() {
	var _ = ginkgoleaf.Register(ginkgoleaf.FormatJest, ginkgoleaf.WithColor(ginkgoleaf.ColorAlways))
}
