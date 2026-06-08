package withgomega_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// This fixture intentionally does NOT register ginkgoleaf in-suite.
// `task test:fixtures` runs ginkgo with --succinct (Ginkgo's own
// per-suite progress line goes to the user's terminal) plus
// --json-report, then pipes the JSON through `ginkgoleaf` to render
// the tree below. That demonstrates the CLI/post-processing workflow
// — the same pipeline downstream users adopt.

func TestWithGomega(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "With Gomega Suite")
}

var _ = Describe("Outer", func() {
	Describe("Inner", func() {
		It("passes", func() {
			Expect(1 + 1).To(Equal(2))
		})
		It("fails", func() {
			Expect(1).To(Equal(2))
		})
		PIt("is pending")
		It("is skipped", func() { Skip("intentional") })
	})
})
