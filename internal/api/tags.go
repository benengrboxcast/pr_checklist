package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/benanderson/pr-checklist/internal/domain"
	"github.com/benanderson/pr-checklist/internal/store"
)

type addTagRequest struct {
	Tag string `json:"tag"`
}

func (s *Server) handleAddTag(w http.ResponseWriter, r *http.Request) {
	owner, repo, number, err := pathParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var req addTagRequest
	if err := parseJSONBody(r, &req); err != nil || req.Tag == "" {
		writeError(w, http.StatusBadRequest, "tag is required")
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

	rec.Tags = domain.EnsureTag(rec.Tags, req.Tag)
	if err := s.Store.Save(rec); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toDetailDTO(rec))
}

func (s *Server) handleRemoveTag(w http.ResponseWriter, r *http.Request) {
	owner, repo, number, err := pathParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	tag := r.PathValue("tag")

	rec, err := s.Store.Load(owner, repo, number)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not tracked")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rec.Tags = domain.RemoveTag(rec.Tags, tag)
	if err := s.Store.Save(rec); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toDetailDTO(rec))
}

type bulkTagRequest struct {
	Tag string `json:"tag"`
	PRs []struct {
		Owner  string `json:"owner"`
		Repo   string `json:"repo"`
		Number int    `json:"number"`
	} `json:"prs"`
}

// handleBulkTag applies one tag to a batch of already-tracked PRs at once,
// for the dashboard's bulk-edit mode. Additive/idempotent per PR (same as a
// single add-tag call) — a PR that already has the tag just isn't counted as
// updated. Failures (e.g. a PR deleted mid-batch) are collected and reported
// rather than aborting the whole batch.
func (s *Server) handleBulkTag(w http.ResponseWriter, r *http.Request) {
	var req bulkTagRequest
	if err := parseJSONBody(r, &req); err != nil || req.Tag == "" || len(req.PRs) == 0 {
		writeError(w, http.StatusBadRequest, "tag and at least one pr are required")
		return
	}

	updated := 0
	var errs []string
	for _, ref := range req.PRs {
		rec, err := s.Store.Load(ref.Owner, ref.Repo, ref.Number)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s/%s#%d: %v", ref.Owner, ref.Repo, ref.Number, err))
			continue
		}
		before := len(rec.Tags)
		rec.Tags = domain.EnsureTag(rec.Tags, req.Tag)
		if len(rec.Tags) == before {
			continue // already had the tag
		}
		if err := s.Store.Save(rec); err != nil {
			errs = append(errs, fmt.Sprintf("%s/%s#%d: %v", ref.Owner, ref.Repo, ref.Number, err))
			continue
		}
		updated++
	}

	writeJSON(w, http.StatusOK, map[string]any{"updated": updated, "errors": errs})
}
