package render

import (
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

// CucumberRenderer emits a Gherkin-shaped view of a suite: one Feature
// (the suite), one Scenario per spec, and one step per Describe/Context
// container plus the leaf It. Every step carries a "# file:line" source
// reference, so a reader can jump to where any container — not only the
// failing leaf — is defined.
//
// The Given/And/Then keywords are SYNTHESISED from container nesting:
// Given for the outermost container, And for each inner container, Then
// for the leaf. They denote STRUCTURE (outer -> inner -> leaf), not
// Gherkin's precondition/action/assertion semantics; there are no step
// definitions behind them. Do not read behavioural intent into them.
type CucumberRenderer struct {
	color bool
}

// NewCucumber returns a CucumberRenderer. With color on it emits ANSI:
// green passed, red failed, cyan skipped, yellow pending.
func NewCucumber(color bool) *CucumberRenderer { return &CucumberRenderer{color: color} }

// WriteAll renders the full report.
func (c *CucumberRenderer) WriteAll(w io.Writer, r Report) error {
	a := NewANSI(w, c.color)
	a.printf("Feature: %s\n", r.Suite.Name)
	for _, s := range r.Specs {
		a.writeString("\n")
		c.writeScenario(a, s)
	}
	a.writeString("\n")
	c.writeFooter(a, r)
	return a.Err()
}

// maxAlignWidth bounds the step-body column used to align the
// "# file:line" comments within a scenario.
const maxAlignWidth = 120

// cukeStep is one rendered step: the "Keyword text" body, an optional
// "file:line" target, and whether it is the leaf Then.
type cukeStep struct {
	text   string
	loc    string
	isLeaf bool
}

// cucumberSteps builds the Given/And/Then step list for a spec.
func cucumberSteps(s SpecRow) []cukeStep {
	steps := make([]cukeStep, 0, len(s.ContainerHier)+1)
	for i, name := range s.ContainerHier {
		kw := "And"
		if i == 0 {
			kw = "Given"
		}
		loc := ""
		if i < len(s.ContainerLocations) {
			loc = s.ContainerLocations[i].String()
		}
		steps = append(steps, cukeStep{text: kw + " " + name, loc: loc})
	}
	steps = append(steps, cukeStep{text: "Then " + s.LeafText, loc: s.Location.String(), isLeaf: true})
	return steps
}

// writeScenario writes the Scenario header, its aligned steps, and any
// failure detail beneath the leaf.
func (c *CucumberRenderer) writeScenario(a *ANSI, s SpecRow) {
	leaf := s.LeafText
	if s.IsFocused {
		leaf = "[FOCUS] " + leaf
	}
	a.printf("  Scenario: %s\n", leaf)

	steps := cucumberSteps(s)
	width := 0
	for _, st := range steps {
		if n := utf8.RuneCountInString(st.text); n > width {
			width = n
		}
	}
	// Cap the alignment column: one pathologically long step name (hostile
	// or generated) would otherwise pad every sibling step to match,
	// amplifying output size. Beyond the cap, the long step simply breaks
	// alignment for its scenario.
	if width > maxAlignWidth {
		width = maxAlignWidth
	}
	for _, st := range steps {
		c.writeStep(a, st, s.State, width)
	}
	if isFailure(s) {
		c.writeFailure(a, s)
	}
}

// writeStep writes one indented, colored step line, padding the body to
// width so the "# file:line" comment column aligns within the scenario.
func (c *CucumberRenderer) writeStep(a *ANSI, st cukeStep, state State, width int) {
	a.writeString("    ")
	c.writeStepText(a, st.text, state, st.isLeaf)
	if st.loc != "" {
		// pad is negative only for a step longer than the capped width;
		// that step gets a single space before its ref instead.
		pad := max(width-utf8.RuneCountInString(st.text), 0)
		a.writeString(strings.Repeat(" ", pad) + " ")
		a.WriteDim("# " + st.loc)
	}
	a.writeString("\n")
}

// writeStepText writes the "Keyword text" body in the color for the spec
// state. On failure only the leaf Then is red while the containers stay
// green — a deliberate simplification keyed off the spec state alone,
// matching the state-only coloring of the sibling renderers. When the
// failure was in a container hook rather than the leaf, the "at file:line"
// detail under the Then step still names the real failure site.
func (c *CucumberRenderer) writeStepText(a *ANSI, text string, state State, isLeaf bool) {
	switch state {
	case StatePassed:
		a.WriteGreen(text)
	case StateSkipped:
		a.WriteCyan(text)
	case StatePending:
		a.WriteYellow(text)
	case StateFailed, StatePanicked, StateInterrupted, StateAborted:
		if isLeaf {
			a.WriteRed(text)
		} else {
			a.WriteGreen(text)
		}
	default:
		a.writeString(text)
	}
}

// writeFailure writes the failure location and detail block (matcher /
// message / stack / captured output) indented beneath the Then step. The
// detail is uncolored — identical in colored and plain modes — so the
// goldens are color-independent and match the text renderer's wording.
func (c *CucumberRenderer) writeFailure(a *ANSI, s SpecRow) {
	const prefix = "      "
	if loc := s.Failure.Location.String(); loc != "" {
		a.writeString(prefix + "at " + loc + "\n")
	}
	a.writeString(indentBlock(prefix, failureDetail(s)))
	a.writeString(indentBlock(prefix, failureExtras(s)))
}

// cukeTally accumulates per-category counts for the footer.
type cukeTally struct{ passed, failed, skipped, pending int }

// breakdown renders " (a passed, b failed, …)" with only non-zero
// categories, or "" when the tally is empty.
func (t cukeTally) breakdown() string {
	parts := make([]string, 0, 4)
	add := func(n int, label string) {
		if n > 0 {
			parts = append(parts, strconv.Itoa(n)+" "+label)
		}
	}
	add(t.passed, "passed")
	add(t.failed, "failed")
	add(t.skipped, "skipped")
	add(t.pending, "pending")
	if len(parts) == 0 {
		return ""
	}
	return " (" + strings.Join(parts, ", ") + ")"
}

// writeFooter writes the scenario tally, step tally, duration, and — when
// a suite setup/teardown node failed — a final Suite failure line.
func (c *CucumberRenderer) writeFooter(a *ANSI, r Report) {
	var sc, st cukeTally
	for _, s := range r.Specs {
		nSteps := len(s.ContainerHier) + 1
		switch s.State {
		case StatePassed:
			sc.passed++
			st.passed += nSteps
		case StateSkipped:
			sc.skipped++
			st.skipped += nSteps
		case StatePending:
			sc.pending++
			st.pending += nSteps
		default: // failed / panicked / interrupted / aborted
			// Mirror the step coloring: the leaf counts as the one failed
			// step regardless of which node actually failed (see
			// writeStepText); the failure detail names the real site.
			sc.failed++
			st.passed += nSteps - 1
			st.failed++
		}
	}
	a.printf("%d %s%s\n", len(r.Specs), pluralize(len(r.Specs), "scenario", "scenarios"), sc.breakdown())
	total := st.passed + st.failed + st.skipped + st.pending
	a.printf("%d %s%s\n", total, pluralize(total, "step", "steps"), st.breakdown())
	a.writeString(formatDurMs(r.EndTime.Sub(r.StartTime)) + "\n")
	if r.Suite.SpecialFailure != "" {
		a.writeString("Suite failure: " + r.Suite.SpecialFailure + "\n")
	}
}

// pluralize returns one when n == 1, otherwise many.
func pluralize(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
