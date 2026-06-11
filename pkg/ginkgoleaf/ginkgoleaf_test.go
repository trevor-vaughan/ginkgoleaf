package ginkgoleaf_test

import (
	"bytes"
	"errors"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/trevor-vaughan/ginkgoleaf/pkg/ginkgoleaf"
)

// firstRegistration wires the ginkgoleaf formatter once at suite-init.
// This keeps registry.attached == true for the duration of the test run,
// which lets the double-registration It block trigger the panic without
// needing to call a second successful Register (which would attempt to add
// Ginkgo reporter hooks after tree-building, causing a Ginkgo error).
//
// Output is discarded: we want the registration side effect (the attached
// flag), not a visible rendering — `task test` drives Ginkgo with
// --json-report and runs the ginkgoleaf CLI separately to produce the
// tree below.
//
// Note: ReportAfterSuite is top-level only, so Register must live here,
// not inside a Describe body.
var _ = func() bool {
	ginkgoleaf.ResetForTest()
	return ginkgoleaf.Register(ginkgoleaf.FormatText, ginkgoleaf.WithWriter(io.Discard))
}()

var _ = Describe("Format", func() {
	It("FormatJest has the string value 'jest'", func() {
		Expect(string(ginkgoleaf.FormatJest)).To(Equal("jest"))
	})

	It("FormatCucumber has the string value 'cucumber'", func() {
		Expect(string(ginkgoleaf.FormatCucumber)).To(Equal("cucumber"))
	})

	Describe("ValidateFormat", func() {
		It("accepts a known format without error", func() {
			Expect(ginkgoleaf.ValidateFormat(ginkgoleaf.FormatJest)).To(Succeed())
		})

		It("returns ErrUnknownFormat for an unrecognised format string", func() {
			err := ginkgoleaf.ValidateFormat(ginkgoleaf.Format("bogus"))
			Expect(errors.Is(err, ginkgoleaf.ErrUnknownFormat)).To(BeTrue(),
				"expected ErrUnknownFormat, got %v", err)
		})
	})
})

var _ = Describe("NewConfig", func() {
	It("applies WithWriter and records the writer on the config", func() {
		var buf bytes.Buffer
		cfg, err := ginkgoleaf.NewConfig(ginkgoleaf.FormatText, ginkgoleaf.WithWriter(&buf))
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Writer()).To(BeIdenticalTo(&buf))
	})

	It("returns ErrUnknownFormat when an unsupported format is requested", func() {
		_, err := ginkgoleaf.NewConfig(ginkgoleaf.Format("bogus"))
		Expect(errors.Is(err, ginkgoleaf.ErrUnknownFormat)).To(BeTrue(),
			"expected ErrUnknownFormat, got %v", err)
	})
})

var _ = Describe("Register", func() {
	// registry.attached is true (set by the top-level var _ above).

	It("panics with ErrAlreadyRegistered on a second call", func() {
		// Calling Register while attached==true must panic immediately —
		// no new Ginkgo hooks are registered.
		Expect(func() {
			_ = ginkgoleaf.Register(ginkgoleaf.FormatText)
		}).To(PanicWith(ginkgoleaf.ErrAlreadyRegistered))
	})

	It("panics when given an unknown format", func() {
		// NewConfig rejects an unknown format before checking attached, so
		// this panics regardless of registration state.
		Expect(func() {
			_ = ginkgoleaf.Register(ginkgoleaf.Format("bogus"))
		}).To(Panic())
	})
})
