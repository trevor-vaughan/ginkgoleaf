// Package testfx provides hand-built canonical Report fixtures and a golden
// file harness shared by every renderer test.
package testfx

import (
	"time"

	"github.com/trevor-vaughan/ginkgoleaf/render"
)

// FixedStart is the deterministic start time used across all fixtures so
// golden files are stable.
var FixedStart = time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC)

// Scenario names — kept in one place so every renderer test references
// the same identifiers.
const (
	ScenarioPass            = "pass"
	ScenarioFailEqual       = "fail-equal"
	ScenarioFailSubstring   = "fail-substring"
	ScenarioFailJSON        = "fail-json"
	ScenarioFailYAML        = "fail-yaml"
	ScenarioSkip            = "skip"
	ScenarioPending         = "pending"
	ScenarioFocused         = "focused"
	ScenarioPanicked        = "panicked"
	ScenarioEventuallyFail  = "eventually-fail"
	ScenarioInterrupted     = "interrupted"
	ScenarioBeforeSuiteFail = "before-suite-fail"
	ScenarioMixed           = "mixed"
	ScenarioInterleaved     = "interleaved"
	ScenarioCaptured        = "captured"
	ScenarioFlaked          = "flaked"
)

// AllScenarios returns every scenario name. Renderer tests iterate over this.
func AllScenarios() []string {
	return []string{
		ScenarioPass,
		ScenarioFailEqual,
		ScenarioFailSubstring,
		ScenarioFailJSON,
		ScenarioFailYAML,
		ScenarioSkip,
		ScenarioPending,
		ScenarioFocused,
		ScenarioPanicked,
		ScenarioEventuallyFail,
		ScenarioInterrupted,
		ScenarioBeforeSuiteFail,
		ScenarioMixed,
		ScenarioInterleaved,
		ScenarioCaptured,
		ScenarioFlaked,
	}
}

// Report returns a canonical render.Report for the given scenario name.
// It panics on an unknown scenario — test code only.
func Report(scenario string) render.Report {
	switch scenario {
	case ScenarioPass:
		return single("My Suite", passingSpec("does the thing"))
	case ScenarioFailEqual:
		return single("My Suite", failingEqualSpec())
	case ScenarioFailSubstring:
		return single("My Suite", failingSubstringSpec())
	case ScenarioFailJSON:
		return single("My Suite", failingJSONSpec())
	case ScenarioFailYAML:
		return single("My Suite", failingYAMLSpec())
	case ScenarioSkip:
		return single("My Suite", skipSpec())
	case ScenarioPending:
		return single("My Suite", pendingSpec())
	case ScenarioFocused:
		return single("My Suite", focusedSpec())
	case ScenarioPanicked:
		return single("My Suite", panickedSpec())
	case ScenarioEventuallyFail:
		return single("My Suite", eventuallyFailSpec())
	case ScenarioInterrupted:
		return single("My Suite", interruptedSpec())
	case ScenarioBeforeSuiteFail:
		return beforeSuiteFailReport()
	case ScenarioMixed:
		return mixedReport()
	case ScenarioInterleaved:
		return interleavedReport()
	case ScenarioCaptured:
		return single("My Suite", capturedSpec())
	case ScenarioFlaked:
		return flakedReport()
	default:
		panic("unknown scenario: " + scenario)
	}
}

func single(name string, s render.SpecRow) render.Report {
	r := render.Report{
		Suite: render.SuiteRow{
			Name:           name,
			Path:           "/example/my_suite_test.go",
			NumSpecs:       1,
			SuiteSucceeded: s.State == render.StatePassed || s.State == render.StateSkipped || s.State == render.StatePending,
			PreRunStats:    render.PreRunStats{TotalSpecs: 1, SpecsThatWillRun: 1},
			RandomSeed:     1717000000,
			ParallelTotal:  1,
			NumWillRun:     1,
		},
		Specs:     []render.SpecRow{s},
		StartTime: FixedStart,
		EndTime:   FixedStart.Add(50 * time.Millisecond),
	}
	bumpCounts(&r)
	return r
}

func bumpCounts(r *render.Report) {
	r.Suite.NumSpecs = len(r.Specs)
	r.Suite.NumPassed = 0
	r.Suite.NumFailed = 0
	r.Suite.NumSkipped = 0
	r.Suite.NumPending = 0
	r.Suite.NumPanicked = 0
	for _, s := range r.Specs {
		switch s.State {
		case render.StatePassed:
			r.Suite.NumPassed++
		case render.StateFailed, render.StateInterrupted, render.StateAborted:
			// Interrupted and aborted aren't separately surfaced in SuiteRow; they
			// count as failures for the purpose of suite success.
			r.Suite.NumFailed++
		case render.StateSkipped:
			r.Suite.NumSkipped++
		case render.StatePending:
			r.Suite.NumPending++
		case render.StatePanicked:
			r.Suite.NumPanicked++
		}
	}
	r.Suite.SuiteSucceeded = r.Suite.NumFailed == 0 && r.Suite.NumPanicked == 0 && r.Suite.SpecialFailure == ""
}

func passingSpec(text string) render.SpecRow {
	return render.SpecRow{
		FullText:      []string{"Outer", "Inner", text},
		ContainerHier: []string{"Outer", "Inner"},
		LeafText:      text,
		State:         render.StatePassed,
		Duration:      12 * time.Millisecond,
		StartTime:     FixedStart,
		EndTime:       FixedStart.Add(12 * time.Millisecond),
		Location: render.CodeLocation{
			FileName:   "/example/inner_test.go",
			LineNumber: 21,
		},
		NumAttempts: 1,
	}
}

func failingEqualSpec() render.SpecRow {
	s := passingSpec("equals fails")
	s.State = render.StateFailed
	s.Duration = 8 * time.Millisecond
	s.EndTime = s.StartTime.Add(s.Duration)
	s.Failure = &render.FailureRow{
		Message: "Expected\n    <int>: 1\nto equal\n    <int>: 2",
		Location: render.CodeLocation{
			FileName:   "/example/inner_test.go",
			LineNumber: 25,
		},
		FailureNodeContext: "It",
		FailureNodeType:    "It",
	}
	s.Matcher = &render.MatcherEvent{
		Kind:     render.MatcherEqual,
		Expected: 2,
		Actual:   1,
		Raw:      s.Failure.Message,
	}
	return s
}

func failingSubstringSpec() render.SpecRow {
	s := passingSpec("substring fails")
	s.State = render.StateFailed
	s.Failure = &render.FailureRow{
		Message:            "Expected\n    <string>: \"hello world\"\nto contain substring\n    <string>: \"goodbye\"",
		Location:           render.CodeLocation{FileName: "/example/inner_test.go", LineNumber: 31},
		FailureNodeContext: "It",
		FailureNodeType:    "It",
	}
	s.Matcher = &render.MatcherEvent{
		Kind:     render.MatcherContainSubstring,
		Expected: "goodbye",
		Actual:   "hello world",
		Raw:      s.Failure.Message,
	}
	return s
}

func failingJSONSpec() render.SpecRow {
	s := passingSpec("json fails")
	s.State = render.StateFailed
	s.Failure = &render.FailureRow{
		Message:            "Expected\n    <string>: {\"a\":1,\"b\":2}\nto match JSON of\n    <string>: {\"a\":1,\"b\":3}",
		Location:           render.CodeLocation{FileName: "/example/inner_test.go", LineNumber: 35},
		FailureNodeContext: "It",
		FailureNodeType:    "It",
	}
	s.Matcher = &render.MatcherEvent{
		Kind:     render.MatcherMatchJSON,
		Expected: `{"a":1,"b":3}`,
		Actual:   `{"a":1,"b":2}`,
		Raw:      s.Failure.Message,
	}
	return s
}

func failingYAMLSpec() render.SpecRow {
	s := passingSpec("yaml fails")
	s.State = render.StateFailed
	s.Failure = &render.FailureRow{
		Message:            "Expected\n    <string>: a: 1\nb: 2\nto match YAML of\n    <string>: a: 1\nb: 3",
		Location:           render.CodeLocation{FileName: "/example/inner_test.go", LineNumber: 39},
		FailureNodeContext: "It",
		FailureNodeType:    "It",
	}
	s.Matcher = &render.MatcherEvent{
		Kind:     render.MatcherMatchYAML,
		Expected: "a: 1\nb: 3",
		Actual:   "a: 1\nb: 2",
		Raw:      s.Failure.Message,
	}
	return s
}

func skipSpec() render.SpecRow {
	s := passingSpec("is skipped")
	s.State = render.StateSkipped
	// Ginkgo's Skip("reason") records the reason as a Failure on the
	// (non-failing) spec. Renderers must NOT treat this as a failure.
	s.Failure = &render.FailureRow{
		Message:            "skipping: needs network",
		Location:           render.CodeLocation{FileName: "/example/inner_test.go", LineNumber: 28},
		FailureNodeContext: "It",
		FailureNodeType:    "It",
	}
	return s
}

func pendingSpec() render.SpecRow {
	s := passingSpec("is pending")
	s.State = render.StatePending
	return s
}

func focusedSpec() render.SpecRow {
	s := passingSpec("is focused")
	s.IsFocused = true
	return s
}

func panickedSpec() render.SpecRow {
	s := passingSpec("panics")
	s.State = render.StatePanicked
	s.Failure = &render.FailureRow{
		Message:            "Test Panicked: runtime error: index out of range [3] with length 2",
		Location:           render.CodeLocation{FileName: "/example/inner_test.go", LineNumber: 44},
		FailureNodeContext: "It",
		FailureNodeType:    "It",
		StackTrace:         "goroutine 1 [running]:\n  example/inner_test.go:44 +0x42",
	}
	return s
}

func eventuallyFailSpec() render.SpecRow {
	s := passingSpec("eventually fails")
	s.State = render.StateFailed
	s.Duration = 2 * time.Second
	s.EndTime = s.StartTime.Add(s.Duration)
	s.Failure = &render.FailureRow{
		Message:            "Timed out after 2.000s.\nExpected\n    <int>: 4\nto equal\n    <int>: 5",
		Location:           render.CodeLocation{FileName: "/example/inner_test.go", LineNumber: 49},
		FailureNodeContext: "It",
		FailureNodeType:    "It",
	}
	s.Matcher = &render.MatcherEvent{
		Kind:     render.MatcherEventually,
		Expected: 5,
		Actual:   4,
		Raw:      s.Failure.Message,
		Retries: &render.RetryContext{
			Duration: 2 * time.Second,
			Polling:  10 * time.Millisecond,
			Attempts: 200,
			LastErr:  "expected 4 to equal 5",
		},
	}
	return s
}

func interruptedSpec() render.SpecRow {
	s := passingSpec("interrupted")
	s.State = render.StateInterrupted
	s.Failure = &render.FailureRow{
		Message:            "Interrupted by user",
		Location:           render.CodeLocation{FileName: "/example/inner_test.go", LineNumber: 55},
		FailureNodeContext: "It",
		FailureNodeType:    "It",
	}
	return s
}

func beforeSuiteFailReport() render.Report {
	r := single("My Suite", passingSpec("never runs"))
	r.Suite.SpecialFailure = "BeforeSuite failed"
	r.Suite.SuiteSucceeded = false
	r.Specs[0].State = render.StateSkipped
	bumpCounts(&r)
	return r
}

// leafSpec builds a passing spec at an arbitrary container path. Used to
// compose reports whose specs share containers out of order.
func leafSpec(container []string, text string) render.SpecRow {
	return render.SpecRow{
		FullText:      append(append([]string{}, container...), text),
		ContainerHier: append([]string{}, container...),
		LeafText:      text,
		State:         render.StatePassed,
		Duration:      time.Millisecond,
		StartTime:     FixedStart,
		EndTime:       FixedStart.Add(time.Millisecond),
		Location:      render.CodeLocation{FileName: "/example/interleaved_test.go", LineNumber: 21},
		NumAttempts:   1,
	}
}

// interleavedReport models a suite run with --randomize-all: specs from one
// container arrive non-contiguously. "Alpha" re-enters after "Beta", and the
// nested "reads" container under Alpha re-enters after "writes". A faithful
// structural render collapses each container to one node; this fixture locks
// that across every renderer's golden so the duplication regression cannot
// silently return.
func interleavedReport() render.Report {
	specs := []render.SpecRow{
		leafSpec([]string{"Alpha", "reads"}, "returns the stored value"),
		leafSpec([]string{"Beta"}, "stands on its own"),
		leafSpec([]string{"Alpha", "writes"}, "persists the value"),
		leafSpec([]string{"Alpha", "reads"}, "falls back to the default"),
		leafSpec([]string{"Beta"}, "still stands on its own"),
	}
	r := render.Report{
		Suite: render.SuiteRow{
			Name:          "Interleaved Suite",
			Path:          "/example/interleaved_test.go",
			RandomSeed:    1717000000,
			ParallelTotal: 1,
			PreRunStats:   render.PreRunStats{TotalSpecs: 5, SpecsThatWillRun: 5},
			NumWillRun:    5,
		},
		Specs:     specs,
		StartTime: FixedStart,
		EndTime:   FixedStart.Add(100 * time.Millisecond),
	}
	bumpCounts(&r)
	return r
}

func capturedSpec() render.SpecRow {
	s := passingSpec("captures output")
	s.State = render.StateFailed
	s.CapturedOut = "log line 1\nlog line 2"
	s.CapturedErr = "stderr noise"
	s.Failure = &render.FailureRow{
		Message:            "Expected\n    <int>: 1\nto equal\n    <int>: 2",
		Location:           render.CodeLocation{FileName: "/example/inner_test.go", LineNumber: 60},
		FailureNodeContext: "It",
		FailureNodeType:    "It",
		ProgressReport:     "spec goroutine:\n  main.It.func1\n    /example/inner_test.go:60",
	}
	s.Matcher = &render.MatcherEvent{Kind: render.MatcherEqual, Expected: 2, Actual: 1, Raw: s.Failure.Message}
	return s
}

func flakedReport() render.Report {
	s := passingSpec("flakes then passes")
	s.NumAttempts = 3
	r := single("My Suite", s)
	r.Suite.NumFlaked = 1
	return r
}

func mixedReport() render.Report {
	specs := []render.SpecRow{
		passingSpec("a passes"),
		failingEqualSpec(),
		skipSpec(),
		pendingSpec(),
	}
	r := render.Report{
		Suite: render.SuiteRow{
			Name:          "Mixed Suite",
			Path:          "/example/mixed_test.go",
			RandomSeed:    1717000000,
			ParallelTotal: 1,
			PreRunStats:   render.PreRunStats{TotalSpecs: 4, SpecsThatWillRun: 4},
			NumWillRun:    4,
		},
		Specs:     specs,
		StartTime: FixedStart,
		EndTime:   FixedStart.Add(100 * time.Millisecond),
	}
	bumpCounts(&r)
	return r
}
