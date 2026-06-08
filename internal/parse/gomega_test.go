package parse_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/trevor-vaughan/ginkgoleaf/internal/parse"
	"github.com/trevor-vaughan/ginkgoleaf/render"
)

var _ = Describe("ParseGomega", func() {
	DescribeTable("identifies the matcher kind from the failure message",
		func(input string, wantKind render.MatcherKind) {
			got := parse.ParseGomega(input)
			Expect(got.Kind).To(Equal(wantKind),
				"input: %q", input)
		},
		Entry("Equal", "Expected\n    <int>: 1\nto equal\n    <int>: 2",
			render.MatcherEqual),
		Entry("ContainSubstring",
			"Expected\n    <string>: \"hello world\"\nto contain substring\n    <string>: \"goodbye\"",
			render.MatcherContainSubstring),
		Entry("MatchJSON",
			"Expected\n    <string>: {\"a\":1}\nto match JSON of\n    <string>: {\"a\":2}",
			render.MatcherMatchJSON),
		Entry("MatchYAML",
			"Expected\n    <string>: a: 1\nto match YAML of\n    <string>: a: 2",
			render.MatcherMatchYAML),
		Entry("generic unrecognized message", "totally unrecognized message",
			render.MatcherGeneric),
	)

	Describe("Equal matcher", func() {
		It("extracts actual and expected values and preserves the raw message", func() {
			in := "Expected\n    <int>: 1\nto equal\n    <int>: 2"
			got := parse.ParseGomega(in)

			Expect(got.Kind).To(Equal(render.MatcherEqual))
			Expect(got.Expected).To(Equal("<int>: 2"))
			Expect(got.Actual).To(Equal("<int>: 1"))
			Expect(got.Raw).To(Equal(in), "Raw field must preserve the original message")
		})
	})

	Describe("ContainSubstring matcher", func() {
		It("extracts the needle as Expected", func() {
			in := "Expected\n    <string>: \"hello world\"\nto contain substring\n    <string>: \"goodbye\""
			got := parse.ParseGomega(in)

			Expect(got.Kind).To(Equal(render.MatcherContainSubstring))
			Expect(got.Expected).To(Equal(`<string>: "goodbye"`))
		})
	})

	Describe("Eventually matcher", func() {
		It("parses the timeout duration and sets Retries", func() {
			in := "Timed out after 2.000s.\nExpected\n    <int>: 4\nto equal\n    <int>: 5"
			got := parse.ParseGomega(in)

			Expect(got.Kind).To(Equal(render.MatcherEventually))
			Expect(got.Retries).NotTo(BeNil(), "Retries must be populated for a timed-out assertion")
			Expect(got.Retries.Duration.String()).To(Equal("2s"))
		})
	})

	Describe("Consistently matcher", func() {
		It("parses the duration and polling interval and classifies as Consistently", func() {
			in := "Failed after 1.0s polling every 0.1s.\nExpected\n    <int>: 1\nto equal\n    <int>: 2"
			got := parse.ParseGomega(in)

			Expect(got.Kind).To(Equal(render.MatcherConsistently),
				"a 'polling every' header is the Consistently signature")
			Expect(got.Retries).NotTo(BeNil())
			Expect(got.Retries.Duration.String()).To(Equal("1s"))
			Expect(got.Retries.Polling.String()).To(Equal("100ms"))
		})
	})

	Describe("Generic matcher", func() {
		It("preserves the raw message verbatim", func() {
			in := "totally unrecognized message"
			got := parse.ParseGomega(in)

			Expect(got.Kind).To(Equal(render.MatcherGeneric))
			Expect(got.Raw).To(Equal(in))
		})
	})

	Describe("empty input", func() {
		It("returns MatcherUnknown for an empty string", func() {
			got := parse.ParseGomega("")
			Expect(got.Kind).To(Equal(render.MatcherUnknown))
		})
	})
})
