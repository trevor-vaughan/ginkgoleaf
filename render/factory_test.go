package render_test

import (
	"errors"
	"testing"

	"github.com/trevor-vaughan/ginkgoleaf/render"
)

func TestNewAllFormats(t *testing.T) {
	for _, f := range []render.Format{
		render.FormatTree, render.FormatJest, render.FormatMarkdown, render.FormatGitHub,
		render.FormatGitLab, render.FormatText, render.FormatShell, render.FormatTAP,
		render.FormatCucumber,
	} {
		r, err := render.New(f, false)
		if err != nil || r == nil {
			t.Fatalf("New(%q): r=%v err=%v", f, r, err)
		}
	}
	if _, err := render.New(render.Format("bogus"), false); !errors.Is(err, render.ErrUnknownFormat) {
		t.Fatalf("New(bogus): want ErrUnknownFormat, got %v", err)
	}
}
