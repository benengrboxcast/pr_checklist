package api

import (
	"testing"

	"github.com/benanderson/pr-checklist/internal/githubapi"
)

func TestMergeReviewRequests_SortsByStateThenAge(t *testing.T) {
	requested := []githubapi.SearchResult{
		{Owner: "acme", Repo: "a", Number: 2, Title: "newer requested", CreatedAt: "2024-01-02T00:00:00Z"},
		{Owner: "acme", Repo: "a", Number: 1, Title: "older requested", CreatedAt: "2024-01-01T00:00:00Z"},
	}
	reviewed := []githubapi.SearchResult{
		{Owner: "acme", Repo: "a", Number: 3, Title: "reviewed", CreatedAt: "2023-01-01T00:00:00Z"}, // older than everything, but reviewed still sorts after review_requested
	}

	out := mergeReviewRequests(requested, reviewed, nil)

	if len(out) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(out))
	}
	// review_requested entries first, oldest of that group before newer
	if out[0].Title != "older requested" || out[0].Status != ReviewRequested {
		t.Errorf("out[0] = %+v, want older requested/review_requested first", out[0])
	}
	if out[1].Title != "newer requested" || out[1].Status != ReviewRequested {
		t.Errorf("out[1] = %+v, want newer requested/review_requested second", out[1])
	}
	// reviewed entry last, even though it's chronologically the oldest PR overall
	if out[2].Title != "reviewed" || out[2].Status != Reviewed {
		t.Errorf("out[2] = %+v, want reviewed last despite being oldest", out[2])
	}
}

func TestMergeReviewRequests_IgnoredSortsLastRegardlessOfAge(t *testing.T) {
	requested := []githubapi.SearchResult{
		{Owner: "acme", Repo: "a", Number: 1, Title: "requested", CreatedAt: "2024-06-01T00:00:00Z"},
		{Owner: "acme", Repo: "a", Number: 2, Title: "ignored but oldest", CreatedAt: "2020-01-01T00:00:00Z"},
	}

	out := mergeReviewRequests(requested, nil, []string{"acme/a#2"})

	if len(out) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(out))
	}
	if out[0].Title != "requested" || out[0].Status != ReviewRequested {
		t.Errorf("out[0] = %+v, want the non-ignored requested PR first", out[0])
	}
	if out[1].Title != "ignored but oldest" || out[1].Status != Ignored {
		t.Errorf("out[1] = %+v, want the ignored PR last despite being oldest", out[1])
	}
}

func TestMergeReviewRequests_RequestedTakesPriorityOverPastReview(t *testing.T) {
	requested := []githubapi.SearchResult{
		{Owner: "acme", Repo: "a", Number: 1, Title: "re-requested", CreatedAt: "2024-01-01T00:00:00Z"},
	}
	reviewed := []githubapi.SearchResult{
		{Owner: "acme", Repo: "a", Number: 1, Title: "re-requested", CreatedAt: "2024-01-01T00:00:00Z"},
	}

	out := mergeReviewRequests(requested, reviewed, nil)

	if len(out) != 1 {
		t.Fatalf("expected the duplicate to be deduplicated into 1 entry, got %d", len(out))
	}
	if out[0].Status != ReviewRequested {
		t.Errorf("status = %v, want ReviewRequested to win over a past Reviewed for the same PR", out[0].Status)
	}
}
