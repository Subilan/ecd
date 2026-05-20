# Verification

构建数据库后的验证命令及预期结果。

## 行数验证

```bash
DB=../ecd.db

# 基本条目数
sqlite3 $DB "SELECT 'collins_entries', COUNT(*) FROM collins_entries"
# → 约 81k（含约 3.5k DRV 条目）

sqlite3 $DB "SELECT 'oxford_entries', COUNT(*) FROM oxford_entries"
# → 约 88k

# 交叉引用
sqlite3 $DB "SELECT COUNT(*) FROM collins_entries WHERE cross_ref IS NOT NULL AND cross_ref != ''"
# → 约 230

sqlite3 $DB "SELECT COUNT(*) FROM oxford_entries WHERE cross_ref IS NOT NULL AND cross_ref != ''"
# → 约 3,300（含 .derived 重定向）

# DRV 条目（Collins 派生形式）
sqlite3 $DB "SELECT COUNT(*) FROM collins_entries WHERE pos = 'DRV'"
# → 约 3,500

# 同义词（两本词典均有）
sqlite3 $DB "SELECT source, COUNT(*) FROM synonyms GROUP BY source"
# → collins 约 85k，oxford 约 7,200

# 反义词
sqlite3 $DB "SELECT source, COUNT(*) FROM antonyms GROUP BY source"
# → oxford 约 1,500

# extra_notes
sqlite3 $DB "SELECT COUNT(*) FROM collins_entries WHERE extra_notes IS NOT NULL"
# → 约 10,000

# 发音
sqlite3 $DB "SELECT COUNT(*) FROM collins_entries WHERE pronunciation IS NOT NULL"
# → 约 60k
```

## 抽查

### 常规条目

```bash
# Collins
sqlite3 $DB "SELECT word, pos, cn_definition FROM collins_entries WHERE word='abject'"

# Oxford（含例句）
sqlite3 $DB "SELECT e.word, e.pos, e.cn_definition, x.en_example, x.cn_example FROM oxford_entries e LEFT JOIN oxford_examples x ON e.id=x.entry_id WHERE e.word='beauty'"

# Oxford 模式 4
sqlite3 $DB "SELECT word, pos, cn_definition FROM oxford_entries WHERE word='incantation'"

# Oxford 习语（模式 4b）
sqlite3 $DB "SELECT word, pos, cn_definition FROM oxford_entries WHERE word='aback'"

# Collins extra_notes
sqlite3 $DB "SELECT word, extra_notes FROM collins_entries WHERE word='prefer'"
```

### 交叉引用

```bash
sqlite3 $DB "SELECT word, cn_definition, cross_ref FROM oxford_entries WHERE word='went'"
# → went | past tense of go | go

sqlite3 $DB "SELECT word, cn_definition, cross_ref FROM collins_entries WHERE word='mice'"
# → mice | Mice is the plural of mouse.（mouse 的复数）| mouse

sqlite3 $DB "SELECT word, cn_definition, cross_ref FROM oxford_entries WHERE word='abasement'"
# → abasement | See 见词条 | abase（派生形式交叉引用）
```

### 发音

```bash
sqlite3 $DB "SELECT word, pos, pronunciation FROM oxford_entries WHERE word='record'"
# → 名词条目 ["ˈrekɔːd","ˈrekərd"]；动词条目 ["rɪˈkɔːd","rɪˈkɔːrd"]
```

## CLI 测试

```bash
./ecd hello
./ecd 水
./ecd -s oxford beauty
./ecd went
```

## 查询历史

```bash
sqlite3 ~/.ecd_lookup.db "SELECT * FROM lookup_history"
```
