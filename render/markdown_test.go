package render_test

import (
	"bytes"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"

	"github.com/trevor-vaughan/ginkgoleaf/internal/testfx"
	"github.com/trevor-vaughan/ginkgoleaf/render"
)

var _ = scenarioTable("Markdown renderer renders each scenario matching the golden file",
	func(scenario string) {
		r := testfx.Report(scenario)
		renderer := render.NewMarkdown()
		var buf bytes.Buffer

		Expect(renderer.WriteAll(&buf, r)).To(Succeed(),
			"WriteAll failed for scenario %q", scenario)

		testfx.Golden(GinkgoT(), "markdown", scenario, buf.Bytes())
	},
)

var _ = Describe("Markdown renderer", func() {
	It("merges a non-adjacent container re-entry within a group into one node", func() {
		// Same structural-hierarchy guarantee as the box-drawing tree: under
		// --randomize-all a nested container arrives interleaved, and the
		// Markdown nested list must still render it once, not duplicated.
		r := render.Report{
			Suite: render.SuiteRow{
				Name: "S", Path: "/s",
				NumSpecs: 3, NumPassed: 3, SuiteSucceeded: true,
			},
			Specs: []render.SpecRow{
				{ContainerHier: []string{"Top", "Alpha"}, LeafText: "one", State: render.StatePassed},
				{ContainerHier: []string{"Top", "Beta"}, LeafText: "two", State: render.StatePassed},
				{ContainerHier: []string{"Top", "Alpha"}, LeafText: "three", State: render.StatePassed},
			},
			StartTime: testfx.FixedStart,
			EndTime:   testfx.FixedStart,
		}
		var buf bytes.Buffer
		Expect(render.NewMarkdown().WriteAll(&buf, r)).To(Succeed())
		out := buf.String()
		Expect(strings.Count(out, "**Alpha**")).To(Equal(1),
			"the Alpha container should render once with both its specs merged under it")
		Expect(out).To(ContainSubstring("one"))
		Expect(out).To(ContainSubstring("three"))
	})
})

func TestMarkdownFenceCannotBeBroken(t *testing.T) {
	in := types.Report{
		SuiteDescription: "S",
		SpecReports: types.SpecReports{{
			State:                   types.SpecStateFailed,
			ContainerHierarchyTexts: []string{"Group"},
			LeafNodeText:            "fails",
			Failure: types.Failure{
				Message:  "before\n```\n<img src=x onerror=alert(1)>\nafter",
				Location: types.CodeLocation{FileName: "f.go", LineNumber: 1},
			},
		}},
	}
	var buf bytes.Buffer
	if err := render.NewMarkdown().WriteAll(&buf, render.Translate(in)); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	// The body contains a 3-backtick run, so the surrounding fence must be at
	// least 4 backticks; otherwise the inner ``` would terminate the block
	// early and the <img> would render as live HTML.
	if !strings.Contains(out, "````text\n") {
		t.Fatalf("expected a >=4-backtick fence around the failure body, got:\n%s", out)
	}
	if strings.Contains(out, "```text\n") && !strings.Contains(out, "````text\n") {
		t.Fatalf("body fence is only 3 backticks; inner ``` can break out:\n%s", out)
	}
}
