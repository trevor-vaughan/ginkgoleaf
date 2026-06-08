package gomega_test

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"

	leafg "github.com/trevor-vaughan/ginkgoleaf/gomega"
	"github.com/trevor-vaughan/ginkgoleaf/render"
)

// findMatcherEntry returns the decoded MatcherEvent the wrapper stashed
// under leafg.EntryName in a spec report, or ok=false if it attached
// nothing.
func findMatcherEntry(r types.SpecReport) (render.MatcherEvent, bool) {
	for _, e := range r.ReportEntries {
		if e.Name != leafg.EntryName {
			continue
		}
		raw, isStr := e.GetRawValue().(string)
		if !isStr {
			continue
		}
		return leafg.Decode(raw)
	}
	return render.MatcherEvent{}, false
}

// failWithPanic mimics Ginkgo's own Fail handler: it panics so the
// assertion's call stack unwinds without ever returning to the caller.
// This is the exact path the wrapper must survive — a handler that
// returns normally (e.g. InterceptGomegaFailure's) would not reproduce
// the defect that left report() unreachable under Ginkgo.
func failWithPanic(message string, _ ...int) {
	panic(message)
}

// driveFailingAssertion runs fn under a panicking fail handler (Ginkgo's
// real semantics), recovers the panic itself so the host spec stays
// green, and restores Ginkgo's Fail handler. It returns whether fn
// panicked — the wrapper MUST re-raise so a real spec still fails.
func driveFailingAssertion(fn func()) (rePanicked bool) {
	RegisterFailHandler(failWithPanic)
	defer RegisterFailHandler(Fail)
	defer func() {
		if recover() != nil {
			rePanicked = true
		}
	}()
	fn()
	return false
}

// wantEvent maps a spec's leaf text to the MatcherEvent its wrapped
// assertion must emit. Gomega exposes matcher identity only via concrete
// type, so classify()/matcherExpected() key off the runtime type string;
// this table pins that fragile contract — kind, the extracted expected
// value, and the captured actual — for every structured kind plus the
// generic fallback (which has no expected field).
var wantEvent = map[string]struct {
	kind     render.MatcherKind
	expected any
	actual   any
}{
	"Equal":            {render.MatcherEqual, "two", "one"},
	"ContainSubstring": {render.MatcherContainSubstring, "xyz", "hello"},
	"MatchJSON":        {render.MatcherMatchJSON, `{"a":2}`, `{"a":1}`},
	"MatchYAML":        {render.MatcherMatchYAML, "a: 2", "a: 1"},
	"generic":          {render.MatcherGeneric, nil, true},
}

var _ = Describe("wrapped assertion structured reporting", func() {
	// Report entries attached during a spec are only observable here —
	// not mid-spec via CurrentSpecReport. Every spec in this container is a
	// wrapped assertion: all but the passing one MUST attach an entry, so a
	// renamed or newly added failing spec can never silently skip the
	// emission check (it falls through to the default BeTrue). wantEvent
	// adds the kind/value assertions for the cases that have them.
	ReportAfterEach(func(r types.SpecReport) {
		ev, found := findMatcherEntry(r)
		if r.LeafNodeText == "passing assertion attaches nothing" {
			Expect(found).To(BeFalse(), "a passing assertion must not attach a matcher entry")
			return
		}
		Expect(found).To(BeTrue(), "every failing wrapped assertion must attach a ginkgoleaf.matcher entry")
		if want, ok := wantEvent[r.LeafNodeText]; ok {
			Expect(ev.Kind).To(Equal(want.kind))
			Expect(ev.Actual).To(Equal(want.actual))
			if want.expected == nil {
				Expect(ev.Expected).To(BeNil(), "a matcher with no expected field must emit a nil Expected")
			} else {
				Expect(ev.Expected).To(Equal(want.expected))
			}
		}
	})

	DescribeTable("To() classifies the matcher and attaches its MatcherEvent",
		func(actual any, matcher OmegaMatcher) {
			rePanicked := driveFailingAssertion(func() {
				leafg.Expect(actual).To(matcher)
			})
			Expect(rePanicked).To(BeTrue(),
				"the wrapper must re-raise the failure so Ginkgo still fails the spec")
		},
		Entry("Equal", "one", Equal("two")),
		Entry("ContainSubstring", "hello", ContainSubstring("xyz")),
		Entry("MatchJSON", `{"a":1}`, MatchJSON(`{"a":2}`)),
		Entry("MatchYAML", "a: 1", MatchYAML("a: 2")),
		Entry("generic", true, BeFalse()),
	)

	// One spec per negated verb so a copy-paste miswire (wrong inner
	// method or dropped report call) in any public entry point is caught.
	It("ToNot fails", func() {
		Expect(driveFailingAssertion(func() {
			leafg.Expect("x").ToNot(Equal("x"))
		})).To(BeTrue())
	})

	It("NotTo fails", func() {
		Expect(driveFailingAssertion(func() {
			leafg.Expect("x").NotTo(Equal("x"))
		})).To(BeTrue())
	})

	It("Should fails", func() {
		Expect(driveFailingAssertion(func() {
			leafg.Expect("x").Should(Equal("y"))
		})).To(BeTrue())
	})

	It("ShouldNot fails", func() {
		Expect(driveFailingAssertion(func() {
			leafg.Expect("x").ShouldNot(Equal("x"))
		})).To(BeTrue())
	})

	It("passing assertion attaches nothing", func() {
		leafg.Expect("one").To(Equal("one"))
	})
})
