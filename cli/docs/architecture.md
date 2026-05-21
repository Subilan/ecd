# Architecture

## 整体架构

CLI 是单二进制文件，零运行时依赖（纯 Go SQLite via `modernc.org/sqlite`）。

```
main.go ──→ flag 解析 ──→ CLI 模式 (cli/)    ──→ 一次查询 + 输出 + 退出
                        ──→ TUI 模式 (tui/)   ──→ Bubble Tea 交互界面
```

模式选择逻辑：当提供了命令行参数或 stdin 为管道时走 CLI 模式；当无参数且 stdin 为 TTY 时启动 TUI。

## 数据流

```
ecd.db (只读, ~80MB)              ~/.ecd_lookup.db (读写)
┌─────────────────────┐         ┌──────────────────────┐
│ collins_entries     │         │ lookup_history       │
│ collins_examples    │         │ flashcards           │
│ oxford_entries      │         └──────────────────────┘
│ oxford_examples     │                  ▲
│ entries_fts (FTS5)  │                  │
│ synonyms / antonyms │                  │
└─────────────────────┘                  │
         ▲                               │
         │     ┌──────────────┐          │
         └─────│  dict.DB     │──────────┘
               │  (只读查询)   │    history.DB (读写)
               └──────┬───────┘
                      │
              ┌───────▼────────┐
              │  search 调度器  │
              └───────┬────────┘
                      │
              ┌───────▼────────┐
              │  tui / cli     │
              │  (展示层)       │
              └────────────────┘
```

两个数据库通过 `config` 包解析路径：
- `ecd.db`：从可执行文件同目录或环境变量 `ECD_DB_PATH` 定位
- `~/.ecd_lookup.db`：固定在用户 home 目录

## 包职责

### `config/`
- `DBPath`：词典数据库路径，默认 `../ecd.db`（相对可执行文件），可通过 `ECD_DB_PATH` 覆盖
- `HistoryDB`：历史数据库路径，固定 `~/.ecd_lookup.db`
- `IsChineseQuery(text) bool`：通过 `\p{Han}` 正则检测中文输入
- SQLite 驱动注册（`modernc.org/sqlite`，无 CGO）

### `dict/`
- `models.go`：核心数据结构 — `Entry`（义项）、`Example`（例句对）、`Note`（额外注释，JSON）、`ChineseResult`（FTS5 聚合结果）
- `db.go`：`DB` 封装只读 SQLite 连接，提供：
  - `SearchExact` / `SearchPrefix` — `COLLATE NOCASE` 精确/前缀匹配
  - `SearchFuzzy` — 首字母索引 + bigram 相似度（≥0.75），最多返回 5 个结果
  - `SearchChinese` — FTS5 MATCH，按 rank 排序，最多 50 条，按 `(source, word, cn_definition)` 去重
  - `RandomWord` — `ORDER BY RANDOM()` 随机取词
- 每次查询同时从 `collins` 和 `oxford` 两表 UNION 风格拉取，逐条附带例句、同义词、反义词

### `search/`
- `Context`：持有 `DictDB`、`HistoryDB`、`LastWord` 指针，贯穿搜索生命周期
- `HandleQuery`：4 层调度（详见 [search.md](./search.md)）
- `SearchSynonyms` / `SearchAntonyms`：先 exact match 找到词条，再按 source 分组合并同/反义词，去重后返回

### `history/`
- `DB`：封装 `~/.ecd_lookup.db` 的读写连接（`SetMaxOpenConns(1)`）
- 自动建表：`lookup_history`（查词统计）、`flashcards`（SM-2 状态）
- 详见 [flashcards.md](./flashcards.md)

### `sm2/`
- 纯数学包，无外部依赖
- `SM2Params`：`Repetitions`、`IntervalDays`、`EaseFactor`
- `Schedule(outcome, params)`：根据 0-3 评分计算新的调度参数
- 详见 [flashcards.md](./flashcards.md)

### `cli/`
- ANSI 着色输出函数（`C(name, text)`）
- `PrintResultsEnglish` / `PrintResultsChinese`：格式化打印搜索结果
- 颜色支持通过 `--no-color` 或非 TTY stdout 自动关闭

### `tui/`
- Bubble Tea 界面，详见 [tui.md](./tui.md)

### `i18n/`
- 中/英文字符串表，详见 [i18n.md](./i18n.md)
