package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
)

// maxJSONBytes caps decoded JSON request bodies; attachments have their own limit.
const maxJSONBytes = 2 << 20

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(io.LimitReader(r.Body, maxJSONBytes)).Decode(v)
}
