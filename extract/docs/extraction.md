# Extraction Logic

基于对实际 tabfile HTML 结构的分析。

## Collins

HTML 结构：`div.word_entry` → `div.collins_en_cn` 块，每块为一个义项。

### 常规条目提取

1. **Word**：`span.word_key` 文本，回退到 `<h1>` 文本。注意：某些条目（如 `'cause`）的 tabfile key 带有前导引号，HTML 中的 `word_key` 才是规范形式。
2. **POS**：`span.st` 文本（COBUILD 编码如 `N-COUNT`、`ADJ-GRADED`、`VERB`）。使用 `.//text()` 处理内部元素。
3. **cn_definition**：每个 `collins_en_cn` div 内 `span.def_cn` 的文本。
4. **Examples**：`collins_en_cn` 内每个 `<li>` → 第一个 `<p>` 为 `en_example`，第二个 `<p>` 为 `cn_example`。使用 `.//text()` 处理 `<dfn>` 标签。**排除 `<figure class="note">` 内的 `<li>`**——这些属于 extra_notes。
5. **extra_notes**：从 `<figure class="note type-*">` 元素提取（**排除 `type-drv`**）。每个 figure 生成一条 note，合并内联文本与 `<li>` 示例。引语类（`type-quotation`）从 `.cit` 元素解析（`blockquote > p` 或 `span.quote` 为引用文本，`cite` 为出处）。`span.quote` 中的 `<br>` 转换为换行。多条引语以双换行分隔。`.collins_en_cn` 外的孤立 figure 附加到首个带释义的条目。存储为 JSON 数组。
6. **派生词**：`<figure class="note type-drv">` 元素由 `_extract_drv_entries()` 处理 → 独立条目，`pos='DRV'`，词头取自 `<b>` 标签，例句来自 `<li><p>` 对。与父条目共享发音。同一父条目内通过 `seen_drv_words` 集合去重。
7. **同义词**：`<div class="synonym">` 块 → `_extract_collins_synonyms()` 提取每个 `<span class="form">`（链接文本或纯文本）。存入 `synonyms` 表。

### 发音

`_extract_collins_pronunciation()` — `word_entry` 级别的 `span.pron` → `span.pron.type_uk` 和 `span.pron.type_us`。IPA 文本中的 HTML 标记（`<u>`、`<sup>`）通过 `clean_ipa()` 去除。在 `parse_tabfile()` 中应用于该词的所有条目。存储为 JSON 数组。

### 交叉引用检测

Collins 条目在以下情况为纯交叉引用：
- 无 `span.st`（POS 标签）
- 无 `<li>` 示例
- 含 `<a class="see" href="bword://...">` 或 caption 文本匹配 "past tense of"、"plural of"、"is the ... of" 等模式

完整模式列表详见 `_COLLINS_XREF_PATTERNS`（位于 `ecd_extract/collins.py`）。

交叉引用条目提取：
1. **cross_ref**：优先从 `<a class="see">` 文本取，其次从 caption 正则匹配
2. **cn_definition**：`div.caption` 完整文本
3. **pos**：NULL
4. **sense_order**：1

## Oxford

HTML 结构：`span.entry` → `span.p-g`（POS 组）→ `span.n-g`（义项组）。

### 常规条目提取

共有四种 HTML 结构模式：

- **Pattern 1**（如 "water"）：`.entry` 下有 `.p-g` 块，每个 `.p-g` 包含 `.pos` + `.n-g` 子元素。`.n-g` 内含 `.gr` 标签和 `.x-g` 示例。
- **Pattern 1b**（如 "cause" 的动词、 "above" 的形容词）：存在 `.p-g` 但 `.def-g` 和 `.x-g` 直接作为 `.p-g` 的子元素（无 `.n-g` 包裹）。语法信息来自 `.p-g` 的直接 `.gr` 子元素。
- **Pattern 2**（如 "beauty"）：无 `.p-g`，`.n-g` 直接位于 `.h-g` 下，POS 从 `top-g > block-g > pos` 获取。延伸条目（如 "radically"）的 POS 回退到 `.top-g > .pos-g > .pos`。
- **Pattern 4**（如 "incantation"、"A1"）：无 `.n-g`——`def-g` 直接位于 `.h-g` 下。POS 从 `top-g > block-g > pos` 获取，语法来自 `top-g > .gr`。子模式 4b：`ids-g` 习语，每个 `id-g > sense-g` 作为独立条目，POS 为 `IDM <短语>`。

POS 字符串构建规则因模式而异：
- 模式 1/2：`_oxford_pos_for_ng()` 合并基础 POS span 与 `.n-g` 中的 `.gr` span
- 模式 1b：`_oxford_parse_pg_direct()` 合并基础 POS span 与 `.p-g` 中的直接 `.gr` 子元素
- 模式 4：`_oxford_make_pos()` 合并基础 POS span 与容器中的 `.gr` span
- 模式 4b：POS 为 `"IDM <短语>"`，短语取自 `.id-g` 中的 `.id` 文本

### 发音

`_extract_oxford_pronunciation()` — `span.ei-g` 块包含 `span.phon-gb`（英式）和 `span.phon-usgb`/`span.phon-us`（美式）。两种位置模式：(a) 单词级别在 `top-g > ei-g`；(b) 按词性在 `p-g > ei-g` 内。先检查 POS 组容器，回退到 `top-g`。模式 4 先检查 `h-g`，回退到 `top-g`。

### Oxford 同义词与反义词

每个条目容器（`n-g`、`p-g`、`h-g`、`sense-g`）的 `.xr-g` 交叉引用元素由 `_extract_oxford_xrefs()` 解析：
- `.symbols-oppsym`（OPP）标记 → 反义词
- `.symbols-synsym`（SYN）标记 → 同义词
- `.z_xr` 元素包含 "synonyms at" → 同义词交叉引用

存入 `synonyms` 和 `antonyms` 表，`source='oxford'`。

### 交叉引用检测

Oxford 条目在以下情况为纯交叉引用：
- 含 `span.xr-g`（在 `span.sense-g` 中）且无 `span.p-g` 块、无 `span.n-g` 块
- 或含 `.entry > span.derived` 且有父词链接，无 `.p-g`、`.n-g`、`.def-g`、`.ids-g`（派生形式重定向，如 "abasement" → "abase"）

交叉引用条目提取：
1. **cross_ref**：`.xr-g` 条目从 `xr-g > xr` 内的 `<a>` 文本取；`.derived` 条目从 `.derived` 内的 `<a>` 文本取
2. **cn_definition**：`.xr-g` 条目取 `span.xr-g` 完整文本；`.derived` 条目合并 `.de_e` 和 `.de_c` span 的文本
3. **pos**：NULL
4. **sense_order**：1

### 条目过滤

两个解析器均跳过既无 `cn_definition` 也无 examples 的条目（词库内部噪音，如词库标题、"See also:" 链接、无中文翻译的缩写展开）。有例句但无中文释义的条目予以保留。
