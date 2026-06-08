package render

// State represents the outcome of a single spec.
type State int

// State constants.
const (
	StateUnknown State = iota
	StatePassed
	StateFailed
	StateSkipped
	StatePending
	StatePanicked
	StateInterrupted
	StateAborted
)

// String returns a lowercase label for the state, suitable for shell/tap output.
func (s State) String() string {
	switch s {
	case StatePassed:
		return "passed"
	case StateFailed:
		return "failed"
	case StateSkipped:
		return "skipped"
	case StatePending:
		return "pending"
	case StatePanicked:
		return "panicked"
	case StateInterrupted:
		return "interrupted"
	case StateAborted:
		return "aborted"
	default:
		return "unknown"
	}
}
