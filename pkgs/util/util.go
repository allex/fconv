package util

import (
	"os"
	"path/filepath"
	"strconv"
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

// ParseValidPort parses and validates a port number from int or string.
// Returns true and the parsed port number (1-65535) if valid, or false and 0 if invalid.
func ParseValidPort(port any) (bool, int) {
	var n int
	switch v := port.(type) {
	case int:
		n = v
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return false, 0
		}
		var err error
		n, err = strconv.Atoi(s)
		if err != nil {
			return false, 0
		}
	default:
		return false, 0
	}
	if n >= 1 && n <= 65535 {
		return true, n
	}
	return false, 0
}

// IsValidPort returns true if port is in the valid range 1-65535.
// Accepts both int and string types.
func IsValidPort(port any) bool {
	valid, _ := ParseValidPort(port)
	return valid
}
