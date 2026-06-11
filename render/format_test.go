package render_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/trevor-vaughan/ginkgoleaf/render"
)

var _ = Describe("Format", func() {
	It("accepts every supported format", func() {
		for _, f := range []render.Format{
			render.FormatTree, render.FormatJest, render.FormatMarkdown,
			render.FormatGitHub, render.FormatGitLab, render.FormatText,
			render.FormatShell, render.FormatTAP,
		} {
			Expect(render.ValidateFormat(f)).To(Succeed(),
				"format %q must validate", f)
		}
	})

	It("returns ErrUnknownFormat for an unrecognised format", func() {
		err := render.ValidateFormat(render.Format("bogus"))
		Expect(errors.Is(err, render.ErrUnknownFormat)).To(BeTrue(),
			"expected ErrUnknownFormat, got %v", err)
	})

	It("names the offending value and the valid set in the message", func() {
		err := render.ValidateFormat(render.Format("nope"))
		Expect(err.Error()).To(ContainSubstring(`"nope"`),
			"the error should echo the bad value")
		Expect(err.Error()).To(ContainSubstring(render.FormatList()),
			"the error should list the supported formats")
	})

	It("carries no program-name prefix on the sentinel message", func() {
		Expect(render.ErrUnknownFormat.Error()).To(Equal("unknown format"),
			"the caller owns the program-name prefix, not the sentinel")
	})

	Describe("FormatList", func() {
		It("leads with tree and lists all nine formats", func() {
			Expect(render.FormatList()).To(Equal(
				"tree|jest|markdown|github|gitlab|text|shell|tap|cucumber"))
		})
	})
})
