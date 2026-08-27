package api

import (
	"errors"
	"net/http"

	"github.com/benanderson/pr-checklist/internal/ghsync"
	"github.com/benanderson/pr-checklist/internal/store"
)

func (s *Server) handleResolveThread(w http.ResponseWriter, r *http.Request) {
	s.updateThreadLocalState(w, r, ghsync.MarkResolvedLocally)
}

func (s *Server) handleUnresolveThread(w http.ResponseWriter, r *http.Request) {
	s.updateThreadLocalState(w, r, ghsync.MarkUnresolvedLocally)
}

func (s *Server) updateThreadLocalState(w http.ResponseWriter, r *http.Request, mutate func(*store.PRRecord, string) error) {
	owner, repo, number, err := pathParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	threadID := r.PathValue("threadId")

	rec, err := s.Store.Load(owner, repo, number)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not tracked")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := mutate(&rec, threadID); err != nil {
		switch {
		case errors.Is(err, ghsync.ErrThreadNotFound):
			writeError(w, http.StatusNotFound, "thread not found")
		case errors.Is(err, ghsync.ErrAlreadyResolvedOnGitHub):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	if err := s.Store.Save(rec); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toDetailDTO(rec))
}
