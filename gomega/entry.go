// Package gomega is the opt-in ginkgoleaf integration that wraps gomega
// assertions to attach structured matcher metadata. Importing this
// package is the only path that pulls gomega into a consumer's go.sum.
package gomega

import (
	"encoding/json"
	"time"

	"github.com/trevor-vaughan/ginkgoleaf/render"
)

// EntryName is the AddReportEntry name used to stash MatcherEvent
// payloads. The renderer looks for this exact string. It is the same
// constant as render.EntryName; re-exported here so callers who import
// only this sub-package do not need to also import render/.
const EntryName = render.EntryName

// Encode serialises a MatcherEvent to a JSON string.
func Encode(ev render.MatcherEvent) string {
	wire := wireForm{
		Kind: int(ev.Kind),
		Raw:  ev.Raw,
	}
	if ev.Expected != nil {
		wire.Expected = encodeAny(ev.Expected)
	}
	if ev.Actual != nil {
		wire.Actual = encodeAny(ev.Actual)
	}
	if ev.Retries != nil {
		wire.Retries = &wireRetry{
			DurationNS: ev.Retries.Duration.Nanoseconds(),
			PollingNS:  ev.Retries.Polling.Nanoseconds(),
			Attempts:   ev.Retries.Attempts,
			LastErr:    ev.Retries.LastErr,
		}
	}
	// wireForm holds only scalars (int/string/int64), so Marshal cannot fail.
	b, _ := json.Marshal(wire)
	return string(b)
}

// Decode reverses Encode. Returns ok=false on malformed input.
func Decode(s string) (render.MatcherEvent, bool) {
	var wire wireForm
	if err := json.Unmarshal([]byte(s), &wire); err != nil {
		return render.MatcherEvent{}, false
	}
	ev := render.MatcherEvent{
		Kind: render.MatcherKind(wire.Kind),
		Raw:  wire.Raw,
	}
	if wire.Expected != "" {
		ev.Expected = decodeAny(wire.Expected)
	}
	if wire.Actual != "" {
		ev.Actual = decodeAny(wire.Actual)
	}
	if wire.Retries != nil {
		ev.Retries = &render.RetryContext{
			Duration: time.Duration(wire.Retries.DurationNS),
			Polling:  time.Duration(wire.Retries.PollingNS),
			Attempts: wire.Retries.Attempts,
			LastErr:  wire.Retries.LastErr,
		}
	}
	return ev, true
}

type wireForm struct {
	Kind     int        `json:"kind"`
	Expected string     `json:"expected,omitempty"`
	Actual   string     `json:"actual,omitempty"`
	Raw      string     `json:"raw,omitempty"`
	Retries  *wireRetry `json:"retries,omitempty"`
}

type wireRetry struct {
	DurationNS int64  `json:"duration_ns"`
	PollingNS  int64  `json:"polling_ns"`
	Attempts   int    `json:"attempts"`
	LastErr    string `json:"last_err,omitempty"`
}

func encodeAny(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func decodeAny(s string) any {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		// Best-effort: a value that does not round-trip as JSON degrades to
		// its raw string form rather than being dropped.
		return s
	}
	return v
}
