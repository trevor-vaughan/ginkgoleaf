package render

import (
	"errors"
	"testing"
)

type countWriter struct {
	n       int
	failAt  int // fail the write that would push total bytes past this
	written int
}

func (c *countWriter) Write(p []byte) (int, error) {
	if c.failAt > 0 && c.written+len(p) > c.failAt {
		return 0, errors.New("boom")
	}
	c.written += len(p)
	c.n++
	return len(p), nil
}

func TestErrWriterRecordsFirstErrorAndStopsWriting(t *testing.T) {
	cw := &countWriter{failAt: 3}
	ew := &errWriter{w: cw}

	_, _ = ew.Write([]byte("ab")) // ok (2 bytes)
	_, _ = ew.Write([]byte("cd")) // fails (would be 4 > 3)
	_, _ = ew.Write([]byte("ef")) // must be a no-op after the error

	if ew.Err() == nil {
		t.Fatal("expected Err() to report the failed write")
	}
	if cw.n != 1 {
		t.Fatalf("expected exactly 1 successful underlying write, got %d", cw.n)
	}
}

func TestErrWriterErrNilWhenAllSucceed(t *testing.T) {
	ew := &errWriter{w: &countWriter{}}
	_, _ = ew.Write([]byte("a"))
	_, _ = ew.Write([]byte("b"))
	if ew.Err() != nil {
		t.Fatalf("expected nil Err() on success, got %v", ew.Err())
	}
}
