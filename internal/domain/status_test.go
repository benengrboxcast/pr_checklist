package domain

import (
	"testing"

	"github.com/benanderson/pr-checklist/internal/store"
)

func rec(state, reviewDecision string, threads ...store.ReviewThread) store.PRRecord {
	return store.PRRecord{State: state, ReviewDecision: reviewDecision, ReviewThreads: threads}
}

func unresolved() store.ReviewThread {
	return store.ReviewThread{ID: "t", IsResolvedOnGitHub: false, LocalResolved: false}
}
func resolvedLocal() store.ReviewThread {
	return store.ReviewThread{ID: "t", IsResolvedOnGitHub: false, LocalResolved: true}
}
func resolved() store.ReviewThread { return store.ReviewThread{ID: "t", IsResolvedOnGitHub: true} }

func TestComputePRStatus(t *testing.T) {
	tests := []struct {
		name string
		rec  store.PRRecord
		want Status
	}{
		{"closed PR is gray regardless of threads", rec("CLOSED", "APPROVED", unresolved()), StatusGray},
		{"merged PR is gray regardless of threads", rec("MERGED", "", unresolved()), StatusGray},
		{"any unresolved thread is red", rec("OPEN", "APPROVED", resolved(), unresolved()), StatusRed},
		{"changes requested is red even with clean threads", rec("OPEN", "CHANGES_REQUESTED", resolved()), StatusRed},
		{"fully resolved and approved is green", rec("OPEN", "APPROVED", resolved(), resolved()), StatusGreen},
		{"fully resolved but null reviewDecision is amber, not green", rec("OPEN", "", resolved()), StatusAmber},
		{"fully resolved but review required is amber", rec("OPEN", "REVIEW_REQUIRED", resolved()), StatusAmber},
		{"resolved locally but approved is amber, not green", rec("OPEN", "APPROVED", resolvedLocal()), StatusAmber},
		{"no threads at all, not approved, is amber", rec("OPEN", ""), StatusAmber},
		{"no threads at all, approved, is green", rec("OPEN", "APPROVED"), StatusGreen},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputePRStatus(tc.rec)
			if got != tc.want {
				t.Errorf("ComputePRStatus() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestComputeThreadState(t *testing.T) {
	tests := []struct {
		name string
		t    store.ReviewThread
		want ThreadState
	}{
		{"github resolved wins over local flag", store.ReviewThread{IsResolvedOnGitHub: true, LocalResolved: true}, ThreadResolved},
		{"github resolved wins even without local flag", store.ReviewThread{IsResolvedOnGitHub: true, LocalResolved: false}, ThreadResolved},
		{"local resolved only", store.ReviewThread{IsResolvedOnGitHub: false, LocalResolved: true}, ThreadResolvedLocal},
		{"neither resolved", store.ReviewThread{IsResolvedOnGitHub: false, LocalResolved: false}, ThreadUnresolved},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ComputeThreadState(tc.t); got != tc.want {
				t.Errorf("ComputeThreadState() = %v, want %v", got, tc.want)
			}
		})
	}
}
