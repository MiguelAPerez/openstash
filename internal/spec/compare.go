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
}

// SchemaDiff is a schema present in both specs with differing fields.
type SchemaDiff struct {
	Name          string   `json:"name"`
	FieldsAdded   []string `json:"fieldsAdded,omitempty"`
	FieldsRemoved []string `json:"fieldsRemoved,omitempty"`
	// DefinitionChanged is set by CompareDetailed when the raw schema definition
	// differs (types, formats, nested properties, ...), not just property names.
	DefinitionChanged bool `json:"definitionChanged,omitempty"`
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
// but CompareDetailed does.
var definitionFields = []string{"parameters", "requestBody", "responses", "deprecated", "security"}

// CompareDetailed is Compare plus definition-level change detection. Operations
// present in both specs whose parameters, request body, responses, deprecation
// or security differ are reported as changed, with an empty FieldChange under
// each differing field name (the values are full sub-documents, so callers fetch
// both sides when they need them). Schemas present in both specs whose raw
// definitions differ get DefinitionChanged set.
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
		b := RawOperation(baseline, op.Path, op.Method)
		t := RawOperation(target, op.Path, op.Method)
		var fields []string
		for _, f := range definitionFields {
			if !reflect.DeepEqual(b[f], t[f]) {
				fields = append(fields, f)
			}
		}
		if len(fields) == 0 {
			continue
		}
		i, ok := changedOps[key]
		if !ok {
			res.Operations.Changed = append(res.Operations.Changed, OperationChange{
				Method:  op.Method,
				Path:    op.Path,
				Changes: map[string]FieldChange{},
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
		res.Schemas.Changed = append(res.Schemas.Changed, SchemaDiff{Name: name, DefinitionChanged: true})
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

// RawOperation returns the operation object at paths[path][method], or nil.
func RawOperation(doc map[string]any, path, method string) map[string]any {
	paths, _ := doc["paths"].(map[string]any)
	item, _ := paths[path].(map[string]any)
	op, _ := item[strings.ToLower(method)].(map[string]any)
	return op
}
