package render

import "fmt"

// New constructs the Renderer for f with color resolved by the caller.
// Returns a wrapped ErrUnknownFormat for an unsupported format. This is
// the single construction point shared by the library and the CLI so the
// format-to-renderer mapping cannot drift.
func New(f Format, color bool) (Renderer, error) {
	switch f {
	case FormatTree:
		return NewTree(color), nil
	case FormatJest:
		return NewJest(color), nil
	case FormatMarkdown:
		return NewMarkdown(), nil
	case FormatGitHub:
		return NewGitHub(), nil
	case FormatGitLab:
		return NewGitLab(nil, color), nil
	case FormatText:
		return NewText(), nil
	case FormatShell:
		return NewShell(), nil
	case FormatTAP:
		return NewTAP(), nil
	case FormatCucumber:
		return NewCucumber(color), nil
	default:
		return nil, fmt.Errorf("%w %q", ErrUnknownFormat, string(f))
	}
}
