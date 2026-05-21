package i18n

import "fmt"

type Lang string

const (
	LangEN Lang = "en"
	LangZH Lang = "zh"
)

var currentLang Lang = LangZH

type stringEntry map[Lang]string

var table = map[string]stringEntry{
	"source.collins": {LangEN: "Collins", LangZH: "柯林斯"},
	"source.oxford":  {LangEN: "Oxford", LangZH: "牛津"},

	"label.definition": {LangEN: "Def", LangZH: "释义"},
	"label.example":    {LangEN: "Ex", LangZH: "例"},
	"label.example_cn": {LangEN: "ExTr", LangZH: "例译"},
	"label.synonym":    {LangEN: "Syn", LangZH: "同义"},
	"label.antonym":    {LangEN: "Ant", LangZH: "反义"},
	"label.pos":        {LangEN: "POS", LangZH: "词性"},
	"label.rate":       {LangEN: "Rate", LangZH: "评分"},

	"note.usage":     {LangEN: "Usage", LangZH: "用法"},
	"note.drv":       {LangEN: "Derived", LangZH: "派生"},
	"note.regional":  {LangEN: "Note", LangZH: "注"},
	"note.sense":     {LangEN: "Extra", LangZH: "释义补充"},
	"note.quotation": {LangEN: "Quote", LangZH: "名言"},
	"note.phrase":    {LangEN: "Phrase", LangZH: "短语"},
	"note.general":   {LangEN: "Note", LangZH: "注"},

	"idiom.not_found": {LangEN: "No idioms found for '%s'.", LangZH: "未找到 '%s' 的习语。"},
	"idiom.found":     {LangEN: "Found %[1]d idiom(s) for %[2]s", LangZH: "找到 %[2]s 的 %[1]d 个习语"},
	"help.item_idm":   {LangEN: "/idm <word>          Show idioms", LangZH: "/idm <单词>         显示习语"},

	"interactive.welcome": {
		LangEN: "ecd interactive mode. Type .help for commands, .exit or Ctrl-D to quit.",
		LangZH: "ecd 交互模式。输入 .help 查看命令，.exit 或 Ctrl-D 退出。",
	},
	"interactive.help_header":   {LangEN: "Commands:", LangZH: "命令："},
	"interactive.help_exit":     {LangEN: "Exit", LangZH: "退出"},
	"interactive.help_help":     {LangEN: "Show this help", LangZH: "显示帮助信息"},
	"interactive.help_add":      {LangEN: "Add word to flashcard deck (or last looked-up word)", LangZH: "将单词加入闪卡牌组（或上次查询的单词）"},
	"interactive.help_del":      {LangEN: "Remove word from flashcard deck (or last looked-up word)", LangZH: "从闪卡牌组中移除单词（或上次查询的单词）"},
	"interactive.help_auto_add": {LangEN: "Toggle auto-add of looked-up words to deck", LangZH: "切换自动将查询的单词加入牌组"},
	"interactive.help_review":   {LangEN: "Review due flashcards", LangZH: "复习到期的闪卡"},
	"interactive.help_deck":     {LangEN: "Show flashcard deck statistics", LangZH: "显示闪卡牌组统计"},
	"interactive.help_reset":    {LangEN: "Reset all flashcard data", LangZH: "重置所有闪卡数据"},
	"interactive.help_syn":      {LangEN: "Show synonyms for a word", LangZH: "显示单词的同义词"},
	"interactive.help_random":   {LangEN: "Show a random word", LangZH: "显示随机单词"},
	"interactive.help_ant":      {LangEN: "Show antonyms for a word", LangZH: "显示单词的反义词"},
	"interactive.help_lang":     {LangEN: "Switch UI language (.lang en|zh)", LangZH: "切换界面语言 (.lang en|zh)"},
	"interactive.search_hint":   {LangEN: "Enter any English or Chinese text to search.", LangZH: "输入英文或中文进行查询。"},
	"interactive.unknown_cmd":   {LangEN: "Unknown command: %s", LangZH: "未知命令：%s"},
	"interactive.auto_add":      {LangEN: "Auto-add: %s", LangZH: "自动添加：%s"},
	"interactive.lang_switched": {LangEN: "Language switched to English.", LangZH: "界面语言已切换为中文。"},

	"synonym.no_entries": {LangEN: "No entries found for '%s'.", LangZH: "未找到 '%s' 的词条。"},
	"synonym.not_found":  {LangEN: "No synonyms found for '%s'.", LangZH: "未找到 '%s' 的同义词。"},
	"synonym.found":      {LangEN: "Found %[1]d synonym(s) for %[2]s", LangZH: "找到 %[2]s 的 %[1]d 个同义词"},
	"synonym.usage":      {LangEN: "Usage: /syn <word>", LangZH: "用法：/syn <单词>"},
	"antonym.not_found":  {LangEN: "No antonyms found for '%s'.", LangZH: "未找到 '%s' 的反义词。"},
	"antonym.found":      {LangEN: "Found %[1]d antonym(s) for %[2]s", LangZH: "找到 %[2]s 的 %[1]d 个反义词"},
	"antonym.usage":      {LangEN: "Usage: /ant <word>", LangZH: "用法：/ant <单词>"},

	"search.no_results":   {LangEN: "No results for: %s", LangZH: "未找到结果：%s"},
	"search.did_you_mean": {LangEN: "Did you mean: %s?", LangZH: "您是不是要找：%s？"},
	"search.random_word":  {LangEN: "Random word:", LangZH: "随机单词："},
	"search.no_words":     {LangEN: "No words found in the database.", LangZH: "数据库中未找到任何单词。"},

	"add.no_word":   {LangEN: "No word to add. Look up a word first, or use .add <word>.", LangZH: "没有可添加的单词。请先查询一个单词，或使用 .add <单词>。"},
	"add.not_found": {LangEN: "'%s' not found in dictionaries.", LangZH: "词典中未找到 '%s'。"},
	"add.anyway":    {LangEN: "Add anyway? (y/n): ", LangZH: "仍然添加？(y/n)："},
	"add.added":     {LangEN: "Added '%s' to flashcard deck.", LangZH: "已将 '%s' 加入闪卡牌组。"},
	"add.already":   {LangEN: "'%s' is already in your flashcard deck.", LangZH: "'%s' 已在闪卡牌组中。"},
	"del.removed":   {LangEN: "Removed '%s' from flashcard deck.", LangZH: "已将 '%s' 从闪卡牌组中移除。"},
	"del.not_found": {LangEN: "'%s' is not in your flashcard deck.", LangZH: "'%s' 不在闪卡牌组中。"},
	"del.usage":     {LangEN: "Usage: .del <word>", LangZH: "用法：.del <单词>"},

	"reset.confirm":   {LangEN: "Reset all flashcard data? This cannot be undone. (y/n): ", LangZH: "重置所有闪卡数据？此操作不可撤销。(y/n)："},
	"reset.done":      {LangEN: "Flashcard data reset.", LangZH: "闪卡数据已重置。"},
	"reset.cancelled": {LangEN: "Cancelled.", LangZH: "已取消。"},

	"deck.empty":       {LangEN: "Deck is empty. Use .add to add words.", LangZH: "牌组为空。使用 .add 添加单词。"},
	"deck.stats_title": {LangEN: "Deck Statistics", LangZH: "牌组统计"},
	"deck.total":       {LangEN: "Total", LangZH: "总数"},
	"deck.due":         {LangEN: "Due", LangZH: "待复习"},
	"deck.new":         {LangEN: "New", LangZH: "新卡"},
	"deck.mature":      {LangEN: "Mature", LangZH: "成熟"},
	"deck.next":        {LangEN: "Next", LangZH: "下次复习"},
	"deck.leeches":     {LangEN: "Leeches", LangZH: "难点卡"},
	"deck.avg_ease":    {LangEN: "Avg ease", LangZH: "平均简易度"},

	"review.no_due":           {LangEN: "No cards due for review!", LangZH: "没有待复习的卡片！"},
	"review.deck_has_overdue": {LangEN: "Deck has %d cards. Next was due %s ago — run again to review.", LangZH: "牌组共 %d 张卡片。最近到期时间为 %s 前 — 请重新运行复习。"},
	"review.deck_has_pending": {LangEN: "Deck has %d cards. Next due in %s.", LangZH: "牌组共 %d 张卡片。下次复习在 %s 后。"},
	"review.entry_n_of":       {LangEN: "Entry %d/%d  [%s]", LangZH: "释义 %d/%d  [%s]"},
	"review.switch_entry":     {LangEN: "← → switch entry", LangZH: "← → 切换释义"},
	"review.press_enter":      {LangEN: "Press Enter to reveal answer...", LangZH: "按回车键显示答案…"},
	"review.cancelled":        {LangEN: "Review session cancelled.", LangZH: "复习会话已取消。"},
	"review.word_not_found":   {LangEN: "(word not found in dictionary database)", LangZH: "（词典数据库中未找到该单词）"},
	"review.again":            {LangEN: "Again", LangZH: "忘记"},
	"review.hard":             {LangEN: "Hard", LangZH: "困难"},
	"review.good":             {LangEN: "Good", LangZH: "良好"},
	"review.easy":             {LangEN: "Easy", LangZH: "简单"},
	"review.invalid_choice":   {LangEN: "Please enter 0, 1, 2, or 3.", LangZH: "请输入 0、1、2 或 3。"},
	"review.invalid_number":   {LangEN: "Please enter a number (0-3).", LangZH: "请输入数字 (0-3)。"},
	"review.complete":         {LangEN: "Review session complete. %d card(s) reviewed.", LangZH: "复习完成。已复习 %d 张卡片。"},
	"review.header_card":      {LangEN: "Card %d/%d", LangZH: "卡片 %d/%d"},

	"footer.hint": {LangEN: "/help for help, /quit to quit", LangZH: "/help 查看帮助，/quit 退出"},

	"help.title":                {LangEN: "Commands", LangZH: "命令"},
	"help.desc":                 {LangEN: "Type any English or Chinese text to search. Commands start with / and are typed in the search bar.", LangZH: "输入英文或中文进行查询。命令以 / 开头，在搜索栏中输入。"},
	"help.section_search":       {LangEN: "Search", LangZH: "搜索"},
	"help.section_flashcards":   {LangEN: "Flashcards", LangZH: "闪卡"},
	"help.section_general":      {LangEN: "General", LangZH: "通用"},
	"help.item_random":          {LangEN: "/random               Show a random word", LangZH: "/random               显示随机单词"},
	"help.item_syn":             {LangEN: "/syn [word]           Show synonyms", LangZH: "/syn [单词]           显示同义词"},
	"help.item_ant":             {LangEN: "/ant [word]           Show antonyms", LangZH: "/ant [单词]           显示反义词"},
	"help.item_add":             {LangEN: "/add [word]           Add word to deck", LangZH: "/add [单词]           将单词加入牌组"},
	"help.item_del":             {LangEN: "/del <word>           Remove word from deck", LangZH: "/del <单词>           从牌组中移除单词"},
	"help.item_auto_add":        {LangEN: "/auto-add [on|off]    Toggle auto-add", LangZH: "/auto-add [on|off]    切换自动添加"},
	"help.item_review":          {LangEN: "/review               Start review session", LangZH: "/review               开始复习"},
	"help.item_deck":            {LangEN: "/deck                 Deck statistics", LangZH: "/deck                 牌组统计"},
	"help.item_reset":           {LangEN: "/reset                Reset all cards", LangZH: "/reset                重置所有卡片"},
	"help.item_help":            {LangEN: "/help                 Show this help", LangZH: "/help                 显示帮助"},
	"help.item_lang":            {LangEN: "/lang [en|zh]         Switch language", LangZH: "/lang [en|zh]         切换语言"},
	"help.item_exit":            {LangEN: "/exit, /quit, /q      Quit", LangZH: "/exit, /quit, /q      退出"},
	"help.item_ctrlc":           {LangEN: "Ctrl+C                Force quit", LangZH: "Ctrl+C                强制退出"},
	"help.item_esc":             {LangEN: "Esc                   Clear input / go back", LangZH: "Esc                   清空输入 / 返回"},
	"common.press_any_key":      {LangEN: "Press any key to return", LangZH: "按任意键返回"},
	"common.initializing":       {LangEN: "Initializing...", LangZH: "初始化中…"},
	"common.search_placeholder": {LangEN: "Search English or Chinese...", LangZH: "输入英文或中文进行查询…"},
	"interactive.lang_usage":    {LangEN: "Usage: /lang en|zh  (current: %s)", LangZH: "用法：/lang en|zh  (当前：%s)"},
}

func T(key string, args ...interface{}) string {
	return TWithLang(currentLang, key, args...)
}

func TWithLang(lang Lang, key string, args ...interface{}) string {
	entry, ok := table[key]
	if !ok {
		return key
	}
	text, ok := entry[lang]
	if !ok {
		text = entry[LangEN]
	}
	if len(args) > 0 {
		return fmt.Sprintf(text, args...)
	}
	return text
}

func SetLang(l Lang) { currentLang = l }
func GetLang() Lang  { return currentLang }
