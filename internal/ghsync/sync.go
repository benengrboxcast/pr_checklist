// Package ghsync is the single place that turns a fresh GitHub fetch into
// persisted store state — used identically by add-by-URL, refresh, and import,
// so auto-tagging and the revert-on-reply rule only exist in one place.
package ghsync

import (
	"errors"
	"time"

	"github.com/benanderson/pr-checklist/internal/domain"
	"github.com/benanderson/pr-checklist/internal/githubapi"
	"github.com/benanderson/pr-checklist/internal/store"
)

// ErrThreadNotFound is returned when a requested thread ID doesn't exist on the PR.
var ErrThreadNotFound = errors.New("thread not found")

// ErrAlreadyResolvedOnGitHub is returned when trying to locally resolve a
// thread GitHub itself already reports as resolved — a meaningless no-op.
var ErrAlreadyResolvedOnGitHub = errors.New("thread is already resolved on github")

// MarkResolvedLocally sets a thread's local-resolved flag and stamps the
// revert-on-reply watermark to its current last comment. This never calls
// GitHub — it only ever mutates the local record.
func MarkResolvedLocally(rec *store.PRRecord, threadID string) error {
	for i := range rec.ReviewThreads {
		t := &rec.ReviewThreads[i]
		if t.ID != threadID {
			continue
		}
		if t.IsResolvedOnGitHub {
			return ErrAlreadyResolvedOnGitHub
		}
		now := time.Now()
		t.LocalResolved = true
		t.LocalResolvedAt = &now
		if n := len(t.Comments); n > 0 {
			t.LastSeenCommentIDAtResolve = t.Comments[n-1].ID
		}
		return nil
	}
	return ErrThreadNotFound
}

// MarkUnresolvedLocally clears a thread's local-resolved flag.
func MarkUnresolvedLocally(rec *store.PRRecord, threadID string) error {
	for i := range rec.ReviewThreads {
		t := &rec.ReviewThreads[i]
		if t.ID != threadID {
			continue
		}
		t.LocalResolved = false
		t.LocalResolvedAt = nil
		t.LastSeenCommentIDAtResolve = ""
		return nil
	}
	return ErrThreadNotFound
}

func toStoreComments(cs []githubapi.Comment) []store.Comment {
	out := make([]store.Comment, 0, len(cs))
	for _, c := range cs {
		createdAt, err := time.Parse(time.RFC3339, c.CreatedAt)
		if err != nil {
			createdAt = time.Time{}
		}
		out = append(out, store.Comment{
			ID:        c.ID,
			Author:    c.Author,
			Body:      c.Body,
			CreatedAt: createdAt,
		})
	}
	return out
}

func applyAutoTag(tags []string, headRefName string) []string {
	if embedTag, ok := domain.ExtractEmbedTag(headRefName); ok {
		return domain.EnsureTag(tags, embedTag)
	}
	return tags
}

// NewRecord builds a brand-new PRRecord from a fresh GitHub fetch. All review
// threads start with no local resolution state.
func NewRecord(owner, repo string, number int, detail githubapi.PRDetail) store.PRRecord {
	threads := make([]store.ReviewThread, 0, len(detail.ReviewThreads))
	for _, t := range detail.ReviewThreads {
		threads = append(threads, store.ReviewThread{
			ID:                 t.ID,
			IsResolvedOnGitHub: t.IsResolved,
			Comments:           toStoreComments(t.Comments),
		})
	}

	return store.PRRecord{
		Owner:              owner,
		Repo:               repo,
		Number:             number,
		NodeID:             detail.NodeID,
		Title:              detail.Title,
		URL:                detail.URL,
		State:              detail.State,
		ReviewDecision:     detail.ReviewDecision,
		HeadRefName:        detail.HeadRefName,
		AuthorLogin:        detail.AuthorLogin,
		Tags:               applyAutoTag(nil, detail.HeadRefName),
		AuthenticatedLogin: detail.AuthenticatedLogin,
		LastSyncedAt:       time.Now(),
		ReviewThreads:      threads,
		Comments:           toStoreComments(detail.Comments),
	}
}

// MergeRefresh merges a fresh GitHub fetch into an existing record, preserving
// tags and local resolution state, applying the revert-on-reply rule, and
// re-ensuring the branch-derived auto-tag.
func MergeRefresh(existing store.PRRecord, detail githubapi.PRDetail) store.PRRecord {
	existingByID := make(map[string]store.ReviewThread, len(existing.ReviewThreads))
	for _, t := range existing.ReviewThreads {
		existingByID[t.ID] = t
	}

	threads := make([]store.ReviewThread, 0, len(detail.ReviewThreads))
	for _, t := range detail.ReviewThreads {
		fresh := store.ReviewThread{
			ID:                 t.ID,
			IsResolvedOnGitHub: t.IsResolved,
			Comments:           toStoreComments(t.Comments),
		}
		if prev, ok := existingByID[t.ID]; ok {
			fresh.LocalResolved = prev.LocalResolved
			fresh.LocalResolvedAt = prev.LocalResolvedAt
			fresh.LastSeenCommentIDAtResolve = prev.LastSeenCommentIDAtResolve
			if fresh.LocalResolved && !fresh.IsResolvedOnGitHub {
				applyRevertOnReply(&fresh, existing.AuthenticatedLogin)
			}
		}
		threads = append(threads, fresh)
	}

	return store.PRRecord{
		Owner:              existing.Owner,
		Repo:               existing.Repo,
		Number:             existing.Number,
		NodeID:             detail.NodeID,
		Title:              detail.Title,
		URL:                detail.URL,
		State:              detail.State,
		ReviewDecision:     detail.ReviewDecision,
		HeadRefName:        detail.HeadRefName,
		AuthorLogin:        detail.AuthorLogin,
		Tags:               applyAutoTag(existing.Tags, detail.HeadRefName),
		AuthenticatedLogin: detail.AuthenticatedLogin,
		LastSyncedAt:       time.Now(),
		ReviewThreads:      threads,
		Comments:           toStoreComments(detail.Comments),
	}
}

// applyRevertOnReply clears a thread's local-resolved state in place if a
// comment authored by someone other than authenticatedLogin was added after
// the watermark comment recorded when it was marked resolved locally. If the
// watermark comment can't be located in the fresh comment list (e.g. it was
// deleted), the local-resolved state is left untouched rather than guessed at.
func applyRevertOnReply(t *store.ReviewThread, authenticatedLogin string) {
	watermarkIdx := -1
	for i, c := range t.Comments {
		if c.ID == t.LastSeenCommentIDAtResolve {
			watermarkIdx = i
			break
		}
	}
	if watermarkIdx == -1 {
		return
	}
	for _, c := range t.Comments[watermarkIdx+1:] {
		if c.Author != authenticatedLogin {
			t.LocalResolved = false
			t.LocalResolvedAt = nil
			t.LastSeenCommentIDAtResolve = ""
			return
		}
	}
}
