# ecd CLI — Documentation

Go CLI for querying the ecd dictionary database. Supports one-shot CLI queries and an interactive Bubble Tea TUI.

## 文档索引

| 文档 | 内容 |
|------|------|
| [architecture.md](./architecture.md) | 整体架构、包职责、数据流、构建与运行 |
| [tui.md](./tui.md) | TUI 设计：状态机、键位绑定、命令、组件树 |
| [search.md](./search.md) | 搜索策略：4 层调度、中文 FTS5、模糊匹配 |
| [flashcards.md](./flashcards.md) | SM-2 算法、闪卡数据库、复习流程 |
| [i18n.md](./i18n.md) | 国际化设计：翻译表结构、语言切换 |

## 项目结构

```
cli/
├── docs/                     # 设计文档（即本目录）
│   ├── README.md
│   ├── architecture.md
│   ├── tui.md
│   ├── search.md
│   ├── flashcards.md
│   └── i18n.md
├── internal/
│   ├── config/               # DB 路径、CJK 检测、驱动初始化
│   ├── dict/                 # 词典 DB 连接与查询（models.go, db.go）
│   ├── search/               # 4 层搜索调度、同义/反义词查询
│   ├── history/              # 查询历史与闪卡数据库
│   ├── sm2/                  # SM-2 间隔重复算法（纯数学）
│   ├── i18n/                 # 中/英文字符串表
│   ├── cli/                  # CLI 模式：ANSI 着色输出
│   └── tui/                  # TUI 模式：Bubble Tea 界面
├── main.go                   # 入口：flag 解析，CLI/TUI 分发
├── go.mod
└── go.sum
```

## 快速开始

```bash
# 开发模式（TUI）
cd cli && go run .

# CLI 模式
go run . hello              # 英→中
go run . 水                 # 中→英 (FTS5)
go run . -s oxford beauty   # 限定词典
go run . -i suffice         # 牛津习语
go run . -r                 # 随机单词
echo hello | go run .       # 管道输入

# 编译
make build                  # → ../ecd-go
./ecd-go hello
```
