package server

import (
	"fmt"
	"net/http"

	"github.com/MiguelAPerez/openstash/internal/spec"
)

// handlePull re-fetches a key's source (the one recorded on its latest stored
// version) and stores it as a new version when its info.version is not stored
// yet. The new version keeps the latest version's source and endpoint.
//
// It answers 201 when a version was added and 200 when the source still
// serves a version already stored.
func (s *Server) handlePull(w http.ResponseWriter, r *http.Request) {
	// Same CSRF guard as POST /v1/specs: a cross-site JSON POST needs a CORS
	// preflight this server never approves.
	if !isJSONContentType(r) {
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return
	}
	specKey := r.PathValue("specKey")
	if err := validatePathSegment("key", specKey); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	key, latest, err := s.resolveLatest(specKey)
	if err != nil {
		writeResolveError(w, err)
		return
	}
	meta, err := s.store.LoadMeta(key, latest)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if meta.Source == "" {
		writeError(w, http.StatusUnprocessableEntity, fmt.Sprintf("%s@%s has no recorded source to pull from", key, latest))
		return
	}

	doc, err := spec.LoadFrom(meta.Source)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	remote := spec.InfoVersion(doc)
	if remote == "" {
		writeError(w, http.StatusUnprocessableEntity, "the source spec has no info.version; add it with an explicit version instead")
		return
	}
	if err := validatePathSegment("version", remote); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	upToDate := func(version string) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":        "up-to-date",
			"key":           key,
			"version":       version,
			"remoteVersion": remote,
			"source":        meta.Source,
		})
	}
	if s.store.Exists(key, remote) {
		upToDate(remote)
		return
	}
	// The latest version may be stored under a custom tag (add --version)
	// while its document carries the same info.version as the source.
	if latestDoc, err := s.store.LoadSpec(key, latest); err == nil && spec.InfoVersion(latestDoc) == remote {
		upToDate(latest)
		return
	}

	added, res, err := s.store.Add(key, remote, meta.Source, meta.Endpoint, doc)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"status":   "added",
		"previous": latest,
		"meta":     added,
		"indexed":  res.Operations,
		"schemas":  res.Schemas,
	})
}
