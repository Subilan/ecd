# Edge Cases

HTML 解析中遇到的主要边界情况及其处理方式。

| Case | Handling |
|------|----------|
| Oxford 空 `p-g` 块（无 pos、无 n-g） | 跳过该块 |
| Oxford 模式 4：`def-g` 直接位于 `h-g` 下（无 `p-g`/`n-g`） | `_oxford_parse_hg_direct()` 处理——POS 取自 `top-g > block-g > pos`，语法取自 `top-g > .gr`。示例来自 `h-g` 中的兄弟 `.x-g` |
| Oxford 模式 4b：`h-g` 下的 `ids-g` 习语 | 每个 `id-g > sense-g` 作为独立条目，POS 为 `IDM <短语>` |
| Oxford 派生形式交叉引用（`.entry > .derived`，如 "abasement"） | 按交叉引用处理——`cn_definition` 取自 `.de_e`/`.de_c`，`cross_ref` 取自 `<a>` 链接 |
| Oxford 延伸条目：含 `.derived` 但同时有 `.n-g` 内容（如 "radically"） | 按普通条目处理（模式 2）。若无 `.block-g`，POS 回退到 `.top-g > .pos-g .pos`。有例句但无释义的条目通过过滤 |
| Oxford 情态动词（`must`）——`span.pos` 在 `p-g` 外 | 在条目级而非 `p-g` 内检查 `block-g > pos-g > pos` |
| Collins `<dfn>` 标签在示例 `<p>` 中 | 使用 `.//text()` 而非 `.text` 收集全部文本节点 |
| Collins `<figure class="note type-*">` 元素 | 从示例中排除；提取为 `extra_notes` JSON。每个 figure 生成一条 note，合并内联文本与 `<li>` 示例。引语 note 从 `.cit > blockquote p` 或 `.cit > span.quote` + `.cit > cite` 解析 |
| Collins `<figure class="note type-drv">` 派生形式 | 创建独立条目，`pos='DRV'`，词头取自 `<b>` 标签，例句来自 `<li><p>` 对。父条目内通过 `seen_drv_words` 集合去重 |
| Collins synonym `<div class="synonym">` 块 | 提取 `<span class="form">` 子元素（`<a>` 链接及纯文本），按条目 ID 存入 `synonyms` 表 |
| Collins 孤立 `figure.note`（`.collins_en_cn` 外的引语） | 附加到首个带释义的条目 |
| Collins level1–level5 频率标记 | 忽略 |
| 纯交叉引用条目 | `pos=NULL`，`cn_definition`=描述，`cross_ref`=目标词 |
| 混合条目（含交叉引用标签但自身有释义，如 Oxford "better"） | 按普通条目处理；忽略交叉引用标签 |
| Collins "see also" 引用（在短语中） | 从 `<a class="see">` 提取首个 `bword://` 目标 |
| **所有文本字段** | **INSERT 前必须 `.strip()`** |
