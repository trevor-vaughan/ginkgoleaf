package gomega

import (
	"fmt"
	"reflect"

	"github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"

	"github.com/trevor-vaughan/ginkgoleaf/render"
)

func init() {
	render.SetEntryDecoder(Decode)
}

// Expect wraps gomega.Expect so that on failure the structured matcher
// data is attached to the current spec via AddReportEntry.
//
// Usage:
//
//	import leafg "github.com/trevor-vaughan/ginkgoleaf/gomega"
//	leafg.Expect(actual).To(Equal(expected))
//
// The standard gomega Expect chain still works alongside; this helper
// is purely additive.
func Expect(actual any, extra ...any) gomega.Assertion {
	return wrapped{inner: gomega.ExpectWithOffset(1, actual, extra...), actual: actual}
}

// wrapped delegates every Assertion method to inner, and emits a
// structured MatcherEvent report entry on assertion failure.
type wrapped struct {
	inner  gomega.Assertion
	actual any
}

// To satisfies gomega.Assertion. Reports structured metadata on failure.
func (w wrapped) To(matcher gomega.OmegaMatcher, optionalDescription ...any) (passed bool) {
	defer w.reportOnFailure(matcher, false, &passed)
	passed = w.inner.To(matcher, optionalDescription...)
	return passed
}

// ToNot satisfies gomega.Assertion. Reports structured metadata on failure.
func (w wrapped) ToNot(matcher gomega.OmegaMatcher, optionalDescription ...any) (passed bool) {
	defer w.reportOnFailure(matcher, true, &passed)
	passed = w.inner.ToNot(matcher, optionalDescription...)
	return passed
}

// NotTo satisfies gomega.Assertion. Reports structured metadata on failure.
func (w wrapped) NotTo(matcher gomega.OmegaMatcher, optionalDescription ...any) (passed bool) {
	defer w.reportOnFailure(matcher, true, &passed)
	passed = w.inner.NotTo(matcher, optionalDescription...)
	return passed
}

// Should satisfies gomega.Assertion. Reports structured metadata on failure.
func (w wrapped) Should(matcher gomega.OmegaMatcher, optionalDescription ...any) (passed bool) {
	defer w.reportOnFailure(matcher, false, &passed)
	passed = w.inner.Should(matcher, optionalDescription...)
	return passed
}

// ShouldNot satisfies gomega.Assertion. Reports structured metadata on failure.
func (w wrapped) ShouldNot(matcher gomega.OmegaMatcher, optionalDescription ...any) (passed bool) {
	defer w.reportOnFailure(matcher, true, &passed)
	passed = w.inner.ShouldNot(matcher, optionalDescription...)
	return passed
}

// reportOnFailure is the deferred half of every wrapped assertion. It
// attaches the structured MatcherEvent whenever the assertion did not
// pass, handling both failure shapes:
//
//   - Ginkgo's Fail handler panics, so the inner assertion never returns;
//     we recover, attach the report, then re-raise the same panic so
//     Ginkgo still records the spec failure.
//   - A non-panicking handler (e.g. gomega.InterceptGomegaFailures) lets
//     the inner assertion return false; *passed is false and we attach
//     the report directly.
//
// A passing assertion (*passed true, no panic) attaches nothing. Without
// this recover the report() call after a panicking inner assertion was
// unreachable, so no metadata was ever emitted under Ginkgo.
func (w wrapped) reportOnFailure(matcher gomega.OmegaMatcher, negated bool, passed *bool) {
	r := recover()
	if r != nil || !*passed {
		w.report(matcher, negated)
	}
	if r != nil {
		panic(r)
	}
}

// WithOffset satisfies gomega.Assertion.
func (w wrapped) WithOffset(offset int) gomega.Assertion {
	return wrapped{inner: w.inner.WithOffset(offset), actual: w.actual}
}

// Error satisfies gomega.Assertion.
func (w wrapped) Error() gomega.Assertion {
	return wrapped{inner: w.inner.Error(), actual: w.actual}
}

// report attaches a MatcherEvent report entry to the running Ginkgo spec.
// negated is true when the assertion is of the "should NOT" family so we
// use NegatedFailureMessage instead of FailureMessage.
func (w wrapped) report(matcher gomega.OmegaMatcher, negated bool) {
	var rawMsg string
	if negated {
		rawMsg = matcher.NegatedFailureMessage(w.actual)
	} else {
		rawMsg = matcher.FailureMessage(w.actual)
	}
	ev := render.MatcherEvent{
		Kind:     classify(matcher),
		Expected: matcherExpected(matcher),
		Actual:   w.actual,
		Raw:      rawMsg,
	}
	ginkgo.AddReportEntry(EntryName, Encode(ev))
}

// matcherMeta records how to interpret one of gomega's built-in
// matchers: the MatcherKind it maps to, and the name of the exported
// struct field that holds its expected value (empty when the matcher has
// none, e.g. boolean matchers).
type matcherMeta struct {
	kind          render.MatcherKind
	expectedField string
}

// matcherTable is the single source of truth for the gomega matchers we
// render structurally. It is keyed by the matcher's runtime type string
// so this package keeps no compile-time dependency on gomega's internal
// matchers package. The field names differ per matcher (verified against
// gomega: Equal.Expected, ContainSubstring.Substr, MatchJSON.JSONToMatch,
// MatchYAML.YAMLToMatch) and are read reflectively by matcherExpected.
var matcherTable = map[string]matcherMeta{
	"*matchers.EqualMatcher":            {render.MatcherEqual, "Expected"},
	"*matchers.ContainSubstringMatcher": {render.MatcherContainSubstring, "Substr"},
	"*matchers.MatchJSONMatcher":        {render.MatcherMatchJSON, "JSONToMatch"},
	"*matchers.MatchYAMLMatcher":        {render.MatcherMatchYAML, "YAMLToMatch"},
}

// classify maps a matcher's concrete type to a MatcherKind constant,
// falling back to MatcherGeneric for anything not in matcherTable.
func classify(m gomega.OmegaMatcher) render.MatcherKind {
	if meta, ok := matcherTable[fmt.Sprintf("%T", m)]; ok {
		return meta.kind
	}
	return render.MatcherGeneric
}

// matcherExpected extracts the expected value from a known matcher by
// reflecting its expected-value field. Gomega's matchers expose that
// value as an exported struct field (not a method), so a method-style
// assertion never matches — reflection is the dependency-free way to
// read it. Returns nil for unknown matchers or matchers with no expected
// field.
func matcherExpected(m gomega.OmegaMatcher) any {
	meta, ok := matcherTable[fmt.Sprintf("%T", m)]
	if !ok || meta.expectedField == "" {
		return nil
	}
	v := reflect.ValueOf(m)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	f := v.FieldByName(meta.expectedField)
	if !f.IsValid() || !f.CanInterface() {
		return nil
	}
	return f.Interface()
}
