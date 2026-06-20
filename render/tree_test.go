package render_test

import (
	"bytes"
	"strings"
	"time"

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

var _ = Describe("Tree roll-up summary", func() {
	It("ends with a verdict line summarising the suite", func() {
		r := render.Report{
			Suite: render.SuiteRow{
				Name: "S", Path: "/s",
				NumSpecs: 4, NumPassed: 1, NumFailed: 1, NumSkipped: 1, NumPending: 1,
				SuiteSucceeded: false,
			},
			Specs: []render.SpecRow{
				{LeafText: "a", State: render.StatePassed},
				{LeafText: "b", State: render.StateFailed, Failure: &render.FailureRow{Message: "boom"}},
				{LeafText: "c", State: render.StateSkipped},
				{LeafText: "d", State: render.StatePending},
			},
			StartTime: testfx.FixedStart,
			EndTime:   testfx.FixedStart.Add(100 * time.Millisecond),
		}
		var buf bytes.Buffer
		Expect(render.NewTree(false).WriteAll(&buf, r)).To(Succeed())
		Expect(buf.String()).To(HaveSuffix(
			"Summary: 1 passed | 1 failed | 1 skipped | 1 pending in 100ms — FAILED\n"))
	})

	It("renders PASSED for a succeeding suite", func() {
		r := render.Report{
			Suite: render.SuiteRow{
				Name: "S", Path: "/s",
				NumSpecs: 1, NumPassed: 1, SuiteSucceeded: true,
			},
			Specs:     []render.SpecRow{{LeafText: "a", State: render.StatePassed}},
			StartTime: testfx.FixedStart,
			EndTime:   testfx.FixedStart.Add(50 * time.Millisecond),
		}
		var buf bytes.Buffer
		Expect(render.NewTree(false).WriteAll(&buf, r)).To(Succeed())
		Expect(buf.String()).To(HaveSuffix("Summary: 1 passed in 50ms — PASSED\n"))
	})

	It("omits flaked counts from the summary even when the header shows them", func() {
		r := render.Report{
			Suite: render.SuiteRow{
				Name: "S", Path: "/s",
				NumSpecs: 1, NumPassed: 1, NumFlaked: 1, SuiteSucceeded: true,
			},
			Specs:     []render.SpecRow{{LeafText: "a", State: render.StatePassed}},
			StartTime: testfx.FixedStart,
			EndTime:   testfx.FixedStart.Add(50 * time.Millisecond),
		}
		var buf bytes.Buffer
		Expect(render.NewTree(false).WriteAll(&buf, r)).To(Succeed())
		out := buf.String()
		summary := out[strings.Index(out, "Summary:"):]
		Expect(summary).NotTo(ContainSubstring("flaked"))
		Expect(summary).To(Equal("Summary: 1 passed in 50ms — PASSED\n"))
	})

	It("colours the verdict red when ANSI is enabled and the suite failed", func() {
		r := render.Report{
			Suite:     render.SuiteRow{Name: "S", Path: "/s", NumSpecs: 1, NumFailed: 1, SuiteSucceeded: false},
			Specs:     []render.SpecRow{{LeafText: "a", State: render.StateFailed, Failure: &render.FailureRow{Message: "x"}}},
			StartTime: testfx.FixedStart,
			EndTime:   testfx.FixedStart.Add(10 * time.Millisecond),
		}
		var buf bytes.Buffer
		Expect(render.NewTree(true).WriteAll(&buf, r)).To(Succeed())
		Expect(buf.String()).To(ContainSubstring("\x1b[31mFAILED\x1b[0m"))
	})
})

var _ = Describe("Tree grand total", func() {
	It("rolls up totals across multiple suites with an overall verdict", func() {
		suites := []render.SuiteRow{
			{NumPassed: 9, NumFailed: 1, SuiteSucceeded: false},
			{NumPassed: 9, NumFailed: 1, NumSkipped: 1, SuiteSucceeded: false},
			{NumPassed: 0, SuiteSucceeded: true},
		}
		var buf bytes.Buffer
		Expect(render.WriteTreeGrandTotal(&buf, suites, 1200*time.Millisecond, false)).To(Succeed())
		Expect(buf.String()).To(Equal(
			"\nTotal: 3 suites | 18 passed | 2 failed | 1 skipped in 1.20s — FAILED\n"))
	})

	It("reports PASSED when every suite succeeded and counts one suite singular", func() {
		suites := []render.SuiteRow{{NumPassed: 3, SuiteSucceeded: true}}
		var buf bytes.Buffer
		Expect(render.WriteTreeGrandTotal(&buf, suites, 200*time.Millisecond, false)).To(Succeed())
		Expect(buf.String()).To(Equal("\nTotal: 1 suite | 3 passed in 200ms — PASSED\n"))
	})
})
