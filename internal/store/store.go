package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MiguelAPerez/openstash/internal/spec"
)

const defaultDirName = ".openstash"

// Store persists specs on disk.
type Store struct {
	Root string
}

func Default() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	root := filepath.Join(home, defaultDirName)
	return New(root)
}

func New(root string) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(root, "specs"), 0o755); err != nil {
		return nil, err
	}
	return &Store{Root: root}, nil
}

func (s *Store) specDir(key, version string) string {
	return filepath.Join(s.Root, "specs", sanitize(key), sanitize(version))
}

func (s *Store) Add(key, version, source, endpoint string, doc map[string]any) (spec.Meta, error) {
	key = strings.TrimSpace(key)
	version = strings.TrimSpace(version)
	if key == "" || version == "" {
		return spec.Meta{}, fmt.Errorf("key and version are required")
	}

	dir := s.specDir(key, version)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return spec.Meta{}, err
	}

	specPath := filepath.Join(dir, "spec.json")
	f, err := os.Create(specPath)
	if err != nil {
		return spec.Meta{}, err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		_ = f.Close()
		return spec.Meta{}, err
	}
	if err := f.Close(); err != nil {
		return spec.Meta{}, err
	}

	index := spec.BuildIndex(doc)
	idxPath := filepath.Join(dir, "index.json")
	if err := writeJSON(idxPath, index); err != nil {
		return spec.Meta{}, err
	}

	schemaIndex := spec.BuildSchemaIndex(doc)
	if err := writeJSON(filepath.Join(dir, "schemas.json"), schemaIndex); err != nil {
		return spec.Meta{}, err
	}

	meta := spec.Meta{
		Key:       key,
		Version:   version,
		Source:    source,
		Endpoint:  endpoint,
		FetchedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := writeJSON(filepath.Join(dir, "meta.json"), meta); err != nil {
		return spec.Meta{}, err
	}
	return meta, nil
}

func (s *Store) LoadMeta(key, version string) (spec.Meta, error) {
	var meta spec.Meta
	err := readJSON(filepath.Join(s.specDir(key, version), "meta.json"), &meta)
	return meta, err
}

func (s *Store) LoadSpec(key, version string) (map[string]any, error) {
	var doc map[string]any
	err := readJSON(filepath.Join(s.specDir(key, version), "spec.json"), &doc)
	return doc, err
}

func (s *Store) LoadIndex(key, version string) ([]spec.OperationIndex, error) {
	var index []spec.OperationIndex
	err := readJSON(filepath.Join(s.specDir(key, version), "index.json"), &index)
	return index, err
}

// LoadSchemaIndex reads schemas.json; returns (nil, nil) if the file does not exist
// (backward-compat with specs added before schema indexing was introduced).
func (s *Store) LoadSchemaIndex(key, version string) ([]spec.SchemaIndex, error) {
	var idx []spec.SchemaIndex
	err := readJSON(filepath.Join(s.specDir(key, version), "schemas.json"), &idx)
	if err != nil && os.IsNotExist(err) {
		return nil, nil
	}
	return idx, err
}

func (s *Store) Exists(key, version string) bool {
	_, err := os.Stat(filepath.Join(s.specDir(key, version), "spec.json"))
	return err == nil
}

type Entry struct {
	spec.Meta
	SpecVersion string `json:"specVersion,omitempty"`
}

func (s *Store) List() ([]Entry, error) {
	base := filepath.Join(s.Root, "specs")
	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var out []Entry
	for _, keyDir := range entries {
		if !keyDir.IsDir() {
			continue
		}
		keyPath := filepath.Join(base, keyDir.Name())
		vers, err := os.ReadDir(keyPath)
		if err != nil {
			continue
		}
		for _, verDir := range vers {
			if !verDir.IsDir() {
				continue
			}
			key := unsanitize(keyDir.Name())
			version := unsanitize(verDir.Name())
			meta, err := s.LoadMeta(key, version)
			if err != nil {
				continue
			}
			doc, err := s.LoadSpec(key, version)
			if err == nil {
				out = append(out, Entry{
					Meta:        meta,
					SpecVersion: spec.InfoVersion(doc),
				})
			} else {
				out = append(out, Entry{Meta: meta})
			}
		}
	}
	return out, nil
}

func (s *Store) VersionsForKey(key string) ([]string, error) {
	keyPath := filepath.Join(s.Root, "specs", sanitize(key))
	vers, err := os.ReadDir(keyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, v := range vers {
		if v.IsDir() {
			out = append(out, unsanitize(v.Name()))
		}
	}
	return out, nil
}

// LatestVersion returns the highest stored version for key using semver when possible.
func (s *Store) LatestVersion(key string) (string, error) {
	vers, err := s.VersionsForKey(key)
	if err != nil {
		return "", err
	}
	if len(vers) == 0 {
		return "", fmt.Errorf("no versions stored for key %q (run: openstash add)", key)
	}
	return maxVersion(vers), nil
}

// ResolveRef parses key, key@version, or key@ and resolves an omitted version to the latest stored one.
func (s *Store) ResolveRef(ref string) (spec.Ref, error) {
	r, err := ParseRef(ref)
	if err != nil {
		return spec.Ref{}, err
	}
	if r.Version != "" {
		return r, nil
	}
	version, err := s.LatestVersion(r.Key)
	if err != nil {
		return spec.Ref{}, err
	}
	return spec.Ref{Key: r.Key, Version: version}, nil
}

func writeJSON(path string, v any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func sanitize(s string) string {
	s = strings.ReplaceAll(s, string(os.PathSeparator), "_")
	return strings.ReplaceAll(s, " ", "_")
}

func unsanitize(s string) string {
	return s
}

// ParseRef splits key@version. A bare key or trailing @ means version is omitted.
func ParseRef(ref string) (spec.Ref, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return spec.Ref{}, fmt.Errorf("empty ref")
	}
	at := strings.LastIndex(ref, "@")
	if at == -1 {
		return spec.Ref{Key: ref}, nil
	}
	if at == 0 {
		return spec.Ref{}, fmt.Errorf("invalid ref: missing key before @")
	}
	return spec.Ref{
		Key:     ref[:at],
		Version: strings.TrimSpace(ref[at+1:]),
	}, nil
}
