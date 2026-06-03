package search

import (
	"strings"

	"github.com/MiguelAPerez/openstash/internal/spec"
)

// Hit is a ranked search result.
type Hit struct {
	Score     int                 `json:"score"`
	Operation spec.OperationIndex `json:"operation"`
}

// SchemaHit is a ranked schema search result.
type SchemaHit struct {
	Score  int              `json:"score"`
	Schema spec.SchemaIndex `json:"schema"`
}

// SearchSchemas matches component schemas against a free-text query.
// Scoring: exact name match highest, name-contains next, property-name match,
// title/description contains, then token matches. When query is empty, all
// entries are returned (score 1) up to limit, mirroring Query behavior.
func SearchSchemas(idx []spec.SchemaIndex, query string, limit int) []SchemaHit {
	query = strings.TrimSpace(strings.ToLower(query))
	if limit <= 0 {
		limit = 5
	}

	tokens := tokenize(query)
	var hits []SchemaHit

	for _, s := range idx {
		score := scoreSchema(s, query, tokens)
		if query != "" && score == 0 {
			continue
		}
		if query == "" {
			score = 1
		}
		hits = append(hits, SchemaHit{Score: score, Schema: s})
	}

	sortSchemaHits(hits)
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}

func scoreSchema(s spec.SchemaIndex, query string, tokens []string) int {
	score := 0
	nameLower := strings.ToLower(s.Name)
	titleLower := strings.ToLower(s.Title)
	descLower := strings.ToLower(s.Description)

	if query != "" {
		if nameLower == query {
			score += 100
		}
		if strings.Contains(nameLower, query) {
			score += 50
		}
		if strings.Contains(titleLower, query) {
			score += 35
		}
		if strings.Contains(descLower, query) {
			score += 15
		}
		for _, prop := range s.Properties {
			if strings.ToLower(prop) == query {
				score += 40
			} else if strings.Contains(strings.ToLower(prop), query) {
				score += 20
			}
		}
	}

	for _, tok := range tokens {
		if len(tok) < 2 {
			continue
		}
		if strings.Contains(nameLower, tok) {
			score += 12
		}
		if strings.Contains(titleLower, tok) {
			score += 10
		}
		if strings.Contains(descLower, tok) {
			score += 5
		}
		for _, prop := range s.Properties {
			if strings.Contains(strings.ToLower(prop), tok) {
				score += 8
			}
		}
	}

	return score
}

func sortSchemaHits(hits []SchemaHit) {
	for i := 0; i < len(hits); i++ {
		for j := i + 1; j < len(hits); j++ {
			if hits[j].Score > hits[i].Score {
				hits[i], hits[j] = hits[j], hits[i]
			}
		}
	}
}

// Query matches operations against a free-text query with optional filters.
func Query(index []spec.OperationIndex, query string, limit int, pathPrefix, method string) []Hit {
	query = strings.TrimSpace(strings.ToLower(query))
	method = strings.ToUpper(strings.TrimSpace(method))
	if limit <= 0 {
		limit = 5
	}

	tokens := tokenize(query)
	var hits []Hit

	for _, op := range index {
		if pathPrefix != "" && !strings.HasPrefix(op.Path, pathPrefix) {
			continue
		}
		if method != "" && op.Method != method {
			continue
		}

		score := scoreOperation(op, query, tokens)
		if query != "" && score == 0 {
			continue
		}
		if query == "" {
			score = 1
		}
		hits = append(hits, Hit{Score: score, Operation: op})
	}

	sortHits(hits)
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}

func tokenize(q string) []string {
	if q == "" {
		return nil
	}
	repl := strings.NewReplacer("/", " ", "-", " ", "_", " ")
	parts := strings.Fields(repl.Replace(q))
	return parts
}

func scoreOperation(op spec.OperationIndex, query string, tokens []string) int {
	score := 0
	pathLower := strings.ToLower(op.Path)
	summaryLower := strings.ToLower(op.Summary)
	descLower := strings.ToLower(op.Description)
	opIDLower := strings.ToLower(op.OperationID)

	if query != "" {
		if pathLower == query {
			score += 100
		}
		if opIDLower == query {
			score += 90
		}
		if strings.Contains(pathLower, query) {
			score += 40
		}
		if strings.Contains(summaryLower, query) {
			score += 35
		}
		if strings.Contains(opIDLower, query) {
			score += 30
		}
		if strings.Contains(descLower, query) {
			score += 15
		}
	}

	for _, tag := range op.Tags {
		tagLower := strings.ToLower(tag)
		if query != "" && strings.Contains(tagLower, query) {
			score += 25
		}
	}

	for _, tok := range tokens {
		if len(tok) < 2 {
			continue
		}
		if strings.Contains(pathLower, tok) {
			score += 12
		}
		if strings.Contains(summaryLower, tok) {
			score += 10
		}
		if strings.Contains(opIDLower, tok) {
			score += 8
		}
		for _, tag := range op.Tags {
			if strings.Contains(strings.ToLower(tag), tok) {
				score += 8
			}
		}
	}

	return score
}

func sortHits(hits []Hit) {
	for i := 0; i < len(hits); i++ {
		for j := i + 1; j < len(hits); j++ {
			if hits[j].Score > hits[i].Score {
				hits[i], hits[j] = hits[j], hits[i]
			}
		}
	}
}
