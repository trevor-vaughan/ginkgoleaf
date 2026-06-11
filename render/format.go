package render

import (
	"errors"
	"fmt"
	"strings"
)

// Format selects an output format. Use the Format* constants — string
// values other than these are rejected by ValidateFormat.
type Format string

// The set of formats this version supports.
const (
	FormatTree     Format = "tree"
	FormatJest     Format = "jest"
	FormatMarkdown Format = "markdown"
	FormatGitHub   Format = "github"
	FormatGitLab   Format = "gitlab"
	FormatText     Format = "text"
	FormatShell    Format = "shell"
	FormatTAP      Format = "tap"
	FormatCucumber Format = "cucumber"
)

// allFormats is the canonical, ordered list of supported formats. It is
// the single source of truth for both validation and the CLI usage
// string, so the two cannot drift apart.
var allFormats = []Format{
	FormatTree, FormatJest, FormatMarkdown, FormatGitHub,
	FormatGitLab, FormatText, FormatShell, FormatTAP, FormatCucumber,
}

// ErrUnknownFormat is returned by ValidateFormat for unsupported values.
// The sentinel carries no program-name prefix; the caller owns that.
var ErrUnknownFormat = errors.New("unknown format")

// ValidateFormat returns nil for a supported format, otherwise a message
// wrapping ErrUnknownFormat that echoes the bad value and the valid set
// (so the message self-corrects). Match the cause with errors.Is.
func ValidateFormat(f Format) error {
	for _, v := range allFormats {
		if v == f {
			return nil
		}
	}
	return fmt.Errorf("%w %q; want one of: %s", ErrUnknownFormat, string(f), FormatList())
}

// FormatList returns the supported formats as a pipe-separated string,
// in canonical order, for use in CLI usage text.
func FormatList() string {
	names := make([]string, len(allFormats))
	for i, f := range allFormats {
		names[i] = string(f)
	}
	return strings.Join(names, "|")
}
