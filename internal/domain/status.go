package domain

import "github.com/benanderson/pr-checklist/internal/store"

// Status is the derived overall color/state for a PR.
type Status string

const (
	StatusGray  Status = "gray"  // PR is closed or merged
	StatusRed   Status = "red"   // unresolved threads, or changes requested
	StatusGreen Status = "green" // everything resolved and approved
	StatusAmber Status = "amber" // clean-ish but not fully green
)

// ThreadCounts tallies review threads by derived state.
type ThreadCounts struct {
	Total         int `json:"total"`
	Unresolved    int `json:"unresolved"`
	ResolvedLocal int `json:"resolvedLocally"`
	Resolved      int `json:"resolved"`
}

// CountThreads tallies a PR's review threads by derived state.
func CountThreads(rec store.PRRecord) ThreadCounts {
	var c ThreadCounts
	c.Total = len(rec.ReviewThreads)
	for _, t := range rec.ReviewThreads {
		switch ComputeThreadState(t) {
		case ThreadUnresolved:
			c.Unresolved++
		case ThreadResolvedLocal:
			c.ResolvedLocal++
		case ThreadResolved:
			c.Resolved++
		}
	}
	return c
}

// ComputePRStatus derives the overall status color for a PR. First matching
// rule wins, in priority order: closed/merged, then unresolved/changes-requested,
// then fully-resolved-and-approved, else amber.
func ComputePRStatus(rec store.PRRecord) Status {
	if rec.State == "CLOSED" || rec.State == "MERGED" {
		return StatusGray
	}

	counts := CountThreads(rec)
	if counts.Unresolved > 0 || rec.ReviewDecision == "CHANGES_REQUESTED" {
		return StatusRed
	}

	// ReviewDecision == "" covers GitHub's null case; only APPROVED counts as approved.
	if counts.ResolvedLocal == 0 && counts.Unresolved == 0 && rec.ReviewDecision == "APPROVED" {
		return StatusGreen
	}

	return StatusAmber
}
