# ecd-extract — Documentation

从 macOS 内置词典包构建 `ecd.db` 的提取工具及相关说明。

## 文档索引

| 文档 | 内容 |
|------|------|
| [schema.md](./schema.md) | 数据库表结构设计，含字段说明、约束、FTS5 及同义/反义词表 |
| [extraction.md](./extraction.md) | Collins 与 Oxford 词典的 HTML 解析逻辑，含发音、派生词、同义词提取规则 |
| [edge-cases.md](./edge-cases.md) | 解析过程中遇到的边界情况汇总 |
| [verification.md](./verification.md) | 构建验证命令、预期行数及抽查用例 |

## 项目结构

```
extract/
├── docs/                     # 设计文档（即本目录）
│   ├── README.md
│   ├── schema.md
│   ├── extraction.md
│   ├── edge-cases.md
│   └── verification.md
├── ecd_extract/              # Python 包（提取与建库代码）
│   ├── config.py             # 路径、常量、pyglossary 定位
│   ├── utils.py              # extract_tabfile、itertext、child_elements、clean_ipa
│   ├── collins.py            # Collins 解析器
│   ├── oxford.py             # Oxford 解析器
│   ├── parse.py              # parse_tabfile() 调度
│   ├── build.py              # build_db() 编排
│   └── cli.py                # main() 命令行入口
├── schema.sql                # DDL
├── pyproject.toml            # 打包元数据
└── requirements.txt          # 依赖
```

## 快速开始

```bash
cd extract
python3 -m venv .venv
.venv/bin/pip install -r requirements.txt
.venv/bin/pip install -e .
.venv/bin/ecd-build           # 或 python -m ecd_extract
```

构建完成后，项目根目录会生成 `ecd.db`。
