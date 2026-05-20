# Schema Design

每个词典有两张表：`{dict}_entries` 存储词条释义，`{dict}_examples` 存储例句（1:N 关系）。另有三张全局表用于全文搜索、同义词和反义词。

## `collins_entries` / `oxford_entries`

| Column | Type | Nullable | Description |
|--------|------|----------|-------------|
| `id` | INTEGER PK | — | 自增 ID |
| `word` | TEXT | NOT NULL | 词头，如 `"abject"`、`"water"`、`"went"` |
| `pos` | TEXT | NULLABLE | 词性。Collins: `"N-COUNT"`、`"ADJ-GRADED"`、`"VERB"`；Oxford: `"noun [U]"`、`"verb [I]"`、`"IDM phrase"`。纯交叉引用条目为 NULL |
| `cn_definition` | TEXT | NULLABLE | 中文释义。交叉引用条目存储描述文本（如 `"past tense of go"`） |
| `cross_ref` | TEXT | NULLABLE | 交叉引用目标词。Oxford 从 `xr-g > xr` 的 `<a>` 提取；Collins 从 `<a class="see">` 或 caption 文本提取。普通条目为 NULL |
| `sense_order` | INTEGER | NOT NULL DEFAULT 1 | 同一 (word, pos) 组内的序号，从 1 开始。交叉引用条目固定为 1 |
| `pronunciation` | TEXT | NULLABLE | JSON 数组格式的 IPA 字符串，如 `["rɪˈfjuːz"]` 或 `["ˈrekɔːd","ˈrekərd"]`（UK+US）。不同 POS 的发音通过各自的 `pos` 字段区分 |
| `extra_notes` | TEXT | NULLABLE | JSON 数组 `{"type": "...", "en": "...", "cn": "..."}`。Collins 从 `<figure class="note type-*">` 提取。Oxford 保留字段 |

**约束与索引：**
- `UNIQUE (word, pos, sense_order)` — 覆盖主要查询模式的复合唯一约束
- `CREATE INDEX idx_{dict}_entries_word ON {dict}_entries(word)` — 前缀/精确查询
- `CREATE INDEX idx_{dict}_entries_cross_ref ON {dict}_entries(cross_ref)` — 反向追踪引用

### 为何用 (word, pos, sense_order) 而非 (word, pos)

同一词性可能有多条释义（如 Oxford "beauty" noun [U] 包含 "美，美丽" 和 "魅力" 两个 `n-g` 块）。`sense_order` 用于区分。

### 交叉引用处理

约 14,700 条 Oxford 条目和 10,100 条 Collins 条目标纯交叉引用（如 "went" → "go"、"mice" → "mouse"）。这些条目 `pos = NULL`，`cn_definition` 为描述文本，`cross_ref` 为规范词，在 examples 表中无记录。

同时具有释义和交叉引用标签的混合条目（如 Oxford "better" 标注 "comparative of good" 但仍包含完整释义）按普通条目处理，即 `cross_ref = NULL`。

## `collins_examples` / `oxford_examples`

| Column | Type | Nullable | Description |
|--------|------|----------|-------------|
| `id` | INTEGER PK | — | 自增 ID |
| `entry_id` | INTEGER | NOT NULL | FK → `{dict}_entries(id)` |
| `en_example` | TEXT | NULLABLE | 英语例句 |
| `cn_example` | TEXT | NULLABLE | 例句中文翻译 |
| `example_order` | INTEGER | NOT NULL DEFAULT 1 | 同一条目内例句序号 |

**约束与索引：**
- `FOREIGN KEY (entry_id) REFERENCES {dict}_entries(id) ON DELETE CASCADE`
- `CREATE INDEX idx_{dict}_examples_entry ON {dict}_examples(entry_id)`

## `entries_fts`（全文搜索）

用于中文反查和模糊搜索：

```sql
CREATE VIRTUAL TABLE entries_fts USING fts5(
    source,         -- 'collins' 或 'oxford'
    word,
    cn_definition,
    en_example,
    cn_example,
    tokenize='unicode61'
);
```

排除交叉引用条目（`cross_ref IS NULL OR cross_ref = ''`），从 entries 与 examples 的 LEFT JOIN 中填充。

## `synonyms`（同义词）

```sql
CREATE TABLE IF NOT EXISTS synonyms (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source TEXT NOT NULL CHECK(source IN ('collins', 'oxford')),
    entry_id INTEGER NOT NULL,
    synonym_word TEXT NOT NULL
);
CREATE INDEX idx_synonyms_entry ON synonyms(source, entry_id);
CREATE INDEX idx_synonyms_word ON synonyms(synonym_word);
```

Collins：从 `<div class="synonym">` 块提取。Oxford：从包含 `.symbols-synsym`（SYN）标记或 `.z_xr "synonyms at"` 的 `.xr-g` 元素提取。

## `antonyms`（反义词）

```sql
CREATE TABLE IF NOT EXISTS antonyms (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source TEXT NOT NULL CHECK(source IN ('collins', 'oxford')),
    entry_id INTEGER NOT NULL,
    antonym_word TEXT NOT NULL
);
CREATE INDEX idx_antonyms_entry ON antonyms(source, entry_id);
CREATE INDEX idx_antonyms_word ON antonyms(antonym_word);
```

主要来自 Oxford 包含 `.symbols-oppsym`（OPP）标记的 `.xr-g` 元素。

## 分表设计的原因

- Collins 使用 COBUILD 语法编码，Oxford 使用传统标签搭配语法标记——POS 体系不同
- HTML 结构差异显著——每种词典单独的解析逻辑更清晰
- 查询单个词典时无需每次加 WHERE 过滤
- 列布局一致，UNION 查询可直接合并两个词典的结果
