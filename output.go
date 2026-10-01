package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// writeOutputs 将结果写入 outDir/result.json 与 outDir/rounds.csv。
func writeOutputs(outDir string, res *Result) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	if err := writeResultJSON(filepath.Join(outDir, "result.json"), res); err != nil {
		return err
	}
	if err := writeRoundsCSV(filepath.Join(outDir, "rounds.csv"), res); err != nil {
		return err
	}
	return nil
}

func writeResultJSON(path string, res *Result) error {
	if res.InvalidDetails == nil {
		res.InvalidDetails = []InvalidBallot{}
	}
	data, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化结果失败: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("写入 %s 失败: %w", path, err)
	}
	return nil
}

// writeRoundsCSV 每行记录一轮：每候选一列得票，外加未耗尽/累计耗尽/当选/淘汰
// 以及淘汰权重向各候选和耗尽池的转移量（列前缀 transfer_ / transfer_exhausted）。
func writeRoundsCSV(path string, res *Result) error {
	if len(res.Rounds) == 0 {
		return fmt.Errorf("内部错误: 无任何轮次记录")
	}

	candidateSet := map[string]bool{}
	for _, r := range res.Rounds {
		for _, c := range r.RemainingCandidates {
			candidateSet[c] = true
		}
	}
	var allCandidates []string
	// 候选列顺序取所有轮次中出现过的候选，按标识字节字典序固定，保证重复运行一致。
	allCandidates = sortedKeys(candidateSet)

	header := []string{"round", "remaining", "non_exhausted_weight", "exhausted_weight_cumulative", "winner", "eliminated"}
	for _, c := range allCandidates {
		header = append(header, "votes_"+c)
	}
	header = append(header, "eliminated_votes", "transfer_exhausted")
	for _, c := range allCandidates {
		header = append(header, "transfer_"+c)
	}

	var rows [][]string
	rows = append(rows, header)
	for _, r := range res.Rounds {
		row := []string{
			strconv.Itoa(r.Round),
			joinComma(r.RemainingCandidates),
			strconv.FormatInt(r.NonExhaustedWeight, 10),
			strconv.FormatInt(r.ExhaustedWeight, 10),
			r.Winner,
			r.Eliminated,
		}
		for _, c := range allCandidates {
			row = append(row, strconv.FormatInt(r.Tallies[c], 10))
		}
		var elimVotes, toExhausted int64
		if r.Transfers != nil {
			elimVotes = r.Transfers.EliminatedVotes
			toExhausted = r.Transfers.ToExhausted
		}
		row = append(row, strconv.FormatInt(elimVotes, 10), strconv.FormatInt(toExhausted, 10))
		for _, c := range allCandidates {
			var v int64
			if r.Transfers != nil {
				v = r.Transfers.ToCandidates[c]
			}
			row = append(row, strconv.FormatInt(v, 10))
		}
		rows = append(rows, row)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("创建 %s 失败: %w", path, err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if err := w.WriteAll(rows); err != nil {
		return fmt.Errorf("写入 %s 失败: %w", path, err)
	}
	return nil
}

// joinComma 用分号连接，避免与 CSV 的逗号分隔冲突。
func joinComma(items []string) string {
	out := ""
	for i, s := range items {
		if i > 0 {
			out += ";"
		}
		out += s
	}
	return out
}
