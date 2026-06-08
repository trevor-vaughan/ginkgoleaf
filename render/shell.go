package render

import (
	"fmt"
	"io"
	"strings"
)

// ShellRenderer emits one tab-separated line per spec for grep/awk
// pipelines. The first line is a self-documenting `#fields:` header.
type ShellRenderer struct {
	headerWritten bool
}

// NewShell returns a new ShellRenderer.
func NewShell() *ShellRenderer { return &ShellRenderer{} }

// WriteAll renders the full report via the streaming entry points.
func (s *ShellRenderer) WriteAll(w io.Writer, r Report) error {
	ew := &errWriter{w: w}
	for _, spec := range r.Specs {
		s.writeSpec(ew, spec)
	}
	s.writeSummary(ew, r)
	return ew.Err()
}

// writeSpec emits one record per spec: state, duration_ms, location, description.
func (s *ShellRenderer) writeSpec(ew *errWriter, spec SpecRow) {
	if !s.headerWritten {
		ew.writeString("#fields: state duration_ms location description\n")
		s.headerWritten = true
	}
	desc := strings.Join(spec.FullText, " > ")
	loc := spec.Location.String()
	if loc == "" {
		loc = "-"
	}
	ew.printf("%s\t%d\t%s\t%s\n",
		spec.State, spec.Duration.Milliseconds(), loc, escapeShell(desc))
}

// writeSummary emits a `#summary` line at end of suite.
func (*ShellRenderer) writeSummary(ew *errWriter, r Report) {
	verdict := "pass"
	if !r.Suite.SuiteSucceeded {
		verdict = "fail"
	}
	flaked := ""
	if r.Suite.NumFlaked > 0 {
		flaked = fmt.Sprintf(" flaked=%d", r.Suite.NumFlaked)
	}
	ew.printf(
		"#summary verdict=%s total=%d pass=%d fail=%d skip=%d pending=%d panic=%d%s duration_ms=%d suite=%q\n",
		verdict, r.Suite.NumSpecs, r.Suite.NumPassed, r.Suite.NumFailed,
		r.Suite.NumSkipped, r.Suite.NumPending, r.Suite.NumPanicked, flaked,
		r.EndTime.Sub(r.StartTime).Milliseconds(), r.Suite.Name,
	)
}

// escapeShell replaces tab and newline so single-record-per-line is invariant.
func escapeShell(s string) string {
	r := strings.NewReplacer("\t", "\\t", "\n", "\\n", "\r", "\\r")
	return r.Replace(s)
}
