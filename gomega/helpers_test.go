package gomega_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	leafg "github.com/trevor-vaughan/ginkgoleaf/gomega"
	"github.com/trevor-vaughan/ginkgoleaf/render"
)

var _ = Describe("Encode/Decode", func() {
	It("round-trips a MatcherEvent through the entry encoding without data loss", func() {
		ev := render.MatcherEvent{
			Kind:     render.MatcherEqual,
			Expected: "two",
			Actual:   "one",
			Raw:      "Expected\n    one\nto equal\n    two",
		}
		enc := leafg.Encode(ev)
		dec, ok := leafg.Decode(enc)

		Expect(ok).To(BeTrue(), "Decode should succeed for a well-formed encoded event")
		Expect(dec.Kind).To(Equal(render.MatcherEqual))
		Expect(dec.Expected).To(Equal("two"))
		Expect(dec.Actual).To(Equal("one"))
	})

	DescribeTable("Decode rejects malformed input at the deserialization boundary",
		func(in string) {
			ev, ok := leafg.Decode(in)
			Expect(ok).To(BeFalse(), "Decode(%q) should report failure", in)
			Expect(ev).To(Equal(render.MatcherEvent{}), "a failed Decode must return a zero event")
		},
		Entry("not JSON", "{not json"),
		Entry("empty string", ""),
		Entry("truncated object", `{"kind":1`),
	)
})

var _ = Describe("EntryName", func() {
	It("is the canonical report entry key 'ginkgoleaf.matcher'", func() {
		Expect(leafg.EntryName).To(Equal("ginkgoleaf.matcher"))
	})
})
