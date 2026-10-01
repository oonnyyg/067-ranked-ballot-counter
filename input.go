package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Input 是选举输入文件的结构。
type Input struct {
	Candidates []string     `json:"candidates"`
	Ballots    []ballotJSON `json:"ballots"`
}

type ballotJSON struct {
	Weight  json.Number `json:"weight"`
	Ranking []string    `json:"ranking"`
}

// Ballot 是一张校验后的有效选票。
type Ballot struct {
	Weight  int64
	Ranking []string
}

// RejectedBallot 记录一张被排除的无效选票。
type RejectedBallot struct {
	Index  int    `json:"index"` // 原始序号（从 0 开始）
	Reason string `json:"reason"`
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
}

// LoadInput 读取并做全局结构校验；结构错误返回 error（应退出非零）。
func LoadInput(path string) (*Input, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取输入文件失败: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var in Input
	if err := dec.Decode(&in); err != nil {
		return nil, fmt.Errorf("输入 JSON 解析失败: %w", err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("输入 JSON 含有多余内容")
	}
	if len(in.Candidates) == 0 {
		return nil, fmt.Errorf("候选人列表为空")
	}
	seen := map[string]bool{}
	for _, c := range in.Candidates {
		if c == "" {
			return nil, fmt.Errorf("候选标识不能为空字符串")
		}
		if !isASCII(c) {
			return nil, fmt.Errorf("候选标识 %q 不是 ASCII 字符串", c)
		}
		if seen[c] {
			return nil, fmt.Errorf("候选标识 %q 重复", c)
		}
		seen[c] = true
	}
	return &in, nil
}

// ValidateBallots 逐票校验，返回有效选票与无效票记录。
func ValidateBallots(in *Input) (valid []Ballot, rejected []RejectedBallot) {
	known := map[string]bool{}
	for _, c := range in.Candidates {
		known[c] = true
	}
	for i, b := range in.Ballots {
		w, err := b.Weight.Int64()
		if err != nil || w <= 0 {
			rejected = append(rejected, RejectedBallot{Index: i, Reason: "权重缺失或不是正整数"})
			continue
		}
		bad := ""
		seenRank := map[string]bool{}
		for _, r := range b.Ranking {
			if !known[r] {
				bad = fmt.Sprintf("引用未知候选 %q", r)
				break
			}
			if seenRank[r] {
				bad = fmt.Sprintf("重复排列候选 %q", r)
				break
			}
			seenRank[r] = true
		}
		if bad != "" {
			rejected = append(rejected, RejectedBallot{Index: i, Reason: bad})
			continue
		}
		valid = append(valid, Ballot{Weight: w, Ranking: b.Ranking})
	}
	return valid, rejected
}
