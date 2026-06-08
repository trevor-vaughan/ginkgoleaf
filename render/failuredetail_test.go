package render_test

import (
	"bytes"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/trevor-vaughan/ginkgoleaf/internal/testfx"
	"github.com/trevor-vaughan/ginkgoleaf/render"
)

// The mixed scenario carries a real failure (with a gomega Matcher) and a
// skipped spec whose Skip("…") reason lives in Failure. Failure detail
// must be confined to real failures across every format (matching tree),
// and the matcher must render in the parsed expected/actual form
// everywhere — not the raw "Expected … to equal …" gomega text.
const skipReason = "skipping: needs network"

func renderMixed(r render.Renderer) string {
	var buf bytes.Buffer
	ExpectWithOffset(1, r.WriteAll(&buf, testfx.Report(testfx.ScenarioMixed))).To(Succeed())
	return buf.String()
}

var _ = Describe("Failure detail is confined to real failures", func() {
	Describe("github", func() {
		It("never emits an ::error annotation for a skipped spec", func() {
			out := renderMixed(render.NewGitHub())
			for _, line := range strings.Split(out, "\n") {
				if strings.HasPrefix(line, "::error") {
					Expect(line).NotTo(ContainSubstring("is skipped"),
						"a skipped spec must not become a GitHub error annotation")
				}
			}
			Expect(out).NotTo(ContainSubstring(skipReason),
				"the skip reason must not surface in github output")
		})

		It("annotates the failure with the parsed matcher form", func() {
			out := renderMixed(render.NewGitHub())
			var errLine string
			for _, line := range strings.Split(out, "\n") {
				if strings.HasPrefix(line, "::error") && strings.Contains(line, "equals fails") {
					errLine = line
				}
			}
			Expect(errLine).NotTo(BeEmpty(), "expected an ::error for the real failure")
			Expect(errLine).To(ContainSubstring("expected:"))
			Expect(errLine).NotTo(ContainSubstring("to equal"),
				"annotation should use the parsed matcher, not the raw gomega text")
		})
	})

	Describe("tap", func() {
		It("emits no severity:fail diagnostic for a skipped spec", func() {
			out := renderMixed(render.NewTAP())
			Expect(strings.Count(out, "severity:")).To(Equal(1),
				"only the single real failure should carry a YAML diagnostic")
			Expect(out).NotTo(ContainSubstring(skipReason),
				"the skip reason must not surface as a TAP diagnostic")
		})

		It("uses the parsed matcher form in the failure diagnostic", func() {
			out := renderMixed(render.NewTAP())
			Expect(out).To(ContainSubstring("expected:"))
			Expect(out).NotTo(ContainSubstring("to equal"))
		})
	})

	Describe("jest", func() {
		It("renders a skipped spec without failure detail", func() {
			Expect(renderMixed(render.NewJest(false))).NotTo(ContainSubstring(skipReason))
		})
	})

	Describe("text", func() {
		It("renders a skipped spec without failure detail", func() {
			Expect(renderMixed(render.NewText())).NotTo(ContainSubstring(skipReason))
		})
	})
})
