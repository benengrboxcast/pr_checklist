package githubapi

import "time"

type pageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

type actor struct {
	Login string `json:"login"`
}

type rawComment struct {
	ID        string    `json:"id"`
	Author    actor     `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

type commentConnection struct {
	PageInfo pageInfo     `json:"pageInfo"`
	Nodes    []rawComment `json:"nodes"`
}

type rawReviewThread struct {
	ID         string            `json:"id"`
	IsResolved bool              `json:"isResolved"`
	Comments   commentConnection `json:"comments"`
}

type reviewThreadConnection struct {
	PageInfo pageInfo          `json:"pageInfo"`
	Nodes    []rawReviewThread `json:"nodes"`
}

type rawLabel struct {
	Name string `json:"name"`
}

type labelConnection struct {
	Nodes []rawLabel `json:"nodes"`
}

type rawPullRequest struct {
	ID             string                 `json:"id"`
	Title          string                 `json:"title"`
	URL            string                 `json:"url"`
	State          string                 `json:"state"`
	ReviewDecision string                 `json:"reviewDecision"`
	HeadRefName    string                 `json:"headRefName"`
	Body           string                 `json:"body"`
	Author         actor                  `json:"author"`
	Labels         labelConnection        `json:"labels"`
	ReviewThreads  reviewThreadConnection `json:"reviewThreads"`
	Comments       commentConnection      `json:"comments"`
}

type prDetailResponse struct {
	Viewer     actor `json:"viewer"`
	Repository struct {
		PullRequest rawPullRequest `json:"pullRequest"`
	} `json:"repository"`
}

type threadCommentsResponse struct {
	Node struct {
		Comments commentConnection `json:"comments"`
	} `json:"node"`
}

type searchNode struct {
	Number     int       `json:"number"`
	Title      string    `json:"title"`
	URL        string    `json:"url"`
	State      string    `json:"state"`
	CreatedAt  time.Time `json:"createdAt"`
	Repository struct {
		Owner actor  `json:"owner"`
		Name  string `json:"name"`
	} `json:"repository"`
}

type searchResponse struct {
	Search struct {
		PageInfo pageInfo     `json:"pageInfo"`
		Nodes    []searchNode `json:"nodes"`
	} `json:"search"`
}

type viewerResponse struct {
	Viewer actor `json:"viewer"`
}

type orgsResponse struct {
	Viewer struct {
		Organizations struct {
			PageInfo pageInfo `json:"pageInfo"`
			Nodes    []actor  `json:"nodes"`
		} `json:"organizations"`
	} `json:"viewer"`
}
