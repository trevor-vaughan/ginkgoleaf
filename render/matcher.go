package render

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
)

// FormatMatcher returns a renderer-agnostic, multi-line string
// representing a MatcherEvent suitable for display under a failing spec.
// Returns the empty string for an empty/unknown event.
func FormatMatcher(ev MatcherEvent) string {
	switch ev.Kind {
	case MatcherEqual:
		return formatEqualDiff(ev)
	case MatcherContainSubstring:
		return formatSubstring(ev)
	case MatcherMatchJSON:
		return formatStructuredDiff(ev, parseJSON)
	case MatcherMatchYAML:
		return formatStructuredDiff(ev, parseYAML)
	case MatcherEventually, MatcherConsistently:
		return formatRetryDiff(ev)
	case MatcherGeneric:
		return ev.Raw
	case MatcherUnknown:
		return ""
	default:
		return ev.Raw
	}
}

func formatEqualDiff(ev MatcherEvent) string {
	a := stringify(ev.Actual)
	e := stringify(ev.Expected)
	if !strings.Contains(a, "\n") && !strings.Contains(e, "\n") {
		return fmt.Sprintf("expected: %s\nactual:   %s", e, a)
	}
	return "diff (-expected +actual):\n" + normalizeDiff(cmp.Diff(e, a))
}

func formatSubstring(ev MatcherEvent) string {
	needle := stringify(ev.Expected)
	haystack := stringify(ev.Actual)
	return fmt.Sprintf("needle:   %q\nhaystack: %q", needle, haystack)
}

type parseFn func(string) (any, error)

func formatStructuredDiff(ev MatcherEvent, fn parseFn) string {
	a, errA := fn(stringify(ev.Actual))
	e, errE := fn(stringify(ev.Expected))
	if errA != nil || errE != nil {
		return formatEqualDiff(ev)
	}
	return "diff (-expected +actual):\n" + normalizeDiff(cmp.Diff(e, a))
}

func parseJSON(s string) (any, error) {
	var v any
	err := json.Unmarshal([]byte(s), &v)
	return v, err
}

func parseYAML(s string) (any, error) {
	var v any
	err := yaml.Unmarshal([]byte(s), &v)
	return v, err
}

func formatRetryDiff(ev MatcherEvent) string {
	sub := ""
	if ev.Expected != nil || ev.Actual != nil {
		sub = formatEqualDiff(ev) + "\n"
	}
	if ev.Retries == nil {
		return sub + ev.Raw
	}
	rc := ev.Retries
	kind := "eventually"
	if ev.Kind == MatcherConsistently {
		kind = "consistently"
	}
	footer := fmt.Sprintf("(%s: %s after %d attempts", kind, rc.Duration, rc.Attempts)
	if rc.Polling > 0 {
		footer += fmt.Sprintf(", polling every %s", rc.Polling)
	}
	footer += ")"
	if rc.LastErr != "" {
		footer += "\nlast error: " + rc.LastErr
	}
	return sub + footer
}

func stringify(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// normalizeDiff replaces the non-breaking spaces (U+00A0, encoded as \xc2\xa0
// in UTF-8) that go-cmp randomly injects in its diff output with regular ASCII
// spaces. This produces stable golden-file output regardless of which random
// branch go-cmp's report_text.go takes at init time.
func normalizeDiff(s string) string {
	return strings.ReplaceAll(s, "\xc2\xa0", " ")
}
