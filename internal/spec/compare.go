package spec

import (
	"reflect"
	"slices"
	"sort"
	"strings"
)

// FieldChange records a single field difference between baseline and target.
type FieldChange struct {
	Baseline any `json:"baseline,omitempty"`
	Target   any `json:"target,omitempty"`
}

// CompareSide summarizes one spec in a diff.
type CompareSide struct {
	Key        string `json:"key"`
	Version    string `json:"version"`
	Operations int    `json:"operations"`
	Schemas    int    `json:"schemas"`
}

// CompareSummary holds high-level diff counts.
type CompareSummary struct {
	Operations CompareCounts `json:"operations"`
	Schemas    CompareCounts `json:"schemas"`
}

// CompareCounts reports added, removed, changed, and unchanged items.
// Added = in target only; removed = in baseline only; changed = in both with differences.
type CompareCounts struct {
	Added     int `json:"added"`
	Removed   int `json:"removed"`
	Changed   int `json:"changed"`
	Unchanged int `json:"unchanged"`
}

// OperationChange is an operation present in both specs with field-level diffs.
type OperationChange struct {
	Method  string                 `json:"method"`
	Path    string                 `json:"path"`
	Changes map[string]FieldChange `json:"changes"`
	// ExtensionsOnly is set by CompareDetailed when the only differences are in
	// vendor extensions (x-* keys), e.g. generator metadata.
	ExtensionsOnly bool `json:"extensionsOnly,omitempty"`
}

// SchemaDiff is a schema present in both specs with differing fields.
type SchemaDiff struct {
	Name          string   `json:"name"`
	FieldsAdded   []string `json:"fieldsAdded,omitempty"`
	FieldsRemoved []string `json:"fieldsRemoved,omitempty"`
	// DefinitionChanged is set by CompareDetailed when the raw schema definition
	// differs (types, formats, nested properties, ...), not just property names.
	DefinitionChanged bool `json:"definitionChanged,omitempty"`
	// ExtensionsOnly is set by CompareDetailed when the definitions differ only
	// in vendor extensions (x-* keys), e.g. x-go-package renames.
	ExtensionsOnly bool `json:"extensionsOnly,omitempty"`
}

// CompareResult is the full diff between two OpenAPI documents.
type CompareResult struct {
	Baseline   CompareSide      `json:"baseline"`
	Target     CompareSide      `json:"target"`
	Summary    CompareSummary   `json:"summary"`
	Operations CompareOpsResult `json:"operations"`
	Schemas    CompareSchemas   `json:"schemas"`
}

// CompareOpsResult lists operation-level changes.
type CompareOpsResult struct {
	Added   []OperationIndex  `json:"added"`
	Removed []OperationIndex  `json:"removed"`
	Changed []OperationChange `json:"changed"`
}

// CompareSchemas lists schema-level changes.
type CompareSchemas struct {
	Added   []string     `json:"added"`
	Removed []string     `json:"removed"`
	Changed []SchemaDiff `json:"changed"`
}

// Compare diffs two OpenAPI documents by operations and component schemas.
// The first document is the baseline; the second is the target.
func Compare(baseline, target map[string]any) CompareResult {
	baselineOps := BuildIndex(baseline)
	targetOps := BuildIndex(target)
	baselineSchemas := BuildSchemaIndex(baseline)
	targetSchemas := BuildSchemaIndex(target)

	opAdded, opRemoved, opChanged := diffOperations(baselineOps, targetOps)
	schemaAdded, schemaRemoved, schemaChanged := diffSchemas(baselineSchemas, targetSchemas)

	return CompareResult{
		Baseline: CompareSide{
			Operations: len(baselineOps),
			Schemas:    len(baselineSchemas),
		},
		Target: CompareSide{
			Operations: len(targetOps),
			Schemas:    len(targetSchemas),
		},
		Summary: CompareSummary{
			Operations: CompareCounts{
				Added:     len(opAdded),
				Removed:   len(opRemoved),
				Changed:   len(opChanged),
				Unchanged: len(baselineOps) - len(opRemoved) - len(opChanged),
			},
			Schemas: CompareCounts{
				Added:     len(schemaAdded),
				Removed:   len(schemaRemoved),
				Changed:   len(schemaChanged),
				Unchanged: len(baselineSchemas) - len(schemaRemoved) - len(schemaChanged),
			},
		},
		Operations: CompareOpsResult{
			Added:   opAdded,
			Removed: opRemoved,
			Changed: opChanged,
		},
		Schemas: CompareSchemas{
			Added:   schemaAdded,
			Removed: schemaRemoved,
			Changed: schemaChanged,
		},
	}
}

func operationKey(op OperationIndex) string {
	return op.Method + " " + op.Path
}

func diffOperations(baseline, target []OperationIndex) (added, removed []OperationIndex, changed []OperationChange) {
	baselineMap := make(map[string]OperationIndex, len(baseline))
	for _, op := range baseline {
		baselineMap[operationKey(op)] = op
	}
	targetMap := make(map[string]OperationIndex, len(target))
	for _, op := range target {
		targetMap[operationKey(op)] = op
	}

	for key, op := range targetMap {
		if baselineOp, ok := baselineMap[key]; ok {
			if changes := operationChanges(baselineOp, op); len(changes) > 0 {
				changed = append(changed, OperationChange{
					Method:  op.Method,
					Path:    op.Path,
					Changes: changes,
				})
			}
			continue
		}
		added = append(added, op)
	}

	for key, op := range baselineMap {
		if _, ok := targetMap[key]; !ok {
			removed = append(removed, op)
		}
	}

	sortOperations(added)
	sortOperations(removed)
	sort.Slice(changed, func(i, j int) bool {
		if changed[i].Method != changed[j].Method {
			return changed[i].Method < changed[j].Method
		}
		return changed[i].Path < changed[j].Path
	})
	return added, removed, changed
}

// operationChanges compares only the fields captured in OperationIndex (operationId, summary,
// description, tags). Parameter and response changes are not detected.
func operationChanges(baseline, target OperationIndex) map[string]FieldChange {
	changes := map[string]FieldChange{}
	if baseline.OperationID != target.OperationID {
		changes["operationId"] = FieldChange{Baseline: baseline.OperationID, Target: target.OperationID}
	}
	if baseline.Summary != target.Summary {
		changes["summary"] = FieldChange{Baseline: baseline.Summary, Target: target.Summary}
	}
	if baseline.Description != target.Description {
		changes["description"] = FieldChange{Baseline: baseline.Description, Target: target.Description}
	}
	if !slices.Equal(baseline.Tags, target.Tags) {
		changes["tags"] = FieldChange{Baseline: baseline.Tags, Target: target.Tags}
	}
	return changes
}

func sortOperations(ops []OperationIndex) {
	sort.Slice(ops, func(i, j int) bool {
		if ops[i].Method != ops[j].Method {
			return ops[i].Method < ops[j].Method
		}
		return ops[i].Path < ops[j].Path
	})
}

func diffSchemas(baseline, target []SchemaIndex) (added, removed []string, changed []SchemaDiff) {
	baselineMap := make(map[string]SchemaIndex, len(baseline))
	for _, s := range baseline {
		baselineMap[s.Name] = s
	}
	targetMap := make(map[string]SchemaIndex, len(target))
	for _, s := range target {
		targetMap[s.Name] = s
	}

	for name, targetSchema := range targetMap {
		baselineSchema, ok := baselineMap[name]
		if !ok {
			added = append(added, name)
			continue
		}
		fieldsAdded, fieldsRemoved := diffStringSets(baselineSchema.Properties, targetSchema.Properties)
		if len(fieldsAdded) > 0 || len(fieldsRemoved) > 0 {
			changed = append(changed, SchemaDiff{
				Name:          name,
				FieldsAdded:   fieldsAdded,
				FieldsRemoved: fieldsRemoved,
			})
		}
	}

	for name := range baselineMap {
		if _, ok := targetMap[name]; !ok {
			removed = append(removed, name)
		}
	}

	sort.Strings(added)
	sort.Strings(removed)
	sort.Slice(changed, func(i, j int) bool {
		return changed[i].Name < changed[j].Name
	})
	return added, removed, changed
}

func diffStringSets(baseline, target []string) (added, removed []string) {
	baselineSet := make(map[string]bool, len(baseline))
	for _, s := range baseline {
		baselineSet[s] = true
	}
	targetSet := make(map[string]bool, len(target))
	for _, s := range target {
		targetSet[s] = true
	}
	for s := range targetSet {
		if !baselineSet[s] {
			added = append(added, s)
		}
	}
	for s := range baselineSet {
		if !targetSet[s] {
			removed = append(removed, s)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	return added, removed
}

// definitionFields are the parts of an operation that Compare does not inspect
// but CompareDetailed does (on the effective operation, see EffectiveOperation).
var definitionFields = []string{
	"parameters", "requestBody", "responses", "deprecated", "security",
	"servers", "callbacks", "consumes", "produces",
}

// CompareDetailed is Compare plus definition-level change detection. Operations
// present in both specs whose effective parameters, request body, responses,
// deprecation, security, servers, callbacks or (Swagger 2) consumes/produces
// differ are reported as changed, with an empty FieldChange under each differing
// field name (the values are full sub-documents, so callers fetch both sides
// with EffectiveOperation when they need them). Changes behind shared $refs and
// in path-level parameters count. Schemas present in both specs whose raw
// definitions differ get DefinitionChanged set. Differences confined to x-*
// vendor extensions are flagged ExtensionsOnly.
//
// Only operations and component schemas are compared: changes to info, servers
// or security scheme definitions on their own are not reported.
func CompareDetailed(baseline, target map[string]any) CompareResult {
	res := Compare(baseline, target)

	changedOps := make(map[string]int, len(res.Operations.Changed))
	for i, c := range res.Operations.Changed {
		changedOps[c.Method+" "+c.Path] = i
	}
	baselineOps := make(map[string]bool)
	for _, op := range BuildIndex(baseline) {
		baselineOps[operationKey(op)] = true
	}
	for _, op := range BuildIndex(target) {
		key := operationKey(op)
		if !baselineOps[key] {
			continue
		}
		b := EffectiveOperation(baseline, op.Path, op.Method)
		t := EffectiveOperation(target, op.Path, op.Method)
		var fields []string
		extensionsOnly := true
		for _, f := range definitionFields {
			if reflect.DeepEqual(b[f], t[f]) {
				continue
			}
			fields = append(fields, f)
			if !reflect.DeepEqual(withoutExtensions(b[f]), withoutExtensions(t[f])) {
				extensionsOnly = false
			}
		}
		if len(fields) == 0 {
			continue
		}
		i, ok := changedOps[key]
		if !ok {
			res.Operations.Changed = append(res.Operations.Changed, OperationChange{
				Method:         op.Method,
				Path:           op.Path,
				Changes:        map[string]FieldChange{},
				ExtensionsOnly: extensionsOnly,
			})
			i = len(res.Operations.Changed) - 1
			changedOps[key] = i
		}
		for _, f := range fields {
			res.Operations.Changed[i].Changes[f] = FieldChange{}
		}
	}
	sort.Slice(res.Operations.Changed, func(i, j int) bool {
		a, b := res.Operations.Changed[i], res.Operations.Changed[j]
		if a.Method != b.Method {
			return a.Method < b.Method
		}
		return a.Path < b.Path
	})

	baselineSchemas, _ := schemasContainer(baseline)
	targetSchemas, _ := schemasContainer(target)
	changedSchemas := make(map[string]int, len(res.Schemas.Changed))
	for i, c := range res.Schemas.Changed {
		changedSchemas[c.Name] = i
	}
	for name, t := range targetSchemas {
		b, ok := baselineSchemas[name]
		if !ok || reflect.DeepEqual(b, t) {
			continue
		}
		if i, ok := changedSchemas[name]; ok {
			res.Schemas.Changed[i].DefinitionChanged = true
			continue
		}
		res.Schemas.Changed = append(res.Schemas.Changed, SchemaDiff{
			Name:              name,
			DefinitionChanged: true,
			ExtensionsOnly:    reflect.DeepEqual(withoutExtensions(b), withoutExtensions(t)),
		})
	}
	sort.Slice(res.Schemas.Changed, func(i, j int) bool {
		return res.Schemas.Changed[i].Name < res.Schemas.Changed[j].Name
	})

	ops := &res.Summary.Operations
	ops.Changed = len(res.Operations.Changed)
	ops.Unchanged = res.Baseline.Operations - ops.Removed - ops.Changed
	schemas := &res.Summary.Schemas
	schemas.Changed = len(res.Schemas.Changed)
	schemas.Unchanged = res.Baseline.Schemas - schemas.Removed - schemas.Changed
	return res
}

// EffectiveOperation returns the operation at paths[path][method] as a client
// sees it, or nil when it doesn't exist:
//   - path-level parameters are merged in (operation entries override by in+name)
//   - $refs to shared parameters, request bodies, responses and response
//     headers are replaced by their targets (schema $refs are left as they are;
//     component schemas are compared on their own)
//   - security, and Swagger 2 consumes/produces, fall back to the document-level
//     defaults when the operation doesn't set them.
//
// The returned map shares nested values with doc; treat it as read-only.
func EffectiveOperation(doc map[string]any, path, method string) map[string]any {
	paths, _ := doc["paths"].(map[string]any)
	item, _ := paths[path].(map[string]any)
	op, _ := item[strings.ToLower(method)].(map[string]any)
	if op == nil {
		return nil
	}
	out := make(map[string]any, len(op)+3)
	for k, v := range op {
		out[k] = v
	}
	delete(out, "parameters")
	if params := mergeParameters(doc, item["parameters"], op["parameters"]); len(params) > 0 {
		out["parameters"] = params
	}
	if rb, ok := op["requestBody"].(map[string]any); ok {
		out["requestBody"] = resolveLocalRef(doc, rb)
	}
	if responses, ok := op["responses"].(map[string]any); ok {
		resolved := make(map[string]any, len(responses))
		for code, v := range responses {
			r, ok := v.(map[string]any)
			if !ok {
				resolved[code] = v
				continue
			}
			r = resolveLocalRef(doc, r)
			if headers, ok := r["headers"].(map[string]any); ok {
				copied := make(map[string]any, len(r))
				for k, v := range r {
					copied[k] = v
				}
				hs := make(map[string]any, len(headers))
				for name, h := range headers {
					if hm, ok := h.(map[string]any); ok {
						hs[name] = resolveLocalRef(doc, hm)
					} else {
						hs[name] = h
					}
				}
				copied["headers"] = hs
				r = copied
			}
			resolved[code] = r
		}
		out["responses"] = resolved
	}
	for _, f := range []string{"security", "consumes", "produces"} {
		if _, ok := out[f]; !ok {
			if v, ok := doc[f]; ok {
				out[f] = v
			}
		}
	}
	return out
}

// mergeParameters resolves path-level and operation-level parameter lists and
// merges them; an operation parameter replaces a path-level one with the same
// in+name.
func mergeParameters(doc map[string]any, shared, own any) []any {
	resolve := func(v any) []any {
		list, _ := v.([]any)
		out := make([]any, 0, len(list))
		for _, p := range list {
			if pm, ok := p.(map[string]any); ok {
				out = append(out, resolveLocalRef(doc, pm))
			} else {
				out = append(out, p)
			}
		}
		return out
	}
	key := func(p any) string {
		pm, _ := p.(map[string]any)
		in, _ := pm["in"].(string)
		name, _ := pm["name"].(string)
		return in + ":" + name
	}
	ownList := resolve(own)
	overridden := make(map[string]bool, len(ownList))
	for _, p := range ownList {
		overridden[key(p)] = true
	}
	var merged []any
	for _, p := range resolve(shared) {
		if !overridden[key(p)] {
			merged = append(merged, p)
		}
	}
	return append(merged, ownList...)
}

// resolveLocalRef follows a chain of local $refs ("#/components/parameters/X",
// Swagger 2 "#/responses/X", ...) to the object it names. Unresolvable or
// external refs are returned unchanged.
func resolveLocalRef(doc, node map[string]any) map[string]any {
	for hops := 0; hops < 8; hops++ {
		ref, ok := node["$ref"].(string)
		if !ok || !strings.HasPrefix(ref, "#/") {
			break
		}
		target, err := ResolveRef(doc, ref)
		if err != nil {
			break
		}
		node = target
	}
	return node
}

// namedMaps are keys whose map values are keyed by user-chosen names (property
// names, media types, header names, ...), so a child key starting with "x-"
// there is data, not a vendor extension.
var namedMaps = map[string]bool{
	"properties": true, "patternProperties": true, "headers": true, "content": true,
	"responses": true, "examples": true, "encoding": true, "links": true,
	"callbacks": true, "mapping": true, "variables": true, "scopes": true,
	"definitions": true, "$defs": true,
}

// withoutExtensions returns v with vendor-extension (x-*) keys removed at every
// level, for telling real changes from generator-metadata noise.
func withoutExtensions(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, child := range t {
			if strings.HasPrefix(k, "x-") {
				continue
			}
			if named, ok := child.(map[string]any); ok && namedMaps[k] {
				m := make(map[string]any, len(named))
				for name, c := range named {
					m[name] = withoutExtensions(c)
				}
				out[k] = m
				continue
			}
			out[k] = withoutExtensions(child)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, c := range t {
			out[i] = withoutExtensions(c)
		}
		return out
	}
	return v
}

// RawSchema returns the raw component schema called name (an object or, in
// OpenAPI 3.1, a boolean) and whether it exists.
func RawSchema(doc map[string]any, name string) (any, bool) {
	schemas, ok := schemasContainer(doc)
	if !ok {
		return nil, false
	}
	v, ok := schemas[name]
	return v, ok
}
