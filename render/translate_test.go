package render_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"

	"github.com/trevor-vaughan/ginkgoleaf/render"
)

var _ = Describe("Translate", func() {
	Describe("a simple passing suite", func() {
		var (
			start = time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC)
			end   = start.Add(50 * time.Millisecond)
		)

		It("produces correct suite-level metadata", func() {
			in := types.Report{
				SuiteDescription: "My Suite",
				SuitePath:        "/some/path",
				SuiteSucceeded:   true,
				StartTime:        start,
				EndTime:          end,
				RunTime:          end.Sub(start),
				PreRunStats:      types.PreRunStats{TotalSpecs: 1, SpecsThatWillRun: 1},
				SuiteConfig:      types.SuiteConfig{RandomSeed: 42, ParallelTotal: 1},
				SpecReports: []types.SpecReport{
					{
						ContainerHierarchyTexts: []string{"Outer", "Inner"},
						LeafNodeText:            "does the thing",
						LeafNodeLocation: types.CodeLocation{
							FileName:   "thing_test.go",
							LineNumber: 17,
						},
						State:     types.SpecStatePassed,
						StartTime: start,
						EndTime:   end,
						RunTime:   end.Sub(start),
					},
				},
			}
			got := render.Translate(in)

			Expect(got.Suite.Name).To(Equal("My Suite"))
			Expect(got.Suite.NumSpecs).To(Equal(1))
			Expect(got.Suite.NumPassed).To(Equal(1))
			Expect(got.Suite.SuiteSucceeded).To(BeTrue())
		})

		It("translates spec fields correctly", func() {
			in := types.Report{
				SuiteDescription: "My Suite",
				SuiteSucceeded:   true,
				SpecReports: []types.SpecReport{
					{
						ContainerHierarchyTexts: []string{"Outer", "Inner"},
						LeafNodeText:            "does the thing",
						LeafNodeLocation: types.CodeLocation{
							FileName:   "thing_test.go",
							LineNumber: 17,
						},
						State:   types.SpecStatePassed,
						RunTime: 50 * time.Millisecond,
					},
				},
			}
			got := render.Translate(in)

			Expect(got.Specs).To(HaveLen(1))
			spec := got.Specs[0]
			Expect(spec.State).To(Equal(render.StatePassed))
			Expect(spec.LeafText).To(Equal("does the thing"))
			Expect(spec.ContainerHier).To(Equal([]string{"Outer", "Inner"}))
			Expect(spec.FullText).To(Equal([]string{"Outer", "Inner", "does the thing"}))
			Expect(spec.Location.FileName).To(Equal("thing_test.go"))
			Expect(spec.Location.LineNumber).To(Equal(17))
			Expect(spec.Duration).To(Equal(50 * time.Millisecond))
		})
	})

	Describe("a failed suite", func() {
		It("counts the failed spec and preserves the failure message and location", func() {
			failMsg := "Expected\n    <int>: 1\nto equal\n    <int>: 2"
			in := types.Report{
				SuiteDescription: "S",
				SuiteSucceeded:   false,
				SpecReports: []types.SpecReport{
					{
						ContainerHierarchyTexts: []string{"C"},
						LeafNodeText:            "fails",
						LeafNodeLocation:        types.CodeLocation{FileName: "x_test.go", LineNumber: 5},
						State:                   types.SpecStateFailed,
						Failure: types.Failure{
							Message:  failMsg,
							Location: types.CodeLocation{FileName: "x_test.go", LineNumber: 9},
						},
					},
				},
			}
			got := render.Translate(in)

			Expect(got.Suite.NumFailed).To(Equal(1))
			Expect(got.Suite.SuiteSucceeded).To(BeFalse())
			Expect(got.Specs).To(HaveLen(1))
			Expect(got.Specs[0].Failure).NotTo(BeNil())
			Expect(got.Specs[0].Failure.Message).To(Equal(failMsg))
			Expect(got.Specs[0].Failure.Location.String()).To(Equal("x_test.go:9"))
		})
	})

	Describe("suite-level setup/teardown nodes", func() {
		It("drops passing BeforeSuite and DeferCleanup nodes and excludes them from counts", func() {
			in := types.Report{
				SuiteDescription: "S",
				SpecReports: []types.SpecReport{
					{LeafNodeType: types.NodeTypeBeforeSuite, State: types.SpecStatePassed},
					{
						ContainerHierarchyTexts: []string{"C"},
						LeafNodeText:            "real spec",
						LeafNodeType:            types.NodeTypeIt,
						State:                   types.SpecStatePassed,
					},
					{LeafNodeType: types.NodeTypeCleanupAfterSuite, State: types.SpecStatePassed},
				},
			}
			got := render.Translate(in)

			Expect(got.Specs).To(HaveLen(1), "only the real It spec should remain")
			Expect(got.Specs[0].LeafText).To(Equal("real spec"))
			Expect(got.Suite.NumSpecs).To(Equal(1), "setup/teardown nodes must not inflate the spec count")
			Expect(got.Suite.NumPassed).To(Equal(1))
		})

		It("keeps a FAILING suite-level setup node and labels it by node type", func() {
			in := types.Report{
				SuiteDescription: "S",
				SpecReports: []types.SpecReport{
					{
						LeafNodeType: types.NodeTypeBeforeSuite,
						State:        types.SpecStateFailed,
						Failure: types.Failure{
							Message:  "setup blew up",
							Location: types.CodeLocation{FileName: "s_test.go", LineNumber: 3},
						},
					},
				},
			}
			got := render.Translate(in)

			Expect(got.Specs).To(HaveLen(1), "a failing setup node carries real signal and must be surfaced")
			Expect(got.Specs[0].LeafText).To(Equal("[BeforeSuite]"),
				"an empty-description setup node should be labelled by its node type")
			Expect(got.Specs[0].FullText).To(Equal([]string{"[BeforeSuite]"}))
			Expect(got.Suite.NumFailed).To(Equal(1))
		})
	})

	Describe("skip, pending, and panicked specs", func() {
		It("correctly bumps the per-state counters", func() {
			in := types.Report{
				SuiteDescription: "S",
				SpecReports: []types.SpecReport{
					{LeafNodeText: "skip", State: types.SpecStateSkipped},
					{LeafNodeText: "pend", State: types.SpecStatePending},
					{LeafNodeText: "boom", State: types.SpecStatePanicked, Failure: types.Failure{Message: "boom"}},
				},
			}
			got := render.Translate(in)

			Expect(got.Suite.NumSkipped).To(Equal(1), "NumSkipped")
			Expect(got.Suite.NumPending).To(Equal(1), "NumPending")
			Expect(got.Suite.NumPanicked).To(Equal(1), "NumPanicked")
		})
	})
})

func TestTranslateSanitizesFailureBody(t *testing.T) {
	const esc = "\x1b"
	in := types.Report{
		SuiteDescription:           "Body Suite",
		SpecialSuiteFailureReasons: []string{"setup blew up\x1b[31m\rred", "second\nreason"},
		SpecReports: types.SpecReports{
			{
				State:                      types.SpecStateFailed,
				ContainerHierarchyTexts:    []string{"Group"},
				LeafNodeText:               "fails",
				CapturedGinkgoWriterOutput: "ginkgo out\x1b[2Jcleared\rover",
				CapturedStdOutErr:          "std out\x1b[31mred",
				Failure: types.Failure{
					Message: "boom\x1b[0Ksection_start:0:evil\rINJECT",
					Location: types.CodeLocation{
						FileName:       "f.go",
						LineNumber:     1,
						FullStackTrace: "frame one\x1b[31m\rframe two",
					},
				},
			},
		},
	}
	r := render.Translate(in)
	spec := r.Specs[0]

	check := func(field, got string) {
		if strings.Contains(got, esc) {
			t.Fatalf("%s retained ESC: %q", field, got)
		}
		if strings.Contains(got, "\r") {
			t.Fatalf("%s retained bare CR: %q", field, got)
		}
	}
	check("Failure.Message", spec.Failure.Message)
	check("Failure.StackTrace", spec.Failure.StackTrace)
	check("CapturedOut", spec.CapturedOut)
	check("CapturedErr", spec.CapturedErr)
	check("Suite.SpecialFailure", r.Suite.SpecialFailure)
	// SpecialFailure is rendered inline; newlines from a multi-reason join
	// must be escaped so they cannot break the line.
	if strings.Contains(r.Suite.SpecialFailure, "\n") {
		t.Fatalf("Suite.SpecialFailure retained raw newline: %q", r.Suite.SpecialFailure)
	}
}

func TestTranslateSanitizesInjection(t *testing.T) {
	in := types.Report{
		SuiteDescription: "Inj Suite\x1b[2J\rwiped",
		SuiteSucceeded:   false,
		SpecReports: types.SpecReports{
			{
				State:                   types.SpecStateFailed,
				ContainerHierarchyTexts: []string{"Group\x1b[0Ksection_start:9:evil\rForged\n::error::ghinject", "Inner"},
				ContainerHierarchyLocations: []types.CodeLocation{
					{FileName: "c\x1b[31m\revil_test.go", LineNumber: 7},
					{FileName: "inner_test.go", LineNumber: 9},
				},
				LeafNodeText:     "fails\nnot ok 99 - forged",
				LeafNodeLocation: types.CodeLocation{FileName: "f\n::warning::x.go", LineNumber: 1},
				Failure: types.Failure{
					Message:  "boom",
					Location: types.CodeLocation{FileName: "f.go", LineNumber: 1},
				},
			},
		},
	}
	r := render.Translate(in)

	// Suite.Name and container-location FileNames cross the same untrusted
	// boundary as the spec texts: cucumber prints the name on its Feature:
	// line and each container location as a "# file:line" ref, so control
	// bytes must be stripped at the translate chokepoint.
	if strings.ContainsAny(r.Suite.Name, "\x1b\r") {
		t.Fatalf("Suite.Name retained control bytes: %q", r.Suite.Name)
	}
	locs := r.Specs[0].ContainerLocations
	if len(locs) != 2 {
		t.Fatalf("ContainerLocations: want 2 entries parallel to ContainerHier, got %d", len(locs))
	}
	if strings.ContainsAny(locs[0].FileName, "\x1b\r") {
		t.Fatalf("ContainerLocations[0].FileName retained control bytes: %q", locs[0].FileName)
	}
	if empty := render.Translate(types.Report{SpecReports: types.SpecReports{{LeafNodeText: "x"}}}); empty.Specs[0].ContainerLocations != nil {
		t.Fatalf("ContainerLocations: want nil for a spec without container locations, got %#v", empty.Specs[0].ContainerLocations)
	}

	var gh bytes.Buffer
	if err := render.NewGitHub().WriteAll(&gh, r); err != nil {
		t.Fatal(err)
	}
	var stopTok string
	for _, line := range strings.Split(gh.String(), "\n") {
		// The failure body is wrapped in ::stop-commands::<tok> … ::<tok>::;
		// lines inside that region are not parsed as commands by the runner.
		if stopTok != "" {
			if line == "::"+stopTok+"::" {
				stopTok = ""
			}
			continue
		}
		if rest, ok := strings.CutPrefix(line, "::stop-commands::"); ok {
			stopTok = rest
			continue
		}
		if strings.HasPrefix(line, "::") &&
			!strings.HasPrefix(line, "::group::") &&
			!strings.HasPrefix(line, "::endgroup::") &&
			!strings.HasPrefix(line, "::error ") &&
			!strings.HasPrefix(line, "::notice ") {
			t.Fatalf("github: injected workflow command at column 0: %q", line)
		}
	}

	var gl bytes.Buffer
	if err := render.NewGitLab(func() int64 { return 1 }, false).WriteAll(&gl, r); err != nil {
		t.Fatal(err)
	}
	// One failed spec yields exactly two legitimate section markers: the
	// per-top-container spec group and the trailing "Failures" section. A
	// forged "section_start" smuggled in through untrusted text would push
	// this above two, so anything other than two means injection got through.
	// Count the full control sequence, not the plain substring: the forged
	// payload carries the real ESC+"[0Ksection_start" prefix, so matching on
	// the bare word would not distinguish it from renderer-emitted markers.
	const sectionCtl = "\x1b[0Ksection_start"
	if got := strings.Count(gl.String(), sectionCtl); got != 2 {
		t.Fatalf("gitlab: expected 2 legitimate section markers, got %d (forged marker leaked):\n%q", got, gl.String())
	}
	// Defense in depth: no raw ESC from untrusted text should survive — every ESC
	// in the output belongs to a renderer-emitted control/color sequence. With the
	// payload's ESC stripped by sanitizeLine, the literal "section_start:9:evil"
	// must appear only as escaped text, never as a control line.
	if strings.Contains(gl.String(), "\x1b[0Ksection_start:9:evil") {
		t.Fatalf("gitlab: forged section_start control sequence survived:\n%q", gl.String())
	}

	var tp bytes.Buffer
	if err := render.NewTAP().WriteAll(&tp, r); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(tp.String(), "\nnot ok 99 - forged") {
		t.Fatalf("tap: forged test point present:\n%s", tp.String())
	}
}
