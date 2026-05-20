"""Interactive REPL mode."""

from .config import _auto_add, _c, _last_word, _save_history, readline
from .db import add_flashcard, _ensure_history_db
from .flashcards import _add_word_with_check, _delete_word_with_check, print_deck_stats, review_session
from .lang import get_lang, set_lang, t
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
    print(t("interactive.auto_add", state=state))


def interactive(db, source):
    import ecdlib.config as cfg

    print(f"\033]0;ecd\007{_c('word', 'ecd')} {t('interactive.welcome')}\n")
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
                print(t("interactive.help_header"))
                print(f"  .exit .quit .q    {t('interactive.help_exit')}")
                print(f"  .help             {t('interactive.help_help')}")
                print(f"  .add [word]       {t('interactive.help_add')}")
                print(f"  .del [word]       {t('interactive.help_del')}")
                print(f"  .auto-add [on|off] {t('interactive.help_auto_add')}")
                print(f"  .review           {t('interactive.help_review')}")
                print(f"  .deck             {t('interactive.help_deck')}")
                print(f"  .reset            {t('interactive.help_reset')}")
                print(f"  .syn [word]       {t('interactive.help_syn')}")
                print(f"  .random           {t('interactive.help_random')}")
                print(f"  .ant [word]       {t('interactive.help_ant')}")
                print(f"  .lang [en|zh]     {t('interactive.help_lang')}")
                print(t("interactive.search_hint"))
                continue
            elif cmd == ".lang" or cmd.startswith(".lang "):
                lang_arg = query[6:].strip().lower()
                if lang_arg in ("en", "zh"):
                    set_lang(lang_arg)
                    cfg._LANG = lang_arg
                    print(t("interactive.lang_switched"))
                else:
                    print(f"Usage: .lang en|zh  (current: {get_lang()})")
                continue
            elif cmd.startswith(".syn"):
                syn_word = query[5:].strip()
                if not syn_word:
                    syn_word = cfg._last_word
                if not syn_word:
                    print(t("synonym.usage"))
                    continue
                results = search_english(db, syn_word)
                if not results:
                    print(t("synonym.no_entries", word=syn_word))
                    continue
                syn_results = [(r, r.get("synonyms", [])) for r in results if r.get("synonyms")]
                if not syn_results:
                    print(t("synonym.not_found", word=syn_word))
                    continue
                print(f"\n{t('synonym.found_groups', count=len(syn_results))}")
                for r, syns in syn_results:
                    pos_str = f" {r['pos']}" if r["pos"] else ""
                    src_label = t(f"source.{r['source']}")
                    print(f"\n{_c('source', src_label)}: 【{_c('word', r['word'])}】{_c('dim', pos_str)}")
                    if r["cn_definition"]:
                        print(f"  {_c('label', t('label.definition') + ':')} {r['cn_definition']}")
                    syn_text = _c('dim', ', ').join(_c('word', s) for s in syns)
                    print(f"  {_c('label', t('label.synonym') + ':')} {syn_text}")
                print()
                continue
            elif cmd == ".random":
                word = random_word(db)
                if word:
                    print(f"\n{_c('dim', t('search.random_word'))} {_c('word', word)}")
                    handle_query(db, word, source)
                else:
                    print(t("search.no_words"))
                continue
            elif cmd == ".ant" or cmd.startswith(".ant "):
                ant_word = query[5:].strip()
                if not ant_word:
                    ant_word = cfg._last_word
                if not ant_word:
                    print(t("antonym.usage"))
                    continue
                results = search_english(db, ant_word)
                if not results:
                    print(t("synonym.no_entries", word=ant_word))
                    continue
                ant_results = [(r, r.get("antonyms", [])) for r in results if r.get("antonyms")]
                if not ant_results:
                    print(t("antonym.not_found", word=ant_word))
                    continue
                print(f"\n{t('antonym.found_groups', count=len(ant_results))}")
                for r, ants in ant_results:
                    pos_str = f" {r['pos']}" if r["pos"] else ""
                    src_label = t(f"source.{r['source']}")
                    print(f"\n{_c('source', src_label)}: 【{_c('word', r['word'])}】{_c('dim', pos_str)}")
                    if r["cn_definition"]:
                        print(f"  {_c('label', t('label.definition') + ':')} {r['cn_definition']}")
                    ant_text = _c('dim', ', ').join(_c('word', s) for s in ants)
                    print(f"  {_c('label', t('label.antonym') + ':')} {ant_text}")
                print()
                continue
            elif cmd == ".add" or cmd.startswith(".add "):
                word = query[5:].strip()
                if not word:
                    word = cfg._last_word
                if not word:
                    print(t("add.no_word"))
                    continue
                added = _add_word_with_check(db, word)
                if added:
                    cfg._last_word = word
                continue
            elif cmd.startswith(".del "):
                word = query[5:].strip()
                if not word:
                    print(t("del.usage"))
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
                    answer = input(t("reset.confirm")).strip().lower()
                except (EOFError, KeyboardInterrupt):
                    print()
                    continue
                if answer in ("y", "yes"):
                    conn = _ensure_history_db()
                    conn.execute("DELETE FROM flashcards")
                    conn.commit()
                    conn.close()
                    print(t("reset.done"))
                else:
                    print(t("reset.cancelled"))
                continue
            else:
                print(t("interactive.unknown_cmd", cmd=query))
                continue

        handle_query(db, query, source)
        if cfg._auto_add and cfg._last_word:
            if add_flashcard(cfg._last_word):
                print(t("add.added", word=cfg._last_word) + " (auto)")
        if readline:
            _save_history()
