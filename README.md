# 排序选票计票器（Ranked Ballot Counter）

按公开、确定的排序复选（instant-runoff）规则从排序选票中选出单一获胜者，
并输出可逐轮复核的 JSON 结果与 CSV 轮次记录。仅使用 Go 标准库。

## 运行

```sh
go run . --input 选举.json --out 输出目录
```

- `--input`：选举输入 JSON 文件（必需）。
- `--out`：输出目录（必需，不存在则自动创建）。
- 成功后在输出目录生成 `result.json` 与 `rounds.csv`，并在标准输出打印获胜者。
- 输入存在全局结构错误时，程序向标准错误打印原因并以非零码退出，**不会**生成成功结果。

自测：

```sh
go test ./...
```

## 输入格式

```json
{
  "candidates": ["Alice", "Bob", "Carol"],
  "ballots": [
    {"weight": 40, "ranking": ["Alice"]},
    {"weight": 30, "ranking": ["Bob", "Alice"]},
    {"weight": 20, "ranking": ["Carol", "Alice"]},
    {"weight": 10, "ranking": ["Carol"]},
    {"weight": 5,  "ranking": []}
  ]
}
```

字段规则：

- `candidates`：非空字符串数组。每个标识必须非空、唯一且为 ASCII；至少一人。
  候选列表本身的任何问题都属于全局结构错误，直接退出非零。
- `ballots`：选票数组，顺序即“原始序号”（从 0 开始，记录在无效票明细中）。
- `weight`：选票权重，必须是严格正整数字面量（不接受 `0`、负数、小数、指数、前导零）。
- `ranking`：候选标识数组，可以只排列部分候选，也可以为空数组。
  空排序是**有效票**，但其权重从第一轮起即计入累计耗尽。

以下问题只排除该张选票（记录原始序号与原因），其他选票继续计数：

- 权重非法；
- `ranking` 中重复列出同一候选；
- `ranking` 引用了不在 `candidates` 中的标识。

选票结构本身损坏（缺少 `weight`/`ranking` 字段、字段类型错误、`ballots` 不是数组等）
属于全局结构错误，直接失败退出。

## 计票规则

每一轮：

1. 每张有效票的全部权重投给其**原排序中尚未淘汰的第一人**；排序永不被改写。
2. 没有可投候选的选票记为**耗尽**，其权重计入累计耗尽，不参与后续多数门槛。
3. 某候选得票**严格大于本轮未耗尽权重的一半**（`votes > floor(nonExhausted/2)`）即当选，计票结束。
4. 若本轮未耗尽权重为 0，计票结束并报告 `winner: null`（无人当选）。
5. 否则淘汰**恰好一人**：得票最少者；最低票并列时，淘汰标识按字节字典序**最大**者。
6. 被淘汰者名下选票按原排序转移给下一个仍在选候选；无后续可选则进入耗尽池。
   每轮的转移量（向各候选与耗尽池）均完整记录。

## 输出

### `result.json`

- `winner`：获胜者标识；无人当选为 `null`。
- `rounds_count`：计票轮数。
- `valid_ballots` / `invalid_ballots`：有效/无效票数。
- `total_valid_weight`：有效票权重总和。
- `invalid_ballot_details`：每张无效票的原始 `index` 与 `reason`。
- `rounds`：逐轮记录，包含：
  - `remaining_candidates`：本轮仍在选候选（标识字典序）；
  - `tallies`：各在选候选得票；
  - `non_exhausted_weight`：本轮未耗尽权重；
  - `exhausted_weight_cumulative`：截至本轮累计耗尽权重；
  - `winner` / `eliminated`：本轮当选者或淘汰者；
  - `transfers`：淘汰权重向各候选（`to_candidates`）与耗尽池（`to_exhausted`）的转移量。

### `rounds.csv`

每行一轮，列为：轮次、在选候选（分号连接）、未耗尽权重、累计耗尽权重、
当选者、淘汰者、每候选得票列（`votes_<候选>`）、被淘汰者得票、
转入耗尽池权重（`transfer_exhausted`）、每候选转入列（`transfer_<候选>`）。
列集合与顺序按候选标识字节字典序固定，重复运行结果逐字节一致。

## 计票示例

示例输入见 `examples/election.json`（含一张空排序票、一张未知候选票、一张重复列名票、
一张负权重票）。运行：

```sh
go run . --input examples/election.json --out examples/out
```

计票过程：

| 轮次 | Alice | Bob | Carol | 未耗尽 | 累计耗尽 | 事件 |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | 40 | 30 | 30 | 100 | 5（空排序票） | 无人过半；Bob/Carol 并列最低，淘汰字典序较大的 Carol；其 30 票中 20 转 Alice、10 耗尽 |
| 2 | 60 | 30 | — | 90 | 15 | Alice 60 > 45，当选 |

三张无效票（原始序号 5、6、7）连同原因记录在 `result.json` 的
`invalid_ballot_details` 中，不参与计票。
