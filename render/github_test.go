package render_test

import (
	"bytes"
	"strings"
	"testing"
	"unicode"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"

	"github.com/trevor-vaughan/ginkgoleaf/internal/testfx"
	"github.com/trevor-vaughan/ginkgoleaf/render"
)

var _ = scenarioTable("GitHub renderer renders each scenario matching the golden file",
	func(scenario string) {
		r := testfx.Report(scenario)
		renderer := render.NewGitHub()
		var buf bytes.Buffer

		Expect(renderer.WriteAll(&buf, r)).To(Succeed(),
			"WriteAll failed for scenario %q", scenario)

		testfx.Golden(GinkgoT(), "github", scenario, buf.Bytes())
	},
)

func TestGitHubFailureBodyCannotForgeCommand(t *testing.T) {
	in := types.Report{
		SuiteDescription: "S",
		SpecReports: types.SpecReports{{
			State:                   types.SpecStateFailed,
			ContainerHierarchyTexts: []string{"Group"},
			LeafNodeText:            "fails",
			Failure: types.Failure{
				Message:  "boom\n::error file=/etc/passwd,line=1,title=PWNED::owned",
				Location: types.CodeLocation{FileName: "f.go", LineNumber: 1},
			},
		}},
	}
	var buf bytes.Buffer
	if err := render.NewGitHub().WriteAll(&buf, render.Translate(in)); err != nil {
		t.Fatal(err)
	}
	// Emulate the Actions runner (ActionCommand.TryParseV2): a log line is a
	// workflow command when, after TrimStart (which strips Unicode whitespace),
	// it begins with "::"; the property segment lies between that "::" and the
	// next "::". ::stop-commands::<tok> suspends parsing until ::<tok>::. The
	// forgery succeeds only if a recognized command carries the attacker's
	// property (file=/etc/passwd) — appearing inside another command's data
	// segment (where the renderer %0A-encodes newlines) is harmless.
	var stopTok string
	for _, line := range strings.Split(buf.String(), "\n") {
		cmd := strings.TrimLeftFunc(line, unicode.IsSpace)
		if stopTok != "" {
			if cmd == "::"+stopTok+"::" {
				stopTok = ""
			}
			continue // command parsing suspended
		}
		if rest, ok := strings.CutPrefix(cmd, "::stop-commands::"); ok {
			stopTok = rest
			continue
		}
		if !strings.HasPrefix(cmd, "::") {
			continue
		}
		end := strings.Index(cmd[2:], "::")
		if end < 0 {
			continue
		}
		props := cmd[2 : 2+end] // "<name> k=v,k=v"
		if strings.Contains(props, "file=/etc/passwd") || strings.Contains(props, "title=PWNED") {
			t.Fatalf("forged workflow command would be recognized by the runner: %q", line)
		}
	}
}
