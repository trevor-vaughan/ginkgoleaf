package render

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// TextRenderer emits a plain-ASCII tree of specs with no ANSI ever.
// It is the simplest renderer and serves as a reference for the others.
type TextRenderer struct{}

// NewText returns a new TextRenderer.
func NewText() *TextRenderer { return &TextRenderer{} }

// WriteAll is a convenience that batches the streaming calls.
func (t *TextRenderer) WriteAll(w io.Writer, r Report) error {
	ew := &errWriter{w: w}
	for _, s := range r.Specs {
		t.writeSpec(ew, s)
	}
	t.writeSummary(ew, r)
	return ew.Err()
}

// writeSpec writes one line per spec, indented by its container depth.
func (*TextRenderer) writeSpec(ew *errWriter, s SpecRow) {
	indent := strings.Repeat("  ", len(s.ContainerHier))
	glyph := textGlyph(s.State)
	dur := formatDurMs(s.Duration)
	leaf := s.LeafText
	if s.IsFocused {
		leaf = "[FOCUS] " + leaf
	}
	ew.printf("%s%s %s (%s)\n", indent, glyph, leaf, dur)
	if isFailure(s) {
		body := indentBlock(indent+"    ", failureDetail(s))
		body += indentBlock(indent+"    ", failureExtras(s))
		loc := s.Failure.Location.String()
		if loc != "" {
			body = indent + "    at " + loc + "\n" + body
		}
		ew.writeString(body)
	}
}

// writeSummary writes the totals line, a compact Failures section (when
// any spec failed), and the final verdict.
func (*TextRenderer) writeSummary(ew *errWriter, r Report) {
	special := ""
	if r.Suite.SpecialFailure != "" {
		special = " | special-failure: " + r.Suite.SpecialFailure
	}
	verdict := "PASS"
	if !r.Suite.SuiteSucceeded {
		verdict = "FAIL"
	}

	seed := ""
	if r.Suite.RandomSeed != 0 {
		seed = fmt.Sprintf(" | seed %d", r.Suite.RandomSeed)
	}
	flaked := ""
	if r.Suite.NumFlaked > 0 {
		flaked = fmt.Sprintf(" | flaked %d", r.Suite.NumFlaked)
	}
	ew.printf(
		"\n%s: %s | total %d | pass %d | fail %d | skip %d | pending %d | panic %d | duration %s%s%s%s\n",
		verdict, r.Suite.Name, r.Suite.NumSpecs, r.Suite.NumPassed, r.Suite.NumFailed,
		r.Suite.NumSkipped, r.Suite.NumPending, r.Suite.NumPanicked,
		formatDurMs(r.EndTime.Sub(r.StartTime)), seed, flaked, special,
	)

	failures := failedSpecs(r.Specs)
	if len(failures) == 0 {
		return
	}
	ew.writeString("\nFailures:\n")
	for _, s := range failures {
		ew.printf("  [X] %s\n", strings.Join(s.FullText, " > "))
		if loc := s.Failure.Location.String(); loc != "" {
			ew.printf("      at %s\n", loc)
		}
		lines := strings.Split(strings.TrimRight(failureMessage(s), "\n"), "\n")
		const maxLines = 2
		for i, ln := range lines {
			if i >= maxLines {
				ew.writeString("      ...\n")
				break
			}
			ew.printf("      %s\n", ln)
		}
	}
}

func textGlyph(s State) string {
	switch s {
	case StatePassed:
		return "[+]"
	case StateFailed, StatePanicked:
		return "[X]"
	case StateSkipped:
		return "[~]"
	case StatePending:
		return "[.]"
	case StateInterrupted:
		return "[!]"
	case StateAborted:
		return "[A]"
	default:
		return "[?]"
	}
}

func indentBlock(prefix, s string) string {
	if s == "" {
		return ""
	}
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(prefix)
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func formatDurMs(d time.Duration) string {
	if d == 0 {
		return "0ms"
	}
	if d >= time.Second {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
	return fmt.Sprintf("%dms", d.Milliseconds())
}
