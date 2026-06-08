package render

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/muesli/termenv"
)

// ColorMode mirrors the public ginkgoleaf.ColorMode without creating a
// dependency on the root package. Callers cast between the two types
// (both are int underneath with the same constant order).
type ColorMode int

// Color modes.
const (
	ColorAuto ColorMode = iota
	ColorAlways
	ColorNever
)

// ErrUnknownColor is returned by ParseColorMode for an unrecognised
// mode string. The sentinel carries no program-name prefix.
var ErrUnknownColor = errors.New("unknown color mode")

// ParseColorMode maps a CLI color string (auto|always|never) to a
// ColorMode, returning a message wrapping ErrUnknownColor otherwise.
func ParseColorMode(s string) (ColorMode, error) {
	switch s {
	case "auto":
		return ColorAuto, nil
	case "always":
		return ColorAlways, nil
	case "never":
		return ColorNever, nil
	default:
		return ColorAuto, fmt.Errorf("%w %q; want auto|always|never", ErrUnknownColor, s)
	}
}

// ColorEnabled reports whether a renderer should emit ANSI escapes when
// writing to w.
//
//   - ColorAlways forces color on; ColorNever forces it off.
//   - ColorAuto colors a color-capable terminal OR a pipe — so that
//     `ginkgoleaf … | less -R` is colored — but stays plain for a regular
//     file redirect (`> out.txt`) so captured output carries no escapes.
//     NO_COLOR / CLICOLOR override everywhere (https://no-color.org,
//     https://bixense.com/clicolors); CLICOLOR_FORCE forces color on.
func ColorEnabled(mode ColorMode, w io.Writer) bool {
	switch mode {
	case ColorAlways:
		return true
	case ColorNever:
		return false
	default:
		if termenv.EnvNoColor() {
			return false
		}
		if termenv.NewOutput(w).EnvColorProfile() != termenv.Ascii {
			return true // color-capable terminal (or CLICOLOR_FORCE)
		}
		return isPipe(w) // e.g. `| less`; regular files stay plain
	}
}

// isPipe reports whether w is an OS pipe (FIFO) — the case for a shell
// pipe such as `ginkgoleaf … | less`. Regular files, character devices
// (including /dev/null), and non-file writers return false.
func isPipe(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeNamedPipe != 0
}

// ANSI is a thin writer wrapper that emits or omits ANSI color codes based
// on the resolved enable flag. It embeds *errWriter, so its Write and Err
// methods (and the sticky-error semantics) are shared with the plain
// renderers: the Write* helpers below swallow per-call errors to stay terse,
// and renderers check Err() once before reporting success, so a broken pipe
// or full disk is not silently dropped.
type ANSI struct {
	*errWriter
	enable bool
}

// NewANSI constructs an ANSI writer.
func NewANSI(w io.Writer, enable bool) *ANSI {
	return &ANSI{errWriter: &errWriter{w: w}, enable: enable}
}

// WriteRed writes s in red when color is enabled, plain otherwise.
func (a *ANSI) WriteRed(s string) { a.writeColored("31", s) }

// WriteGreen writes s in green when color is enabled, plain otherwise.
func (a *ANSI) WriteGreen(s string) { a.writeColored("32", s) }

// WriteYellow writes s in yellow when color is enabled, plain otherwise.
func (a *ANSI) WriteYellow(s string) { a.writeColored("33", s) }

// WriteDim writes s dimmed when color is enabled, plain otherwise.
func (a *ANSI) WriteDim(s string) { a.writeColored("2", s) }

// WriteBold writes s in bold when color is enabled, plain otherwise.
func (a *ANSI) WriteBold(s string) { a.writeColored("1", s) }

func (a *ANSI) writeColored(code, s string) {
	// Writes go through the embedded errWriter, which records the first error
	// and no-ops afterward; a.Err() surfaces it.
	if a.enable {
		_, _ = io.WriteString(a.errWriter, "\x1b["+code+"m"+s+"\x1b[0m")
		return
	}
	_, _ = io.WriteString(a.errWriter, s)
}
