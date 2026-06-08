package render

import (
	"fmt"
	"io"
	"strings"
)

// MarkdownRenderer emits a tree-shaped Markdown report:
//
//	# Suite Name
//
//	**FAIL** — 4 specs · 1 passed · 1 failed · 1 skipped · 1 pending · 100ms
//
//	<details open><summary><strong>Outer</strong></summary>
//
//	- **Inner**
//	  - ✅ a passes _(12ms)_
//	  - ❌ equals fails _(8ms)_
//	  ...
//
//	</details>
//
//	## Failures
//
//	### ❌ Outer › Inner › equals fails
//
//	`at /example/inner_test.go:25`
//
//	```text
//	expected: 2
//	actual:   1
//	```
//
// Each top-level container becomes a <details> block (expanded when it
// holds at least one failure). Inner containers nest as bold list items
// so the hierarchy reads top-to-bottom like the tree format. A trailing
// "Failures" section repeats every failed spec with its location and
// body, mirroring tree's failures block.
type MarkdownRenderer struct{}

// NewMarkdown returns a new MarkdownRenderer.
func NewMarkdown() *MarkdownRenderer { return &MarkdownRenderer{} }

// WriteAll renders the full report.
func (m *MarkdownRenderer) WriteAll(w io.Writer, r Report) error {
	ew := &errWriter{w: w}
	ew.printf("# %s\n\n", mdEscape(r.Suite.Name))
	writeMarkdownStatus(ew, r)
	if r.Suite.SpecialFailure != "" {
		ew.printf("> ⚠️ Suite-level failure: %s\n\n", mdEscape(r.Suite.SpecialFailure))
	}

	for _, g := range groupByTopContainer(r.Specs) {
		m.writeGroup(ew, g)
	}

	if failures := failedSpecs(r.Specs); len(failures) > 0 {
		ew.writeString("## Failures\n\n")
		for _, s := range failures {
			writeMarkdownFailureBlock(ew, s)
		}
	}
	return ew.Err()
}

// writeMarkdownStatus emits the verdict + counts + duration line:
//
//	**FAIL** — 4 specs · 1 passed · 1 failed · 1 skipped · 1 pending · 100ms
func writeMarkdownStatus(ew *errWriter, r Report) {
	verdict := "PASS"
	if !r.Suite.SuiteSucceeded {
		verdict = "FAIL"
	}
	var parts []string
	parts = append(parts, specCount(r.Suite.NumSpecs))
	if r.Suite.NumPassed > 0 {
		parts = append(parts, fmt.Sprintf("%d passed", r.Suite.NumPassed))
	}
	if r.Suite.NumFailed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", r.Suite.NumFailed))
	}
	if r.Suite.NumPanicked > 0 {
		parts = append(parts, fmt.Sprintf("%d panicked", r.Suite.NumPanicked))
	}
	if r.Suite.NumSkipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", r.Suite.NumSkipped))
	}
	if r.Suite.NumPending > 0 {
		parts = append(parts, fmt.Sprintf("%d pending", r.Suite.NumPending))
	}
	if r.Suite.NumFlaked > 0 {
		parts = append(parts, fmt.Sprintf("%d flaked", r.Suite.NumFlaked))
	}
	parts = append(parts, formatDurMs(r.EndTime.Sub(r.StartTime)))
	ew.printf("**%s** — %s\n\n", verdict, strings.Join(parts, " · "))
}

type mdGroup struct {
	name    string
	specs   []SpecRow
	hasFail bool
}

func groupByTopContainer(specs []SpecRow) []mdGroup {
	var groups []mdGroup
	idx := map[string]int{}
	for _, s := range specs {
		key := "(no container)"
		if len(s.ContainerHier) > 0 {
			key = s.ContainerHier[0]
		}
		i, ok := idx[key]
		if !ok {
			groups = append(groups, mdGroup{name: key})
			i = len(groups) - 1
			idx[key] = i
		}
		groups[i].specs = append(groups[i].specs, s)
		if isFailure(s) {
			groups[i].hasFail = true
		}
	}
	return groups
}

func (m *MarkdownRenderer) writeGroup(ew *errWriter, g mdGroup) {
	openAttr := ""
	if g.hasFail {
		openAttr = " open"
	}
	ew.printf("<details%s><summary><strong>%s</strong></summary>\n\n", openAttr, mdEscape(g.name))
	// Build the inner tree by walking each spec's container hierarchy
	// past the top-level container we already named in the <summary>.
	root := newMDNode("")
	for _, s := range g.specs {
		inner := s.ContainerHier
		if len(inner) > 0 {
			inner = inner[1:]
		}
		root.addSpec(inner, s)
	}
	root.writeChildren(ew, 0)
	ew.writeString("\n</details>\n\n")
}

// mdNode is a tiny container/leaf tree used to render nested Markdown
// bullet lists with stable, container-grouped ordering.
type mdNode struct {
	label    string   // container name; empty for the root
	spec     *SpecRow // non-nil for leaves
	children []*mdNode
}

func newMDNode(label string) *mdNode { return &mdNode{label: label} }

// addSpec walks (and grows) the tree along containerPath, then attaches
// spec as a leaf. Container ordering follows first-seen, matching the
// box-drawing tree's behavior in tree.go.
func (n *mdNode) addSpec(containerPath []string, spec SpecRow) {
	cur := n
	for _, seg := range containerPath {
		// Match any existing child container with this label — same rule
		// as appendOrFindBranch — so a container re-entered out of order
		// (as happens under --randomize-all) merges into one node instead
		// of splitting into duplicate siblings.
		var match *mdNode
		for _, child := range cur.children {
			if child.spec == nil && child.label == seg {
				match = child
				break
			}
		}
		if match == nil {
			match = newMDNode(seg)
			cur.children = append(cur.children, match)
		}
		cur = match
	}
	leaf := spec
	cur.children = append(cur.children, &mdNode{spec: &leaf})
}

// writeChildren emits this node's children at the given depth.
// Indentation is two spaces per level — the standard for nested Markdown
// lists; both GitHub and most browsers render that correctly.
func (n *mdNode) writeChildren(ew *errWriter, depth int) {
	indent := strings.Repeat("  ", depth)
	for _, c := range n.children {
		if c.spec != nil {
			writeMDLeaf(ew, indent, *c.spec)
			continue
		}
		ew.printf("%s- **%s**\n", indent, mdEscape(c.label))
		c.writeChildren(ew, depth+1)
	}
}

func writeMDLeaf(ew *errWriter, indent string, s SpecRow) {
	leaf := mdEscape(s.LeafText)
	if s.IsFocused {
		leaf = "**\\[FOCUS\\]** " + leaf
	}
	ew.printf("%s- %s %s _(%s)_\n", indent, mdGlyph(s.State), leaf, formatDurMs(s.Duration))
}

// writeMarkdownFailureBlock emits one failure detail block. Each gets
// its own ### heading so a reader can jump from the table of contents
// straight to the failure, mirroring tree's Failures: list.
func writeMarkdownFailureBlock(ew *errWriter, s SpecRow) {
	desc := strings.Join(s.FullText, " › ")
	ew.printf("### ❌ %s\n\n", mdEscape(desc))
	if loc := s.Failure.Location.String(); loc != "" {
		ew.printf("`at %s`\n\n", loc)
	}
	body := strings.TrimRight(failureDetail(s), "\n")
	if extras := strings.TrimRight(failureExtras(s), "\n"); extras != "" {
		body += "\n" + extras
	}
	fence := mdFence(body)
	ew.printf("%stext\n%s\n%s\n\n", fence, body, fence)
}

// mdFence returns a backtick fence longer than the longest run of backticks
// in body, so attacker-controlled body content containing ``` cannot
// terminate the code block early (CommonMark: a fenced block ends only at a
// fence of equal-or-greater length). Minimum three backticks.
func mdFence(body string) string {
	longest, run := 0, 0
	for _, r := range body {
		if r == '`' {
			run++
			if run > longest {
				longest = run
			}
		} else {
			run = 0
		}
	}
	return strings.Repeat("`", max(longest+1, 3))
}

func mdGlyph(s State) string {
	switch s {
	case StatePassed:
		return "✅"
	case StateFailed, StatePanicked:
		return "❌"
	case StateSkipped:
		return "⏭️"
	case StatePending:
		return "🕒"
	case StateInterrupted:
		return "⚠️"
	default:
		return "❓"
	}
}

func mdEscape(s string) string {
	r := strings.NewReplacer(
		"\\", "\\\\",
		"<", "&lt;",
		">", "&gt;",
		"&", "&amp;",
	)
	return r.Replace(s)
}
