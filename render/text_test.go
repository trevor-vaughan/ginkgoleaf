package render_test

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/trevor-vaughan/ginkgoleaf/internal/testfx"
	"github.com/trevor-vaughan/ginkgoleaf/render"
)

var _ = scenarioTable("Text renderer renders each scenario matching the golden file",
	func(scenario string) {
		r := testfx.Report(scenario)
		renderer := render.NewText()
		var buf bytes.Buffer

		Expect(renderer.WriteAll(&buf, r)).To(Succeed(),
			"WriteAll failed for scenario %q", scenario)

		testfx.Golden(GinkgoT(), "text", scenario, buf.Bytes())
	},
)
