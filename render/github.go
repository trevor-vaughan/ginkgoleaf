package render

import (
	"fmt"
	"io"
	"strings"
)

// GitHubRenderer emits a tree-shaped report for GitHub Actions.
//
//	Mixed Suite (4 specs, 100ms)
//	  1 passed | 1 failed | 1 skipped | 1 pending
//	::group::Outer
//	└── Inner
//	    ├── ✓ a passes (12ms)
//	    ├── ✗ equals fails (8ms)
//	    ├── - is skipped (12ms)
//	    └── ◌ is pending (12ms)
//	::endgroup::
//	::group::Failures
//	  ✗ Outer > Inner > equals fails
//	    at /example/inner_test.go:25
//	::stop-commands::ginkgoleaf-end-output
//	    expected: 2
//	    actual:   1
//	::ginkgoleaf-end-output::
//	::endgroup::
//	::error file=...,line=...,title=...::<message>
//	::notice title=ginkgoleaf::FAIL Mixed Suite — total ...
//
// The header + counts line gives at-a-glance status in the log. Each
// top-level container is its own collapsible group with a box-drawing
// subtree inside, mirroring the tree format. ::error annotations after
// the groups still drive GitHub's inline PR annotations.
type GitHubRenderer struct{}

// NewGitHub returns a new GitHubRenderer.
func NewGitHub() *GitHubRenderer { return &GitHubRenderer{} }

// WriteAll renders the entire report.
func (g *GitHubRenderer) WriteAll(w io.Writer, r Report) error {
	ew := &errWriter{w: w}
	writeGHHeader(ew, r)

	for _, grp := range groupByTopContainer(r.Specs) {
		ew.printf("::group::%s\n", ghEscapeData(grp.name))
		ew.writeString(buildSpecTree(stripTopContainer(grp.specs),
			func(seg string) string { return seg },
			ghLeafLabel,
		))
		ew.writeString("::endgroup::\n")
	}

	if failures := failedSpecs(r.Specs); len(failures) > 0 {
		ew.writeString("::group::Failures\n")
		for _, s := range failures {
			writeGHFailureBlock(ew, s)
		}
		ew.writeString("::endgroup::\n")
	}

	// Annotations after groups so the GH UI shows them inline at the top.
	// Only real failures become ::error annotations — a skipped spec's
	// Skip("…") reason lives in Failure but is not an error.
	for _, s := range r.Specs {
		if !isFailure(s) {
			continue
		}
		writeGHAnnotation(ew, s)
	}

	if r.Suite.SpecialFailure != "" {
		ew.printf("::error title=ginkgoleaf::Suite-level failure: %s\n", ghEscapeData(r.Suite.SpecialFailure))
	}

	verdict := "PASS"
	if !r.Suite.SuiteSucceeded {
		verdict = "FAIL"
	}
	flaked := ""
	if r.Suite.NumFlaked > 0 {
		flaked = fmt.Sprintf(" / flaked %d", r.Suite.NumFlaked)
	}
	ew.printf(
		"::notice title=ginkgoleaf::%s %s — total %d / pass %d / fail %d / skip %d / pending %d / panic %d%s\n",
		verdict, ghEscapeData(r.Suite.Name),
		r.Suite.NumSpecs, r.Suite.NumPassed, r.Suite.NumFailed,
		r.Suite.NumSkipped, r.Suite.NumPending, r.Suite.NumPanicked, flaked,
	)
	return ew.Err()
}

// writeGHHeader emits the suite name + counts line — the same shape the
// tree format uses, minus ANSI (GitHub strips it from log output).
func writeGHHeader(ew *errWriter, r Report) {
	header := r.Suite.Path
	if header == "" {
		header = r.Suite.Name
	}
	dur := formatDurMs(r.EndTime.Sub(r.StartTime))
	ew.printf("%s (%s, %s)\n", header, specCount(r.Suite.NumSpecs), dur)
	ew.printf("  %s\n", plainStatus(r.Suite))
}

// ghLeafLabel renders one leaf inside a github group as
// "<glyph> <leaf text> (<duration>)" — same glyph set as the tree
// format, plain (no ANSI) since GitHub strips it.
func ghLeafLabel(s SpecRow) string {
	leaf := s.LeafText
	if s.IsFocused {
		leaf = "[FOCUS] " + leaf
	}
	return fmt.Sprintf("%s %s (%s)", treeGlyph(s.State), leaf, formatDurMs(s.Duration))
}

// writeGHFailureBlock mirrors tree.writeFailureBlock but without ANSI.
// Two indented body lines max keeps the group compact in the log; full
// messages still appear on the inline PR annotation.
func writeGHFailureBlock(ew *errWriter, s SpecRow) {
	ew.printf("  %s %s\n", treeGlyph(s.State), strings.Join(s.FullText, " > "))
	if loc := s.Failure.Location.String(); loc != "" {
		ew.printf("    at %s\n", loc)
	}
	detail := failureDetail(s)
	if detail == "" {
		return
	}
	// The failure body is attacker-influenced. The Actions runner parses a log
	// line as a workflow command when, after TrimStart, it begins with "::"
	// (ActionCommand.TryParseV2) — so an indented "::error file=…::" body line
	// would forge an annotation. Wrap the body in stop-commands/<token>: the
	// runner suspends command parsing until it sees "::<token>::". The token is
	// chosen to not occur in the body, so attacker content cannot emit the
	// resume marker to re-enable injection. Both markers are consumed (not
	// shown in the log), so the body still renders verbatim.
	token := ghStopToken(detail)
	ew.printf("::stop-commands::%s\n", token)
	ew.writeString(indentBlock("    ", detail))
	ew.printf("::%s::\n", token)
}

// ghStopToken returns a stop-commands token guaranteed not to appear in
// body, so attacker-controlled body content cannot re-enable command
// processing by emitting the resume marker itself.
func ghStopToken(body string) string {
	const base = "ginkgoleaf-end-output"
	token := base
	for i := 0; strings.Contains(body, token); i++ {
		token = fmt.Sprintf("%s-%d", base, i)
	}
	return token
}

func writeGHAnnotation(ew *errWriter, s SpecRow) {
	loc := s.Failure.Location
	desc := strings.Join(s.FullText, " > ")
	msg := failureMessage(s)
	if loc.FileName == "" {
		ew.printf("::error title=%s::%s\n", ghEscapeProp(desc), ghEscapeData(msg))
		return
	}
	ew.printf("::error file=%s,line=%d,title=%s::%s\n",
		ghEscapeProp(loc.FileName), loc.LineNumber,
		ghEscapeProp(desc), ghEscapeData(msg))
}

// ghEscapeData escapes a workflow command's data segment.
// See https://docs.github.com/en/actions/using-workflows/workflow-commands-for-github-actions#example-of-masking-a-value-in-a-log
func ghEscapeData(s string) string {
	r := strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A")
	return r.Replace(s)
}

// ghEscapeProp also escapes : and , which are property separators.
func ghEscapeProp(s string) string {
	r := strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A", ":", "%3A", ",", "%2C")
	return r.Replace(s)
}
