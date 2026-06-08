// Command ginkgoleaf renders a Ginkgo JSON report into one of the
// supported output formats.
//
//	ginkgoleaf [--in PATH|-] [--out PATH|-] [--format FMT] [--color auto|always|never] [--exit-code]
//	ginkgoleaf --version
//
// The CLI deliberately does NOT import the ginkgoleaf library package
// (which links the Ginkgo runtime); it depends only on the leaf render
// and parse packages, so `--help` shows the six real flags rather than
// the Ginkgo framework's test flags, and the binary stays small.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"github.com/onsi/ginkgo/v2/types"
	"golang.org/x/term"

	"github.com/trevor-vaughan/ginkgoleaf/internal/parse"
	"github.com/trevor-vaughan/ginkgoleaf/render"
)

// Build-time variables populated by GoReleaser via -ldflags -X.
// Local `go build` / `go install` leave them at their defaults; the
// version then falls back to runtime/debug build info.
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

// errInvalidInput marks input that can't be rendered (malformed JSON, no
// reports). It maps to exit code 2 — "invalid input" — per the README
// contract, distinct from a render error (exit 1).
var errInvalidInput = errors.New("invalid input")

// maxInputBytes bounds the report we will buffer/parse from an untrusted
// source so a giant or unbounded stream cannot exhaust memory.
const maxInputBytes = 256 << 20

// decodeReports parses the Ginkgo report array from r, refusing input larger
// than max bytes. The cap is a parameter so it is testable without a
// max-sized fixture.
func decodeReports(r io.Reader, max int64) ([]types.Report, error) {
	lr := &io.LimitedReader{R: r, N: max + 1}
	var reports []types.Report
	if err := json.NewDecoder(lr).Decode(&reports); err != nil {
		if lr.N <= 0 {
			return nil, fmt.Errorf("%w: input exceeds %d bytes", errInvalidInput, max)
		}
		return nil, fmt.Errorf("%w: %v", errInvalidInput, err)
	}
	if len(reports) == 0 {
		return nil, fmt.Errorf("%w: no reports", errInvalidInput)
	}
	return reports, nil
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run is the testable entry point: it parses args, wires I/O, and returns
// the process exit code. main is a thin os.Exit(run(...)) wrapper so that
// run's defers (closing input/output) always execute — os.Exit would skip
// them. stdin/stdout/stderr are injected so tests can drive it in-process.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) (code int) {
	fs := flag.NewFlagSet("ginkgoleaf", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { writeUsage(stderr) }
	var (
		inPath      = fs.String("in", "-", "input path (- for stdin)")
		outPath     = fs.String("out", "-", "output path (- for stdout)")
		format      = fs.String("format", "tree", "format ("+render.FormatList()+")")
		color       = fs.String("color", "auto", "color mode (auto|always|never)")
		exitCode    = fs.Bool("exit-code", false, "exit 1 if the report contains failures")
		showVersion = fs.Bool("version", false, "print version information and exit")
	)
	if err := fs.Parse(args); err != nil {
		return 2 // flag.ContinueOnError already printed the error + usage
	}

	if *showVersion {
		info, _ := debug.ReadBuildInfo()
		v, c, d := resolveVersion(version, commit, date, info)
		_, _ = fmt.Fprintf(stdout, "ginkgoleaf %s (commit %s, built %s)\n", v, c, d)
		return 0
	}

	// Bare invocation: no --in and stdin is an interactive terminal. Reading
	// stdin would block forever waiting for keystrokes, so show usage instead
	// of hanging. Only meaningful when stdin is a real *os.File (production).
	if f, ok := stdin.(*os.File); ok && noInputOnTTY(*inPath, f) {
		writeUsage(stderr)
		return 2
	}

	in, err := openInput(*inPath, stdin)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "ginkgoleaf:", err)
		return 2
	}
	defer func() { _ = in.Close() }()

	out, closeOut, err := openOutput(*outPath, stdout)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "ginkgoleaf:", err)
		return 2
	}
	// A file's flush-on-close is where a delayed write error (full disk,
	// quota) surfaces, so a close failure must not be reported as success.
	defer func() {
		if cerr := closeOut(); cerr != nil && code == 0 {
			_, _ = fmt.Fprintln(stderr, "ginkgoleaf:", cerr)
			code = 1
		}
	}()

	ok, err := Run(in, out, *format, *color)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "ginkgoleaf:", err)
		if errors.Is(err, render.ErrUnknownFormat) ||
			errors.Is(err, render.ErrUnknownColor) ||
			errors.Is(err, errInvalidInput) {
			return 2
		}
		return 1
	}
	// --exit-code is opt-in: it overlays exit 1 (otherwise "render error")
	// with "the report contained failures", giving a single-command gate
	// without changing the default contract.
	if *exitCode && !ok {
		return 1
	}
	return 0
}

// writeUsage prints the CLI's own short usage — the six real flags and
// the exit-code contract — instead of the default flag dump. It is wired
// to flag.Usage so a typo'd flag prints this rather than a wall of
// inherited framework flags.
func writeUsage(w io.Writer) {
	// Best-effort: usage text goes to the flag output stream; a write
	// failure there is not actionable and flag.Usage cannot propagate it.
	_, _ = fmt.Fprint(w, `ginkgoleaf — render a Ginkgo JSON report into a chosen format

Usage:
  ginkgoleaf [--in PATH|-] [--out PATH|-] [--format FMT] [--color MODE]
  ginkgoleaf --version

Flags:
  -in      PATH | -   Ginkgo JSON report (default: stdin)
  -out     PATH | -   output path; a file path writes there only (stdout
                      stays silent), - is stdout (default)
  -format  FMT        `+render.FormatList()+` (default: tree)
  -color   MODE       auto | always | never (default: auto)
  -exit-code          exit 1 if the report contains failures (default: off)
  -version            print build info and exit

Exit codes:
  0  rendered cleanly
  1  render error (or report contained failures, with -exit-code)
  2  invalid input / unknown format or color
`)
}

// resolveVersion picks the version triple to report. When GoReleaser
// injected a real version via ldflags it is used verbatim; otherwise the
// values fall back to the module's build info (module version, VCS
// revision and commit time), which `go install` and `go build` embed.
func resolveVersion(ldVersion, ldCommit, ldDate string, info *debug.BuildInfo) (v, c, d string) {
	v, c, d = ldVersion, ldCommit, ldDate
	if ldVersion != "dev" || info == nil {
		return v, c, d
	}
	if mv := info.Main.Version; mv != "" && mv != "(devel)" {
		v = mv
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if s.Value != "" {
				c = s.Value
			}
		case "vcs.time":
			if s.Value != "" {
				d = s.Value
			}
		}
	}
	return v, c, d
}

// Run reads a Ginkgo JSON report from r, renders it in the named format,
// and writes the bytes to w. It is the testable core of the CLI.
//
// Ginkgo's --json-report produces an array of types.Report; Run expects
// the same shape.
// It also reports ok: whether every rendered suite succeeded. Callers
// that want a pass/fail exit code (the --exit-code flag) consult it; the
// render itself is independent of pass/fail.
func Run(r io.Reader, w io.Writer, format, color string) (ok bool, err error) {
	if err := render.ValidateFormat(render.Format(format)); err != nil {
		return false, err
	}
	if _, err := render.ParseColorMode(color); err != nil {
		return false, err
	}

	reports, err := decodeReports(r, maxInputBytes)
	if err != nil {
		return false, err
	}

	ok = true
	for _, rep := range reports {
		canon := render.TranslateWithParser(rep, parse.ParseGomega)
		if !canon.Suite.SuiteSucceeded {
			ok = false
		}
		if err := renderOne(w, render.Format(format), color, canon); err != nil {
			return false, err
		}
	}
	return ok, nil
}

func renderOne(w io.Writer, f render.Format, color string, canon render.Report) error {
	rdr, err := render.New(f, resolveCLIColor(color, w))
	if err != nil {
		return err
	}
	return rdr.WriteAll(w, canon)
}

// resolveCLIColor decides whether to emit ANSI for the given writer. The
// mode string is validated up front in Run via ParseColorMode, so any
// parse error here is unreachable and an unknown mode degrades to auto.
func resolveCLIColor(mode string, w io.Writer) bool {
	m, _ := render.ParseColorMode(mode)
	return render.ColorEnabled(m, w)
}

// noInputOnTTY reports whether the CLI was invoked with no input source:
// the default stdin (--in "-") attached to an interactive terminal.
// A file redirect, a pipe, or /dev/null is not a terminal, so those
// still flow through to the reader.
func noInputOnTTY(inPath string, stdin *os.File) bool {
	return inPath == "-" && term.IsTerminal(int(stdin.Fd()))
}

func openInput(path string, stdin io.Reader) (io.ReadCloser, error) {
	if path == "-" {
		return io.NopCloser(stdin), nil
	}
	f, err := os.Open(path)
	if err != nil {
		// os.Open's *PathError already reads "open <path>: <reason>";
		// add only the role, not a second "open <path>".
		return nil, fmt.Errorf("reading input: %w", err)
	}
	return f, nil
}

func openOutput(path string, stdout io.Writer) (io.Writer, func() error, error) {
	if path == "-" {
		return stdout, func() error { return nil }, nil
	}
	f, err := os.Create(path)
	if err != nil {
		// os.Create's *PathError already names the path; add only the role.
		return nil, nil, fmt.Errorf("writing output: %w", err)
	}
	return f, f.Close, nil
}
