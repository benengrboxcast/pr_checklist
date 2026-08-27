package api

import "net/http"

func (s *Server) handleGetViewer(w http.ResponseWriter, r *http.Request) {
	login, err := s.GitHub.FetchViewer(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"login": login})
}

func (s *Server) handleGetOrgs(w http.ResponseWriter, r *http.Request) {
	orgs, err := s.GitHub.FetchViewerOrgs(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"orgs": orgs})
}
