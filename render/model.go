package render

import (
	"strconv"
	"time"
)

// Report is the canonical, Ginkgo-agnostic representation of a finished suite.
// Renderers consume only this type; translate.go is the single chokepoint that
// builds it from types.Report.
type Report struct {
	Suite     SuiteRow
	Specs     []SpecRow
	StartTime time.Time
	EndTime   time.Time
}

// SuiteRow holds suite-level metadata and totals.
type SuiteRow struct {
	Name           string
	Path           string
	NumSpecs       int
	NumPassed      int
	NumFailed      int
	NumSkipped     int
	NumPending     int
	NumPanicked    int
	NumFlaked      int
	NumWillRun     int
	PreRunStats    PreRunStats
	SuiteSucceeded bool
	SpecialFailure string // non-empty when BeforeSuite/AfterSuite/etc. failed
	RandomSeed     int64
	ParallelTotal  int
}

// PreRunStats are computed by Ginkgo before specs execute.
type PreRunStats struct {
	TotalSpecs       int
	SpecsThatWillRun int
}

// SpecRow holds the data for a single spec, in renderer-ready form.
type SpecRow struct {
	FullText      []string // ["Outer container", "Inner container", "It does X"]
	State         State
	Duration      time.Duration
	StartTime     time.Time
	EndTime       time.Time
	Location      CodeLocation
	IsFocused     bool
	Failure       *FailureRow
	Matcher       *MatcherEvent
	CapturedOut   string
	CapturedErr   string
	ReportEntries []ReportEntry
	NumAttempts   int
	ContainerHier []string // FullText minus the leaf "It"
	LeafText      string   // the "It" / leaf description
}

// FailureRow is the data we need to render a failure.
type FailureRow struct {
	Message            string
	Location           CodeLocation
	FailureNodeContext string // "It", "BeforeEach", "AfterSuite", etc.
	FailureNodeType    string // ginkgo's node type string
	StackTrace         string
	ProgressReport     string // ginkgo's interrupt/progress dump if any
}

// CodeLocation pinpoints a file:line:column in the user's source.
type CodeLocation struct {
	FileName   string
	LineNumber int
}

// String formats as "file:line" — used for clickable links and TAP diagnostics.
func (cl CodeLocation) String() string {
	if cl.FileName == "" {
		return ""
	}
	if cl.LineNumber == 0 {
		return cl.FileName
	}
	return cl.FileName + ":" + strconv.Itoa(cl.LineNumber)
}

// MatcherKind is the family of a gomega matcher we recognize.
type MatcherKind int

// MatcherKind constants.
const (
	MatcherUnknown MatcherKind = iota
	MatcherGeneric             // failure with no recognizable matcher; render Raw verbatim
	MatcherEqual
	MatcherContainSubstring
	MatcherMatchJSON
	MatcherMatchYAML
	MatcherEventually
	MatcherConsistently
)

// MatcherEvent is the structured matcher payload populated either by the
// opt-in gomega helper (via AddReportEntry) or the best-effort parser.
type MatcherEvent struct {
	Kind     MatcherKind
	Expected any
	Actual   any
	Retries  *RetryContext
	Raw      string // original gomega failure string; always preserved
}

// RetryContext is set for Eventually/Consistently failures.
type RetryContext struct {
	Duration time.Duration
	Polling  time.Duration
	Attempts int
	LastErr  string
}

// ReportEntry is a generic per-spec attachment surfaced through Ginkgo's
// AddReportEntry mechanism. We do not interpret its value; the renderer
// shows it as a labelled block under the failing spec.
type ReportEntry struct {
	Name string
	Repr string // human-readable rendering of Value
}
