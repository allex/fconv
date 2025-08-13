package util

import (
	"os"
	"path/filepath"
	"strings"
)

// Getenv returns the value of the environment variable named by the key, or def if the variable is empty.
func Getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// EnvBool parses common boolean-like strings from environment variables.
// Accepted truthy: 1, true, yes, on, enable, enabled
// Accepted falsy: 0, false, no, off, disable, disabled
func EnvBool(key string, def bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return def
	}
	switch v {
	case "1", "true", "yes", "on", "enable", "enabled":
		return true
	case "0", "false", "no", "off", "disable", "disabled":
		return false
	default:
		return def
	}
}

// ValidateBearer checks that the Authorization header contains a matching Bearer token.
func ValidateBearer(authHeader string, expected string) bool {
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(authHeader)), "bearer ") {
		return false
	}
	token := strings.TrimSpace(authHeader[len("Bearer "):])
	return token == expected
}

// SafeOutNameWithExt returns the input file name with its extension replaced by the target extension.
func SafeOutNameWithExt(in string, targetExt string) string {
	base := filepath.Base(in)
	cleanExt := strings.ToLower(strings.TrimPrefix(targetExt, "."))
	if cleanExt == "" {
		cleanExt = "bin"
	}
	return strings.TrimSuffix(base, filepath.Ext(base)) + "." + cleanExt
}
