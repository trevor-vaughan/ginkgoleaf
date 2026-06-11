package render

import (
	"strconv"
	"strings"

	"github.com/onsi/ginkgo/v2/types"
)

// Translate converts in into the canonical Report without running the
// gomega parser. Useful when callers want to fill Matcher themselves.
func Translate(in types.Report) Report {
	return TranslateWithParser(in, nil)
}

// TranslateWithParser is Translate plus a fallback gomega parser. The
// parser is injected to avoid an import cycle (internal/parse imports
// this package).
func TranslateWithParser(in types.Report, parseFn func(string) MatcherEvent) Report {
	out := Report{
		StartTime: in.StartTime,
		EndTime:   in.EndTime,
		Suite: SuiteRow{
			Name:           sanitizeLine(in.SuiteDescription),
			Path:           in.SuitePath,
			SuiteSucceeded: in.SuiteSucceeded,
			PreRunStats: PreRunStats{
				TotalSpecs:       in.PreRunStats.TotalSpecs,
				SpecsThatWillRun: in.PreRunStats.SpecsThatWillRun,
			},
			RandomSeed:    in.SuiteConfig.RandomSeed,
			ParallelTotal: in.SuiteConfig.ParallelTotal,
		},
	}
	if len(in.SpecialSuiteFailureReasons) > 0 {
		// sanitizeLine (not block): SpecialFailure is rendered inline by the
		// text/tree renderers, so newlines and control bytes must be escaped.
		out.Suite.SpecialFailure = sanitizeLine(strings.Join(in.SpecialSuiteFailureReasons, "; "))
	}
	// Ginkgo emits SpecReports for ReportBefore/After {Suite,Each} nodes
	// alongside real specs. They are bookkeeping events, not specs the
	// user wrote, so skip them — counting them in totals or rendering
	// them in the spec list would just be noise (and would inflate the
	// total spec count by one for every report node we register
	// ourselves).
	const reportNodes = types.NodeTypeReportBeforeSuite |
		types.NodeTypeReportAfterSuite |
		types.NodeTypeReportBeforeEach |
		types.NodeTypeReportAfterEach
	// Suite-level setup/teardown nodes (BeforeSuite, AfterSuite, their
	// Synchronized variants, DeferCleanup at suite scope) are also emitted
	// as standalone SpecReports — with an empty LeafNodeText. When they
	// pass they are pure noise: rendering them as blank-description rows and
	// counting them inflates the spec tally (e.g. a "72 specs" suite reports
	// 74). Drop the passing ones. A *failing* setup node, however, carries
	// critical signal — a BeforeSuite that blows up explains why nothing
	// ran — so keep it and label it by node type, since it has no
	// description of its own.
	const suiteSetupNodes = types.NodeTypeBeforeSuite |
		types.NodeTypeSynchronizedBeforeSuite |
		types.NodeTypeAfterSuite |
		types.NodeTypeSynchronizedAfterSuite |
		types.NodeTypeCleanupAfterSuite
	for _, sr := range in.SpecReports {
		if sr.LeafNodeType.Is(reportNodes) {
			continue
		}
		if sr.LeafNodeType.Is(suiteSetupNodes) {
			if !failedState(sr.State) {
				continue
			}
			labelSetupNode(&sr)
		}
		row := translateSpec(sr)
		FillMatcher(&row, parseFn)
		out.Specs = append(out.Specs, row)
	}
	for _, s := range out.Specs {
		out.Suite.NumSpecs++
		switch s.State {
		case StatePassed:
			out.Suite.NumPassed++
		case StateFailed, StateInterrupted, StateAborted:
			// Interrupted and aborted have no separate counter; treat as
			// failed so suite totals balance with NumSpecs and the verdict
			// reflects the failure (mirrors testfx fixture behaviour).
			out.Suite.NumFailed++
		case StateSkipped:
			out.Suite.NumSkipped++
		case StatePending:
			out.Suite.NumPending++
		case StatePanicked:
			out.Suite.NumPanicked++
		}
		if s.NumAttempts > 1 && s.State == StatePassed {
			out.Suite.NumFlaked++
		}
	}
	out.Suite.NumWillRun = out.Suite.PreRunStats.SpecsThatWillRun
	return out
}

// FillMatcher populates s.Matcher from explicit ReportEntries (the
// opt-in gomega helper path) or, failing that, parses Failure.Message.
// Translate calls this after constructing s; it is exported so the CLI
// can call it after JSON decoding.
//
// parseFn allows callers to inject the parser (avoids an import cycle
// when the parse package depends on render but render imports nothing
// from parse).
func FillMatcher(s *SpecRow, parseFn func(string) MatcherEvent) {
	if s.Matcher != nil {
		return
	}
	for _, e := range s.ReportEntries {
		if e.Name == EntryName {
			if ev, ok := decodeEntry(e.Repr); ok {
				s.Matcher = &ev
				return
			}
		}
	}
	if s.Failure == nil || s.Failure.Message == "" || parseFn == nil {
		return
	}
	ev := parseFn(s.Failure.Message)
	s.Matcher = &ev
}

// EntryName is the report entry key used to identify ginkgoleaf matcher
// events. Defined here so render/ can reference it without importing
// the gomega/ sub-package (which would create an import cycle).
const EntryName = "ginkgoleaf.matcher"

// decodeEntry is the wired entry decoder. This package-var initializer is
// the no-op default; ginkgoleaf/gomega's init overrides it with the real
// JSON decoder.
var decodeEntry = func(string) (MatcherEvent, bool) { return MatcherEvent{}, false }

// SetEntryDecoder wires a decoder for structured report entries created by
// the ginkgoleaf/gomega sub-package. It must be called only during package
// initialization: the sole caller is gomega's init, and Go guarantees this
// package's var init (the no-op default above) completes before gomega's
// init runs, so there is a single ordered writer and no data race.
func SetEntryDecoder(fn func(string) (MatcherEvent, bool)) {
	if fn != nil {
		decodeEntry = fn
	}
}

func translateSpec(in types.SpecReport) SpecRow {
	out := SpecRow{
		FullText:           sanitizeLines(append(append([]string{}, in.ContainerHierarchyTexts...), in.LeafNodeText)),
		ContainerHier:      sanitizeLines(in.ContainerHierarchyTexts),
		LeafText:           sanitizeLine(in.LeafNodeText),
		State:              translateState(in.State),
		Duration:           in.RunTime,
		StartTime:          in.StartTime,
		EndTime:            in.EndTime,
		ContainerLocations: translateLocations(in.ContainerHierarchyLocations),
		Location: CodeLocation{
			FileName:   sanitizeLine(in.LeafNodeLocation.FileName),
			LineNumber: in.LeafNodeLocation.LineNumber,
		},
		IsFocused: false,
		// Captured output is untrusted, multi-line, and the most
		// attacker-influenceable data in a report; sanitizeBlock strips
		// control bytes while preserving line structure.
		CapturedOut: sanitizeBlock(in.CapturedGinkgoWriterOutput),
		CapturedErr: sanitizeBlock(in.CapturedStdOutErr),
		NumAttempts: in.NumAttempts,
	}
	if in.Failure.Message != "" || in.State == types.SpecStateFailed || in.State == types.SpecStatePanicked {
		out.Failure = &FailureRow{
			// Message/StackTrace/ProgressReport are untrusted multi-line text;
			// sanitize at this single chokepoint so every renderer (and the
			// gomega parser, which reads the sanitized Message) is protected.
			Message: sanitizeBlock(in.Failure.Message),
			Location: CodeLocation{
				FileName:   sanitizeLine(in.Failure.Location.FileName),
				LineNumber: in.Failure.Location.LineNumber,
			},
			FailureNodeContext: in.Failure.FailureNodeContext.String(),
			FailureNodeType:    in.Failure.FailureNodeType.String(),
			StackTrace:         sanitizeBlock(in.Failure.Location.FullStackTrace),
			ProgressReport:     sanitizeBlock(formatProgressReport(in.Failure.ProgressReport)),
		}
	}
	for _, e := range in.ReportEntries {
		out.ReportEntries = append(out.ReportEntries, ReportEntry{
			Name: e.Name,
			Repr: e.StringRepresentation(),
		})
	}
	return out
}

// failedState reports whether a setup node's state warrants surfacing it
// despite having no description — i.e. it failed, panicked, or timed out.
func failedState(s types.SpecState) bool {
	switch s {
	case types.SpecStateFailed, types.SpecStatePanicked,
		types.SpecStateTimedout, types.SpecStateInterrupted, types.SpecStateAborted:
		return true
	default:
		return false
	}
}

// labelSetupNode gives an empty-description suite-level node a synthetic
// description — its node type in brackets, e.g. "[BeforeSuite]" — so the
// renderers have something meaningful to print for a kept failure.
func labelSetupNode(sr *types.SpecReport) {
	label := "[" + sr.LeafNodeType.String() + "]"
	sr.LeafNodeText = label
}

func translateState(s types.SpecState) State {
	switch s {
	case types.SpecStatePassed:
		return StatePassed
	case types.SpecStateFailed, types.SpecStateTimedout:
		return StateFailed
	case types.SpecStatePending:
		return StatePending
	case types.SpecStateSkipped:
		return StateSkipped
	case types.SpecStatePanicked:
		return StatePanicked
	case types.SpecStateInterrupted:
		return StateInterrupted
	case types.SpecStateAborted:
		return StateAborted
	default:
		return StateUnknown
	}
}

// translateLocations converts Ginkgo's container CodeLocations into the
// canonical model, sanitizing each FileName at this single untrusted-input
// boundary exactly as the leaf location is. Returns nil for an empty input
// so a spec with no containers carries no slice.
func translateLocations(in []types.CodeLocation) []CodeLocation {
	if len(in) == 0 {
		return nil
	}
	out := make([]CodeLocation, len(in))
	for i, cl := range in {
		out[i] = CodeLocation{
			FileName:   sanitizeLine(cl.FileName),
			LineNumber: cl.LineNumber,
		}
	}
	return out
}

// formatProgressReport renders a ProgressReport to a compact string.
// ProgressReport.SpecGoroutine() is a method that returns the goroutine running
// the spec; its Stack is []FunctionCall, each with Function/Filename/Line fields.
func formatProgressReport(p types.ProgressReport) string {
	if p.IsZero() {
		return ""
	}
	var b strings.Builder
	if p.Message != "" {
		b.WriteString(p.Message)
		b.WriteByte('\n')
	}
	if sg := p.SpecGoroutine(); len(sg.Stack) > 0 {
		b.WriteString("spec goroutine:\n")
		for _, fc := range sg.Stack {
			b.WriteString("  " + fc.Function + "\n")
			if fc.Filename != "" {
				b.WriteString("    " + fc.Filename + ":" + strconv.Itoa(fc.Line) + "\n")
			}
		}
	}
	return b.String()
}
