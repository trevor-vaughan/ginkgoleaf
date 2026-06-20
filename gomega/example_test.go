package gomega_test

import (
	. "github.com/onsi/gomega"

	leafg "github.com/trevor-vaughan/ginkgoleaf/gomega"
)

// ExampleExpect is the drop-in replacement for gomega.Expect inside a Ginkgo
// spec. On failure it attaches a structured MatcherEvent to the running spec
// via AddReportEntry, which ginkgoleaf's renderers turn into a clean
// expected/actual diff; on success it behaves exactly like gomega.Expect, so
// the standard chain (To, ToNot, Should, ...) works unchanged.
//
// It needs a running Ginkgo spec, so this example is compiled as documentation
// rather than executed.
func ExampleExpect() {
	leafg.Expect(2 + 2).To(Equal(4))
}
