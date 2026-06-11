package render

import (
	"fmt"
	"io"
	"slices"
	"strings"
)

// JestRenderer emits jest-style output: glyphs, indented BDD nesting,
// inline failure blocks, and optionally ANSI color.
type JestRenderer struct {
	color   bool
	lastHdr []string // last container hierarchy printed, to avoid reprinting unchanged headers
}

// NewJest constructs a JestRenderer. color toggles ANSI output.
func NewJest(color bool) *JestRenderer { return &JestRenderer{color: color} }

// WriteAll renders the full report by replaying through writeSpec/writeSummary.
func (j *JestRenderer) WriteAll(w io.Writer, r Report) error {
	for _, s := range r.Specs {
		if err := j.writeSpec(w, s); err != nil {
			return err
		}
	}
	return j.writeSummary(w, r)
}

// writeSpec emits container headers (when changed) + the spec line + an
// indented failure block when present.
func (j *JestRenderer) writeSpec(w io.Writer, s SpecRow) error {
	a := NewANSI(w, j.color)
	if !slices.Equal(j.lastHdr, s.ContainerHier) {
		for i, h := range s.ContainerHier {
			if i < len(j.lastHdr) && j.lastHdr[i] == h {
				continue
			}
			indent := strings.Repeat("  ", i)
			if _, err := io.WriteString(w, indent); err != nil {
				return err
			}
			a.WriteBold(h)
			if _, err := io.WriteString(w, "\n"); err != nil {
				return err
			}
		}
		j.lastHdr = slices.Clone(s.ContainerHier)
	}
	indent := strings.Repeat("  ", len(s.ContainerHier))
	if _, err := io.WriteString(w, indent); err != nil {
		return err
	}
	switch s.State {
	case StatePassed:
		a.WriteGreen("✓")
	case StateFailed, StatePanicked:
		a.WriteRed("✗")
	case StateSkipped:
		a.WriteYellow("-")
	case StatePending:
		a.WriteYellow("◌")
	case StateInterrupted:
		a.WriteYellow("!")
	default:
		a.WriteDim("?")
	}
	leaf := " " + s.LeafText
	if s.IsFocused {
		leaf += " [FOCUS]"
	}
	dur := " " + dim(j.color, fmt.Sprintf("(%s)", formatDurMs(s.Duration)))
	if _, err := io.WriteString(w, leaf+dur+"\n"); err != nil {
		return err
	}
	if isFailure(s) {
		if err := j.writeFailure(w, indent+"  ", s); err != nil {
			return err
		}
	}
	return nil
}

func (j *JestRenderer) writeFailure(w io.Writer, prefix string, s SpecRow) error {
	a := NewANSI(w, j.color)
	loc := s.Failure.Location.String()
	if loc != "" {
		if _, err := io.WriteString(w, prefix); err != nil {
			return err
		}
		a.WriteDim("at " + loc)
		if _, err := io.WriteString(w, "\n"); err != nil {
			return err
		}
	}
	body := indentBlock(prefix, failureDetail(s))
	if _, err := io.WriteString(w, body); err != nil {
		return err
	}
	return nil
}

// writeSummary prints the end-of-suite footer. The footer is structured
// for quick scanning when scrolling up from the end:
//
//   - Totals line
//   - Time line
//   - Seed line (when non-zero — useful for reproducing parallel runs)
//   - Optional Suite-level failure line
//   - Failures: block listing every failed spec compactly, so the user
//     does not have to scroll back through the per-spec rendering to
//     find what went wrong
//   - PASS/FAIL verdict line
func (j *JestRenderer) writeSummary(w io.Writer, r Report) error {
	a := NewANSI(w, j.color)
	if _, err := io.WriteString(w, "\n"); err != nil {
		return err
	}
	totals := fmt.Sprintf("Tests: %d total, %d passed, %d failed, %d skipped, %d pending, %d panicked\n",
		r.Suite.NumSpecs, r.Suite.NumPassed, r.Suite.NumFailed,
		r.Suite.NumSkipped, r.Suite.NumPending, r.Suite.NumPanicked)
	if _, err := io.WriteString(w, totals); err != nil {
		return err
	}
	if r.Suite.NumFlaked > 0 {
		if _, err := fmt.Fprintf(w, "Flaked: %d\n", r.Suite.NumFlaked); err != nil {
			return err
		}
	}
	dur := fmt.Sprintf("Time:  %s\n", formatDurMs(r.EndTime.Sub(r.StartTime)))
	if _, err := io.WriteString(w, dur); err != nil {
		return err
	}
	if r.Suite.RandomSeed != 0 {
		if _, err := fmt.Fprintf(w, "Seed:  %d\n", r.Suite.RandomSeed); err != nil {
			return err
		}
	}
	if r.Suite.SpecialFailure != "" {
		a.WriteRed("Suite-level failure: " + r.Suite.SpecialFailure + "\n")
	}

	failures := failedSpecs(r.Specs)
	if len(failures) > 0 {
		if _, err := io.WriteString(w, "\nFailures:\n"); err != nil {
			return err
		}
		for _, s := range failures {
			if err := j.writeFailureSummary(w, s); err != nil {
				return err
			}
		}
	}

	if _, err := io.WriteString(w, "\n"); err != nil {
		return err
	}
	if r.Suite.SuiteSucceeded {
		a.WriteGreen("PASS")
	} else {
		a.WriteRed("FAIL")
	}
	if _, err := io.WriteString(w, " "+r.Suite.Name+"\n"); err != nil {
		return err
	}
	// Surface any error from the colored Write* calls (glyphs, SpecialFailure,
	// the verdict), which swallow their errors individually.
	return a.Err()
}

// writeFailureSummary emits a compact per-failure record for the
// trailing Failures: section. Shows the full spec path, location, and
// up to two lines of the failure body (truncated with "..." when
// longer) — enough to identify and start debugging without scrolling.
func (j *JestRenderer) writeFailureSummary(w io.Writer, s SpecRow) error {
	a := NewANSI(w, j.color)
	if _, err := io.WriteString(w, "  "); err != nil {
		return err
	}
	a.WriteRed("✗")
	if _, err := io.WriteString(w, " "+strings.Join(s.FullText, " > ")+"\n"); err != nil {
		return err
	}
	if loc := s.Failure.Location.String(); loc != "" {
		if _, err := io.WriteString(w, "    "); err != nil {
			return err
		}
		a.WriteDim("at " + loc)
		if _, err := io.WriteString(w, "\n"); err != nil {
			return err
		}
	}
	lines := strings.Split(strings.TrimRight(failureMessage(s), "\n"), "\n")
	const maxLines = 2
	for i, line := range lines {
		if i >= maxLines {
			if _, err := io.WriteString(w, "    ...\n"); err != nil {
				return err
			}
			break
		}
		if _, err := io.WriteString(w, "    "+line+"\n"); err != nil {
			return err
		}
	}
	return nil
}

// isFailure reports whether a spec ended in a real failure carrying a
// failure record. It is the single predicate that gates failure detail
// across all renderers, so a skipped/pending spec — whose Skip("…")
// reason also lands in Failure — is never rendered as a failure.
func isFailure(s SpecRow) bool {
	switch s.State {
	case StateFailed, StatePanicked, StateInterrupted, StateAborted:
		return s.Failure != nil
	default:
		return false
	}
}

// failedSpecs returns the subset of specs that ended in a non-success
// terminal state (failed/panicked/interrupted/aborted). Used by the
// Failures: summary section.
func failedSpecs(specs []SpecRow) []SpecRow {
	var out []SpecRow
	for _, s := range specs {
		if isFailure(s) {
			out = append(out, s)
		}
	}
	return out
}

// failureMessage returns the human body of a failure: the parsed matcher
// when one is attached (clean expected/actual), otherwise the raw
// failure message. Shared by every renderer so the matcher reads the
// same everywhere.
func failureMessage(s SpecRow) string {
	if s.Matcher != nil {
		if formatted := FormatMatcher(*s.Matcher); formatted != "" {
			// A matcher attached via the gomega entry path carries
			// Expected/Actual/Raw that never passed through the translate-time
			// message sanitizer, so strip control bytes here. Idempotent on
			// the already-sanitized Failure.Message below.
			return sanitizeBlock(formatted)
		}
	}
	return sanitizeBlock(s.Failure.Message)
}

// failureDetail is failureMessage plus a trailing stack-trace block when
// the failure carries one (panics). Used by the per-failure detail
// blocks; compact recap lists and the github annotation use
// failureMessage so they stay concise.
func failureDetail(s SpecRow) string {
	body := failureMessage(s)
	if st := strings.TrimRight(s.Failure.StackTrace, "\n"); st != "" {
		body += "\nstack trace:\n" + st
	}
	return body
}

// failureExtras returns optional labelled blocks for a failing spec —
// captured GinkgoWriter output, captured stdout/stderr, and the progress
// report — or "" when none are present. Rendered only by the verbose and
// structured renderers (tree, text, markdown, tap, cucumber); the compact
// CI formats omit it to keep annotations and one-line records lean.
func failureExtras(s SpecRow) string {
	var b strings.Builder
	if s.CapturedOut != "" {
		b.WriteString("captured GinkgoWriter output:\n")
		b.WriteString(strings.TrimRight(s.CapturedOut, "\n"))
		b.WriteByte('\n')
	}
	if s.CapturedErr != "" {
		b.WriteString("captured stdout/stderr:\n")
		b.WriteString(strings.TrimRight(s.CapturedErr, "\n"))
		b.WriteByte('\n')
	}
	if s.Failure != nil && s.Failure.ProgressReport != "" {
		b.WriteString("progress report:\n")
		b.WriteString(strings.TrimRight(s.Failure.ProgressReport, "\n"))
		b.WriteByte('\n')
	}
	return b.String()
}

// dim wraps s in ANSI dim codes when color is enabled, passes through otherwise.
func dim(color bool, s string) string {
	if !color {
		return s
	}
	return "\x1b[2m" + s + "\x1b[0m"
}
