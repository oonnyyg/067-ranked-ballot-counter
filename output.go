package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

// Report 是写入结果 JSON 的完整结构。
type Report struct {
	Candidates      []string         `json:"candidates"`
	ValidBallots    int              `json:"valid_ballots"`
	InvalidBallots  int              `json:"invalid_ballots"`
	Rejected        []RejectedBallot `json:"rejected"`
	TotalValidWeight int64           `json:"total_valid_weight"`
	Elected         bool             `json:"elected"`
	Winner          string           `json:"winner"`
	Rounds          []RoundRecord    `json:"rounds"`
}

// WriteOutputs 在 outDir 生成 result.json 与 rounds.csv，输出排序固定。
func WriteOutputs(outDir string, rep *Report) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(outDir, "result.json"), data, 0o644); err != nil {
		return fmt.Errorf("写入 result.json 失败: %w", err)
	}

	f, err := os.Create(filepath.Join(outDir, "rounds.csv"))
	if err != nil {
		return fmt.Errorf("写入 rounds.csv 失败: %w", err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	if err := w.Write([]string{
		"round", "candidate", "votes", "active_weight", "exhausted_weight",
		"eliminated", "transfer_to", "transfer_weight", "elected",
	}); err != nil {
		return err
	}
	for _, r := range rep.Rounds {
		cands := make([]string, 0, len(r.Tallies))
		for c := range r.Tallies {
			cands = append(cands, c)
		}
		sort.Strings(cands)
		for _, c := range cands {
			if err := w.Write([]string{
				strconv.Itoa(r.Round), c, strconv.FormatInt(r.Tallies[c], 10),
				strconv.FormatInt(r.ActiveWeight, 10),
				strconv.FormatInt(r.ExhaustedWeight, 10),
				r.Eliminated, "", "", r.Elected,
			}); err != nil {
				return err
			}
		}
		targets := make([]string, 0, len(r.Transfers))
		for t := range r.Transfers {
			targets = append(targets, t)
		}
		sort.Strings(targets)
		for _, t := range targets {
			if err := w.Write([]string{
				strconv.Itoa(r.Round), "", "",
				strconv.FormatInt(r.ActiveWeight, 10),
				strconv.FormatInt(r.ExhaustedWeight, 10),
				r.Eliminated, t, strconv.FormatInt(r.Transfers[t], 10), r.Elected,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}
