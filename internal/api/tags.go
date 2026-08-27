package api

import (
	"errors"
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
