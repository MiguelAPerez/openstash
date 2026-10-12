package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/MiguelAPerez/openstash/internal/spec"
	"github.com/MiguelAPerez/openstash/internal/store"
)

// timelineVersion summarizes one stored version of a key.
type timelineVersion struct {
	Version     string `json:"version"`
	Title       string `json:"title,omitempty"`
	InfoVersion string `json:"infoVersion,omitempty"`
	Source      string `json:"source"`
	FetchedAt   string `json:"fetchedAt"`
	Operations  int    `json:"operations"`
	Schemas     int    `json:"schemas"`
}

// timelineStep is the diff summary between two consecutive versions.
type timelineStep struct {
	From    string              `json:"from"`
	To      string              `json:"to"`
	Summary spec.CompareSummary `json:"summary"`
}

// handleTimeline lists every stored version of a key oldest to newest, with a
// change summary for each consecutive pair.
func (s *Server) handleTimeline(w http.ResponseWriter, r *http.Request) {
	specKey := r.PathValue("specKey")
	if err := validatePathSegment("key", specKey); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	vers, err := s.store.VersionsForKey(specKey)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if len(vers) == 0 {
		writeError(w, http.StatusNotFound, fmt.Sprintf("not found: %s (run: openstash add)", specKey))
		return
	}
	store.SortVersions(vers)

	versions := make([]timelineVersion, 0, len(vers))
	steps := make([]timelineStep, 0, len(vers))
	var prevDoc map[string]any
	for i, v := range vers {
		meta, err := s.store.LoadMeta(specKey, v)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		doc, err := s.store.LoadSpec(specKey, v)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		versions = append(versions, timelineVersion{
			Version:     v,
			Title:       infoTitle(doc),
			InfoVersion: spec.InfoVersion(doc),
			Source:      meta.Source,
			FetchedAt:   meta.FetchedAt,
			Operations:  len(spec.BuildIndex(doc)),
			Schemas:     len(spec.BuildSchemaIndex(doc)),
		})
		if i > 0 {
			steps = append(steps, timelineStep{
				From:    vers[i-1],
				To:      v,
				Summary: spec.CompareDetailed(prevDoc, doc).Summary,
			})
		}
		prevDoc = doc
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"key":      specKey,
		"versions": versions,
		"steps":    steps,
	})
}

// handleCompare diffs two stored versions of a key (from = baseline, to = target)
// with definition-level change detection.
func (s *Server) handleCompare(w http.ResponseWriter, r *http.Request) {
	key, from, to, docs, ok := s.loadComparePair(w, r)
	if !ok {
		return
	}
	result := spec.CompareDetailed(docs[0], docs[1])
	result.Baseline.Key, result.Baseline.Version = key, from
	result.Target.Key, result.Target.Version = key, to
	writeJSON(w, http.StatusOK, result)
}

// handleCompareDefinition returns the raw definition of one operation
// (method + path) or one component schema (schema) from both versions, so a
// client can render a line diff. A side is null when the item is absent there.
func (s *Server) handleCompareDefinition(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	method, path, schema := strings.TrimSpace(q.Get("method")), q.Get("path"), q.Get("schema")
	if (method == "" || path == "") && schema == "" {
		writeError(w, http.StatusBadRequest, "provide method and path, or schema")
		return
	}
	key, from, to, docs, ok := s.loadComparePair(w, r)
	if !ok {
		return
	}

	sides := make([]any, 2)
	for i, doc := range docs {
		if schema != "" {
			if node, err := spec.GetSchema(doc, schema); err == nil {
				sides[i] = node
			}
			continue
		}
		if op := spec.RawOperation(doc, path, method); op != nil {
			sides[i] = op
		}
	}
	if sides[0] == nil && sides[1] == nil {
		writeError(w, http.StatusNotFound, "not found in either version")
		return
	}

	item := map[string]any{"schema": schema}
	if schema == "" {
		item = map[string]any{"method": strings.ToUpper(method), "path": path}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"key":      key,
		"from":     from,
		"to":       to,
		"item":     item,
		"baseline": sides[0],
		"target":   sides[1],
	})
}

// loadComparePair validates specKey/from/to and loads both documents, writing
// an error response and returning ok=false on failure.
func (s *Server) loadComparePair(w http.ResponseWriter, r *http.Request) (key, from, to string, docs [2]map[string]any, ok bool) {
	key = r.PathValue("specKey")
	from = strings.TrimSpace(r.URL.Query().Get("from"))
	to = strings.TrimSpace(r.URL.Query().Get("to"))
	if from == "" || to == "" {
		writeError(w, http.StatusBadRequest, "from and to query parameters are required")
		return
	}
	if err := validateRefParams(w, key, from); err != nil {
		return
	}
	if err := validatePathSegment("version", to); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	for i, v := range []string{from, to} {
		if !s.store.Exists(key, v) {
			writeError(w, http.StatusNotFound, fmt.Sprintf("not found: %s@%s", key, v))
			return
		}
		doc, err := s.store.LoadSpec(key, v)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		docs[i] = doc
	}
	return key, from, to, docs, true
}

// handleSchema returns one component schema with $refs inlined to depth
// (default 1), mirroring `openstash schema`.
func (s *Server) handleSchema(w http.ResponseWriter, r *http.Request) {
	specKey := r.PathValue("specKey")
	version := r.PathValue("version")
	name := r.PathValue("name")
	if err := validateRefParams(w, specKey, version); err != nil {
		return
	}
	if !s.store.Exists(specKey, version) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("not found: %s@%s", specKey, version))
		return
	}
	doc, err := s.store.LoadSpec(specKey, version)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	node, err := spec.GetSchema(doc, name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	depth := queryInt(r, "depth", 1)
	writeJSON(w, http.StatusOK, map[string]any{
		"ref":     formatRef(specKey, version),
		"key":     specKey,
		"version": version,
		"name":    name,
		"depth":   depth,
		"schema":  spec.ResolveSchema(doc, node, depth),
	})
}

func infoTitle(doc map[string]any) string {
	info, _ := doc["info"].(map[string]any)
	title, _ := info["title"].(string)
	return title
}
