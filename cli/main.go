package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Subilan/ecd/internal/cli"
	"github.com/Subilan/ecd/internal/config"
	"github.com/Subilan/ecd/internal/dict"
	"github.com/Subilan/ecd/internal/history"
	"github.com/Subilan/ecd/internal/i18n"
	"github.com/Subilan/ecd/internal/search"
	"github.com/Subilan/ecd/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
)

func main() {
	args := parseArgs()
	if err := run(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

type args struct {
	Source     string
	Random     bool
	NoColor    bool
	Query      string
	ConfigPath string
}

func parseArgs() args {
	var a args
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-s", "--source":
			if i+1 < len(os.Args) {
				a.Source = os.Args[i+1]
				i++
			}
		case "-r", "--random":
			a.Random = true
		case "--no-color":
			a.NoColor = true
		case "--config":
			if i+1 < len(os.Args) {
				a.ConfigPath = os.Args[i+1]
				i++
			}
		default:
			if !strings.HasPrefix(os.Args[i], "-") {
				if a.Query != "" {
					a.Query += " "
				}
				a.Query += os.Args[i]
			}
		}
	}
	return a
}

func run(a args) error {
	if a.NoColor || !isatty.IsTerminal(os.Stdout.Fd()) {
		cli.DisableColor()
	}

	// Load config
	configPath := a.ConfigPath
	if configPath == "" {
		configPath = "ecd.toml"
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	config.DBPath = cfg.DBPath
	config.HistoryDB = cfg.LookupDB

	// Validate dictionary DB
	if err := config.ValidateDB(config.DBPath); err != nil {
		isCLI := a.Random || a.Query != "" || !isatty.IsTerminal(os.Stdin.Fd())
		if isCLI {
			return fmt.Errorf("dictionary database not found at %s (configure with --config or edit %s)", config.DBPath, configPath)
		}
		// TUI mode — interactive prompt
		promptForDB(cfg)
		config.DBPath = cfg.DBPath
	}

	dictDB, err := dict.OpenDictDB()
	if err != nil {
		return err
	}
	defer dictDB.Close()

	historyDB, err := history.OpenHistoryDB()
	if err != nil {
		return fmt.Errorf("open history db: %w", err)
	}
	defer historyDB.Close()

	var lastWord string
	ctx := &search.Context{
		DictDB:    dictDB,
		HistoryDB: historyDB,
		LastWord:  &lastWord,
	}

	var srcPtr *string
	if a.Source != "" {
		srcPtr = &a.Source
	}

	switch {
	case a.Random:
		word, err := dictDB.RandomWord(srcPtr)
		if err != nil {
			return err
		}
		fmt.Printf("%s %s\n", cli.C("dim", i18n.T("search.random_word")), cli.C("word", word))
		runQuery(ctx, word, srcPtr)

	case a.Query != "":
		runQuery(ctx, a.Query, srcPtr)

	default:
		if isatty.IsTerminal(os.Stdin.Fd()) {
			model := tui.NewModel(dictDB, historyDB)
			p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
			if _, err := p.Run(); err != nil {
				return err
			}
		} else {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line != "" {
					runQuery(ctx, line, srcPtr)
				}
			}
		}
	}
	return nil
}

func promptForDB(cfg *config.Config) {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Fprintf(os.Stderr, "Dictionary database (ecd.db) not found.\n")
		fmt.Fprintf(os.Stderr, "Enter path to ecd.db: ")
		if !scanner.Scan() {
			fmt.Fprintln(os.Stderr, "\nExiting.")
			os.Exit(1)
		}
		path := strings.TrimSpace(scanner.Text())
		if path == "" {
			continue
		}
		if err := config.ValidateDB(path); err != nil {
			fmt.Fprintf(os.Stderr, "Could not open database: %s\n\n", err)
			continue
		}
		cfg.DBPath = path
		if err := config.SaveConfig(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save config: %s\n", err)
		}
		return
	}
}

func flashcardStatuses(ctx *search.Context, entries []dict.Entry, extra []string) map[string]string {
	words := make([]string, 0, len(entries)+len(extra))
	for _, e := range entries {
		words = append(words, e.Word)
	}
	words = append(words, extra...)
	if ctx.HistoryDB == nil || len(words) == 0 {
		return nil
	}
	return ctx.HistoryDB.GetFlashcardStatuses(words)
}

func runQuery(ctx *search.Context, query string, source *string) {
	result := search.HandleQuery(ctx, query, source)

	switch {
	case result.Entries != nil:
		statuses := flashcardStatuses(ctx, result.Entries, nil)
		cli.PrintResultsEnglish(result.Entries, statuses)
	case result.Chinese != nil:
		words := make([]string, len(result.Chinese))
		for i, r := range result.Chinese {
			words[i] = r.Word
		}
		var statuses map[string]string
		if ctx.HistoryDB != nil {
			statuses = ctx.HistoryDB.GetFlashcardStatuses(words)
		}
		cli.PrintResultsChinese(result.Chinese, statuses)
	case result.Suggestions != nil:
		msg := search.DidYouMeanMessage(search.FormatSuggestions(result.Suggestions))
		fmt.Println(cli.C("label", msg))
	case result.NotFound:
		fmt.Println(search.NotFoundMessage(result.Query))
	}
}
