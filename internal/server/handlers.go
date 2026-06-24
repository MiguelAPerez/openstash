package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/MiguelAPerez/openstash/internal/search"
	"github.com/MiguelAPerez/openstash/internal/spec"
)

var validInScopes = map[string]bool{
	"paths":   true,
	"schemas": true,
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleListSpecs(w http.ResponseWriter, _ *http.Request) {
	entries, err := s.store.List()
	if err != nil {
		writeStoreError(w, err)
		return
	}

	enriched := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		enriched = append(enriched, map[string]any{
			"key":         e.Key,
			"version":     e.Version,
			"source":      e.Source,
			"endpoint":    e.Endpoint,
			"fetchedAt":   e.FetchedAt,
			"specVersion": e.SpecVersion,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"entries": enriched})
}

type addSpecRequest struct {
	Key      string `json:"key"`
	From     string `json:"from"`
	Version  string `json:"version"`
	Endpoint string `json:"endpoint"`
}

func (s *Server) handleAddSpec(w http.ResponseWriter, r *http.Request) {
	var req addSpecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req.Key = strings.TrimSpace(req.Key)
	req.From = strings.TrimSpace(req.From)
	req.Version = strings.TrimSpace(req.Version)
	req.Endpoint = strings.TrimSpace(req.Endpoint)

	if req.Key == "" || req.From == "" {
		writeError(w, http.StatusBadRequest, "key and from are required")
		return
	}

	doc, err := spec.LoadFrom(req.From)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	version := req.Version
	if version == "" {
		version = spec.InfoVersion(doc)
		if version == "" {
			writeError(w, http.StatusUnprocessableEntity, "version required (spec has no info.version)")
			return
		}
	}

	if s.store.Exists(req.Key, version) {
		writeError(w, http.StatusConflict, fmt.Sprintf("already exists: %s@%s", req.Key, version))
		return
	}

	meta, res, err := s.store.Add(req.Key, version, req.From, req.Endpoint, doc)
	if err != nil {
		writeStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"status":  "added",
		"meta":    meta,
		"indexed": res.Operations,
		"schemas": res.Schemas,
	})
}

func (s *Server) handleDumpLatest(w http.ResponseWriter, r *http.Request) {
	specKey := r.PathValue("specKey")
	key, version, err := s.resolveLatest(specKey)
	if err != nil {
		writeResolveError(w, err)
		return
	}
	s.writeDump(w, key, version)
}

func (s *Server) handleListVersions(w http.ResponseWriter, r *http.Request) {
	specKey := r.PathValue("specKey")
	vers, err := s.store.VersionsForKey(specKey)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if len(vers) == 0 {
		writeError(w, http.StatusNotFound, fmt.Sprintf("not found: %s (run: openstash add)", specKey))
		return
	}
	sort.Strings(vers)
	writeJSON(w, http.StatusOK, map[string]any{
		"key":      specKey,
		"versions": vers,
	})
}

func (s *Server) handleDumpVersion(w http.ResponseWriter, r *http.Request) {
	specKey := r.PathValue("specKey")
	version := r.PathValue("version")
	if !s.store.Exists(specKey, version) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("not found: %s@%s", specKey, version))
		return
	}
	s.writeDump(w, specKey, version)
}

func (s *Server) writeDump(w http.ResponseWriter, key, version string) {
	doc, err := s.store.LoadSpec(key, version)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (s *Server) handleOperations(w http.ResponseWriter, r *http.Request) {
	specKey := r.PathValue("specKey")
	version := r.PathValue("version")

	key, ver, doc, index, err := s.loadSpec(specKey, version)
	if err != nil {
		writeResolveError(w, err)
		return
	}

	detail := strings.TrimSpace(r.URL.Query().Get("detail"))
	if detail == "" {
		detail = "search"
	}

	switch detail {
	case "search":
		s.handleOperationsSearch(w, r, key, ver, doc, index)
	case "show":
		s.handleOperationsShow(w, r, key, ver, doc)
	case "gather":
		s.handleOperationsGather(w, r, key, ver, doc, index)
	default:
		writeError(w, http.StatusBadRequest, "detail must be search, show, or gather")
	}
}

func (s *Server) handleOperationsSearch(w http.ResponseWriter, r *http.Request, key, version string, doc map[string]any, index []spec.OperationIndex) {
	q := r.URL.Query().Get("q")
	limit := queryInt(r, "limit", 5)
	pathPrefix := r.URL.Query().Get("pathPrefix")
	method := r.URL.Query().Get("method")

	scopeSet, err := parseInScopes(r.URL.Query()["in"])
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result := map[string]any{
		"ref":     formatRef(key, version),
		"key":     key,
		"version": version,
		"query":   q,
		"detail":  "search",
	}

	if scopeSet["paths"] {
		result["hits"] = search.Query(index, q, limit, pathPrefix, method)
	}

	if scopeSet["schemas"] {
		schemaIdx, loadErr := s.store.LoadSchemaIndex(key, version)
		if loadErr != nil {
			writeStoreError(w, loadErr)
			return
		}
		if schemaIdx == nil {
			schemaIdx = spec.BuildSchemaIndex(doc)
		}
		result["schemas"] = search.SearchSchemas(schemaIdx, q, limit)
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleOperationsShow(w http.ResponseWriter, r *http.Request, key, version string, doc map[string]any) {
	path := r.URL.Query().Get("path")
	method := r.URL.Query().Get("method")
	if path == "" || method == "" {
		writeError(w, http.StatusBadRequest, "path and method query parameters are required for detail=show")
		return
	}

	depth, expand := queryDepth(r)
	op, err := spec.GetOperationDepth(doc, path, method, depth)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ref":       formatRef(key, version),
		"key":       key,
		"version":   version,
		"depth":     depth,
		"detail":    "show",
		"operation": op,
		"expand":    expand,
	})
}

func (s *Server) handleOperationsGather(w http.ResponseWriter, r *http.Request, key, version string, doc map[string]any, index []spec.OperationIndex) {
	q := r.URL.Query().Get("q")
	limit := queryInt(r, "limit", 5)
	pathPrefix := r.URL.Query().Get("pathPrefix")
	method := r.URL.Query().Get("method")
	path := r.URL.Query().Get("path")
	depth, expand := queryDepth(r)

	if path != "" && method != "" {
		op, err := spec.GetOperationDepth(doc, path, method, depth)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ref":        formatRef(key, version),
			"key":        key,
			"version":    version,
			"depth":      depth,
			"detail":     "gather",
			"mode":       "exact",
			"expand":     expand,
			"operations": []any{op},
		})
		return
	}

	if q == "" && pathPrefix == "" && method == "" {
		writeError(w, http.StatusBadRequest, "provide q, pathPrefix, method, or path with method")
		return
	}

	hits := search.Query(index, q, limit, pathPrefix, method)
	if len(hits) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{
			"ref": formatRef(key, version), "key": key, "version": version,
			"depth": depth, "detail": "gather", "mode": "search", "query": q,
			"hits": hits, "operations": []any{},
		})
		return
	}

	var operations []*spec.OperationDetail
	for _, h := range hits {
		op, err := spec.GetOperationDepth(doc, h.Operation.Path, h.Operation.Method, depth)
		if err != nil {
			continue
		}
		operations = append(operations, op)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ref":        formatRef(key, version),
		"key":        key,
		"version":    version,
		"depth":      depth,
		"detail":     "gather",
		"mode":       "search",
		"query":      q,
		"hits":       hits,
		"operations": operations,
	})
}

func (s *Server) resolveLatest(specKey string) (string, string, error) {
	ref, err := s.store.ResolveRef(specKey)
	if err != nil {
		return "", "", err
	}
	if !s.store.Exists(ref.Key, ref.Version) {
		return "", "", fmt.Errorf("not found: %s@%s (run: openstash add)", ref.Key, ref.Version)
	}
	return ref.Key, ref.Version, nil
}

func (s *Server) loadSpec(specKey, version string) (string, string, map[string]any, []spec.OperationIndex, error) {
	key := specKey
	ver := version
	if !s.store.Exists(key, ver) {
		return "", "", nil, nil, fmt.Errorf("not found: %s@%s (run: openstash add)", key, ver)
	}
	doc, err := s.store.LoadSpec(key, ver)
	if err != nil {
		return "", "", nil, nil, err
	}
	index, err := s.store.LoadIndex(key, ver)
	if err != nil {
		return "", "", nil, nil, err
	}
	return key, ver, doc, index, nil
}

func parseInScopes(raw []string) (map[string]bool, error) {
	scopeSet := make(map[string]bool)
	for _, part := range raw {
		for _, item := range strings.Split(part, ",") {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if !validInScopes[item] {
				return nil, fmt.Errorf("unknown in value %q; valid values: paths, schemas", item)
			}
			scopeSet[item] = true
		}
	}
	if len(scopeSet) == 0 {
		scopeSet["paths"] = true
	}
	return scopeSet, nil
}

func queryInt(r *http.Request, name string, defaultVal int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return defaultVal
	}
	return n
}

func queryDepth(r *http.Request) (int, bool) {
	depth := queryInt(r, "depth", 0)
	expand := r.URL.Query().Get("expand") == "true"
	if depth == 0 && expand {
		depth = 1
	}
	return depth, expand
}

func writeResolveError(w http.ResponseWriter, err error) {
	msg := err.Error()
	if strings.Contains(msg, "not found") || strings.Contains(msg, "no versions stored") {
		writeError(w, http.StatusNotFound, msg)
		return
	}
	writeStoreError(w, err)
}

func writeStoreError(w http.ResponseWriter, err error) {
	writeError(w, http.StatusInternalServerError, err.Error())
}
