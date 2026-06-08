// Package parse contains the best-effort gomega failure-text parser.
// It is intentionally permissive: any pattern it does not recognize
// returns MatcherGeneric with Raw set, so renderers degrade to verbatim.
package parse

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/trevor-vaughan/ginkgoleaf/render"
)

// ParseGomega extracts structured matcher data from a gomega failure
// string. Returns a zero-valued MatcherEvent with Kind=MatcherUnknown
// for empty input, MatcherGeneric for unrecognized non-empty input.
func ParseGomega(s string) render.MatcherEvent {
	if s == "" {
		return render.MatcherEvent{Kind: render.MatcherUnknown}
	}
	ev := render.MatcherEvent{Raw: s}

	if rc, body, ok := parseRetryHeader(s); ok {
		ev.Retries = rc
		if rc.Polling > 0 {
			ev.Kind = render.MatcherConsistently
		} else {
			ev.Kind = render.MatcherEventually
		}
		body2 := ParseGomega(body)
		if body2.Kind != render.MatcherGeneric && body2.Kind != render.MatcherUnknown {
			ev.Expected = body2.Expected
			ev.Actual = body2.Actual
		}
		return ev
	}

	if kind, act, exp, ok := parseExpectedToX(s); ok {
		ev.Kind = kind
		ev.Actual = act
		ev.Expected = exp
		return ev
	}

	ev.Kind = render.MatcherGeneric
	return ev
}

var (
	rxTimedOut   = regexp.MustCompile(`^Timed out after ([0-9.]+)s\.\n`)
	rxConsistent = regexp.MustCompile(`^Failed after ([0-9.]+)s polling every ([0-9.]+)s\.\n`)
	rxAttempts   = regexp.MustCompile(`Attempted ([0-9]+) times`)
)

func parseRetryHeader(s string) (*render.RetryContext, string, bool) {
	if m := rxTimedOut.FindStringSubmatch(s); m != nil {
		dur := parseSeconds(m[1])
		body := s[len(m[0]):]
		rc := &render.RetryContext{Duration: dur}
		if am := rxAttempts.FindStringSubmatch(s); am != nil {
			if n, err := strconv.Atoi(am[1]); err == nil {
				rc.Attempts = n
			}
		}
		return rc, body, true
	}
	if m := rxConsistent.FindStringSubmatch(s); m != nil {
		dur := parseSeconds(m[1])
		pol := parseSeconds(m[2])
		body := s[len(m[0]):]
		rc := &render.RetryContext{Duration: dur, Polling: pol}
		if am := rxAttempts.FindStringSubmatch(s); am != nil {
			if n, err := strconv.Atoi(am[1]); err == nil {
				rc.Attempts = n
			}
		}
		return rc, body, true
	}
	return nil, "", false
}

func parseSeconds(s string) time.Duration {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return time.Duration(f * float64(time.Second))
}

var rxExpectedToX = regexp.MustCompile(`(?s)^Expected\n(.*?)\nto (equal|contain substring|match JSON of|match YAML of)\n(.*)$`)

func parseExpectedToX(s string) (kind render.MatcherKind, actual, expected string, ok bool) {
	m := rxExpectedToX.FindStringSubmatch(s)
	if m == nil {
		return render.MatcherUnknown, "", "", false
	}
	switch m[2] {
	case "equal":
		kind = render.MatcherEqual
	case "contain substring":
		kind = render.MatcherContainSubstring
	case "match JSON of":
		kind = render.MatcherMatchJSON
	case "match YAML of":
		kind = render.MatcherMatchYAML
	}
	return kind, strings.TrimSpace(m[1]), strings.TrimSpace(m[3]), true
}
