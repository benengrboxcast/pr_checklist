package api

import (
	"net/http"

	"github.com/benanderson/pr-checklist/internal/store"
)

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.Store.LoadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	var cfg store.Config
	if err := parseJSONBody(r, &cfg); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if cfg.ImportRepos == nil {
		cfg.ImportRepos = []string{}
	}
	if cfg.ImportOrgs == nil {
		cfg.ImportOrgs = []string{}
	}
	if cfg.TagTitles == nil {
		cfg.TagTitles = map[string]string{}
	}
	if cfg.IgnoredReviewPRs == nil {
		cfg.IgnoredReviewPRs = []string{}
	}
	if cfg.RepoGroups == nil {
		cfg.RepoGroups = []store.RepoGroup{}
	}
	for i, g := range cfg.RepoGroups {
		if g.Repos == nil {
			cfg.RepoGroups[i].Repos = []string{}
		}
	}
	if err := s.Store.SaveConfig(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}
