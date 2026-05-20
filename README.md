# ecd

ecd 是一个命令行英汉字典，支持 interactive mode。查询基于 SQL，支持英文精确、前缀、模糊查询和中文反查，并可以跟踪单词的查询次数和查询时间以供将来分析使用。ecd 适合用来在专注的英文写作、阅读或视频观看的过程中快速查询不熟悉的单词。

![](./screenshot.png)

## 依赖

- Python 3.6+
- 本地的词汇数据库 `ecd.db`（约 96MB），可通过解压 `ecd.db.xz` 得到
    ```sh
    xz -d ecd.db.xz
    ```

## 使用方法

推荐在使用之前为脚本加上别名，写入到 `~/.zshrc`、`~/.bashrc` 等。

```sh
# 精确
ecd hello
# 前缀
ecd surprisingl
# 模糊
ecd rondevus

# 指定词典
ecd -s collins beauty
ecd -s oxford beauty

# 中文反查
ecd 全面的
ecd 精确的
ecd 疑惑
ecd 统一
ecd 变化

# 进入交互模式
ecd
```

交互模式下支持以下命令：

| 命令 | 说明 |
|------|------|
| `.add` | 将最近查询的单词加入记忆卡片组 |
| `.review` | 复习到期的记忆卡片 |
| `.deck` | 查看卡片组统计 |
| `.syn [word]` | 查询同义词 |
| `.exit` `.quit` `.q` Ctrl+C Ctrl+D | 退出 |

### 记忆卡片

ecd 内置了基于 SM-2 算法的间隔重复记忆功能，适合结合查词过程积累生词。

- 查完一个单词后，输入 `.add` 即可将其加入卡片组。
- 输入 `.review` 开始复习。每张卡片先显示单词和发音，按回车后显示释义、例句和同义词，然后选择评分：

- `0` Again — 完全忘记（重置进度）
- `1` Hard — 勉强想起（缩短间隔）
- `2` Good — 正常想起
- `3` Easy — 轻松想起（延长间隔）

所有卡片数据存储在 `~/.ecd_lookup.db` 中。

## 协议

MIT

注：词典内容受版权保护，仅供个人学习使用