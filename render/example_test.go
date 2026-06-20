package render_test

import (
	"fmt"
	"log"
	"os"

	"github.com/trevor-vaughan/ginkgoleaf/render"
)

// ExampleNew selects a renderer by format. New is the single construction
// point shared by the library and the CLI, so the format-to-renderer mapping
// cannot drift; an unsupported format returns a wrapped [render.ErrUnknownFormat].
func ExampleNew() {
	r, err := render.New(render.FormatText, false)
	fmt.Printf("%T err=%v\n", r, err)

	_, err = render.New("bogus", false)
	fmt.Println(err)
	// Output:
	// *render.TextRenderer err=<nil>
	// unknown format "bogus"
}

// ExampleRenderer shows the build-your-own-pipeline path: construct a canonical
// [render.Report] by hand (or via [render.Translate] from a Ginkgo report),
// pick a [render.Renderer] with [render.New], and call WriteAll. This never
// links the Ginkgo runtime.
func ExampleRenderer() {
	report := render.Report{
		Suite: render.SuiteRow{
			Name:           "math suite",
			NumSpecs:       2,
			NumPassed:      1,
			NumFailed:      1,
			SuiteSucceeded: false,
		},
		Specs: []render.SpecRow{
			{LeafText: "adds numbers", State: render.StatePassed},
			{
				LeafText: "divides numbers",
				State:    render.StateFailed,
				FullText: []string{"math", "divides numbers"},
				Failure: &render.FailureRow{
					Message:  "expected 2 to equal 3",
					Location: render.CodeLocation{FileName: "math_test.go", LineNumber: 42},
				},
			},
		},
	}

	r, err := render.New(render.FormatText, false)
	if err != nil {
		log.Fatal(err)
	}
	if err := r.WriteAll(os.Stdout, report); err != nil {
		log.Fatal(err)
	}
	// Output:
	// [+] adds numbers (0ms)
	// [X] divides numbers (0ms)
	//     at math_test.go:42
	//     expected 2 to equal 3
	//
	// FAIL: math suite | total 2 | pass 1 | fail 1 | skip 0 | pending 0 | panic 0 | duration 0ms
	//
	// Failures:
	//   [X] math > divides numbers
	//       at math_test.go:42
	//       expected 2 to equal 3
}
