package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
)

func passingReportJSON() string {
	in := types.Report{
		SuiteDescription: "My Suite",
		SuiteSucceeded:   true,
		StartTime:        time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC),
		EndTime:          time.Date(2026, 5, 28, 12, 0, 0, 50_000_000, time.UTC),
		SpecReports: []types.SpecReport{
			{
				ContainerHierarchyTexts: []string{"Outer"},
				LeafNodeText:            "does the thing",
				LeafNodeLocation:        types.CodeLocation{FileName: "t.go", LineNumber: 1},
				State:                   types.SpecStatePassed,
				RunTime:                 12 * time.Millisecond,
			},
		},
	}
	j, err := json.Marshal([]types.Report{in})
	Expect(err).NotTo(HaveOccurred())
	return string(j)
}

func failingReportJSON() string {
	in := types.Report{
		SuiteDescription: "My Suite",
		SuiteSucceeded:   false,
		StartTime:        time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC),
		EndTime:          time.Date(2026, 5, 28, 12, 0, 0, 50_000_000, time.UTC),
		SpecReports: []types.SpecReport{
			{
				ContainerHierarchyTexts: []string{"Outer"},
				LeafNodeText:            "breaks",
				LeafNodeLocation:        types.CodeLocation{FileName: "t.go", LineNumber: 9},
				State:                   types.SpecStateFailed,
				RunTime:                 8 * time.Millisecond,
			},
		},
	}
	j, err := json.Marshal([]types.Report{in})
	Expect(err).NotTo(HaveOccurred())
	return string(j)
}

var _ = Describe("openInput", func() {
	It("does not duplicate the os PathError prefix for a missing file", func() {
		_, err := openInput("/no/such/ginkgoleaf/report.json", nil)
		Expect(err).To(HaveOccurred())
		// os.Open returns a *PathError that already reads
		// "open /no/such/...: no such file or directory". Wrapping it with
		// our own "open %s:" produced "open X: open X: ...". The op+path
		// must appear exactly once.
		Expect(strings.Count(err.Error(), "/no/such/ginkgoleaf/report.json")).To(Equal(1),
			"the file path should appear once, not be double-wrapped")
		Expect(strings.Count(err.Error(), "open ")).To(Equal(1),
			"the 'open' verb should appear once")
	})
})

var _ = Describe("openOutput", func() {
	It("does not duplicate the os PathError path when the target is unwritable", func() {
		_, _, err := openOutput("/no/such/ginkgoleaf/dir/out.txt", nil)
		Expect(err).To(HaveOccurred())
		Expect(strings.Count(err.Error(), "/no/such/ginkgoleaf/dir/out.txt")).To(Equal(1),
			"the output path should appear once, not be double-wrapped")
	})
})

var _ = Describe("Run", func() {
	It("writes the leaf text and PASS, reporting success for a passing report", func() {
		var out bytes.Buffer
		ok, err := Run(strings.NewReader(passingReportJSON()), &out, "text", "never")
		Expect(err).To(Succeed())
		Expect(ok).To(BeTrue(), "a passing suite must report ok=true")
		Expect(out.String()).To(ContainSubstring("does the thing"),
			"leaf node text must appear in the output")
		Expect(out.String()).To(ContainSubstring("PASS"),
			"PASS summary must appear in the output")
	})

	It("reports failure (ok=false) when the report contains a failed suite", func() {
		var out bytes.Buffer
		ok, err := Run(strings.NewReader(failingReportJSON()), &out, "text", "never")
		Expect(err).To(Succeed(), "rendering a failing report is not itself an error")
		Expect(ok).To(BeFalse(), "a failed suite must report ok=false")
	})

	It("classifies malformed JSON as invalid input (exit 2)", func() {
		var out bytes.Buffer
		_, err := Run(strings.NewReader("not json"), &out, "text", "never")
		Expect(errors.Is(err, errInvalidInput)).To(BeTrue(),
			"malformed JSON should map to invalid input, got %v", err)
	})

	It("classifies an empty report set as invalid input (exit 2)", func() {
		var out bytes.Buffer
		_, err := Run(strings.NewReader("[]"), &out, "text", "never")
		Expect(errors.Is(err, errInvalidInput)).To(BeTrue(),
			"no reports should map to invalid input, got %v", err)
	})

	It("returns an error when an unknown format is requested", func() {
		var out bytes.Buffer
		_, err := Run(strings.NewReader("[]"), &out, "bogus", "never")
		Expect(err).To(HaveOccurred(),
			"Run should return an error for an unrecognised format name")
	})

	It("returns an error when an unknown color mode is requested", func() {
		var out bytes.Buffer
		_, err := Run(strings.NewReader(passingReportJSON()), &out, "text", "purple")
		Expect(err).To(HaveOccurred(),
			"Run should reject an unrecognised color mode")
	})
})

var _ = Describe("decodeReports", func() {
	It("rejects input larger than the cap as invalid input", func() {
		// A valid report that is well over the tiny cap; the decoder hits the
		// limit mid-parse and we report it as oversized, not generic garbage.
		_, err := decodeReports(strings.NewReader(passingReportJSON()), 16)
		Expect(errors.Is(err, errInvalidInput)).To(BeTrue(), "got %v", err)
		Expect(err.Error()).To(ContainSubstring("exceeds"))
	})

	It("accepts input within the cap", func() {
		reports, err := decodeReports(strings.NewReader(passingReportJSON()), 1<<20)
		Expect(err).To(Succeed())
		Expect(reports).To(HaveLen(1))
	})
})

var _ = Describe("run", func() {
	It("returns 0 rendering a passing report from stdin", func() {
		var out, errOut bytes.Buffer
		code := run([]string{"--format=text", "--color=never"},
			strings.NewReader(passingReportJSON()), &out, &errOut)
		Expect(code).To(Equal(0))
		Expect(out.String()).To(ContainSubstring("PASS"))
	})

	It("returns 2 for an unknown format", func() {
		var out, errOut bytes.Buffer
		code := run([]string{"--format=bogus"}, strings.NewReader("[]"), &out, &errOut)
		Expect(code).To(Equal(2))
	})

	It("returns 1 with --exit-code when the report contains failures", func() {
		var out, errOut bytes.Buffer
		code := run([]string{"--format=text", "--color=never", "--exit-code"},
			strings.NewReader(failingReportJSON()), &out, &errOut)
		Expect(code).To(Equal(1))
	})

	It("returns 2 when the output path is unwritable", func() {
		var out, errOut bytes.Buffer
		code := run([]string{"--out=/no/such/ginkgoleaf/dir/out.txt"},
			strings.NewReader(passingReportJSON()), &out, &errOut)
		Expect(code).To(Equal(2))
	})
})
