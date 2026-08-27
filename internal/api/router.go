// Package api implements the HTTP surface consumed by the embedded frontend.
package api

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"

	"github.com/benanderson/pr-checklist/internal/githubapi"
	"github.com/benanderson/pr-checklist/internal/store"
)

// Server holds the dependencies shared by every HTTP handler.
type Server struct {
	Store  *store.Store
	GitHub *githubapi.Client
}

// NewRouter builds the full HTTP handler: /api/* endpoints plus the embedded
// static frontend at /.
func NewRouter(s *Server, staticFS fs.FS) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/prs", s.handleListPRs)
	mux.HandleFunc("POST /api/prs", s.handleAddPR)
	mux.HandleFunc("GET /api/prs/{owner}/{repo}/{number}", s.handleGetPR)
	mux.HandleFunc("DELETE /api/prs/{owner}/{repo}/{number}", s.handleDeletePR)
	mux.HandleFunc("POST /api/prs/bulk-remove-closed", s.handleBulkRemoveClosed)
	mux.HandleFunc("POST /api/prs/refresh-all", s.handleRefreshAll)
	mux.HandleFunc("POST /api/prs/{owner}/{repo}/{number}/refresh", s.handleRefreshPR)
	mux.HandleFunc("POST /api/prs/{owner}/{repo}/{number}/threads/{threadId}/resolve", s.handleResolveThread)
	mux.HandleFunc("POST /api/prs/{owner}/{repo}/{number}/threads/{threadId}/unresolve", s.handleUnresolveThread)
	mux.HandleFunc("POST /api/prs/{owner}/{repo}/{number}/tags", s.handleAddTag)
	mux.HandleFunc("DELETE /api/prs/{owner}/{repo}/{number}/tags/{tag}", s.handleRemoveTag)

	mux.HandleFunc("POST /api/import", s.handleImport)

	mux.HandleFunc("GET /api/config", s.handleGetConfig)
	mux.HandleFunc("PUT /api/config", s.handlePutConfig)

	mux.Handle("/", http.FileServerFS(staticFS))

	return logMiddleware(mux)
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func parseJSONBody(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}
