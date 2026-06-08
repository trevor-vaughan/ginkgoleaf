package ginkgoleaf_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Note: we do NOT register ginkgoleaf in this suite. The ginkgoleaf_test.go
// file tests double-registration panics, and a var-level Register call here
// would pre-populate the registry state and break those tests.
func TestGinkgoleaf(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ginkgoleaf")
}
