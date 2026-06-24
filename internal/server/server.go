package server

import (
	"net/http"
	"time"

	"github.com/MiguelAPerez/openstash/internal/store"
)

// DefaultMaxBodyBytes caps the POST /v1/specs request body when no override is
// configured. The body only carries a handful of short string fields, so the
// default is deliberately small; raise it via OPENSTASH_MAX_BODY_BYTES if you
// post specs by value through a field rather than a URL/path.
const DefaultMaxBodyBytes int64 = 64 << 10 // 64 KiB

// Server exposes the openstash store over HTTP.
type Server struct {
	store        *store.Store
	http         *http.Server
	maxBodyBytes int64
}

// New builds an HTTP server listening on addr. A maxBodyBytes <= 0 falls back to
// DefaultMaxBodyBytes.
func New(st *store.Store, addr string, maxBodyBytes int64) *Server {
	if maxBodyBytes <= 0 {
		maxBodyBytes = DefaultMaxBodyBytes
	}
	s := &Server{store: st, maxBodyBytes: maxBodyBytes}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /v1/specs", s.handleListSpecs)
	mux.HandleFunc("POST /v1/specs", s.handleAddSpec)
	mux.HandleFunc("GET /v1/specs/{specKey}", s.handleDumpLatest)
	mux.HandleFunc("GET /v1/specs/{specKey}/versions", s.handleListVersions)
	mux.HandleFunc("GET /v1/specs/{specKey}/versions/{version}", s.handleDumpVersion)
	mux.HandleFunc("GET /v1/specs/{specKey}/versions/{version}/operations", s.handleOperations)
	s.http = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
	}
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
