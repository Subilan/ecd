# Search Strategy

`HandleQuery()` 实现 4 层搜索调度，从精确到模糊逐层降级。

## 调度流程

```
输入文本
    │
    ▼
┌─────────────────────┐
│ 中文？(CJK 检测)    │──是──► SearchChinese (FTS5)
└─────────┬───────────┘              │
          │ 否                   有结果？
          ▼                          │
┌─────────────────────┐         ┌────▼────┐
│ 1. Exact Match      │         │ 返回结果 │
│    word = ? COLLATE  │         └─────────┘
│    NOCASE            │
└─────────┬───────────┘
     有结果？──是──► 返回结果
          │ 否
          ▼
┌─────────────────────┐
│ 2. Prefix Match     │
│    word LIKE ?%     │
└─────────┬───────────┘
     有结果？
          │
    ┌─────┴─────┐
 唯一词头？   多个词头？
    │             │
    ▼             ▼
 返回结果    返回建议列表（最多 10 个）
              │
              ▼ 无结果
┌─────────────────────┐
│ 3. Fuzzy Match      │
│    首字母索引 +      │
│    bigram 相似度     │
│    (cutoff ≥ 0.75)  │
└─────────┬───────────┘
     有结果？──是──► 返回建议列表（最多 5 个）
          │ 否
          ▼
┌─────────────────────┐
│ 4. FTS5 中文回退    │
│    (拼音/混合输入)   │
└─────────┬───────────┘
     有结果？──是──► 返回结果
          │ 否
          ▼
      返回 NotFound
```

## CJK 检测

通过 `\p{Han}` Unicode 正则检测输入是否包含汉字。纯中文输入直接走 FTS5 路径，跳过英文的 3 层。

## 各层实现细节

### Exact Match（`dict.SearchExact`）

```sql
SELECT ... FROM {dict}_entries WHERE word = ? COLLATE NOCASE
ORDER BY pos, sense_order
```

同时从 `collins` 和 `oxford` 两表查询，合并结果。结果按词典源（Collins 在前）、词性、义项序号排序。

### Prefix Match（`dict.SearchPrefix`）

```sql
SELECT ... FROM {dict}_entries WHERE word LIKE ? COLLATE NOCASE
ORDER BY word, pos, sense_order
```

结果按词头字母序排列。`HandleQuery` 对结果去重统计：
- 仅 1 个唯一词头 → 直接展示结果（如输入 "hello" 匹配到 "hello"）
- 多个唯一词头 → 展示建议列表，最多 10 个（如输入 "ab" 匹配到 abandon, ability, ...）

### Fuzzy Match（`dict.SearchFuzzy`）

1. 取输入首字母作为前缀，从两表拉取所有以该字母开头的词（去重）
2. 对每个候选词计算 bigram 相似度（0.0-1.0），公式：`2 * matches / (len(a) + len(b))`
3. 过滤 score ≥ 0.75 的候选，按分数降序排列
4. 最多返回 5 个结果

### FTS5 Chinese（`dict.SearchChinese`）

```sql
SELECT source, word, cn_definition, en_example, cn_example
FROM entries_fts
WHERE entries_fts MATCH ?
ORDER BY rank
LIMIT 50
```

使用 `unicode61` tokenizer，支持：
- 中文单字/词语匹配
- 拼音输入（如果 tokenizer 能切分）
- 混合中英文输入

结果按 `(source, word, cn_definition)` 去重，每个分组保留下所有例句文本。

## 同义词 / 反义词查询

`SearchSynonyms(ctx, word, source)` / `SearchAntonyms(ctx, word, source)`：

1. 先 exact match 目标词，找到所有词条
2. 从每个词条的 `Synonyms` / `Antonyms` 字段中提取项目
3. 按 `source`（Collins/Oxford）分组，组内去重
4. 返回 `[]SynonymResult`，每个元素包含 source、原词、项目列表

## 源过滤

`-s` 参数或 `source *string` 参数不为空时，所有搜索只查指定词典。中文搜索在 FTS5 结果上应用后置过滤；英文搜索直接在 SQL WHERE 中限制。
