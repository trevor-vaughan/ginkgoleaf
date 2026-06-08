package render_test

import (
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/trevor-vaughan/ginkgoleaf/render"
)

var _ = Describe("FormatMatcher", func() {
	DescribeTable("produces a diff for structured matcher events",
		func(ev render.MatcherEvent, mustContain []string) {
			got := render.FormatMatcher(ev)
			for _, s := range mustContain {
				Expect(got).To(ContainSubstring(s), "expected diff to contain %q in:\n%s", s, got)
			}
		},
		Entry("Equal — diff mentions both sides",
			render.MatcherEvent{
				Kind:     render.MatcherEqual,
				Expected: "alpha\nbravo\n",
				Actual:   "alpha\nzulu\n",
			},
			[]string{"bravo", "zulu"},
		),
		Entry("ContainSubstring — output includes the needle",
			render.MatcherEvent{
				Kind:     render.MatcherContainSubstring,
				Expected: "goodbye",
				Actual:   "hello world",
			},
			[]string{"goodbye"},
		),
		Entry("MatchJSON — diff shows changed key with old and new values",
			render.MatcherEvent{
				Kind:     render.MatcherMatchJSON,
				Expected: `{"a":1,"b":3}`,
				Actual:   `{"a":1,"b":2}`,
			},
			[]string{"b", "2", "3"},
		),
		Entry("MatchYAML — diff shows changed key with old and new values",
			render.MatcherEvent{
				Kind:     render.MatcherMatchYAML,
				Expected: "a: 1\nb: 3",
				Actual:   "a: 1\nb: 2",
			},
			[]string{"b", "2", "3"},
		),
	)

	It("returns the raw message verbatim for a Generic event", func() {
		ev := render.MatcherEvent{Kind: render.MatcherGeneric, Raw: "boom"}
		Expect(render.FormatMatcher(ev)).To(Equal("boom"))
	})

	It("returns an empty string for a zero-value event", func() {
		Expect(render.FormatMatcher(render.MatcherEvent{})).To(
			Equal(""),
			"empty MatcherEvent should produce no output but got non-empty",
		)
	})

	It("returns an empty string for an Unknown event", func() {
		// Explicitly verifying the default branch is separate from zero-value
		// so both the zero and explicit MatcherUnknown paths are covered.
		Expect(render.FormatMatcher(render.MatcherEvent{Kind: render.MatcherUnknown})).To(
			Equal(""),
			func() string { return "MatcherUnknown should produce no output" },
		)
	})

	// Sanity-check: ContainSubstring output is formatted, not a bare diff.
	It("formats ContainSubstring with labelled needle and haystack", func() {
		ev := render.MatcherEvent{
			Kind:     render.MatcherContainSubstring,
			Expected: "goodbye",
			Actual:   "hello world",
		}
		got := render.FormatMatcher(ev)
		Expect(got).To(ContainSubstring("needle"))
		Expect(strings.Contains(got, "haystack")).To(BeTrue())
	})
})

func TestFormatRetryDiffIncludesLastErr(t *testing.T) {
	ev := render.MatcherEvent{
		Kind:    render.MatcherEventually,
		Raw:     "timed out",
		Retries: &render.RetryContext{Duration: 2 * time.Second, Attempts: 5, LastErr: "expected 4 to equal 5"},
	}
	out := render.FormatMatcher(ev)
	if !strings.Contains(out, "last error: expected 4 to equal 5") {
		t.Fatalf("missing last error in:\n%s", out)
	}
}
