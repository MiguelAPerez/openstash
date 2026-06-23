package server

import (
	"net/http"

	"github.com/MiguelAPerez/openstash/internal/store"
)

// Server exposes the openstash store over HTTP.
type Server struct {
	store *store.Store
	http  *http.Server
}

// New builds an HTTP server listening on addr.
func New(st *store.Store, addr string) *Server {
	s := &Server{store: st}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /v1/specs", s.handleListSpecs)
	mux.HandleFunc("POST /v1/specs", s.handleAddSpec)
	mux.HandleFunc("GET /v1/specs/{specKey}", s.handleDumpLatest)
	mux.HandleFunc("GET /v1/specs/{specKey}/versions", s.handleListVersions)
	mux.HandleFunc("GET /v1/specs/{specKey}/versions/{version}", s.handleDumpVersion)
	mux.HandleFunc("GET /v1/specs/{specKey}/versions/{version}/operations", s.handleOperations)
	s.http = &http.Server{Addr: addr, Handler: mux}
	return s
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	return s.http.ListenAndServe()
}

// Handler returns the root handler (for tests).
func (s *Server) Handler() http.Handler {
	return s.http.Handler
}
