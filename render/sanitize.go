package render

import "strings"

// sanitizeLine neutralizes control characters in an untrusted single-line
// field (a spec description, container name, or file path) before it is
// written into line-structured output. \t, \n, \r are escaped to their
// two-character forms so they cannot break the line or inject CI
// directives (GitHub `::command::`, GitLab `section_start`, TAP `not ok`);
// ESC (0x1b) and every other C0/C1 control byte is stripped so untrusted
// text cannot emit ANSI escapes. A string with no control characters is
// returned unchanged, and the function is idempotent.
func sanitizeLine(s string) string {
	if !strings.ContainsFunc(s, isControl) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\t':
			b.WriteString(`\t`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		default:
			if isControl(r) {
				continue
			}
			b.WriteRune(r)
		}
	}
	return b.String()
}

// isControl reports whether r is a C0 or C1 control character.
func isControl(r rune) bool {
	return r < 0x20 || (r >= 0x7f && r <= 0x9f)
}

// sanitizeBlock neutralizes control characters in an untrusted multi-line
// field (a failure message, stack trace, captured output, progress report)
// before it is written into output. Real line structure is preserved — \n
// and \t pass through — but ESC (0x1b), bare \r, and every other C0/C1
// control byte is stripped so untrusted text cannot emit ANSI escapes,
// drive GitLab \r section sequences, or overwrite terminal lines. A string
// with no such characters is returned unchanged; the function is idempotent.
func sanitizeBlock(s string) string {
	if !strings.ContainsFunc(s, isBlockControl) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '\n' || r == '\t' {
			b.WriteRune(r)
			continue
		}
		if isControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// isBlockControl reports whether r is a control character that is unsafe
// inside a multi-line block — a C0/C1 control other than \n or \t.
func isBlockControl(r rune) bool {
	return isControl(r) && r != '\n' && r != '\t'
}

// sanitizeLines returns a new slice with sanitizeLine applied to each
// element. The input is not modified.
func sanitizeLines(ss []string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = sanitizeLine(s)
	}
	return out
}
