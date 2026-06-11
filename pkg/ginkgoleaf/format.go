package ginkgoleaf

import "github.com/trevor-vaughan/ginkgoleaf/render"

// Format selects an output format. The canonical definition lives in the
// leaf render package (which does not link the Ginkgo runtime); these
// aliases keep the library's public surface — ginkgoleaf.FormatTree and
// friends — stable for in-suite callers.
type Format = render.Format

// The set of formats this version supports.
const (
	FormatTree     = render.FormatTree
	FormatJest     = render.FormatJest
	FormatMarkdown = render.FormatMarkdown
	FormatGitHub   = render.FormatGitHub
	FormatGitLab   = render.FormatGitLab
	FormatText     = render.FormatText
	FormatShell    = render.FormatShell
	FormatTAP      = render.FormatTAP
	FormatCucumber = render.FormatCucumber
)

// ValidateFormat returns ErrUnknownFormat for unsupported values.
var ValidateFormat = render.ValidateFormat
