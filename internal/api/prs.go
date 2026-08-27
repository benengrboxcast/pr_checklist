package api

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"

	"github.com/benanderson/pr-checklist/internal/ghsync"
	"github.com/benanderson/pr-checklist/internal/store"
)

var prURLPattern = regexp.MustCompile(`github\.com/([^/]+)/([^/]+)/pull/(\d+)`)

func parsePRURL(url string) (owner, repo string, number int, err error) {
	m := prURLPattern.FindStringSubmatch(url)
	if m == nil {
		return "", "", 0, fmt.Errorf("not a recognizable GitHub PR URL: %s", url)
	}
	n, err := strconv.Atoi(m[3])
	if err != nil {
		return "", "", 0, err
	}
	return m[1], m[2], n, nil
}

func pathParams(r *http.Request) (owner, repo string, number int, err error) {
	owner = r.PathValue("owner")
	repo = r.PathValue("repo")
	n, err := strconv.Atoi(r.PathValue("number"))
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid PR number")
	}
	return owner, repo, n, nil
}

func (s *Server) handleListPRs(w http.ResponseWriter, r *http.Request) {
	recs, err := s.Store.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]PRSummaryDTO, 0, len(recs))
	for _, rec := range recs {
		out = append(out, toSummaryDTO(rec))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Owner != out[j].Owner {
			return out[i].Owner < out[j].Owner
		}
		if out[i].Repo != out[j].Repo {
			return out[i].Repo < out[j].Repo
		}
		return out[i].Number < out[j].Number
	})
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetPR(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusOK, toDetailDTO(rec))
}

type addPRRequest struct {
	URL string `json:"url"`
}

func (s *Server) handleAddPR(w http.ResponseWriter, r *http.Request) {
	var req addPRRequest
	if err := parseJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	owner, repo, number, err := parsePRURL(req.URL)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if s.Store.Exists(owner, repo, number) {
		writeError(w, http.StatusConflict, "PR is already tracked")
		return
	}

	detail, err := s.GitHub.FetchPR(r.Context(), owner, repo, number)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	rec := ghsync.NewRecord(owner, repo, number, detail)
	if err := s.Store.Save(rec); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.deriveTagTitlesFor(rec)
	writeJSON(w, http.StatusCreated, toDetailDTO(rec))
}

func (s *Server) handleDeletePR(w http.ResponseWriter, r *http.Request) {
	owner, repo, number, err := pathParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.Store.Delete(owner, repo, number); errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not tracked")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleBulkRemoveClosed(w http.ResponseWriter, r *http.Request) {
	recs, err := s.Store.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	var removed []PRSummaryDTO
	for _, rec := range recs {
		if rec.State != "CLOSED" && rec.State != "MERGED" {
			continue
		}
		if err := s.Store.Delete(rec.Owner, rec.Repo, rec.Number); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		removed = append(removed, toSummaryDTO(rec))
	}
	writeJSON(w, http.StatusOK, map[string]any{"removed": removed})
}

func (s *Server) refreshOne(r *http.Request, owner, repo string, number int) (store.PRRecord, error) {
	existing, err := s.Store.Load(owner, repo, number)
	if err != nil {
		return store.PRRecord{}, err
	}
	detail, err := s.GitHub.FetchPR(r.Context(), owner, repo, number)
	if err != nil {
		return store.PRRecord{}, err
	}
	rec := ghsync.MergeRefresh(existing, detail)
	if err := s.Store.Save(rec); err != nil {
		return store.PRRecord{}, err
	}
	s.deriveTagTitlesFor(rec)
	return rec, nil
}

func (s *Server) handleRefreshPR(w http.ResponseWriter, r *http.Request) {
	owner, repo, number, err := pathParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	rec, err := s.refreshOne(r, owner, repo, number)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not tracked")
		return
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toDetailDTO(rec))
}

func (s *Server) handleRefreshAll(w http.ResponseWriter, r *http.Request) {
	recs, err := s.Store.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]PRSummaryDTO, 0, len(recs))
	for _, rec := range recs {
		refreshed, err := s.refreshOne(r, rec.Owner, rec.Repo, rec.Number)
		if err != nil {
			// keep going with the rest; report using the stale record on failure
			out = append(out, toSummaryDTO(rec))
			continue
		}
		out = append(out, toSummaryDTO(refreshed))
	}
	writeJSON(w, http.StatusOK, out)
}
