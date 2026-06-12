package spec

import (
	"sort"
	"strings"
)

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
type CompareCounts struct {
	Added     int `json:"added"`
	Removed   int `json:"removed"`
	Changed   int `json:"changed"`
	Unchanged int `json:"unchanged"`
}

// OperationDiff is an operation present in both specs with differing metadata.
type OperationDiff struct {
	Method string         `json:"method"`
	Path   string         `json:"path"`
	Left   OperationIndex `json:"left"`
	Right  OperationIndex `json:"right"`
}

// SchemaDiff is a schema present in both specs with differing fields.
type SchemaDiff struct {
	Name          string   `json:"name"`
	FieldsAdded   []string `json:"fieldsAdded,omitempty"`
	FieldsRemoved []string `json:"fieldsRemoved,omitempty"`
}

// CompareResult is the full diff between two OpenAPI documents.
type CompareResult struct {
	Left       CompareSide      `json:"left"`
	Right      CompareSide      `json:"right"`
	Summary    CompareSummary   `json:"summary"`
	Operations CompareOpsResult `json:"operations"`
	Schemas    CompareSchemas   `json:"schemas"`
}

// CompareOpsResult lists operation-level changes.
type CompareOpsResult struct {
	Added   []OperationIndex `json:"added"`
	Removed []OperationIndex `json:"removed"`
	Changed []OperationDiff  `json:"changed"`
}

// CompareSchemas lists schema-level changes.
type CompareSchemas struct {
	Added   []string     `json:"added"`
	Removed []string     `json:"removed"`
	Changed []SchemaDiff `json:"changed"`
}

// Compare diffs two OpenAPI documents by operations and component schemas.
func Compare(left, right map[string]any) CompareResult {
	leftOps := BuildIndex(left)
	rightOps := BuildIndex(right)
	leftSchemas := BuildSchemaIndex(left)
	rightSchemas := BuildSchemaIndex(right)

	opAdded, opRemoved, opChanged := diffOperations(leftOps, rightOps)
	schemaAdded, schemaRemoved, schemaChanged := diffSchemas(leftSchemas, rightSchemas)

	return CompareResult{
		Left: CompareSide{
			Operations: len(leftOps),
			Schemas:    len(leftSchemas),
		},
		Right: CompareSide{
			Operations: len(rightOps),
			Schemas:    len(rightSchemas),
		},
		Summary: CompareSummary{
			Operations: CompareCounts{
				Added:     len(opAdded),
				Removed:   len(opRemoved),
				Changed:   len(opChanged),
				Unchanged: len(leftOps) - len(opRemoved) - len(opChanged),
			},
			Schemas: CompareCounts{
				Added:     len(schemaAdded),
				Removed:   len(schemaRemoved),
				Changed:   len(schemaChanged),
				Unchanged: len(leftSchemas) - len(schemaRemoved) - len(schemaChanged),
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

func diffOperations(left, right []OperationIndex) (added, removed []OperationIndex, changed []OperationDiff) {
	leftMap := make(map[string]OperationIndex, len(left))
	for _, op := range left {
		leftMap[operationKey(op)] = op
	}
	rightMap := make(map[string]OperationIndex, len(right))
	for _, op := range right {
		rightMap[operationKey(op)] = op
	}

	for key, op := range rightMap {
		if leftOp, ok := leftMap[key]; ok {
			if !operationsEqual(leftOp, op) {
				changed = append(changed, OperationDiff{
					Method: op.Method,
					Path:   op.Path,
					Left:   leftOp,
					Right:  op,
				})
			}
			continue
		}
		added = append(added, op)
	}

	for key, op := range leftMap {
		if _, ok := rightMap[key]; !ok {
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

func operationsEqual(a, b OperationIndex) bool {
	return a.OperationID == b.OperationID &&
		a.Summary == b.Summary &&
		a.Description == b.Description &&
		strings.Join(a.Tags, "\x00") == strings.Join(b.Tags, "\x00")
}

func sortOperations(ops []OperationIndex) {
	sort.Slice(ops, func(i, j int) bool {
		if ops[i].Method != ops[j].Method {
			return ops[i].Method < ops[j].Method
		}
		return ops[i].Path < ops[j].Path
	})
}

func diffSchemas(left, right []SchemaIndex) (added, removed []string, changed []SchemaDiff) {
	leftMap := make(map[string]SchemaIndex, len(left))
	for _, s := range left {
		leftMap[s.Name] = s
	}
	rightMap := make(map[string]SchemaIndex, len(right))
	for _, s := range right {
		rightMap[s.Name] = s
	}

	for name, rightSchema := range rightMap {
		leftSchema, ok := leftMap[name]
		if !ok {
			added = append(added, name)
			continue
		}
		fieldsAdded, fieldsRemoved := diffStringSets(leftSchema.Properties, rightSchema.Properties)
		if len(fieldsAdded) > 0 || len(fieldsRemoved) > 0 {
			changed = append(changed, SchemaDiff{
				Name:          name,
				FieldsAdded:   fieldsAdded,
				FieldsRemoved: fieldsRemoved,
			})
		}
	}

	for name := range leftMap {
		if _, ok := rightMap[name]; !ok {
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

func diffStringSets(left, right []string) (added, removed []string) {
	leftSet := make(map[string]bool, len(left))
	for _, s := range left {
		leftSet[s] = true
	}
	rightSet := make(map[string]bool, len(right))
	for _, s := range right {
		rightSet[s] = true
	}
	for s := range rightSet {
		if !leftSet[s] {
			added = append(added, s)
		}
	}
	for s := range leftSet {
		if !rightSet[s] {
			removed = append(removed, s)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	return added, removed
}
