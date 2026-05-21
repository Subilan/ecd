package ai

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func cacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".ecd_ai_cache")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

// orderIndependentCmds lists commands whose arguments are order-independent.
var orderIndependentCmds = map[string]bool{
	"/diff": true,
}

func cacheKey(input string) string {
	normalized := normalizeInput(input)
	hash := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(hash[:]) + ".json"
}

func normalizeInput(input string) string {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return ""
	}

	cmd := strings.ToLower(parts[0])
	cmd = strings.TrimSuffix(cmd, "!")

	if orderIndependentCmds[cmd] && len(parts) > 1 {
		args := make([]string, len(parts)-1)
		copy(args, parts[1:])
		sort.Strings(args)
		return cmd + " " + strings.Join(args, " ")
	}

	return strings.ToLower(parts[0]) + " " + strings.Join(parts[1:], " ")
}

// CacheGet returns the cached response for input, or nil if not found.
func CacheGet(input string) ([]byte, bool) {
	dir, err := cacheDir()
	if err != nil {
		return nil, false
	}
	data, err := os.ReadFile(filepath.Join(dir, cacheKey(input)))
	if err != nil {
		return nil, false
	}
	return data, true
}

// CacheSet stores a response for input.
func CacheSet(input string, response []byte) error {
	dir, err := cacheDir()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, cacheKey(input)), response, 0644)
}
