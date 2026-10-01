package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
)

// ballot 是一张经过校验的有效选票。
type ballot struct {
	ranking []string
	weight  int64
}

// electionInput 是已经过结构与规则校验的计票输入。
type electionInput struct {
	candidates []string
	valid      []ballot
	invalid    []InvalidBallot
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

// parseElectionFile 读取并校验选举文件。结构级错误（无法解析、字段缺失或类型错误、
// 候选不合法等）返回 error；单张选票问题记录在 invalid 中，不中断其他选票的校验。
func parseElectionFile(path string) (*electionInput, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取输入文件失败: %w", err)
	}

	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return nil, fmt.Errorf("输入不是合法 JSON: %w", err)
	}

	candRaw, ok := top["candidates"]
	if !ok {
		return nil, fmt.Errorf("缺少 candidates 字段")
	}
	var candidates []string
	if err := json.Unmarshal(candRaw, &candidates); err != nil {
		return nil, fmt.Errorf("candidates 必须是字符串数组: %w", err)
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("candidates 至少需要一人")
	}

	seen := make(map[string]bool, len(candidates))
	for _, c := range candidates {
		if c == "" {
			return nil, fmt.Errorf("候选标识不能为空")
		}
		if !isASCII(c) {
			return nil, fmt.Errorf("候选标识必须是 ASCII: %q", c)
		}
		if seen[c] {
			return nil, fmt.Errorf("候选标识重复: %q", c)
		}
		seen[c] = true
	}

	ballotRaw, ok := top["ballots"]
	if !ok {
		return nil, fmt.Errorf("缺少 ballots 字段")
	}
	var raws []json.RawMessage
	if err := json.Unmarshal(ballotRaw, &raws); err != nil {
		return nil, fmt.Errorf("ballots 必须是数组: %w", err)
	}

	input := &electionInput{candidates: candidates}
	for i, br := range raws {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(br, &obj); err != nil {
			return nil, fmt.Errorf("第 %d 张选票结构非法: %w", i, err)
		}
		weightRaw, ok := obj["weight"]
		if !ok {
			return nil, fmt.Errorf("第 %d 张选票缺少 weight 字段", i)
		}
		var weightNum json.Number
		if err := json.Unmarshal(weightRaw, &weightNum); err != nil {
			return nil, fmt.Errorf("第 %d 张选票的 weight 必须是数字", i)
		}
		weight, reason := parsePositiveInt64(weightNum.String())
		if reason != "" {
			input.invalid = append(input.invalid, InvalidBallot{Index: i, Reason: "权重非法: " + reason})
			continue
		}

		rankingRaw, ok := obj["ranking"]
		if !ok {
			return nil, fmt.Errorf("第 %d 张选票缺少 ranking 字段", i)
		}
		var ranking []string
		if err := json.Unmarshal(rankingRaw, &ranking); err != nil {
			return nil, fmt.Errorf("第 %d 张选票的 ranking 必须是字符串数组", i)
		}

		dup := false
		listed := make(map[string]bool, len(ranking))
		for _, name := range ranking {
			if !seen[name] {
				input.invalid = append(input.invalid, InvalidBallot{Index: i, Reason: fmt.Sprintf("引用未知候选 %q", name)})
				dup = true
				break
			}
			if listed[name] {
				input.invalid = append(input.invalid, InvalidBallot{Index: i, Reason: fmt.Sprintf("重复列出候选 %q", name)})
				dup = true
				break
			}
			listed[name] = true
		}
		if dup {
			continue
		}

		input.valid = append(input.valid, ballot{ranking: ranking, weight: weight})
	}

	return input, nil
}

// parsePositiveInt64 要求字面量为严格的正整数（不接受小数、指数、前导符号、非 0 开头的前导零）。
func parsePositiveInt64(s string) (int64, string) {
	if _, ok := big.NewInt(0).SetString(s, 10); !ok {
		return 0, "不是整数"
	}
	if len(s) > 1 && s[0] == '0' {
		return 0, "含前导零"
	}
	n := new(big.Int)
	n.SetString(s, 10)
	if n.Sign() <= 0 {
		return 0, "不是正整数"
	}
	if !n.IsInt64() {
		return 0, "超出 int64 范围"
	}
	return n.Int64(), ""
}
