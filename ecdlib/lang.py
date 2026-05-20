"""I18n strings for the ecd CLI. Switch with .lang en|zh in interactive mode."""

_LANG = "zh"

STRINGS = {
    # --- Source names ---
    "source.collins":      {"en": "Collins", "zh": "柯林斯"},
    "source.oxford":       {"en": "Oxford", "zh": "牛津"},

    # --- Display labels ---
    "label.definition":    {"en": "Def", "zh": "释义"},
    "label.example":       {"en": "Ex", "zh": "例"},
    "label.example_cn":    {"en": "ExTr", "zh": "例译"},
    "label.synonym":       {"en": "Syn", "zh": "同义"},
    "label.antonym":       {"en": "Ant", "zh": "反义"},
    "label.pos":           {"en": "POS", "zh": "词性"},
    "label.rate":          {"en": "Rate", "zh": "评分"},

    # --- Note type labels ---
    "note.usage":          {"en": "Usage", "zh": "用法"},
    "note.drv":            {"en": "Derived", "zh": "派生"},
    "note.regional":       {"en": "Note", "zh": "注"},
    "note.sense":          {"en": "Extra", "zh": "释义补充"},
    "note.quotation":      {"en": "Quote", "zh": "名言"},
    "note.phrase":         {"en": "Phrase", "zh": "短语"},
    "note.general":        {"en": "Note", "zh": "注"},

    # --- Interactive mode ---
    "interactive.welcome": {
        "en": "ecd interactive mode. Type .help for commands, .exit or Ctrl-D to quit.",
        "zh": "ecd 交互模式。输入 .help 查看命令，.exit 或 Ctrl-D 退出。",
    },
    "interactive.help_header":    {"en": "Commands:", "zh": "命令："},
    "interactive.help_exit":      {"en": "Exit", "zh": "退出"},
    "interactive.help_help":      {"en": "Show this help", "zh": "显示帮助信息"},
    "interactive.help_add":       {"en": "Add word to flashcard deck (or last looked-up word)", "zh": "将单词加入闪卡牌组（或上次查询的单词）"},
    "interactive.help_del":       {"en": "Remove word from flashcard deck (or last looked-up word)", "zh": "从闪卡牌组中移除单词（或上次查询的单词）"},
    "interactive.help_auto_add":  {"en": "Toggle auto-add of looked-up words to deck", "zh": "切换自动将查询的单词加入牌组"},
    "interactive.help_review":    {"en": "Review due flashcards", "zh": "复习到期的闪卡"},
    "interactive.help_deck":      {"en": "Show flashcard deck statistics", "zh": "显示闪卡牌组统计"},
    "interactive.help_reset":     {"en": "Reset all flashcard data", "zh": "重置所有闪卡数据"},
    "interactive.help_syn":       {"en": "Show synonyms for a word", "zh": "显示单词的同义词"},
    "interactive.help_random":    {"en": "Show a random word", "zh": "显示随机单词"},
    "interactive.help_ant":       {"en": "Show antonyms for a word", "zh": "显示单词的反义词"},
    "interactive.help_lang":      {"en": "Switch UI language (.lang en|zh)", "zh": "切换界面语言 (.lang en|zh)"},
    "interactive.search_hint":    {"en": "Enter any English or Chinese text to search.", "zh": "输入英文或中文进行查询。"},
    "interactive.unknown_cmd":    {"en": "Unknown command: {cmd}", "zh": "未知命令：{cmd}"},
    "interactive.auto_add":       {"en": "Auto-add: {state}", "zh": "自动添加：{state}"},
    "interactive.lang_switched":  {"en": "Language switched to English.", "zh": "界面语言已切换为中文。"},

    # --- Synonym / Antonym display ---
    "synonym.no_entries":        {"en": "No entries found for '{word}'.", "zh": "未找到 '{word}' 的词条。"},
    "synonym.not_found":         {"en": "No synonyms found for '{word}'.", "zh": "未找到 '{word}' 的同义词。"},
    "synonym.found_groups":      {"en": "Found {count} synonym group(s):", "zh": "找到 {count} 个同义词组："},
    "synonym.usage":             {"en": "Usage: .syn <word>", "zh": "用法：.syn <单词>"},
    "antonym.not_found":         {"en": "No antonyms found for '{word}'.", "zh": "未找到 '{word}' 的反义词。"},
    "antonym.found_groups":      {"en": "Found {count} antonym group(s):", "zh": "找到 {count} 个反义词组："},
    "antonym.usage":             {"en": "Usage: .ant <word>", "zh": "用法：.ant <单词>"},

    # --- Search ---
    "search.no_results":         {"en": "No results for: {query}", "zh": "未找到结果：{query}"},
    "search.did_you_mean":       {"en": "Did you mean: {words}?", "zh": "您是不是要找：{words}？"},
    "search.random_word":        {"en": "Random word:", "zh": "随机单词："},
    "search.no_words":           {"en": "No words found in the database.", "zh": "数据库中未找到任何单词。"},

    # --- Add / Delete ---
    "add.no_word":               {"en": "No word to add. Look up a word first, or use .add <word>.", "zh": "没有可添加的单词。请先查询一个单词，或使用 .add <单词>。"},
    "add.not_found":             {"en": "'{word}' not found in dictionaries.", "zh": "词典中未找到 '{word}'。"},
    "add.anyway":                {"en": "Add anyway? (y/n): ", "zh": "仍然添加？(y/n)："},
    "add.added":                 {"en": "Added '{word}' to flashcard deck.", "zh": "已将 '{word}' 加入闪卡牌组。"},
    "add.already":               {"en": "'{word}' is already in your flashcard deck.", "zh": "'{word}' 已在闪卡牌组中。"},
    "del.removed":               {"en": "Removed '{word}' from flashcard deck.", "zh": "已将 '{word}' 从闪卡牌组中移除。"},
    "del.not_found":             {"en": "'{word}' is not in your flashcard deck.", "zh": "'{word}' 不在闪卡牌组中。"},
    "del.usage":                 {"en": "Usage: .del <word>", "zh": "用法：.del <单词>"},

    # --- Reset ---
    "reset.confirm":             {"en": "Reset all flashcard data? This cannot be undone. (y/n): ", "zh": "重置所有闪卡数据？此操作不可撤销。(y/n)："},
    "reset.done":                {"en": "Flashcard data reset.", "zh": "闪卡数据已重置。"},
    "reset.cancelled":           {"en": "Cancelled.", "zh": "已取消。"},

    # --- Deck stats ---
    "deck.empty":                {"en": "Deck is empty. Use .add to add words.", "zh": "牌组为空。使用 .add 添加单词。"},
    "deck.stats_title":          {"en": "Deck Statistics", "zh": "牌组统计"},
    "deck.total":                {"en": "Total", "zh": "总数"},
    "deck.due":                  {"en": "Due", "zh": "待复习"},
    "deck.new":                  {"en": "New", "zh": "新卡"},
    "deck.mature":               {"en": "Mature", "zh": "成熟"},
    "deck.next":                 {"en": "Next", "zh": "下次复习"},
    "deck.leeches":              {"en": "Leeches", "zh": "难点卡"},
    "deck.avg_ease":             {"en": "Avg ease", "zh": "平均简易度"},

    # --- Review session ---
    "review.no_due":             {"en": "No cards due for review!", "zh": "没有待复习的卡片！"},
    "review.deck_has_overdue":   {"en": "Deck has {total} cards. Next was due {due} ago — run again to review.", "zh": "牌组共 {total} 张卡片。最近到期时间为 {due} 前 — 请重新运行复习。"},
    "review.deck_has_pending":   {"en": "Deck has {total} cards. Next due in {due}.", "zh": "牌组共 {total} 张卡片。下次复习在 {due} 后。"},
    "review.entry_n_of":         {"en": "Entry {idx}/{total}  [{src}]", "zh": "释义 {idx}/{total}  [{src}]"},
    "review.nav_hint_multi":     {"en": "← → switch entry  |  0-3 rate", "zh": "← → 切换释义  |  0-3 评分"},
    "review.nav_hint_single":    {"en": "0-3 rate", "zh": "0-3 评分"},
    "review.press_enter":        {"en": "Press Enter to reveal answer...", "zh": "按回车键显示答案…"},
    "review.cancelled":          {"en": "Review session cancelled.", "zh": "复习会话已取消。"},
    "review.word_not_found":     {"en": "(word not found in dictionary database)", "zh": "（词典数据库中未找到该单词）"},
    "review.again":              {"en": "Again", "zh": "忘记"},
    "review.hard":               {"en": "Hard", "zh": "困难"},
    "review.good":               {"en": "Good", "zh": "良好"},
    "review.easy":               {"en": "Easy", "zh": "简单"},
    "review.invalid_choice":     {"en": "Please enter 0, 1, 2, or 3.", "zh": "请输入 0、1、2 或 3。"},
    "review.invalid_number":     {"en": "Please enter a number (0-3).", "zh": "请输入数字 (0-3)。"},
    "review.complete":           {"en": "Review session complete. {count} card(s) reviewed.", "zh": "复习完成。已复习 {count} 张卡片。"},
    "review.header_card":        {"en": "Card {i}/{total}", "zh": "卡片 {i}/{total}"},
}


def t(key, **kwargs):
    """Return the translated string for the current language."""
    entry = STRINGS.get(key)
    if entry is None:
        return key
    text = entry.get(_LANG, entry.get("en", key))
    if kwargs:
        return text.format(**kwargs)
    return text


def set_lang(lang):
    global _LANG
    _LANG = lang


def get_lang():
    return _LANG
