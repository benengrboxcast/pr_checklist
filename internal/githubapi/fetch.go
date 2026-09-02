package githubapi

import (
	"context"
	"fmt"
	"time"
)

// Comment is a normalized comment (review-thread or top-level).
type Comment struct {
	ID        string
	Author    string
	Body      string
	CreatedAt string // RFC3339, kept as string to avoid an extra time import at call sites
}

// ReviewThread is a normalized inline review-comment thread.
type ReviewThread struct {
	ID         string
	IsResolved bool
	Comments   []Comment
}

// PRDetail is the normalized result of fetching one PR from GitHub.
type PRDetail struct {
	NodeID             string
	Title              string
	URL                string
	State              string
	ReviewDecision     string
	HeadRefName        string
	Body               string
	AuthorLogin        string
	AuthenticatedLogin string
	Labels             []string // GitHub's own PR labels, read-only from this app's perspective
	ReviewThreads      []ReviewThread
	Comments           []Comment
}

// PRRef identifies a pull request by owner/repo/number.
type PRRef struct {
	Owner  string
	Repo   string
	Number int
}

// SearchResult is one hit from a GitHub PR search, with enough detail to
// display it directly without a follow-up FetchPR call.
type SearchResult struct {
	Owner     string
	Repo      string
	Number    int
	Title     string
	URL       string
	State     string
	CreatedAt string // RFC3339
}

func (r SearchResult) ref() PRRef {
	return PRRef{Owner: r.Owner, Repo: r.Repo, Number: r.Number}
}

func toComments(nodes []rawComment) []Comment {
	out := make([]Comment, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, Comment{
			ID:        n.ID,
			Author:    n.Author.Login,
			Body:      n.Body,
			CreatedAt: n.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return out
}

// FetchPR fetches a single PR's full detail, following pagination cursors for
// review threads, each thread's comments, and top-level comments.
func (c *Client) FetchPR(ctx context.Context, owner, repo string, number int) (PRDetail, error) {
	var (
		threadsCursor  string
		commentsCursor string
		detail         PRDetail
		threads        []ReviewThread
		topComments    []Comment
	)

	for {
		var resp prDetailResponse
		vars := map[string]any{
			"owner":  owner,
			"repo":   repo,
			"number": number,
		}
		if threadsCursor != "" {
			vars["threadsCursor"] = threadsCursor
		}
		if commentsCursor != "" {
			vars["commentsCursor"] = commentsCursor
		}
		if err := c.Do(ctx, prDetailQuery, vars, &resp); err != nil {
			return PRDetail{}, fmt.Errorf("fetching PR %s/%s#%d: %w", owner, repo, number, err)
		}

		pr := resp.Repository.PullRequest
		detail.NodeID = pr.ID
		detail.Title = pr.Title
		detail.URL = pr.URL
		detail.State = pr.State
		detail.ReviewDecision = pr.ReviewDecision
		detail.HeadRefName = pr.HeadRefName
		detail.Body = pr.Body
		detail.AuthorLogin = pr.Author.Login
		detail.AuthenticatedLogin = resp.Viewer.Login
		detail.Labels = make([]string, 0, len(pr.Labels.Nodes))
		for _, l := range pr.Labels.Nodes {
			detail.Labels = append(detail.Labels, l.Name)
		}

		for _, rt := range pr.ReviewThreads.Nodes {
			thread := ReviewThread{
				ID:         rt.ID,
				IsResolved: rt.IsResolved,
				Comments:   toComments(rt.Comments.Nodes),
			}
			if rt.Comments.PageInfo.HasNextPage {
				more, err := c.fetchRemainingThreadComments(ctx, rt.ID, rt.Comments.PageInfo.EndCursor)
				if err != nil {
					return PRDetail{}, err
				}
				thread.Comments = append(thread.Comments, more...)
			}
			threads = append(threads, thread)
		}
		topComments = append(topComments, toComments(pr.Comments.Nodes)...)

		threadsDone := !pr.ReviewThreads.PageInfo.HasNextPage
		commentsDone := !pr.Comments.PageInfo.HasNextPage
		if threadsDone && commentsDone {
			break
		}
		if !threadsDone {
			threadsCursor = pr.ReviewThreads.PageInfo.EndCursor
		}
		if !commentsDone {
			commentsCursor = pr.Comments.PageInfo.EndCursor
		}
	}

	detail.ReviewThreads = threads
	detail.Comments = topComments
	return detail, nil
}

func (c *Client) fetchRemainingThreadComments(ctx context.Context, threadID, cursor string) ([]Comment, error) {
	var out []Comment
	for {
		var resp threadCommentsResponse
		vars := map[string]any{"threadId": threadID, "cursor": cursor}
		if err := c.Do(ctx, threadCommentsQuery, vars, &resp); err != nil {
			return nil, fmt.Errorf("fetching thread %s comments: %w", threadID, err)
		}
		out = append(out, toComments(resp.Node.Comments.Nodes)...)
		if !resp.Node.Comments.PageInfo.HasNextPage {
			return out, nil
		}
		cursor = resp.Node.Comments.PageInfo.EndCursor
	}
}

// FetchViewer returns the GitHub login the configured token authenticates as.
func (c *Client) FetchViewer(ctx context.Context) (string, error) {
	var resp viewerResponse
	if err := c.Do(ctx, viewerQuery, nil, &resp); err != nil {
		return "", fmt.Errorf("fetching viewer: %w", err)
	}
	return resp.Viewer.Login, nil
}

// SearchOpenAuthoredPRs finds open PRs authored by the token's own user in a
// single repo, returning just enough to identify each PR for a follow-up FetchPR.
func (c *Client) SearchOpenAuthoredPRs(ctx context.Context, owner, repo string) ([]PRRef, error) {
	q := fmt.Sprintf("is:pr is:open author:@me repo:%s/%s", owner, repo)
	results, err := c.runSearch(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("searching %s/%s: %w", owner, repo, err)
	}
	return toRefs(results), nil
}

// SearchOpenAuthoredPRsInOrg finds open PRs authored by the token's own user
// across every repo in an org, returning just enough to identify each PR for
// a follow-up FetchPR.
func (c *Client) SearchOpenAuthoredPRsInOrg(ctx context.Context, org string) ([]PRRef, error) {
	q := fmt.Sprintf("is:pr is:open author:@me org:%s", org)
	results, err := c.runSearch(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("searching org %s: %w", org, err)
	}
	return toRefs(results), nil
}

// SearchReviewRequestedPRs finds open PRs, across every repo the token can
// see, where the token's own user currently has a pending review request.
func (c *Client) SearchReviewRequestedPRs(ctx context.Context) ([]SearchResult, error) {
	results, err := c.runSearch(ctx, "is:pr is:open review-requested:@me")
	if err != nil {
		return nil, fmt.Errorf("searching review-requested PRs: %w", err)
	}
	return results, nil
}

// SearchReviewedPRs finds open PRs, across every repo the token can see,
// where the token's own user has already submitted at least one review.
func (c *Client) SearchReviewedPRs(ctx context.Context) ([]SearchResult, error) {
	results, err := c.runSearch(ctx, "is:pr is:open reviewed-by:@me")
	if err != nil {
		return nil, fmt.Errorf("searching reviewed PRs: %w", err)
	}
	return results, nil
}

func toRefs(results []SearchResult) []PRRef {
	refs := make([]PRRef, 0, len(results))
	for _, r := range results {
		refs = append(refs, r.ref())
	}
	return refs
}

func (c *Client) runSearch(ctx context.Context, q string) ([]SearchResult, error) {
	var results []SearchResult
	var cursor string
	for {
		var resp searchResponse
		vars := map[string]any{"query": q}
		if cursor != "" {
			vars["cursor"] = cursor
		}
		if err := c.Do(ctx, searchQuery, vars, &resp); err != nil {
			return nil, err
		}
		for _, n := range resp.Search.Nodes {
			results = append(results, SearchResult{
				Owner:     n.Repository.Owner.Login,
				Repo:      n.Repository.Name,
				Number:    n.Number,
				Title:     n.Title,
				URL:       n.URL,
				State:     n.State,
				CreatedAt: n.CreatedAt.Format(time.RFC3339),
			})
		}
		if !resp.Search.PageInfo.HasNextPage {
			return results, nil
		}
		cursor = resp.Search.PageInfo.EndCursor
	}
}

// FetchViewerOrgs returns the login of every organization the token's own
// user belongs to. Requires the token to have the read:org scope for private
// organization memberships to be visible.
func (c *Client) FetchViewerOrgs(ctx context.Context) ([]string, error) {
	var orgs []string
	var cursor string
	for {
		var resp orgsResponse
		vars := map[string]any{}
		if cursor != "" {
			vars["cursor"] = cursor
		}
		if err := c.Do(ctx, viewerOrgsQuery, vars, &resp); err != nil {
			return nil, fmt.Errorf("fetching viewer organizations: %w", err)
		}
		for _, n := range resp.Viewer.Organizations.Nodes {
			orgs = append(orgs, n.Login)
		}
		if !resp.Viewer.Organizations.PageInfo.HasNextPage {
			return orgs, nil
		}
		cursor = resp.Viewer.Organizations.PageInfo.EndCursor
	}
}
