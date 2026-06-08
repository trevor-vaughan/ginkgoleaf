package render

import (
	"fmt"
	"io"
	"strings"
)

// TAPRenderer emits TAP 14:
//
//	TAP version 14
//	1..N
//	ok 1 - desc
//	not ok 2 - desc
//	  ---
//	  message: |-
//	    Expected ...
//	  severity: fail
//	  at:
//	    file: ...
//	    line: ...
//	  ...
//
// See https://testanything.org/tap-version-14-specification.html
type TAPRenderer struct{}

// NewTAP returns a new TAPRenderer.
func NewTAP() *TAPRenderer { return &TAPRenderer{} }

// WriteAll renders the report as TAP 14.
func (*TAPRenderer) WriteAll(w io.Writer, r Report) error {
	ew := &errWriter{w: w}
	ew.writeString("TAP version 14\n")
	ew.printf("1..%d\n", len(r.Specs))
	for i, s := range r.Specs {
		tapWriteSpec(ew, i+1, s)
	}
	if r.Suite.SpecialFailure != "" {
		ew.printf("# Suite-level failure: %s\n", r.Suite.SpecialFailure)
	}
	ew.printf("# total %d, pass %d, fail %d, skip %d, pending %d, panic %d\n",
		r.Suite.NumSpecs, r.Suite.NumPassed, r.Suite.NumFailed,
		r.Suite.NumSkipped, r.Suite.NumPending, r.Suite.NumPanicked)
	if r.Suite.NumFlaked > 0 {
		ew.printf("# flaked %d\n", r.Suite.NumFlaked)
	}
	return ew.Err()
}

func tapWriteSpec(ew *errWriter, n int, s SpecRow) {
	desc := strings.Join(s.FullText, " > ")
	directive := ""
	switch s.State {
	case StateSkipped:
		directive = " # SKIP"
	case StatePending:
		// TAP directive for a not-yet-implemented spec (TAP14, "Directives");
		// this is emitted protocol text, not a source annotation.
		directive = " # TODO pending" // DevSkim: ignore DS176209
	}
	header := "ok"
	switch s.State {
	case StateFailed, StatePanicked, StateInterrupted, StateAborted:
		header = "not ok"
	}
	ew.printf("%s %d - %s%s\n", header, n, desc, directive)
	// TAP has no focus concept; surface it as a diagnostic comment so the
	// footgun (a stray FDescribe/FIt) is visible, like the other formats.
	if s.IsFocused {
		ew.writeString("# focused\n")
	}
	// Only real failures carry a YAML diagnostics block. A skipped or
	// pending spec's Skip("…") reason also lands in Failure, but a
	// "severity: fail" block on an "ok … # SKIP" line is contradictory.
	if !isFailure(s) {
		return
	}
	// YAML diagnostics block.
	ew.writeString("  ---\n")
	ew.printf("  message: |-\n%s", indentBlock("    ", failureMessage(s)))
	ew.printf("  severity: %s\n", tapSeverity(s.State))
	if s.Failure.Location.FileName != "" {
		ew.printf("  at:\n    file: %s\n    line: %d\n",
			yamlEscape(s.Failure.Location.FileName), s.Failure.Location.LineNumber)
	}
	if st := strings.TrimRight(s.Failure.StackTrace, "\n"); st != "" {
		ew.printf("  stacktrace: |-\n%s", indentBlock("    ", st))
	}
	if s.Matcher != nil && s.Matcher.Retries != nil {
		rc := s.Matcher.Retries
		ew.printf("  eventually:\n    duration: %s\n    attempts: %d\n", rc.Duration, rc.Attempts)
	}
	if s.CapturedOut != "" {
		ew.printf("  stdout: |-\n%s", indentBlock("    ", s.CapturedOut))
	}
	if s.CapturedErr != "" {
		ew.printf("  stderr: |-\n%s", indentBlock("    ", s.CapturedErr))
	}
	if s.Failure.ProgressReport != "" {
		ew.printf("  progress: |-\n%s", indentBlock("    ", s.Failure.ProgressReport))
	}
	ew.writeString("  ...\n")
}

func tapSeverity(s State) string {
	switch s {
	case StatePanicked:
		return "panic"
	case StateInterrupted:
		return "interrupt"
	case StateAborted:
		return "abort"
	default:
		return "fail"
	}
}

func yamlEscape(s string) string {
	if strings.ContainsAny(s, ":#\n\"'") {
		return fmt.Sprintf("%q", s)
	}
	return s
}
