package render_test

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/trevor-vaughan/ginkgoleaf/internal/testfx"
	"github.com/trevor-vaughan/ginkgoleaf/render"
)

var _ = Describe("GitLab renderer color resolution", func() {
	It("emits no color escapes when color is disabled, but keeps CI section markers", func() {
		r := testfx.Report(testfx.ScenarioMixed)
		var buf bytes.Buffer
		Expect(render.NewGitLab(fixedClock, false).WriteAll(&buf, r)).To(Succeed())
		out := buf.String()

		Expect(out).To(ContainSubstring("section_start"),
			"section_start/section_end markers are GitLab CI structure, not color")
		Expect(out).NotTo(MatchRegexp("\x1b\\[3[0-9]m"),
			"--color=never must suppress foreground color escapes")
		Expect(out).NotTo(ContainSubstring("\x1b[2m"),
			"--color=never must suppress the dim 'at <loc>' escape")
	})

	It("emits color escapes when color is enabled", func() {
		r := testfx.Report(testfx.ScenarioMixed)
		var buf bytes.Buffer
		Expect(render.NewGitLab(fixedClock, true).WriteAll(&buf, r)).To(Succeed())
		Expect(buf.String()).To(MatchRegexp("\x1b\\[3[0-9]m"),
			"--color=always must keep colored glyphs/verdict")
	})
})
