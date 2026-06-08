package testfx

import (
	"flag"
	"os"
	"path/filepath"
)

// UpdateGolden is true when `go test -update` is passed.
var UpdateGolden = flag.Bool("update", false, "rewrite golden files from actual output")

// TestingT is the minimal slice of *testing.T we need. Defined locally
// (instead of using testing.TB) so Ginkgo's GinkgoT() shim — which
// can't satisfy testing.TB because of its sealed private() method — can
// plug in. Both *testing.T and Ginkgo's interface satisfy this set.
type TestingT interface {
	Helper()
	Fatalf(format string, args ...any)
}

// Golden reads (or writes when -update is set) the golden file at
// testdata/<format>/<scenario>.golden, compares its bytes to got, and
// fails the test on mismatch with a diff hint.
func Golden(t TestingT, format, scenario string, got []byte) {
	t.Helper()
	dir := filepath.Join("testdata", format)
	path := filepath.Join(dir, scenario+".golden")

	if *UpdateGolden {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with -update to create)", path, err)
	}
	if string(got) != string(want) {
		_ = os.MkdirAll(".test-output", 0o755)
		actualPath := filepath.Join(".test-output", format+"-"+scenario+".actual")
		_ = os.WriteFile(actualPath, got, 0o644)
		t.Fatalf("golden mismatch for %s/%s\n  want: %s\n  actual written to: %s\n  run `diff %s %s` to inspect",
			format, scenario, path, actualPath, path, actualPath)
	}
}
