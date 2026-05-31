package spec

// OperationIndex is a searchable slice of an OpenAPI operation.
type OperationIndex struct {
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	OperationID string   `json:"operationId,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// Meta describes a stored spec entry.
type Meta struct {
	Key       string `json:"key"`
	Version   string `json:"version"`
	Source    string `json:"source"`
	Endpoint  string `json:"endpoint,omitempty"`
	FetchedAt string `json:"fetchedAt"`
}

// Ref identifies key@version.
type Ref struct {
	Key     string
	Version string
}
