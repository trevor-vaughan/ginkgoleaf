package render_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/trevor-vaughan/ginkgoleaf/render"
)

var _ = Describe("State", func() {
	DescribeTable("String returns the lowercase label",
		func(s render.State, want string) {
			Expect(s.String()).To(Equal(want))
		},
		Entry("Passed", render.StatePassed, "passed"),
		Entry("Failed", render.StateFailed, "failed"),
		Entry("Skipped", render.StateSkipped, "skipped"),
		Entry("Pending", render.StatePending, "pending"),
		Entry("Panicked", render.StatePanicked, "panicked"),
		Entry("Interrupted", render.StateInterrupted, "interrupted"),
		Entry("Aborted", render.StateAborted, "aborted"),
		Entry("Unknown", render.StateUnknown, "unknown"),
		Entry("out-of-range value falls through to 'unknown'", render.State(99), "unknown"),
	)
})
