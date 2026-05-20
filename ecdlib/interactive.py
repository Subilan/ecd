"""Interactive REPL mode."""

from .config import _auto_add, _c, _last_word, _save_history, readline
from .db import add_flashcard, _ensure_history_db
from .flashcards import _add_word_with_check, _delete_word_with_check, print_deck_stats, review_session
from .search import handle_query, random_word, search_english


def _handle_auto_add(query):
    """Handle .auto-add [on|off] command."""
    import ecdlib.config as cfg
    arg = query[10:].strip().lower()
    if arg == "on":
        cfg._auto_add = True
    elif arg == "off":
        cfg._auto_add = False
    else:
        cfg._auto_add = not cfg._auto_add
    state = "ON" if cfg._auto_add else "OFF"
    print(f"Auto-add: {state}")


def interactive(db, source):
    import ecdlib.config as cfg

    print(f"\033]0;ecd\007{_c('word', 'ecd')} interactive mode. Type .help for commands, .exit or Ctrl-D to quit.\n")
    while True:
        try:
            query = input("\n> ").strip()
        except (EOFError, KeyboardInterrupt):
            print()
            break

        if not query:
            continue

        if query.startswith("."):
            cmd = query.lower()
            if cmd in (".exit", ".quit", ".q"):
                break
            elif cmd == ".help":
                print("Commands:")
                print("  .exit .quit .q    Exit")
                print("  .help             Show this help")
                print("  .add [word]       Add word to flashcard deck (or last looked-up word)")
                print("  .del [word]       Remove word from flashcard deck (or last looked-up word)")
                print("  .auto-add [on|off] Toggle auto-add of looked-up words to deck")
                print("  .review           Review due flashcards")
                print("  .deck             Show flashcard deck statistics")
                print("  .reset            Reset all flashcard data")
                print("  .syn [word]       Show synonyms for a word")
                print("  .random           Show a random word")
                print("  .ant [word]       Show antonyms for a word")
                print("Enter any English or Chinese text to search.")
                continue
            elif cmd.startswith(".syn"):
                syn_word = query[5:].strip()
                if not syn_word:
                    syn_word = cfg._last_word
                if not syn_word:
                    print("Usage: .syn <word>")
                    continue
                results = search_english(db, syn_word)
                if not results:
                    print(f"No entries found for '{syn_word}'.")
                    continue
                syn_results = [(r, r.get("synonyms", [])) for r in results if r.get("synonyms")]
                if not syn_results:
                    print(f"No synonyms found for '{syn_word}'.")
                    continue
                source_names = {"collins": "柯林斯", "oxford": "牛津"}
                print(f"\n找到 {len(syn_results)} 个同义词组：")
                for r, syns in syn_results:
                    pos_str = f" {r['pos']}" if r["pos"] else ""
                    src_label = source_names.get(r["source"], r["source"])
                    print(f"\n{_c('source', src_label)}: 【{_c('word', r['word'])}】{_c('dim', pos_str)}")
                    if r["cn_definition"]:
                        print(f"  {_c('label', '释义:')} {r['cn_definition']}")
                    syn_text = _c('dim', ', ').join(_c('word', s) for s in syns)
                    print(f"  {_c('label', '同义:')} {syn_text}")
                print()
                continue
            elif cmd == ".random":
                word = random_word(db)
                if word:
                    print(f"\n{_c('dim', 'Random word:')} {_c('word', word)}")
                    handle_query(db, word, source)
                else:
                    print("No words found in the database.")
                continue
            elif cmd == ".ant" or cmd.startswith(".ant "):
                ant_word = query[5:].strip()
                if not ant_word:
                    ant_word = cfg._last_word
                if not ant_word:
                    print("Usage: .ant <word>")
                    continue
                results = search_english(db, ant_word)
                if not results:
                    print(f"No entries found for '{ant_word}'.")
                    continue
                ant_results = [(r, r.get("antonyms", [])) for r in results if r.get("antonyms")]
                if not ant_results:
                    print(f"No antonyms found for '{ant_word}'.")
                    continue
                source_names = {"collins": "柯林斯", "oxford": "牛津"}
                print(f"\n找到 {len(ant_results)} 个反义词组：")
                for r, ants in ant_results:
                    pos_str = f" {r['pos']}" if r["pos"] else ""
                    src_label = source_names.get(r["source"], r["source"])
                    print(f"\n{_c('source', src_label)}: 【{_c('word', r['word'])}】{_c('dim', pos_str)}")
                    if r["cn_definition"]:
                        print(f"  {_c('label', '释义:')} {r['cn_definition']}")
                    ant_text = _c('dim', ', ').join(_c('word', s) for s in ants)
                    print(f"  {_c('label', '反义:')} {ant_text}")
                print()
                continue
            elif cmd == ".add" or cmd.startswith(".add "):
                word = query[5:].strip()
                if not word:
                    word = cfg._last_word
                if not word:
                    print("No word to add. Look up a word first, or use .add <word>.")
                    continue
                added = _add_word_with_check(db, word)
                if added:
                    cfg._last_word = word
                continue
            elif cmd.startswith(".del "):
                word = query[5:].strip()
                if not word:
                    print("Usage: .del <word>")
                    continue
                _delete_word_with_check(word)
                continue
            elif cmd == ".auto-add" or cmd.startswith(".auto-add "):
                _handle_auto_add(query)
                continue
            elif cmd == ".review":
                review_session(db)
                continue
            elif cmd == ".deck":
                print_deck_stats()
                continue
            elif cmd == ".reset":
                try:
                    answer = input("Reset all flashcard data? This cannot be undone. (y/n): ").strip().lower()
                except (EOFError, KeyboardInterrupt):
                    print()
                    continue
                if answer in ("y", "yes"):
                    conn = _ensure_history_db()
                    conn.execute("DELETE FROM flashcards")
                    conn.commit()
                    conn.close()
                    print("Flashcard data reset.")
                else:
                    print("Cancelled.")
                continue
            else:
                print(f"Unknown command: {query}")
                continue

        handle_query(db, query, source)
        if cfg._auto_add and cfg._last_word:
            if add_flashcard(cfg._last_word):
                print(f"(auto-added '{cfg._last_word}' to deck)")
        if readline:
            _save_history()
