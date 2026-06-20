package render

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/xlab/treeprint"
)

// TreeRenderer emits a box-drawing tree of the spec hierarchy:
//
//	suite/path (20 specs, 24ms)
//	  20 passed
//	├── Describe
//	│   ├── when context A
//	│   │   ├── ✓ first leaf
//	│   │   └── ✓ second leaf
//	│   └── when context B
//	│       └── ✗ third leaf
//	└── Describe two
//	    └── ✓ fourth leaf
//
//	Summary: 20 passed in 24ms — PASSED
//
// Container hierarchy (Describe / Context / When) becomes inner nodes;
// each It block becomes a leaf with a glyph indicating outcome. The
// tree layout itself is built by github.com/xlab/treeprint — a small,
// well-established library that handles the box-drawing characters and
// nested last-child connectors so we never have to.
type TreeRenderer struct {
	color bool
}

// NewTree constructs a TreeRenderer. color toggles ANSI on the leaf
// glyphs and the failure summary lines.
func NewTree(color bool) *TreeRenderer { return &TreeRenderer{color: color} }

// WriteAll renders the suite header, status line, hierarchy tree,
// (if any failures) a block listing each failed spec, and a closing
// roll-up line — "Summary: <tally> in <dur> — PASSED|FAILED" — so the
// verdict is visible without scrolling back to the header.
func (t *TreeRenderer) WriteAll(w io.Writer, r Report) error {
	a := NewANSI(w, t.color)
	header := r.Suite.Path
	if header == "" {
		header = r.Suite.Name
	}
	dur := formatDurMs(r.EndTime.Sub(r.StartTime))
	a.WriteBold(header)
	if _, err := fmt.Fprintf(w, " %s\n", dim(t.color, fmt.Sprintf("(%s, %s)", specCount(r.Suite.NumSpecs), dur))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "  "+coloredStatus(t.color, r.Suite, true)+"\n"); err != nil {
		return err
	}

	body := buildSpecTree(r.Specs,
		func(segment string) string { return boldLabel(t.color, segment) },
		func(s SpecRow) string { return leafLabel(s, t.color) },
	)
	if _, err := io.WriteString(w, body); err != nil {
		return err
	}

	failures := failedSpecs(r.Specs)
	if len(failures) > 0 {
		if _, err := io.WriteString(w, "\n"); err != nil {
			return err
		}
		a.WriteRed("Failures:")
		if _, err := io.WriteString(w, "\n"); err != nil {
			return err
		}
		for _, s := range failures {
			if err := t.writeFailureBlock(w, s); err != nil {
				return err
			}
		}
	}

	if r.Suite.SpecialFailure != "" {
		a.WriteRed("\nSuite-level failure: " + r.Suite.SpecialFailure + "\n")
	}

	// Closing roll-up: a verdict line so the outcome is visible without
	// scrolling back to the header. Flaked/seed are intentionally omitted to
	// keep it terse.
	if _, err := fmt.Fprintf(w, "\nSummary: %s in %s — %s\n",
		coloredStatus(t.color, r.Suite, false),
		formatDurMs(r.EndTime.Sub(r.StartTime)),
		verdictLabel(t.color, r.Suite.SuiteSucceeded),
	); err != nil {
		return err
	}
	// Surface any error from the colored Write* calls above (header, Failures
	// label, glyphs, SpecialFailure), which swallow their errors individually.
	return a.Err()
}

// verdictLabel returns the PASSED/FAILED token for a roll-up line, colored
// green or red when ANSI is enabled.
func verdictLabel(color, succeeded bool) string {
	text, sgr := "FAILED", "31"
	if succeeded {
		text, sgr = "PASSED", "32"
	}
	if !color {
		return text
	}
	return "\x1b[" + sgr + "m" + text + "\x1b[0m"
}

// suiteCount renders a suite tally with correct singular/plural for the
// grand-total line.
func suiteCount(n int) string {
	if n == 1 {
		return "1 suite"
	}
	return fmt.Sprintf("%d suites", n)
}

// WriteTreeGrandTotal writes a cross-suite roll-up for the tree format: the
// suite count, the aggregated per-state tally (flaked/seed omitted to match
// the per-suite summary), the summed duration, and an overall verdict that
// fails if any suite failed. The CLI calls this once after every suite's
// tree has rendered, so a multi-package run ends with a single headline
// total. color toggles ANSI.
func WriteTreeGrandTotal(w io.Writer, suites []SuiteRow, total time.Duration, color bool) error {
	agg := SuiteRow{SuiteSucceeded: true}
	for _, s := range suites {
		agg.NumPassed += s.NumPassed
		agg.NumFailed += s.NumFailed
		agg.NumPanicked += s.NumPanicked
		agg.NumSkipped += s.NumSkipped
		agg.NumPending += s.NumPending
		if !s.SuiteSucceeded {
			agg.SuiteSucceeded = false
		}
	}
	_, err := fmt.Fprintf(w, "\nTotal: %s | %s in %s — %s\n",
		suiteCount(len(suites)),
		coloredStatus(color, agg, false),
		formatDurMs(total),
		verdictLabel(color, agg.SuiteSucceeded),
	)
	return err
}

// buildSpecTree renders the container hierarchy across all specs as a
// box-drawing tree. labelFn is applied to each container segment so
// callers can add bold/color; leafFn renders each spec row. The empty
// root line that treeprint always emits is stripped so callers can use
// their own header.
func buildSpecTree(specs []SpecRow, labelFn func(string) string, leafFn func(SpecRow) string) string {
	tree := treeprint.NewWithRoot("")
	for _, s := range specs {
		branch := tree
		for _, segment := range s.ContainerHier {
			branch = appendOrFindBranch(branch, labelFn(segment))
		}
		branch.AddNode(leafFn(s))
	}
	out := tree.String()
	if idx := strings.IndexByte(out, '\n'); idx >= 0 {
		out = out[idx+1:]
	}
	return out
}

// boldLabel returns label wrapped in ANSI bold when color is on; plain
// otherwise. Used for container node labels (Describe / Context /
// When) so the structural words pop visually against the leaves.
func boldLabel(color bool, label string) string {
	if !color {
		return label
	}
	return "\x1b[1m" + label + "\x1b[0m"
}

// coloredStatus renders the per-state tally — each category in its
// conventional color (green for passed, red for failed, yellow for
// skipped/pending) when ANSI is enabled. includeFlaked appends the
// flaked count; the header status line shows it, the roll-up summaries
// suppress it to stay terse.
func coloredStatus(color bool, s SuiteRow, includeFlaked bool) string {
	type seg struct {
		text  string
		color string // ANSI SGR digits, or "" for plain
	}
	var segs []seg
	if s.NumPassed > 0 {
		segs = append(segs, seg{fmt.Sprintf("%d passed", s.NumPassed), "32"})
	}
	if s.NumFailed > 0 {
		segs = append(segs, seg{fmt.Sprintf("%d failed", s.NumFailed), "31"})
	}
	if s.NumPanicked > 0 {
		segs = append(segs, seg{fmt.Sprintf("%d panicked", s.NumPanicked), "31"})
	}
	if s.NumSkipped > 0 {
		segs = append(segs, seg{fmt.Sprintf("%d skipped", s.NumSkipped), "33"})
	}
	if s.NumPending > 0 {
		segs = append(segs, seg{fmt.Sprintf("%d pending", s.NumPending), "33"})
	}
	if includeFlaked && s.NumFlaked > 0 {
		segs = append(segs, seg{fmt.Sprintf("%d flaked", s.NumFlaked), "33"})
	}
	if len(segs) == 0 {
		return "no specs"
	}
	parts := make([]string, len(segs))
	for i, sg := range segs {
		if color && sg.color != "" {
			parts[i] = "\x1b[" + sg.color + "m" + sg.text + "\x1b[0m"
		} else {
			parts[i] = sg.text
		}
	}
	return strings.Join(parts, " | ")
}

// appendOrFindBranch returns the existing child branch carrying this
// label if one is already present under parent, otherwise creates a new
// branch. Matching against the full child list — rather than only the
// last child — merges non-contiguous re-entries of the same container
// into a single sub-tree. This keeps the tree a faithful structural
// hierarchy even when specs arrive interleaved, as they do under
// Ginkgo's --randomize-all: a container always renders once, gathering
// all of its specs, regardless of execution order.
func appendOrFindBranch(parent treeprint.Tree, label string) treeprint.Tree {
	// treeprint.Node.Nodes is the only way to inspect children; the
	// interface itself does not expose them, so type-assert to *Node.
	// A container we want to merge into always has at least one child
	// (every spec adds a leaf), so the len(child.Nodes) > 0 guard
	// distinguishes container branches from same-named leaf nodes.
	if n, ok := parent.(*treeprint.Node); ok {
		for _, child := range n.Nodes {
			if len(child.Nodes) == 0 {
				continue
			}
			if v, ok := child.Value.(string); ok && v == label {
				return child
			}
		}
	}
	return parent.AddBranch(label)
}

// leafLabel returns the rendered leaf text — `<glyph> <description>` —
// with optional ANSI color on the glyph.
func leafLabel(s SpecRow, color bool) string {
	buf := &strings.Builder{}
	a := NewANSI(buf, color)
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
	buf.WriteByte(' ')
	if s.IsFocused {
		buf.WriteString("[FOCUS] ")
	}
	buf.WriteString(s.LeafText)
	return buf.String()
}

// treeGlyph returns the plain leaf glyph for a state — the same set
// used by the tree format, but without ANSI. Used by the github and
// gitlab renderers, which either strip ANSI (github) or need to wrap
// the glyph in their own colors (gitlab).
func treeGlyph(s State) string {
	switch s {
	case StatePassed:
		return "✓"
	case StateFailed, StatePanicked:
		return "✗"
	case StateSkipped:
		return "-"
	case StatePending:
		return "◌"
	case StateInterrupted:
		return "!"
	default:
		return "?"
	}
}

// plainStatus is coloredStatus with ANSI suppressed — used by renderers
// that emit logs where escape codes would be stripped (github) or where
// the caller wraps the line in its own color sequences.
func plainStatus(s SuiteRow) string {
	return coloredStatus(false, s, true)
}

// stripTopContainer returns copies of specs with the first ContainerHier
// segment removed. The github and gitlab renderers use it when building
// per-group sub-trees: the top-level container already names the group,
// so the inner tree should start one level deeper.
func stripTopContainer(specs []SpecRow) []SpecRow {
	out := make([]SpecRow, len(specs))
	for i, s := range specs {
		if len(s.ContainerHier) > 0 {
			s.ContainerHier = s.ContainerHier[1:]
		}
		out[i] = s
	}
	return out
}

// writeFailureBlock emits a post-tree record per failed spec: the full
// description (with the state's own glyph/color), the file:line, and the
// full matcher / failure body (plus any panic stack trace) so the user
// has the complete failure in one place without scrolling back through
// the tree.
func (t *TreeRenderer) writeFailureBlock(w io.Writer, s SpecRow) error {
	a := NewANSI(w, t.color)
	if _, err := io.WriteString(w, "  "); err != nil {
		return err
	}
	writeStateGlyph(a, s.State)
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
	body := indentBlock("    ", failureDetail(s))
	body += indentBlock("    ", failureExtras(s))
	_, err := io.WriteString(w, body)
	return err
}

// writeStateGlyph writes the colored leaf glyph for a failed state so the
// Failures block matches the inline tree (✗ for fail/panic, ! for
// interrupt) rather than a single hard-coded marker.
func writeStateGlyph(a *ANSI, state State) {
	switch state {
	case StateInterrupted:
		a.WriteYellow("!")
	default:
		a.WriteRed("✗")
	}
}

// specCount renders a spec tally with correct singular/plural agreement.
func specCount(n int) string {
	if n == 1 {
		return "1 spec"
	}
	return fmt.Sprintf("%d specs", n)
}
