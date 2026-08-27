package api

import (
	"net/http"
	"strconv"
)

// ReviewRequestStatus is the two-state status for the "Review requested"
// view — distinct from the tracked-PR red/amber/green/gray status, since
// these PRs aren't tracked/stored locally at all.
type ReviewRequestStatus string

const (
	ReviewRequested ReviewRequestStatus = "review_requested"
	Reviewed        ReviewRequestStatus = "reviewed"
)

// ReviewRequestDTO is one entry in the "Review requested" view.
type ReviewRequestDTO struct {
	Owner  string              `json:"owner"`
	Repo   string              `json:"repo"`
	Number int                 `json:"number"`
	Title  string              `json:"title"`
	URL    string              `json:"url"`
	Status ReviewRequestStatus `json:"status"`
}

func prKey(owner, repo string, number int) string {
	return owner + "/" + repo + "#" + strconv.Itoa(number)
}

// handleGetReviewRequests is a live, unstored view: every open PR where the
// token's own user currently has a pending review request ("Review
// Requested"), plus every open PR they've already submitted a review on
// that isn't also currently awaiting a fresh review ("Reviewed"). Nothing
// here touches local storage — it's fetched fresh from GitHub every time.
func (s *Server) handleGetReviewRequests(w http.ResponseWriter, r *http.Request) {
	requested, err := s.GitHub.SearchReviewRequestedPRs(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	reviewed, err := s.GitHub.SearchReviewedPRs(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	seen := make(map[string]bool, len(requested)+len(reviewed))
	out := make([]ReviewRequestDTO, 0, len(requested)+len(reviewed))

	for _, res := range requested {
		seen[prKey(res.Owner, res.Repo, res.Number)] = true
		out = append(out, ReviewRequestDTO{
			Owner: res.Owner, Repo: res.Repo, Number: res.Number,
			Title: res.Title, URL: res.URL, Status: ReviewRequested,
		})
	}
	for _, res := range reviewed {
		k := prKey(res.Owner, res.Repo, res.Number)
		if seen[k] {
			continue // currently re-requested takes priority over a past review
		}
		seen[k] = true
		out = append(out, ReviewRequestDTO{
			Owner: res.Owner, Repo: res.Repo, Number: res.Number,
			Title: res.Title, URL: res.URL, Status: Reviewed,
		})
	}

	writeJSON(w, http.StatusOK, out)
}
