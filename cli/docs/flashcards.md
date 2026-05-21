# SM-2 Flashcards

## 数据库

`~/.ecd_lookup.db`（SQLite），包含两张表：

### `lookup_history`

| Column | Type | Description |
|--------|------|-------------|
| `word` | TEXT PK | 查询过的单词（小写） |
| `count` | INTEGER | 查询次数 |
| `last_query` | TEXT | 最近查询时间 |

每次查词时 `INSERT OR REPLACE` 更新计数和时间。

### `flashcards`

| Column | Type | Default | Description |
|--------|------|---------|-------------|
| `word` | TEXT PK | — | 单词 |
| `created` | TEXT | `datetime('now')` | 添加时间 |
| `ease_factor` | REAL | 2.5 | SM-2 简易度因子 |
| `interval_days` | INTEGER | 0 | 当前复习间隔（天） |
| `repetitions` | INTEGER | 0 | 连续正确次数 |
| `next_review` | TEXT | `datetime('now', '+10 minutes')` | 下次复习时间 |
| `last_review` | TEXT | NULL | 上次复习时间 |
| `total_reviews` | INTEGER | 0 | 总复习次数 |
| `total_correct` | INTEGER | 0 | 总正确次数（≥Good） |

## SM-2 算法

`sm2/` 包实现修订版 SM-2 算法（纯数学，无依赖）。

### 评分

| 按键 | 评分 | 含义 |
|------|------|------|
| 0 | Again | 完全忘记 — 重置进度 |
| 1 | Hard | 困难 — 减半间隔，降低 EF |
| 2 | Good | 良好 — 正常推进 |
| 3 | Easy | 简单 — 1.3× 间隔加成，提高 EF |

### 调度逻辑（`Schedule()`）

```
Again (0):
  repetitions = 0
  interval = 0
  EF -= 根据公式

Hard (1):
  interval = last_interval * 1.2 (≥ 1)
  repetitions 不变
  EF -= 0.15

Good (2):
  rep=0 → interval=1
  rep=1 → interval=6
  rep≥2 → interval = last_interval * EF
  repetitions += 1
  EF += 基本公式

Easy (3):
  同 Good 的间隔计算
  interval *= 1.3 (EasyBonus)
  repetitions += 1
  EF += 0.15
```

### 常数

| 常数 | 值 | 说明 |
|------|-----|------|
| `MinEF` | 1.3 | 最低 EF（低于此值标记为 leech） |
| `EasyBonus` | 1.3 | 简单评分的间隔乘数 |
| `HardMultiplier` | 1.2 | 困难评分的间隔乘数 |
| `HardEFPenalty` | 0.15 | 困难评分 EF 惩罚 |
| `EasyEFBonus` | 0.15 | 简单评分 EF 奖励 |

## 复习流程

1. `/review` → `startReview()`：查询 `next_review <= now` 的卡片，最多 20 张
2. 无到期卡片 → 显示提示消息
3. `reviewFront`：显示单词和发音，按 `Enter` 翻面
4. `reviewBack`：显示全部释义（`←` / `→` 切换义项），按 `0`-`3` 评分
5. 评分后 `applyRating()` → `Schedule()` → `UpdateFlashcard()`，进入下一张
6. 所有卡片完成 → `reviewComplete`，显示统计，任意键返回

## 牌组统计 (`/deck`)

| 指标 | 说明 |
|------|------|
| Total | 卡片总数 |
| Due | 到期卡片数（`next_review <= now`） |
| New | 未复习过的卡片（`repetitions = 0`） |
| Mature | 成熟卡片（`interval_days ≥ 21`） |
| Leeches | 难点卡（`ease_factor ≤ 1.3`） |
| Avg ease | 平均简易度（百分比） |
| Next | 下次复习时间的 delta 显示 |

## 自动添加 (`/auto-add`)

当 `autoAdd = true` 时，每次查词成功后自动调用 `AddFlashcard(word)`。使用 `INSERT OR IGNORE` 语义，重复添加不报错。
