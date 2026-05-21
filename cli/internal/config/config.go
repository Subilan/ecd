package config

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pelletier/go-toml/v2"

	_ "modernc.org/sqlite"
)

// DB paths — set by main after LoadConfig.
var (
	DBPath    string
	HistoryDB string
)

// LoadedPath is the path to the config file that was loaded (for SaveConfig).
var LoadedPath string

// Config holds the configuration fields.
type Config struct {
	DBPath   string   `toml:"db_path"`
	LookupDB string   `toml:"lookup_db"`
	AI       AIConfig `toml:"ai"`
}

// AIConfig holds AI feature configuration.
type AIConfig struct {
	APIKey       string `toml:"api_key"`
	BaseURL      string `toml:"base_url"`
	Model        string `toml:"model"`
	CacheEnabled bool   `toml:"cache_enabled"`
}

// IsConfigured returns true if AI is ready to use.
func (c AIConfig) IsConfigured() bool {
	return c.APIKey != ""
}

var cjkRE = regexp.MustCompile(`[\x{4e00}-\x{9fff}\x{3400}-\x{4dbf}\x{f900}-\x{faff}]`)

// LoadConfig reads config from path. If the file does not exist, defaults are
// returned and LoadedPath is set so SaveConfig can write a new file later.
func LoadConfig(path string) (*Config, error) {
	LoadedPath = path

	cfg := &Config{
		DBPath:   "ecd.db",
		LookupDB: "~/.ecd_lookup.db",
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg.expand()
			return cfg, nil
		}
		return nil, err
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	// Fill defaults for empty fields (silently, after unmarshal)
	if cfg.DBPath == "" {
		cfg.DBPath = "ecd.db"
	}
	if cfg.LookupDB == "" {
		cfg.LookupDB = "~/.ecd_lookup.db"
	}

	cfg.expand()
	return cfg, nil
}

func (c *Config) expand() {
	c.DBPath = expandHome(c.DBPath)
	c.LookupDB = expandHome(c.LookupDB)
}

// SaveConfig writes cfg to LoadedPath.
func SaveConfig(cfg *Config) error {
	data, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(LoadedPath, data, 0644)
}

// ValidateDB checks that the dictionary database file exists and can be opened.
func ValidateDB(path string) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}
	db, err := sql.Open("sqlite", path+"?mode=ro")
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()
	return db.Ping()
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") || p == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			if p == "~" {
				return home
			}
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

// IsChineseQuery returns true if the text contains CJK characters.
func IsChineseQuery(text string) bool {
	return cjkRE.MatchString(text)
}
