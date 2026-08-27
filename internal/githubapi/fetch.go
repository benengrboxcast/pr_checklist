package githubapi

import (
	"context"
	"fmt"
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
	AuthorLogin        string
	AuthenticatedLogin string
	ReviewThreads      []ReviewThread
	Comments           []Comment
}

// PRRef identifies a pull request by owner/repo/number.
type PRRef struct {
	Owner  string
	Repo   string
	Number int
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
		detail.AuthorLogin = pr.Author.Login
		detail.AuthenticatedLogin = resp.Viewer.Login

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

// SearchOpenAuthoredPRs finds open PRs authored by the token's own user in a
// single repo, returning just enough to identify each PR for a follow-up FetchPR.
func (c *Client) SearchOpenAuthoredPRs(ctx context.Context, owner, repo string) ([]PRRef, error) {
	q := fmt.Sprintf("is:pr is:open author:@me repo:%s/%s", owner, repo)
	var refs []PRRef
	var cursor string
	for {
		var resp searchResponse
		vars := map[string]any{"query": q}
		if cursor != "" {
			vars["cursor"] = cursor
		}
		if err := c.Do(ctx, searchOpenAuthoredQuery, vars, &resp); err != nil {
			return nil, fmt.Errorf("searching %s/%s: %w", owner, repo, err)
		}
		for _, n := range resp.Search.Nodes {
			refs = append(refs, PRRef{
				Owner:  n.Repository.Owner.Login,
				Repo:   n.Repository.Name,
				Number: n.Number,
			})
		}
		if !resp.Search.PageInfo.HasNextPage {
			return refs, nil
		}
		cursor = resp.Search.PageInfo.EndCursor
	}
}
