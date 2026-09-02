// Package store persists and loads PR tracking data as JSON files on disk.
package store

import "time"

// PRRecord is the full persisted state for one tracked pull request.
type PRRecord struct {
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	Number int    `json:"number"`

	NodeID         string `json:"nodeId"`
	Title          string `json:"title"`
	URL            string `json:"url"`
	State          string `json:"state"`          // OPEN | CLOSED | MERGED
	ReviewDecision string `json:"reviewDecision"` // APPROVED | CHANGES_REQUESTED | REVIEW_REQUIRED | ""
	HeadRefName    string `json:"headRefName"`
	AuthorLogin    string `json:"authorLogin"`
	Body           string `json:"body"` // raw PR description, used to find Jira issue keys in it

	Labels []string `json:"labels"` // GitHub's own PR labels — read-only mirror of GitHub, distinct from Tags below

	Tags               []string  `json:"tags"`
	AuthenticatedLogin string    `json:"authenticatedLogin"`
	LastSyncedAt       time.Time `json:"lastSyncedAt"`

	ReviewThreads []ReviewThread `json:"reviewThreads"`
	Comments      []Comment      `json:"comments"`
}

// ReviewThread is one inline review-comment thread on a PR.
type ReviewThread struct {
	ID                         string     `json:"id"`
	IsResolvedOnGitHub         bool       `json:"isResolvedOnGitHub"`
	LocalResolved              bool       `json:"localResolved"`
	LocalResolvedAt            *time.Time `json:"localResolvedAt,omitempty"`
	LastSeenCommentIDAtResolve string     `json:"lastSeenCommentIdAtResolve,omitempty"`
	Comments                   []Comment  `json:"comments"`
}

// Comment is a single comment, either in a review thread or a top-level PR comment.
type Comment struct {
	ID        string    `json:"id"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

// Config is the user-editable import allow-list, stored separately from PR records.
type Config struct {
	ImportRepos      []string          `json:"importRepos"`
	ImportOrgs       []string          `json:"importOrgs"`       // org logins to import every authored-open PR from
	TagTitles        map[string]string `json:"tagTitles"`        // tag -> custom display title, shown instead of the raw tag
	IgnoredReviewPRs []string          `json:"ignoredReviewPRs"` // "owner/repo#number" keys marked ignored in the Review Requested view
}
