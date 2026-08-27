package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/benanderson/pr-checklist/internal/ghsync"
)

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.Store.LoadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var added []PRSummaryDTO
	var errs []string

	for _, ownerRepo := range cfg.ImportRepos {
		owner, repo, ok := strings.Cut(ownerRepo, "/")
		if !ok {
			errs = append(errs, fmt.Sprintf("invalid importRepos entry %q, expected owner/repo", ownerRepo))
			continue
		}

		refs, err := s.GitHub.SearchOpenAuthoredPRs(r.Context(), owner, repo)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}

		for _, ref := range refs {
			if s.Store.Exists(ref.Owner, ref.Repo, ref.Number) {
				continue
			}
			detail, err := s.GitHub.FetchPR(r.Context(), ref.Owner, ref.Repo, ref.Number)
			if err != nil {
				errs = append(errs, err.Error())
				continue
			}
			rec := ghsync.NewRecord(ref.Owner, ref.Repo, ref.Number, detail)
			if err := s.Store.Save(rec); err != nil {
				errs = append(errs, err.Error())
				continue
			}
			added = append(added, toSummaryDTO(rec))
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"added": added, "errors": errs})
}
