package api

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/benanderson/pr-checklist/internal/domain"
	"github.com/benanderson/pr-checklist/internal/githubapi"
)

// ReviewRequestStatus is the status shown in the "Review requested" view —
// distinct from the tracked-PR red/amber/green/gray status, since these PRs
// aren't tracked/stored locally at all (aside from the ignored-PR list).
type ReviewRequestStatus string

const (
	ReviewRequested ReviewRequestStatus = "review_requested"
	Reviewed        ReviewRequestStatus = "reviewed"
	Ignored         ReviewRequestStatus = "ignored"
)

// ReviewRequestDTO is one entry in the "Review requested" view.
type ReviewRequestDTO struct {
	Owner     string              `json:"owner"`
	Repo      string              `json:"repo"`
	Number    int                 `json:"number"`
	Title     string              `json:"title"`
	URL       string              `json:"url"`
	Status    ReviewRequestStatus `json:"status"`
	CreatedAt string              `json:"-"` // RFC3339, used only for server-side sorting, not sent to the frontend
}

// reviewRequestSortRank orders the view: currently-requested first, then
// already-reviewed, then anything the user has ignored.
var reviewRequestSortRank = map[ReviewRequestStatus]int{
	ReviewRequested: 0,
	Reviewed:        1,
	Ignored:         2,
}

func prKey(owner, repo string, number int) string {
	return owner + "/" + repo + "#" + strconv.Itoa(number)
}

// handleGetReviewRequests is a live view: every open PR where the token's
// own user currently has a pending review request ("Review Requested"),
// plus every open PR they've already submitted a review on that isn't also
// currently awaiting a fresh review ("Reviewed"). The only local state that
// affects this is the ignored-PR list — anything on it always shows as
// "Ignored" regardless of its live GitHub state, until un-ignored. Nothing
// else here touches local storage; the PR list itself is fetched fresh from
// GitHub every time.
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
	cfg, err := s.Store.LoadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	out := mergeReviewRequests(requested, reviewed, cfg.IgnoredReviewPRs)
	writeJSON(w, http.StatusOK, out)
}

// mergeReviewRequests combines the requested/reviewed search results into
// one deduplicated list (a currently-pending request takes priority over a
// past review of the same PR), applies the ignored-PR override, and sorts
// the result: first by state (Review Requested -> Reviewed -> Ignored), then
// by age within each group, oldest first. Pure and side-effect-free so it's
// unit-testable without a live GitHub call.
func mergeReviewRequests(requested, reviewed []githubapi.SearchResult, ignoredKeys []string) []ReviewRequestDTO {
	ignored := make(map[string]bool, len(ignoredKeys))
	for _, k := range ignoredKeys {
		ignored[k] = true
	}

	byKey := make(map[string]ReviewRequestDTO, len(requested)+len(reviewed))

	for _, res := range requested {
		byKey[prKey(res.Owner, res.Repo, res.Number)] = ReviewRequestDTO{
			Owner: res.Owner, Repo: res.Repo, Number: res.Number,
			Title: res.Title, URL: res.URL, CreatedAt: res.CreatedAt, Status: ReviewRequested,
		}
	}
	for _, res := range reviewed {
		k := prKey(res.Owner, res.Repo, res.Number)
		if _, exists := byKey[k]; exists {
			continue // currently re-requested takes priority over a past review
		}
		byKey[k] = ReviewRequestDTO{
			Owner: res.Owner, Repo: res.Repo, Number: res.Number,
			Title: res.Title, URL: res.URL, CreatedAt: res.CreatedAt, Status: Reviewed,
		}
	}
	for k, dto := range byKey {
		if ignored[k] {
			dto.Status = Ignored
			byKey[k] = dto
		}
	}

	out := make([]ReviewRequestDTO, 0, len(byKey))
	for _, dto := range byKey {
		out = append(out, dto)
	}
	sort.Slice(out, func(i, j int) bool {
		ri, rj := reviewRequestSortRank[out[i].Status], reviewRequestSortRank[out[j].Status]
		if ri != rj {
			return ri < rj
		}
		return out[i].CreatedAt < out[j].CreatedAt
	})
	return out
}

// handleIgnoreReviewRequest marks a PR as ignored in the Review Requested
// view — purely a local UI marker, never touches GitHub (it doesn't remove
// you as a reviewer or submit anything). Once ignored, a PR always shows as
// "Ignored" regardless of its live review-requested/reviewed state, until
// un-ignored.
func (s *Server) handleIgnoreReviewRequest(w http.ResponseWriter, r *http.Request) {
	owner, repo, number, err := pathParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	cfg, err := s.Store.LoadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	cfg.IgnoredReviewPRs = domain.EnsureTag(cfg.IgnoredReviewPRs, prKey(owner, repo, number))
	if err := s.Store.SaveConfig(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": string(Ignored)})
}

// handleUnignoreReviewRequest removes a PR from the ignored list, so it goes
// back to reflecting its live review-requested/reviewed state.
func (s *Server) handleUnignoreReviewRequest(w http.ResponseWriter, r *http.Request) {
	owner, repo, number, err := pathParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	cfg, err := s.Store.LoadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	cfg.IgnoredReviewPRs = domain.RemoveTag(cfg.IgnoredReviewPRs, prKey(owner, repo, number))
	if err := s.Store.SaveConfig(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "unignored"})
}
