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
# 英查中
ecd hello

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

## 协议

MIT

注：词典内容受版权保护，仅供个人学习使用