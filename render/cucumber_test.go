package render_test

import (
	"bytes"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/trevor-vaughan/ginkgoleaf/internal/testfx"
	"github.com/trevor-vaughan/ginkgoleaf/render"
)

var _ = scenarioTable("Cucumber renderer renders each scenario matching the golden file",
	func(scenario string) {
		r := testfx.Report(scenario)
		renderer := render.NewCucumber(false) // color off for stable goldens
		var buf bytes.Buffer

		Expect(renderer.WriteAll(&buf, r)).To(Succeed(),
			"WriteAll failed for scenario %q", scenario)

		testfx.Golden(GinkgoT(), "cucumber", scenario, buf.Bytes())
	},
)

var _ = Describe("Cucumber renderer color", func() {
	It("colors passed green, the failing leaf red, skipped cyan, and pending yellow", func() {
		var buf bytes.Buffer

		Expect(render.NewCucumber(true).WriteAll(&buf, testfx.Report(testfx.ScenarioPass))).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("\x1b[32m"), "green for passed steps")

		buf.Reset()
		Expect(render.NewCucumber(true).WriteAll(&buf, testfx.Report(testfx.ScenarioFailEqual))).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("\x1b[31m"), "red for the failing leaf step")

		buf.Reset()
		Expect(render.NewCucumber(true).WriteAll(&buf, testfx.Report(testfx.ScenarioSkip))).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("\x1b[36m"), "cyan for skipped steps")

		buf.Reset()
		Expect(render.NewCucumber(true).WriteAll(&buf, testfx.Report(testfx.ScenarioPending))).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("\x1b[33m"), "yellow for pending steps")
	})

	It("keeps containers green and only the leaf red on a failure", func() {
		var buf bytes.Buffer
		Expect(render.NewCucumber(true).WriteAll(&buf, testfx.Report(testfx.ScenarioFailEqual))).To(Succeed())
		out := buf.String()
		Expect(out).To(ContainSubstring("\x1b[32mGiven Outer\x1b[0m"),
			"a container step on a failing spec stays green (execution reached the leaf)")
		Expect(out).To(ContainSubstring("\x1b[31mThen equals fails\x1b[0m"),
			"only the leaf Then step is red")
	})
})

var _ = Describe("Cucumber renderer step references", func() {
	// A SpecRow may carry fewer ContainerLocations than containers (the model
	// documents this for hand-built or truncated reports). The renderer must
	// omit the # ref for the unlocated container without panicking, while the
	// located container and the leaf keep theirs.
	It("omits the # ref for a container that has no location", func() {
		r := render.Report{
			Suite: render.SuiteRow{Name: "S", NumSpecs: 1, NumPassed: 1, SuiteSucceeded: true},
			Specs: []render.SpecRow{{
				FullText:           []string{"Outer", "Inner", "does it"},
				ContainerHier:      []string{"Outer", "Inner"},
				ContainerLocations: []render.CodeLocation{{FileName: "o.go", LineNumber: 10}}, // Inner has none
				LeafText:           "does it",
				State:              render.StatePassed,
				Location:           render.CodeLocation{FileName: "i.go", LineNumber: 21},
			}},
		}
		var buf bytes.Buffer
		Expect(render.NewCucumber(false).WriteAll(&buf, r)).To(Succeed())

		var given, and, then string
		for _, ln := range strings.Split(buf.String(), "\n") {
			switch {
			case strings.Contains(ln, "Given Outer"):
				given = ln
			case strings.Contains(ln, "And Inner"):
				and = ln
			case strings.Contains(ln, "Then does it"):
				then = ln
			}
		}
		Expect(given).To(ContainSubstring("# o.go:10"), "located container keeps its ref")
		Expect(then).To(ContainSubstring("# i.go:21"), "the leaf keeps its ref")
		Expect(and).NotTo(ContainSubstring("#"), "a container with no location omits the ref")
	})

	It("caps the alignment column for pathologically long step text", func() {
		long := strings.Repeat("x", 400)
		r := render.Report{
			Suite: render.SuiteRow{Name: "S", NumSpecs: 1, NumPassed: 1, SuiteSucceeded: true},
			Specs: []render.SpecRow{{
				FullText:      []string{long, "y"},
				ContainerHier: []string{long},
				LeafText:      "y",
				State:         render.StatePassed,
				Location:      render.CodeLocation{FileName: "i.go", LineNumber: 2},
			}},
		}
		var buf bytes.Buffer
		Expect(render.NewCucumber(false).WriteAll(&buf, r)).To(Succeed())

		var then string
		for _, ln := range strings.Split(buf.String(), "\n") {
			if strings.Contains(ln, "Then y") {
				then = ln
			}
		}
		Expect(then).To(ContainSubstring("# i.go:2"), "the leaf keeps its ref")
		Expect(len(then)).To(BeNumerically("<", 200),
			"a short step is not padded out to a pathological sibling's length")
	})
})

var _ = Describe("Cucumber renderer edge paths", func() {
	It("renders an unrecognised state as plain text even with color on", func() {
		r := render.Report{
			Suite: render.SuiteRow{Name: "S", NumSpecs: 1},
			Specs: []render.SpecRow{{
				FullText: []string{"weird"},
				LeafText: "weird",
				State:    render.StateUnknown,
			}},
		}
		var buf bytes.Buffer
		Expect(render.NewCucumber(true).WriteAll(&buf, r)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("    Then weird\n"),
			"an unknown state falls back to uncolored text")
	})

	It("renders an empty report with zero tallies and no breakdown", func() {
		var buf bytes.Buffer
		Expect(render.NewCucumber(false).WriteAll(&buf, render.Report{Suite: render.SuiteRow{Name: "Empty"}})).To(Succeed())
		out := buf.String()
		Expect(out).To(ContainSubstring("0 scenarios\n0 steps\n"))
		Expect(out).NotTo(ContainSubstring("("), "an empty tally renders no breakdown")
	})
})
