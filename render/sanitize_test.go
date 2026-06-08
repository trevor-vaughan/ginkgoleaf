package render

import (
	"strings"
	"testing"
)

func TestSanitizeLine(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"plain", "Outer > Inner", "Outer > Inner"},
		{"newline", "a\nb", "a\\nb"},
		{"carriage return", "a\rb", "a\\rb"},
		{"tab", "a\tb", "a\\tb"},
		{"escape stripped", "a\x1b[31mb", "a[31mb"},
		{"github command", "\n::error::x", "\\n::error::x"},
		{"gitlab section", "\x1b[0Ksection_start:9:x\rY", "[0Ksection_start:9:x\\rY"},
		{"other C0 stripped", "a\x00\x07b", "ab"},
		{"unicode kept", "café ✓", "café ✓"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := sanitizeLine(c.in); got != c.want {
				t.Fatalf("sanitizeLine(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
	once := sanitizeLine("a\nb\x1b[31m")
	if twice := sanitizeLine(once); twice != once {
		t.Fatalf("not idempotent: %q -> %q", once, twice)
	}
}

func TestSanitizeBlock(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"plain", "expected: 2\nactual: 1", "expected: 2\nactual: 1"},
		{"newline preserved", "a\nb", "a\nb"},
		{"tab preserved", "a\tb", "a\tb"},
		{"bare carriage return stripped", "a\rb", "ab"},
		{"crlf collapses to lf", "a\r\nb", "a\nb"},
		{"escape stripped", "a\x1b[31mb", "a[31mb"},
		{"gitlab section sequence neutralized", "x\x1b[0Ksection_start:9:evil\rY", "x[0Ksection_start:9:evilY"},
		{"other C0 stripped", "a\x00\x07b", "ab"},
		{"C1 control rune stripped", "a\u009fb", "ab"},
		{"unicode kept", "café ✓\nnext", "café ✓\nnext"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := sanitizeBlock(c.in); got != c.want {
				t.Fatalf("sanitizeBlock(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
	once := sanitizeBlock("a\nb\x1b[31m\rc")
	if twice := sanitizeBlock(once); twice != once {
		t.Fatalf("not idempotent: %q -> %q", once, twice)
	}
}

// failureMessage must strip control bytes from a matcher attached via the
// gomega entry path, whose Expected/Actual/Raw do not pass through the
// translate-time message sanitizer.
func TestFailureMessageSanitizesEntryMatcher(t *testing.T) {
	s := SpecRow{
		State:   StateFailed,
		Failure: &FailureRow{Message: "ignored"},
		Matcher: &MatcherEvent{
			Kind:     MatcherEqual,
			Expected: "want\x1b[31m",
			Actual:   "got\x1b[0Ksection_start:0:x\rY",
		},
	}
	got := failureMessage(s)
	if strings.Contains(got, "\x1b") {
		t.Fatalf("entry-path matcher leaked ESC: %q", got)
	}
	if strings.Contains(got, "\r") {
		t.Fatalf("entry-path matcher leaked bare CR: %q", got)
	}
}
