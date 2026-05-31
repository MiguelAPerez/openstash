package cli

import (
	"fmt"

	"github.com/MiguelAPerez/openstash/internal/spec"
	"github.com/MiguelAPerez/openstash/internal/store"
)

func openStore() (*store.Store, error) {
	if storeRoot != "" {
		return store.New(storeRoot)
	}
	return store.Default()
}

func mustRef(ref string) (string, string, error) {
	st, err := openStore()
	if err != nil {
		return "", "", err
	}
	r, err := st.ResolveRef(ref)
	if err != nil {
		return "", "", err
	}
	return r.Key, r.Version, nil
}

func formatRef(key, version string) string {
	return key + "@" + version
}

func mustLoad(ref string) (*store.Store, string, string, map[string]any, []spec.OperationIndex, error) {
	st, err := openStore()
	if err != nil {
		return nil, "", "", nil, nil, err
	}
	key, version, err := mustRef(ref)
	if err != nil {
		return st, "", "", nil, nil, err
	}
	if !st.Exists(key, version) {
		return st, key, version, nil, nil, fmt.Errorf("not found: %s@%s (run: openstash add)", key, version)
	}
	doc, err := st.LoadSpec(key, version)
	if err != nil {
		return st, key, version, nil, nil, err
	}
	index, err := st.LoadIndex(key, version)
	if err != nil {
		return st, key, version, doc, nil, err
	}
	return st, key, version, doc, index, nil
}
