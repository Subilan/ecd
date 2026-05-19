# ecd

ecd 是一个自用的命令行英汉字典，支持 interactive mode。查询基于 SQL，支持英文精确、前缀、模糊查询和中文反查。

![](./screenshot.png)

ecd 的运行需要本地的词汇数据库 `ecd.db`（约 96MB），可通过解压 `ecd.db.xz` 得到。
```sh
xz -d ecd.db.xz
```
