package render

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// GitLabRenderer emits a tree-shaped report for GitLab CI:
//
//	Mixed Suite (4 specs, 100ms)
//	  \e[32m1 passed\e[0m | \e[31m1 failed\e[0m | \e[33m1 skipped\e[0m | \e[33m1 pending\e[0m
//	\e[0Ksection_start:<ts>:ginkgoleaf_1\r\e[0KOuter
//	└── Inner
//	    ├── \e[32m✓\e[0m a passes (12ms)
//	    ├── \e[31m✗\e[0m equals fails (8ms)
//	    ├── \e[33m-\e[0m is skipped (12ms)
//	    └── \e[33m◌\e[0m is pending (12ms)
//	\e[0Ksection_end:<ts>:ginkgoleaf_1\r\e[0K
//	\e[0Ksection_start:<ts>:ginkgoleaf_2\r\e[0KFailures
//	  \e[31m✗\e[0m Outer > Inner > equals fails
//	    \e[2mat /example/inner_test.go:25\e[0m
//	    expected: 2
//	    actual:   1
//	\e[0Ksection_end:<ts>:ginkgoleaf_2\r\e[0K
//	\e[31mFAIL\e[0m Mixed Suite — 1 passed | 1 failed | 1 skipped | 1 pending (100ms)
//
// Clock is parameterized so tests get a deterministic epoch. Glyphs and
// container hierarchy match the tree format; GitLab renders the ANSI
// codes inline in its log viewer.
type GitLabRenderer struct {
	clock     func() int64
	color     bool
	sectionID int
}

// NewGitLab constructs a renderer with a custom clock (test override)
// and a resolved color decision. Pass time.Now().Unix as clock in
// production. When color is false, foreground/dim escapes are
// suppressed (honouring --color=never / NO_COLOR); the GitLab
// section_start/section_end control sequences are always emitted because
// they are CI log structure, not color.
func NewGitLab(clock func() int64, color bool) *GitLabRenderer {
	if clock == nil {
		clock = func() int64 { return time.Now().Unix() }
	}
	return &GitLabRenderer{clock: clock, color: color}
}

// gitlabColor wraps s in the ANSI code when enabled; returns s unchanged
// otherwise (or when code is empty).
func gitlabColor(enable bool, code, s string) string {
	if !enable || code == "" {
		return s
	}
	return code + s + "\x1b[0m"
}

// WriteAll renders the entire report.
func (g *GitLabRenderer) WriteAll(w io.Writer, r Report) error {
	ew := &errWriter{w: w}
	g.writeHeader(ew, r)

	ts := g.clock()
	for _, grp := range groupByTopContainer(r.Specs) {
		g.sectionID++
		id := fmt.Sprintf("ginkgoleaf_%d", g.sectionID)
		collapsed := ""
		if !grp.hasFail {
			collapsed = "[collapsed=true]"
		}
		ew.printf("\x1b[0Ksection_start:%d:%s%s\r\x1b[0K%s\n", ts, id, collapsed, grp.name)
		ew.writeString(buildSpecTree(stripTopContainer(grp.specs),
			func(seg string) string { return seg },
			func(s SpecRow) string { return gitlabLeafLabel(s, g.color) },
		))
		ew.printf("\x1b[0Ksection_end:%d:%s\r\x1b[0K\n", ts, id)
	}

	if failures := failedSpecs(r.Specs); len(failures) > 0 {
		g.sectionID++
		id := fmt.Sprintf("ginkgoleaf_%d", g.sectionID)
		ew.printf("\x1b[0Ksection_start:%d:%s\r\x1b[0KFailures\n", ts, id)
		for _, s := range failures {
			writeGitLabFailureBlock(ew, s, g.color)
		}
		ew.printf("\x1b[0Ksection_end:%d:%s\r\x1b[0K\n", ts, id)
	}

	if r.Suite.SpecialFailure != "" {
		ew.printf("%s\n", gitlabColor(g.color, "\x1b[31m", "Suite-level failure: "+r.Suite.SpecialFailure))
	}

	verdict := "PASS"
	code := "\x1b[32m"
	if !r.Suite.SuiteSucceeded {
		verdict = "FAIL"
		code = "\x1b[31m"
	}
	ew.printf("%s %s — %s (%s)\n",
		gitlabColor(g.color, code, verdict), r.Suite.Name,
		coloredStatus(g.color, r.Suite),
		formatDurMs(r.EndTime.Sub(r.StartTime)),
	)
	return ew.Err()
}

// writeHeader emits the suite path + spec count + duration header line
// and the colored counts line — same layout as tree's header.
func (g *GitLabRenderer) writeHeader(ew *errWriter, r Report) {
	header := r.Suite.Path
	if header == "" {
		header = r.Suite.Name
	}
	dur := formatDurMs(r.EndTime.Sub(r.StartTime))
	ew.printf("%s (%s, %s)\n", header, specCount(r.Suite.NumSpecs), dur)
	ew.printf("  %s\n", coloredStatus(g.color, r.Suite))
}

// gitlabLeafLabel renders one leaf with an ANSI-colored glyph that
// matches the tree format's color choices. When enable is false the
// glyph is emitted plain.
func gitlabLeafLabel(s SpecRow, enable bool) string {
	code := gitlabStateColor(s.State)
	glyph := treeGlyph(s.State)
	leaf := s.LeafText
	if s.IsFocused {
		leaf = "[FOCUS] " + leaf
	}
	return fmt.Sprintf("%s %s (%s)", gitlabColor(enable, code, glyph), leaf, formatDurMs(s.Duration))
}

// writeGitLabFailureBlock mirrors tree.writeFailureBlock, with ANSI
// gated on enable.
func writeGitLabFailureBlock(ew *errWriter, s SpecRow, enable bool) {
	code := gitlabStateColor(s.State)
	ew.printf("  %s %s\n", gitlabColor(enable, code, treeGlyph(s.State)), strings.Join(s.FullText, " > "))
	if loc := s.Failure.Location.String(); loc != "" {
		ew.printf("    %s\n", gitlabColor(enable, "\x1b[2m", "at "+loc))
	}
	body := indentBlock("    ", failureDetail(s))
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	ew.writeString(body)
}

// gitlabStateColor returns the ANSI color prefix for a state's glyph.
func gitlabStateColor(s State) string {
	switch s {
	case StatePassed:
		return "\x1b[32m"
	case StateFailed, StatePanicked:
		return "\x1b[31m"
	case StateSkipped, StatePending, StateInterrupted:
		return "\x1b[33m"
	default:
		return "\x1b[2m"
	}
}
