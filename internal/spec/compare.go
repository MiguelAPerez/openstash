package spec

import (
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
	if strings.Join(baseline.Tags, "\x00") != strings.Join(target.Tags, "\x00") {
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
