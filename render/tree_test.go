package render_test

import (
	"bytes"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/trevor-vaughan/ginkgoleaf/internal/testfx"
	"github.com/trevor-vaughan/ginkgoleaf/render"
)

var _ = Describe("Tree renderer", func() {
	It("merges non-adjacent re-entries of the same container into one subtree", func() {
		// With --randomize-all, specs from one container are interleaved
		// with others. The tree is a structural hierarchy, so a container
		// must render once regardless of run order, gathering all its specs.
		r := render.Report{
			Suite: render.SuiteRow{
				Name: "S", Path: "/s",
				NumSpecs: 3, NumPassed: 3, SuiteSucceeded: true,
			},
			Specs: []render.SpecRow{
				{ContainerHier: []string{"Alpha"}, LeafText: "one", State: render.StatePassed},
				{ContainerHier: []string{"Beta"}, LeafText: "two", State: render.StatePassed},
				{ContainerHier: []string{"Alpha"}, LeafText: "three", State: render.StatePassed},
			},
			StartTime: testfx.FixedStart,
			EndTime:   testfx.FixedStart,
		}
		var buf bytes.Buffer
		Expect(render.NewTree(false).WriteAll(&buf, r)).To(Succeed())
		out := buf.String()
		Expect(strings.Count(out, "Alpha")).To(Equal(1),
			"the Alpha container should render once with both its specs merged under it")
		Expect(out).To(ContainSubstring("one"))
		Expect(out).To(ContainSubstring("three"))
	})

	It("emits green ANSI codes for passing specs when color is enabled", func() {
		r := testfx.Report(testfx.ScenarioPass)
		var buf bytes.Buffer
		Expect(render.NewTree(true).WriteAll(&buf, r)).To(Succeed())
		Expect(buf.Bytes()).To(ContainSubstring("\x1b[32m"),
			"expected green ANSI escape for the passing-leaf glyph")
	})
})

var _ = scenarioTable("Tree renderer renders each scenario matching the golden file",
	func(scenario string) {
		r := testfx.Report(scenario)
		var buf bytes.Buffer
		Expect(render.NewTree(false).WriteAll(&buf, r)).To(Succeed())
		testfx.Golden(GinkgoT(), "tree", scenario, buf.Bytes())
	},
)
