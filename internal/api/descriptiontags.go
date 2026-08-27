package api

import (
	"errors"
	"net/http"

	"github.com/benanderson/pr-checklist/internal/ghsync"
	"github.com/benanderson/pr-checklist/internal/store"
)

// handleTagFromDescription applies Jira-issue tags found in one PR's already
// stored description to that PR, without any GitHub call. Useful for PRs
// tracked before this feature existed, or whose description changed since
// the last refresh (refresh first to pick up the latest description).
func (s *Server) handleTagFromDescription(w http.ResponseWriter, r *http.Request) {
	owner, repo, number, err := pathParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	rec, err := s.Store.Load(owner, repo, number)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not tracked")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rec, added := ghsync.ApplyDescriptionTags(rec)
	if added > 0 {
		if err := s.Store.Save(rec); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"pr": toDetailDTO(rec), "added": added})
}

// handleAutoTagFromDescriptions retroactively applies the same
// description-based tagging across every currently tracked PR's stored
// description — for PRs added before this feature existed. Never touches
// GitHub; PRs whose description hasn't been fetched yet (added before the
// description field existed) simply won't have anything to find until
// they're refreshed.
func (s *Server) handleAutoTagFromDescriptions(w http.ResponseWriter, r *http.Request) {
	recs, err := s.Store.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	totalAdded := 0
	var errs []string
	for _, rec := range recs {
		updated, added := ghsync.ApplyDescriptionTags(rec)
		if added == 0 {
			continue
		}
		if err := s.Store.Save(updated); err != nil {
			errs = append(errs, err.Error())
			continue
		}
		totalAdded += added
	}
	writeJSON(w, http.StatusOK, map[string]any{"added": totalAdded, "errors": errs})
}
