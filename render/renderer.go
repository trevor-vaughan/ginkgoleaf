package render

import "io"

// Renderer turns a canonical Report into bytes on the given writer.
// Implementations must be safe to call sequentially; concurrency is
// handled by the registration layer.
type Renderer interface {
	// WriteAll renders the full report.
	WriteAll(w io.Writer, r Report) error
}
