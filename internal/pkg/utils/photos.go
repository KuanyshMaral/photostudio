package utils

import (
	"encoding/json"
	"strings"
)

// PhotosToString converts []string to JSON string (safe for DB)
func PhotosToString(photos []string) string {
	if len(photos) == 0 {
		return "[]"
	}
	data, _ := json.Marshal(photos)
	return string(data)
}

// StringToPhotos converts DB string back to []string
func StringToPhotos(s string) []string {
	if s == "" || s == "[]" {
		return []string{}
	}
	var photos []string
	if err := json.Unmarshal([]byte(s), &photos); err != nil {
		// Fallback: treat as comma-separated if invalid JSON
		return strings.Split(s, ",")
	}
	return photos
}