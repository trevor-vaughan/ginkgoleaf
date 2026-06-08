package render_test

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/trevor-vaughan/ginkgoleaf/internal/testfx"
	"github.com/trevor-vaughan/ginkgoleaf/render"
)

var _ = scenarioTable("TAP renderer renders each scenario matching the golden file",
	func(scenario string) {
		r := testfx.Report(scenario)
		renderer := render.NewTAP()
		var buf bytes.Buffer

		Expect(renderer.WriteAll(&buf, r)).To(Succeed(),
			"WriteAll failed for scenario %q", scenario)

		testfx.Golden(GinkgoT(), "tap", scenario, buf.Bytes())
	},
)

var _ = Describe("TAP renderer envelope structure", func() {
	It("starts with the TAP version header and contains a valid plan line", func() {
		r := testfx.Report(testfx.ScenarioMixed)
		var buf bytes.Buffer
		Expect(render.NewTAP().WriteAll(&buf, r)).To(Succeed())
		out := buf.String()

		Expect(out).To(HavePrefix("TAP version 14\n"),
			"first line must be the TAP version declaration")
		Expect(out).To(ContainSubstring("1..4\n"),
			"plan line 1..4 must appear for the four-spec mixed scenario")
	})
})
