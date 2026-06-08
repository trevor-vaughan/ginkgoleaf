// Ginkgo suite entry point for the render package's tests.
//
// All tests are in the external `render_test` package so we can import
// ginkgoleaf itself and register a live formatter — running `go test`
// here exercises the same library the user runs against their suites.
// That is the strongest possible dogfood: our test output IS our
// production output.
package render_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/trevor-vaughan/ginkgoleaf/internal/testfx"
)

// No in-suite Register here. `task test` runs Ginkgo with
// --json-report and then pipes the JSON through the ginkgoleaf CLI to
// render the tree below — that mirrors the post-processing workflow
// downstream users adopt, and avoids interleaving our renderer with
// Ginkgo's own succinct output.

func TestRender(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "render")
}

// scenarioTable calls DescribeTable with one Entry per scenario name.
// The body function must accept a single string (the scenario name).
//
// Ginkgo's DescribeTable takes (description string, ...any) where the
// variadic holds the body function and Entry values all together. Because
// Go does not allow mixing non-spread and spread arguments in a variadic
// call, we build the complete args slice here and pass it via
// reflect-free two-step: body first, then entries.
func scenarioTable(description string, body interface{}) bool {
	args := make([]interface{}, 0, 1+len(testfx.AllScenarios()))
	args = append(args, body)
	for _, name := range testfx.AllScenarios() {
		args = append(args, Entry(name, name))
	}
	return DescribeTable(description, args...)
}
