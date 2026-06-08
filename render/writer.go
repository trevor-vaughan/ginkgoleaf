package render

import (
	"fmt"
	"io"
)

// errWriter wraps an io.Writer and records the first write error, becoming a
// no-op afterward. It lets a renderer issue a run of writes and check Err()
// once at the end instead of after every call — the "errors are values"
// idiom (https://go.dev/blog/errors-are-values). The colored ANSI writer
// embeds it so every renderer shares one error-accumulation mechanism.
type errWriter struct {
	w   io.Writer
	err error
}

// Write forwards to the underlying writer until the first error, then stops.
func (e *errWriter) Write(p []byte) (int, error) {
	if e.err != nil {
		return 0, e.err
	}
	n, err := e.w.Write(p)
	if err != nil {
		e.err = err
	}
	return n, err
}

// Err returns the first write error encountered, or nil.
func (e *errWriter) Err() error { return e.err }

// printf writes a formatted string through the sticky Write; the error is
// captured in e.err and surfaced by Err(), so callers need not check it.
func (e *errWriter) printf(format string, a ...any) { _, _ = fmt.Fprintf(e, format, a...) }

// writeString writes s through the sticky Write; see printf.
func (e *errWriter) writeString(s string) { _, _ = io.WriteString(e, s) }
