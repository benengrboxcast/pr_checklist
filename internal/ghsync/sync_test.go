package ghsync

import (
	"testing"
	"time"

	"github.com/benanderson/pr-checklist/internal/githubapi"
	"github.com/benanderson/pr-checklist/internal/store"
)

func TestMarkResolvedLocallyThenMergeRefresh_OwnReplyDoesNotRevert(t *testing.T) {
	existing := store.PRRecord{
		Owner: "acme", Repo: "widgets", Number: 1,
		AuthenticatedLogin: "me",
		ReviewThreads: []store.ReviewThread{
			{ID: "t1", Comments: []store.Comment{{ID: "c1", Author: "reviewer", CreatedAt: time.Now()}}},
		},
	}
	if err := MarkResolvedLocally(&existing, "t1"); err != nil {
		t.Fatalf("MarkResolvedLocally: %v", err)
	}
	if existing.ReviewThreads[0].LastSeenCommentIDAtResolve != "c1" {
		t.Fatalf("expected watermark c1, got %q", existing.ReviewThreads[0].LastSeenCommentIDAtResolve)
	}

	detail := githubapi.PRDetail{
		AuthenticatedLogin: "me",
		ReviewThreads: []githubapi.ReviewThread{
			{
				ID:         "t1",
				IsResolved: false,
				Comments: []githubapi.Comment{
					{ID: "c1", Author: "reviewer", CreatedAt: "2024-01-01T00:00:00Z"},
					{ID: "c2", Author: "me", CreatedAt: "2024-01-02T00:00:00Z"}, // my own follow-up
				},
			},
		},
	}

	merged := MergeRefresh(existing, detail)
	th := merged.ReviewThreads[0]
	if !th.LocalResolved {
		t.Errorf("expected LocalResolved to remain true after own reply, got false")
	}
}

func TestMarkResolvedLocallyThenMergeRefresh_OthersReplyReverts(t *testing.T) {
	existing := store.PRRecord{
		Owner: "acme", Repo: "widgets", Number: 1,
		AuthenticatedLogin: "me",
		ReviewThreads: []store.ReviewThread{
			{ID: "t1", Comments: []store.Comment{{ID: "c1", Author: "reviewer", CreatedAt: time.Now()}}},
		},
	}
	if err := MarkResolvedLocally(&existing, "t1"); err != nil {
		t.Fatalf("MarkResolvedLocally: %v", err)
	}

	detail := githubapi.PRDetail{
		AuthenticatedLogin: "me",
		ReviewThreads: []githubapi.ReviewThread{
			{
				ID:         "t1",
				IsResolved: false,
				Comments: []githubapi.Comment{
					{ID: "c1", Author: "reviewer", CreatedAt: "2024-01-01T00:00:00Z"},
					{ID: "c2", Author: "reviewer", CreatedAt: "2024-01-02T00:00:00Z"}, // someone else replied
				},
			},
		},
	}

	merged := MergeRefresh(existing, detail)
	th := merged.ReviewThreads[0]
	if th.LocalResolved {
		t.Errorf("expected LocalResolved to revert to false after other's reply, got true")
	}
	if th.LastSeenCommentIDAtResolve != "" {
		t.Errorf("expected watermark cleared, got %q", th.LastSeenCommentIDAtResolve)
	}
}

func TestMergeRefresh_GitHubResolvedOverridesLocalState(t *testing.T) {
	existing := store.PRRecord{
		AuthenticatedLogin: "me",
		ReviewThreads: []store.ReviewThread{
			{ID: "t1", LocalResolved: true, LastSeenCommentIDAtResolve: "c1"},
		},
	}
	detail := githubapi.PRDetail{
		AuthenticatedLogin: "me",
		ReviewThreads: []githubapi.ReviewThread{
			{ID: "t1", IsResolved: true, Comments: []githubapi.Comment{
				{ID: "c1", Author: "reviewer", CreatedAt: "2024-01-01T00:00:00Z"},
				{ID: "c2", Author: "reviewer", CreatedAt: "2024-01-02T00:00:00Z"},
			}},
		},
	}
	merged := MergeRefresh(existing, detail)
	if !merged.ReviewThreads[0].IsResolvedOnGitHub {
		t.Fatalf("expected IsResolvedOnGitHub true")
	}
}

func TestAutoTagIsAdditiveAndIdempotentAcrossRefreshes(t *testing.T) {
	rec := NewRecord("acme", "widgets", 1, githubapi.PRDetail{
		HeadRefName:        "EMBED-1234-fix-thing",
		AuthenticatedLogin: "me",
	})
	if len(rec.Tags) != 1 || rec.Tags[0] != "EMBED-1234" {
		t.Fatalf("expected auto-tag on new record, got %v", rec.Tags)
	}

	rec.Tags = append(rec.Tags, "backend") // simulate a manually-added tag

	refreshed := MergeRefresh(rec, githubapi.PRDetail{
		HeadRefName:        "EMBED-1234-fix-thing",
		AuthenticatedLogin: "me",
	})
	if len(refreshed.Tags) != 2 {
		t.Fatalf("expected auto-tag to stay additive (no duplicate) across refresh, got %v", refreshed.Tags)
	}

	refreshedAgain := MergeRefresh(refreshed, githubapi.PRDetail{
		HeadRefName:        "EMBED-1234-fix-thing",
		AuthenticatedLogin: "me",
	})
	if len(refreshedAgain.Tags) != 2 {
		t.Fatalf("expected repeated refresh to remain idempotent, got %v", refreshedAgain.Tags)
	}
}

func TestMarkResolvedLocally_RejectsAlreadyGitHubResolved(t *testing.T) {
	rec := store.PRRecord{ReviewThreads: []store.ReviewThread{{ID: "t1", IsResolvedOnGitHub: true}}}
	if err := MarkResolvedLocally(&rec, "t1"); err != ErrAlreadyResolvedOnGitHub {
		t.Fatalf("expected ErrAlreadyResolvedOnGitHub, got %v", err)
	}
}

func TestMarkUnresolvedLocally_ClearsState(t *testing.T) {
	now := time.Now()
	rec := store.PRRecord{ReviewThreads: []store.ReviewThread{
		{ID: "t1", LocalResolved: true, LocalResolvedAt: &now, LastSeenCommentIDAtResolve: "c1"},
	}}
	if err := MarkUnresolvedLocally(&rec, "t1"); err != nil {
		t.Fatalf("MarkUnresolvedLocally: %v", err)
	}
	th := rec.ReviewThreads[0]
	if th.LocalResolved || th.LocalResolvedAt != nil || th.LastSeenCommentIDAtResolve != "" {
		t.Fatalf("expected state fully cleared, got %+v", th)
	}
}
