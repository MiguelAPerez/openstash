package spec

import (
	"fmt"
	"sort"
	"strings"
)

// SchemaIndex is a searchable summary of one component schema.
type SchemaIndex struct {
	Name        string   `json:"name"`
	Type        string   `json:"type,omitempty"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Properties  []string `json:"properties,omitempty"`
	Required    []string `json:"required,omitempty"`
}

// Field is a flattened view of one property within a schema.
type Field struct {
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`
	Format      string `json:"format,omitempty"`
	Ref         string `json:"ref,omitempty"`
	Enum        []any  `json:"enum,omitempty"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
	Items       string `json:"items,omitempty"`
}

// FieldLookup is the result of LookupFieldPath.
type FieldLookup struct {
	Exists    bool     `json:"exists"`
	Resolved  string   `json:"resolved"`
	Missing   string   `json:"missing,omitempty"`
	Type      string   `json:"type,omitempty"`
	Ref       string   `json:"ref,omitempty"`
	Available []string `json:"available,omitempty"`
	Checked   []string `json:"checked,omitempty"`
}

// schemasContainer returns the map of component schemas and whether one was
// found. Handles both OpenAPI 3 (components.schemas) and Swagger 2.0 (definitions).
func schemasContainer(doc map[string]any) (map[string]any, bool) {
	if components, ok := doc["components"].(map[string]any); ok {
		if schemas, ok := components["schemas"].(map[string]any); ok {
			return schemas, true
		}
	}
	if defs, ok := doc["definitions"].(map[string]any); ok {
		return defs, true
	}
	return nil, false
}

// SchemaNames returns sorted names of all component schemas.
func SchemaNames(doc map[string]any) []string {
	schemas, ok := schemasContainer(doc)
	if !ok {
		return nil
	}
	names := make([]string, 0, len(schemas))
	for name := range schemas {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetSchema returns the raw schema node for the given name.
// Returns an error if the name is missing, including the count of available schemas.
func GetSchema(doc map[string]any, name string) (map[string]any, error) {
	schemas, ok := schemasContainer(doc)
	if !ok {
		return nil, fmt.Errorf("spec has no schemas container")
	}
	node, ok := schemas[name].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("schema %q not found (%d schemas available)", name, len(schemas))
	}
	return node, nil
}

// RefName returns the last path segment of a $ref string, with JSON-pointer
// escapes decoded so the true component name is returned.
// E.g. "#/definitions/Foo" -> "Foo", "#/components/schemas/Foo~1Bar" -> "Foo/Bar".
func RefName(ref string) string {
	name := ref
	if idx := strings.LastIndex(ref, "/"); idx >= 0 {
		name = ref[idx+1:]
	}
	name = strings.ReplaceAll(name, "~1", "/")
	name = strings.ReplaceAll(name, "~0", "~")
	return name
}

// ResolveRef resolves an internal JSON-pointer ref ("#/...") within doc.
// Handles ~1 -> / and ~0 -> ~ unescaping. Returns an error for external refs.
func ResolveRef(doc map[string]any, ref string) (map[string]any, error) {
	if !strings.HasPrefix(ref, "#/") && ref != "#" {
		return nil, fmt.Errorf("external refs are not supported: %s", ref)
	}
	if ref == "#" {
		// Whole-document ref.
		return doc, nil
	}
	pointer := strings.TrimPrefix(ref, "#/")
	segments := strings.Split(pointer, "/")
	var current any = doc
	for _, seg := range segments {
		// unescape JSON pointer
		seg = strings.ReplaceAll(seg, "~1", "/")
		seg = strings.ReplaceAll(seg, "~0", "~")
		m, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("ref %q: cannot descend into non-object at segment %q", ref, seg)
		}
		val, exists := m[seg]
		if !exists {
			return nil, fmt.Errorf("ref %q: segment %q not found", ref, seg)
		}
		current = val
	}
	result, ok := current.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("ref %q: resolved to non-object", ref)
	}
	return result, nil
}

// ResolveSchema returns a COPY of node with $ref targets inlined up to depth hops.
// depth counts only $ref dereferences. Descending into properties/items/etc does not consume depth.
// At depth 0 a $ref renders as {"$ref":"<Name>"}.
// When a ref IS expanded, adds "$from":"<Name>" to the expanded object for provenance.
// Guards against cycles; on unresolvable ref renders {"$ref":"<Name>","$error":"unresolved"}.
func ResolveSchema(doc, node map[string]any, depth int) map[string]any {
	seen := make(map[string]bool)
	return resolveNode(doc, node, depth, seen)
}

func resolveNode(doc, node map[string]any, depth int, seen map[string]bool) map[string]any {
	// If node has a $ref, handle it first.
	if ref, ok := node["$ref"].(string); ok {
		name := RefName(ref)
		if depth <= 0 {
			return map[string]any{"$ref": name}
		}
		if seen[ref] {
			// Cycle detected — render as bare ref.
			return map[string]any{"$ref": name}
		}
		target, err := ResolveRef(doc, ref)
		if err != nil {
			return map[string]any{"$ref": name, "$error": "unresolved"}
		}
		// Mark as being expanded.
		newSeen := make(map[string]bool, len(seen)+1)
		for k, v := range seen {
			newSeen[k] = v
		}
		newSeen[ref] = true
		expanded := resolveNode(doc, target, depth-1, newSeen)
		expanded["$from"] = name
		return expanded
	}

	out := make(map[string]any, len(node))
	for k, v := range node {
		switch k {
		case "properties":
			if props, ok := v.(map[string]any); ok {
				resolvedProps := make(map[string]any, len(props))
				for propName, propVal := range props {
					if propMap, ok := propVal.(map[string]any); ok {
						resolvedProps[propName] = resolveNode(doc, propMap, depth, seen)
					} else {
						resolvedProps[propName] = propVal
					}
				}
				out[k] = resolvedProps
			} else {
				out[k] = v
			}
		case "items":
			if items, ok := v.(map[string]any); ok {
				out[k] = resolveNode(doc, items, depth, seen)
			} else {
				out[k] = v
			}
		case "additionalProperties":
			if ap, ok := v.(map[string]any); ok {
				out[k] = resolveNode(doc, ap, depth, seen)
			} else {
				out[k] = v
			}
		case "allOf", "anyOf", "oneOf":
			if arr, ok := v.([]any); ok {
				resolved := make([]any, len(arr))
				for i, item := range arr {
					if im, ok := item.(map[string]any); ok {
						resolved[i] = resolveNode(doc, im, depth, seen)
					} else {
						resolved[i] = item
					}
				}
				out[k] = resolved
			} else {
				out[k] = v
			}
		default:
			out[k] = v
		}
	}
	return out
}

// SchemaFields flattens a schema's properties into sorted Fields,
// marking which appear in the node's required array.
func SchemaFields(node map[string]any) []Field {
	props, _ := node["properties"].(map[string]any)
	if props == nil {
		return nil
	}
	requiredSet := make(map[string]bool)
	for _, r := range stringSliceField(node, "required") {
		requiredSet[r] = true
	}

	fields := make([]Field, 0, len(props))
	for name, raw := range props {
		pm, ok := raw.(map[string]any)
		if !ok {
			fields = append(fields, Field{Name: name, Required: requiredSet[name]})
			continue
		}
		f := Field{
			Name:        name,
			Required:    requiredSet[name],
			Type:        strField(pm, "type"),
			Format:      strField(pm, "format"),
			Description: strField(pm, "description"),
		}
		if ref, ok := pm["$ref"].(string); ok {
			f.Ref = RefName(ref)
		}
		if enum, ok := pm["enum"].([]any); ok {
			f.Enum = enum
		}
		// Handle array items.
		if f.Type == "array" {
			if items, ok := pm["items"].(map[string]any); ok {
				if itemRef, ok := items["$ref"].(string); ok {
					f.Items = RefName(itemRef)
				} else if itemType, ok := items["type"].(string); ok {
					f.Items = itemType
				}
			}
		}
		fields = append(fields, f)
	}
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Name < fields[j].Name
	})
	return fields
}

// LookupFieldPath walks a dotted path like "Schema.field.subfield" within doc.
func LookupFieldPath(doc map[string]any, path string) (*FieldLookup, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("empty schema path")
	}
	segments := strings.Split(path, ".")

	schemas, hasContainer := schemasContainer(doc)
	if !hasContainer {
		return nil, fmt.Errorf("spec has no schemas container")
	}

	rootName := segments[0]
	rootNode, ok := schemas[rootName].(map[string]any)
	if !ok {
		// Missing root schema — return fuzzy suggestions, no error.
		query := strings.ToLower(rootName)
		var suggestions []string
		for name := range schemas {
			if strings.Contains(strings.ToLower(name), query) {
				suggestions = append(suggestions, name)
			}
		}
		sort.Strings(suggestions)
		if len(suggestions) > 10 {
			suggestions = suggestions[:10]
		}
		return &FieldLookup{
			Exists:    false,
			Resolved:  "",
			Missing:   rootName,
			Available: suggestions,
		}, nil
	}

	if len(segments) == 1 {
		// Just the schema name — it exists.
		return &FieldLookup{
			Exists:   true,
			Resolved: rootName,
			Checked:  []string{rootName},
		}, nil
	}

	checked := []string{rootName}
	current := rootNode
	resolved := rootName

	for _, seg := range segments[1:] {
		// Deref any $ref at current level first, guarding against ref cycles
		// (e.g. Foo -> Bar -> Foo) so the loop always terminates.
		seenRefs := make(map[string]bool)
		for {
			ref, hasRef := current["$ref"].(string)
			if !hasRef {
				break
			}
			if seenRefs[ref] {
				break
			}
			seenRefs[ref] = true
			target, err := ResolveRef(doc, ref)
			if err != nil {
				break
			}
			name := RefName(ref)
			if !containsStr(checked, name) {
				checked = append(checked, name)
			}
			current = target
		}
		// Auto-descend through array items.
		if strField(current, "type") == "array" {
			if items, ok := current["items"].(map[string]any); ok {
				// Deref items $ref if present.
				if itemRef, ok := items["$ref"].(string); ok {
					target, err := ResolveRef(doc, itemRef)
					if err == nil {
						name := RefName(itemRef)
						if !containsStr(checked, name) {
							checked = append(checked, name)
						}
						current = target
					} else {
						current = items
					}
				} else {
					current = items
				}
			}
		}

		props, _ := current["properties"].(map[string]any)
		propNode, exists := props[seg]
		if !exists {
			// Field missing — collect available siblings.
			available := sortedKeys(props)
			return &FieldLookup{
				Exists:    false,
				Resolved:  resolved,
				Missing:   seg,
				Available: available,
				Checked:   checked,
			}, nil
		}
		resolved = resolved + "." + seg
		pm, ok := propNode.(map[string]any)
		if !ok {
			// Can't descend further; treat as found.
			return &FieldLookup{
				Exists:   true,
				Resolved: resolved,
				Checked:  checked,
			}, nil
		}
		current = pm
	}

	// Successfully walked the whole path.
	result := &FieldLookup{
		Exists:   true,
		Resolved: resolved,
		Checked:  checked,
		Type:     strField(current, "type"),
	}
	if ref, ok := current["$ref"].(string); ok {
		result.Ref = RefName(ref)
	}
	return result, nil
}

// BuildSchemaIndex builds sorted SchemaIndex entries for every component schema.
func BuildSchemaIndex(doc map[string]any) []SchemaIndex {
	schemas, ok := schemasContainer(doc)
	if !ok {
		return nil
	}
	names := make([]string, 0, len(schemas))
	for name := range schemas {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]SchemaIndex, 0, len(names))
	for _, name := range names {
		node, ok := schemas[name].(map[string]any)
		if !ok {
			out = append(out, SchemaIndex{Name: name})
			continue
		}
		si := SchemaIndex{
			Name:        name,
			Type:        strField(node, "type"),
			Title:       strField(node, "title"),
			Description: strField(node, "description"),
		}
		if props, ok := node["properties"].(map[string]any); ok {
			si.Properties = sortedKeys(props)
		}
		si.Required = stringSliceField(node, "required")
		if len(si.Required) > 0 {
			sort.Strings(si.Required)
		}
		out = append(out, si)
	}
	return out
}

// sortedKeys returns the sorted keys of a map.
func sortedKeys(m map[string]any) []string {
	if m == nil {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
