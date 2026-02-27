package config

import (
	"fmt"
	"os"
	"strconv"
)

// UploadConfig holds settings for the file upload service.
// Read from environment variables; sensible local-dev defaults are provided.
type UploadConfig struct {
	// LocalPath is the base directory on disk where uploaded files are stored.
	// Example: ./uploads  (default, relative to the binary)
	LocalPath string

	// PublicURL is the URL prefix that maps to LocalPath via the static file server.
	// Example: /static/uploads
	PublicURL string

	// MaxFileMB is the maximum allowed upload size in megabytes.
	// Default: 50
	MaxFileMB int64
}

// LoadUploadConfig reads upload settings from environment variables.
// All variables have safe defaults for local development.
func LoadUploadConfig() UploadConfig {
	return UploadConfig{
		LocalPath: getEnv("UPLOAD_LOCAL_PATH", "./uploads"),
		PublicURL: getEnv("UPLOAD_PUBLIC_URL", "/static/uploads"),
		MaxFileMB: parseInt64Env("UPLOAD_MAX_MB", 50),
	}
}

func parseInt64Env(name string, fallback int64) int64 {
	v := os.Getenv(name)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		_ = fmt.Sprintf("WARN: %s is not a valid integer, using default %d", name, fallback)
		return fallback
	}
	return n
}
