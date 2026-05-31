package out

import (
	"encoding/json"
	"os"
)

func JSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func ErrorJSON(msg string, extra map[string]any) {
	payload := map[string]any{"error": msg}
	for k, v := range extra {
		payload[k] = v
	}
	_ = JSON(payload)
}
