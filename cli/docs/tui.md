# TUI Design

基于 [Bubble Tea](https://github.com/charmbracelet/bubbletea) 的交互界面。

## 状态机

```
                    ┌──────────────┐
          ┌────────►│ StateSearch  │◄────────────┐
          │         └──────┬───────┘             │
          │                │                     │
     Esc / any key    ┌────┼────────────┐        │
          │           │    │            │        │
   ┌──────┴──────┐    │    │       ┌────▼────┐   │
   │ StateHelp   │    │    │       │StateDeck│   │
   └─────────────┘    │    │       └─────────┘   │
        /help         │    │          /deck      │
                      │    │                     │
                  ┌───▼──┐ │  ┌────────────┐     │
                  │/add  │ │  │/review     │     │
                  │/del  │ │  │            │     │
                  │/syn  │ │  │┌──────────┐│     │
                  │/ant  │ │  ││reviewFront││     │
                  │/idm  │ │  ││reviewFront││     │
                  │/random│ │  │└────┬─────┘│     │
                  │/lang  │ │  │     │Enter  │     │
                  └───────┘ │  │┌────▼─────┐│     │
                            │  ││reviewBack││     │
                            │  │└────┬─────┘│     │
                            │  │     │0-3   │     │
                            │  │┌────▼──────┐│    │
                            │  ││reviewDone ││    │
                            │  │└───────────┘│    │
                            │  └─────────────┘    │
                            │        any key      │
                            └─────────────────────┘
```

所有状态定义在 `constants.go`：
- `StateSearch` — 主搜索界面
- `StateEntryDetail` — （预留）词条详情
- `StateReview` / `reviewFront` / `reviewBack` / `reviewComplete` — SM-2 复习
- `StateDeckStats` — 牌组统计
- `StateHelp` — 命令帮助
- `StateConfirmReset` — 重置确认

## 组件树

```
Model (model.go)
├── searchModel (search_view.go)
│   ├── textinput.Model   — 搜索输入框
│   └── viewport.Model    — 结果滚动区域
├── detailModel           — 词条详情（内含 viewport.Model）
├── reviewModel           — 闪卡复习（内含 viewport.Model，支持滚动和自动换行）
├── deckModel             — 牌组统计
└── helpModel             — 帮助页面
```

每个子 model 实现 `Init()` / `Update()` / `View()` 接口。

## 键位绑定

### 全局键

| 键 | 动作 |
|----|------|
| `Ctrl+C` | 强制退出 |

### StateSearch

| 键 | 动作 |
|----|------|
| `Esc` | 清空输入框 |
| `Enter` | 提交搜索（普通文本）或执行命令（以 `/` 开头） |
| `Tab` | 切换输入/滚动模式（底部栏显示紫色 SCROLL 指示） |
| `↑` / `↓` | 输入模式：浏览搜索历史；滚动模式：滚动结果 |
| `PgUp` / `PgDn` | 翻页（仅滚动区域） |
| `Ctrl+D` / `Ctrl+U` | 半页滚动 |

### StateHelp / StateDeckStats

| 键 | 动作 |
|----|------|
| 任意键 | 返回搜索界面 |

### StateReview

| 键 | 动作 |
|----|------|
| `Enter` | 翻面（显示答案） |
| `←` / `→` | 切换义项（多义项时） |
| `0`-`3` | 评分：Again / Hard / Good / Easy |
| `↑` / `↓` | 滚动释义（内容超出屏幕时） |
| `PgUp` / `PgDn` | 翻页滚动 |
| 任意键 | 复习完成后返回搜索界面 |

## 命令列表

所有命令以 `/` 开头，在搜索框中输入后按 Enter 执行：

| 命令 | 功能 |
|------|------|
| `/help` | 显示帮助 |
| `/add [word]` | 将单词加入闪卡牌组（省略时使用上次查询的单词） |
| `/del <word>` | 从牌组中移除单词 |
| `/auto-add [on\|off]` | 切换自动添加（查词时自动加入牌组） |
| `/review` | 开始 SM-2 复习 |
| `/deck` | 显示牌组统计 |
| `/reset` | 重置所有卡片（需 `/reset-confirm` 确认） |
| `/syn [word]` | 显示同义词 |
| `/ant [word]` | 显示反义词 |
| `/idm <word>` | 显示牛津习语 |
| `/random` | 随机单词 |
| `/lang [en\|zh]` | 切换界面语言 |
| `/exit` `/quit` `/q` | 退出 |

## 底部栏 (Footer)

底部栏由 `renderFooter()` 渲染，包含：
- 基础提示信息（`footer.hint`）
- 自动添加指示（`[auto]`，灰色）
- 滚动模式指示（`SCROLL`，紫色，仅 Tab 切换后显示）
- 状态消息（绿色，5 秒后自动消失）

## 状态消息机制

`statusMsg` 字段 + `statusSeq` 计数器实现自动清除：每次 `setStatus()` 递增序号，`tea.Tick(5s)` 发送 `clearStatusMsg{seq}`，只有序号匹配的才清除消息，防止旧消息覆盖新消息。

## 搜索历史

`searchModel` 维护一个最多 100 条的环形历史缓冲区：
- 每次提交搜索时 `addHistory()` 追加（自动去重连续重复项）
- `↑` / `↓` 在输入模式下浏览：从最新向最旧/从旧向最新
- 浏览时按其他键自动退出历史模式（当输入与当前历史条目不匹配时）
- `historySaved` 保存浏览前的输入文本，浏览到底可恢复
