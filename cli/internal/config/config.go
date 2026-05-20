package config

import (
	"os"
	"path/filepath"
	"regexp"

	_ "modernc.org/sqlite" // register sqlite driver
)

// DB paths
var (
	DBPath    string
	HistoryDB string
)

var cjkRE = regexp.MustCompile(`[\x{4e00}-\x{9fff}\x{3400}-\x{4dbf}\x{f900}-\x{faff}]`)

func init() {
	// Default DB path: ../ecd.db relative to the binary (cli/ directory)
	DBPath = os.Getenv("ECD_DB_PATH")
	if DBPath == "" {
		// Find ecd.db relative to the binary or cwd
		if _, err := os.Stat("ecd.db"); err == nil {
			DBPath = "ecd.db"
		} else if _, err := os.Stat(filepath.Join("..", "ecd.db")); err == nil {
			DBPath = filepath.Join("..", "ecd.db")
		}
	}

	home, err := os.UserHomeDir()
	if err == nil {
		HistoryDB = filepath.Join(home, ".ecd_lookup.db")
	}
	if HistoryDB == "" {
		HistoryDB = ".ecd_lookup.db"
	}
}

func IsChineseQuery(text string) bool {
	return cjkRE.MatchString(text)
}
