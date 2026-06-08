package nogomega_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"

	// Importing ginkgoleaf without calling Register is the load-bearing
	// part of this fixture: the go.sum hygiene check verifies that a
	// consumer who pulls in ginkgoleaf does NOT transitively pull in
	// gomega. Removing this import would make the check vacuous.
	_ "github.com/trevor-vaughan/ginkgoleaf/pkg/ginkgoleaf"
)

// This fixture intentionally does not register a ginkgoleaf format
// in-suite. `task test:fixtures` renders the result via the ginkgoleaf
// CLI after Ginkgo writes its JSON report.

func TestNoGomega(t *testing.T) {
	RunSpecs(t, "No Gomega Suite")
}

var _ = Describe("Pure DSL", func() {
	It("passes", func() {
		if 1+1 != 2 {
			Fail("math broken")
		}
	})
	It("fails", func() {
		Fail("intentional failure")
	})
	PIt("is pending")
	It("is skipped", func() { Skip("intentional") })
})
