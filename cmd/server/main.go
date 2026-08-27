// Command server runs the PR checklist tool: a local dashboard for tracking
// GitHub PR review-thread resolution independently of GitHub's own UI state.
package main

import (
	"log"
	"net/http"

	"github.com/benanderson/pr-checklist/internal/api"
	"github.com/benanderson/pr-checklist/internal/config"
	"github.com/benanderson/pr-checklist/internal/githubapi"
	"github.com/benanderson/pr-checklist/internal/store"
	"github.com/benanderson/pr-checklist/web"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	st, err := store.New(cfg.DataDir)
	if err != nil {
		log.Fatalf("store: %v", err)
	}

	srv := &api.Server{
		Store:  st,
		GitHub: githubapi.NewClient(cfg.GitHubToken),
	}

	handler := api.NewRouter(srv, web.StaticFS)

	addr := ":" + cfg.Port
	log.Printf("pr-checklist listening on %s (data dir: %s)", addr, cfg.DataDir)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
