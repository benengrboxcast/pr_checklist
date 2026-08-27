// Package domain holds pure derivation logic: thread state, PR status color,
// and tag rules. None of it touches the network or disk, so it's fully unit-testable.
package domain

import "github.com/benanderson/pr-checklist/internal/store"

// ThreadState is the derived (never stored) state of a review thread.
type ThreadState string

const (
	ThreadUnresolved    ThreadState = "unresolved"
	ThreadResolvedLocal ThreadState = "resolved_locally"
	ThreadResolved      ThreadState = "resolved"
)

// ComputeThreadState derives a thread's display state. GitHub's own resolved
// flag always wins, regardless of any local flag.
func ComputeThreadState(t store.ReviewThread) ThreadState {
	if t.IsResolvedOnGitHub {
		return ThreadResolved
	}
	if t.LocalResolved {
		return ThreadResolvedLocal
	}
	return ThreadUnresolved
}
