package search

import (
	"strings"

	"github.com/MiguelAPerez/openstash/internal/spec"
)

// Hit is a ranked search result.
type Hit struct {
	Score       int                `json:"score"`
	Operation   spec.OperationIndex `json:"operation"`
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
