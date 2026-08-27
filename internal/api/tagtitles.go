package api

import (
	"net/http"

	"github.com/benanderson/pr-checklist/internal/ghsync"
	"github.com/benanderson/pr-checklist/internal/store"
)

// deriveTagTitlesFor best-effort derives and persists any missing EMBED tag
// titles implied by recs' branch names. This runs automatically after every
// add/refresh/import so titles show up without the user needing to click
// "Auto-generate". Failures are silently ignored — title display just falls
// back to showing the raw tag if this doesn't succeed.
func (s *Server) deriveTagTitlesFor(recs ...store.PRRecord) {
	if len(recs) == 0 {
		return
	}
	cfg, err := s.Store.LoadConfig()
	if err != nil {
		return
	}
	updated, added := ghsync.FillMissingTagTitles(cfg, recs)
	if added > 0 {
		_ = s.Store.SaveConfig(updated)
	}
}

// handleAutoGenerateTagTitles retroactively applies the same branch-name
// title derivation across every currently tracked PR, for PRs that were
// added before this feature existed (or whose title wasn't derivable at add
// time). Never overwrites a title that's already set.
func (s *Server) handleAutoGenerateTagTitles(w http.ResponseWriter, r *http.Request) {
	recs, err := s.Store.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	cfg, err := s.Store.LoadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	updated, added := ghsync.FillMissingTagTitles(cfg, recs)
	if added > 0 {
		if err := s.Store.SaveConfig(updated); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"tagTitles": updated.TagTitles, "added": added})
}
