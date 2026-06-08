package render_test

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/trevor-vaughan/ginkgoleaf/internal/testfx"
	"github.com/trevor-vaughan/ginkgoleaf/render"
)

var _ = scenarioTable("Jest renderer renders each scenario matching the golden file",
	func(scenario string) {
		r := testfx.Report(scenario)
		renderer := render.NewJest(false) // color off for stable goldens
		var buf bytes.Buffer

		Expect(renderer.WriteAll(&buf, r)).To(Succeed(),
			"WriteAll failed for scenario %q", scenario)

		testfx.Golden(GinkgoT(), "jest", scenario, buf.Bytes())
	},
)

var _ = Describe("Jest renderer", func() {
	It("emits green ANSI codes when color is enabled", func() {
		r := testfx.Report(testfx.ScenarioPass)
		renderer := render.NewJest(true)
		var buf bytes.Buffer
		Expect(renderer.WriteAll(&buf, r)).To(Succeed())
		Expect(buf.Bytes()).To(ContainSubstring("\x1b[32m"),
			"expected green ANSI escape in jest output with color enabled")
	})
})
