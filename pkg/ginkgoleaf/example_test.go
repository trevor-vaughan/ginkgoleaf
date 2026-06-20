package ginkgoleaf_test

import (
	"fmt"
	"log"

	"github.com/trevor-vaughan/ginkgoleaf/pkg/ginkgoleaf"
)

// ExampleRegister shows the in-suite registration call. Place it once at file
// scope in a *_test.go file; ginkgoleaf renders at end-of-suite via Ginkgo's
// ReportAfterSuite. (Compiled as documentation; not run, so it does not
// register a real reporter.)
func ExampleRegister() {
	var _ = ginkgoleaf.Register(ginkgoleaf.FormatJest, ginkgoleaf.WithColor(ginkgoleaf.ColorAlways))
}

// ExampleNewConfig validates a format and resolves options up front. Register
// uses this internally, but it is exported so callers can pre-flight a
// configuration; an unknown format returns a wrapped [ginkgoleaf.ErrUnknownFormat]
// whose message lists the valid set.
func ExampleNewConfig() {
	cfg, err := ginkgoleaf.NewConfig(
		ginkgoleaf.FormatMarkdown,
		ginkgoleaf.WithColor(ginkgoleaf.ColorNever),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("format:", cfg.Format())
	fmt.Println("color is ColorNever:", cfg.Color() == ginkgoleaf.ColorNever)

	_, err = ginkgoleaf.NewConfig("bogus")
	fmt.Println(err)
	// Output:
	// format: markdown
	// color is ColorNever: true
	// unknown format "bogus"; want one of: tree|jest|markdown|github|gitlab|text|shell|tap|cucumber
}
