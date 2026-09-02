package api

import (
	"time"

	"github.com/benanderson/pr-checklist/internal/domain"
	"github.com/benanderson/pr-checklist/internal/store"
)

// CommentDTO is a comment as sent to the frontend.
type CommentDTO struct {
	ID        string    `json:"id"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

// ReviewThreadDTO is a review thread with its derived state, as sent to the frontend.
type ReviewThreadDTO struct {
	ID       string             `json:"id"`
	State    domain.ThreadState `json:"state"`
	Comments []CommentDTO       `json:"comments"`
}

// PRSummaryDTO is the dashboard-list shape for one tracked PR.
type PRSummaryDTO struct {
	Owner          string              `json:"owner"`
	Repo           string              `json:"repo"`
	Number         int                 `json:"number"`
	Title          string              `json:"title"`
	URL            string              `json:"url"`
	State          string              `json:"state"`
	ReviewDecision string              `json:"reviewDecision"`
	Tags           []string            `json:"tags"`
	Labels         []string            `json:"labels"` // GitHub's own PR labels, read-only
	Status         domain.Status       `json:"status"`
	ThreadCounts   domain.ThreadCounts `json:"threadCounts"`
	LastSyncedAt   time.Time           `json:"lastSyncedAt"`
}

// PRDetailDTO is the full single-PR-view shape.
type PRDetailDTO struct {
	PRSummaryDTO
	HeadRefName   string            `json:"headRefName"`
	AuthorLogin   string            `json:"authorLogin"`
	ReviewThreads []ReviewThreadDTO `json:"reviewThreads"`
	Comments      []CommentDTO      `json:"comments"`
}

func commentDTOs(cs []store.Comment) []CommentDTO {
	out := make([]CommentDTO, 0, len(cs))
	for _, c := range cs {
		out = append(out, CommentDTO{ID: c.ID, Author: c.Author, Body: c.Body, CreatedAt: c.CreatedAt})
	}
	return out
}

func toSummaryDTO(rec store.PRRecord) PRSummaryDTO {
	return PRSummaryDTO{
		Owner:          rec.Owner,
		Repo:           rec.Repo,
		Number:         rec.Number,
		Title:          rec.Title,
		URL:            rec.URL,
		State:          rec.State,
		ReviewDecision: rec.ReviewDecision,
		Tags:           rec.Tags,
		Labels:         rec.Labels,
		Status:         domain.ComputePRStatus(rec),
		ThreadCounts:   domain.CountThreads(rec),
		LastSyncedAt:   rec.LastSyncedAt,
	}
}

func toDetailDTO(rec store.PRRecord) PRDetailDTO {
	threads := make([]ReviewThreadDTO, 0, len(rec.ReviewThreads))
	for _, t := range rec.ReviewThreads {
		threads = append(threads, ReviewThreadDTO{
			ID:       t.ID,
			State:    domain.ComputeThreadState(t),
			Comments: commentDTOs(t.Comments),
		})
	}
	return PRDetailDTO{
		PRSummaryDTO:  toSummaryDTO(rec),
		HeadRefName:   rec.HeadRefName,
		AuthorLogin:   rec.AuthorLogin,
		ReviewThreads: threads,
		Comments:      commentDTOs(rec.Comments),
	}
}
