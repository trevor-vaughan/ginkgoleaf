package gomega_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGomega(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ginkgoleaf/gomega")
}
