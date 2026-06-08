package ginkgoleaf

// ResetForTest clears registration state so an external test package can
// register again. It lives in a _test.go file so it is compiled only under
// test and never appears in the public API surface.
func ResetForTest() {
	registry.mu.Lock()
	registry.attached = false
	registry.mu.Unlock()
}
