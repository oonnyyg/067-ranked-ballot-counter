package main

import "sort"

// RoundRecord 记录一轮计票的可复核信息。
type RoundRecord struct {
	Round              int              `json:"round"`
	Tallies            map[string]int64 `json:"tallies"`             // 仍在选候选的本轮得票
	ActiveWeight       int64            `json:"active_weight"`       // 本轮未耗尽权重
	ExhaustedWeight    int64            `json:"exhausted_weight"`    // 截至本轮累计耗尽权重
	Eliminated         string           `json:"eliminated,omitempty"`
	Transfers          map[string]int64 `json:"transfers,omitempty"` // 淘汰者权重去向，键 "EXHAUSTED" 表示耗尽池
	Elected            string           `json:"elected,omitempty"`
}

// TallyResult 是完整计票结果。
type TallyResult struct {
	Winner        string        `json:"winner"` // 无人当选时为空字符串
	Elected       bool          `json:"elected"`
	Rounds        []RoundRecord `json:"rounds"`
	ValidBallots  int           `json:"valid_ballots"`
	TotalWeight   int64         `json:"total_valid_weight"`
}

const exhaustedKey = "EXHAUSTED"

// RunTally 按即时复选（IRV）规则计票，规则确定且可逐轮复核。
func RunTally(candidates []string, ballots []Ballot) TallyResult {
	eliminated := map[string]bool{}
	var totalWeight int64
	for _, b := range ballots {
		totalWeight += b.Weight
	}
	res := TallyResult{
		ValidBallots: len(ballots),
		TotalWeight:  totalWeight,
	}
	for round := 1; ; round++ {
		tallies := map[string]int64{}
		for _, c := range candidates {
			if !eliminated[c] {
				tallies[c] = 0
			}
		}
		var active int64
		for _, b := range ballots {
			for _, r := range b.Ranking {
				if !eliminated[r] {
					tallies[r] += b.Weight
					active += b.Weight
					break
				}
			}
		}
		rec := RoundRecord{
			Round:           round,
			Tallies:         tallies,
			ActiveWeight:    active,
			ExhaustedWeight: totalWeight - active,
		}
		if active == 0 {
			res.Rounds = append(res.Rounds, rec)
			return res // 未耗尽权重为零：无人当选
		}
		// 严格大于未耗尽权重的一半即当选。
		winner := ""
		for _, c := range sortedKeys(tallies) {
			if tallies[c]*2 > active {
				winner = c
				break
			}
		}
		if winner != "" {
			rec.Elected = winner
			res.Rounds = append(res.Rounds, rec)
			res.Winner = winner
			res.Elected = true
			return res
		}
		// 淘汰得票最少者；并列时淘汰标识按字节字典序最大者。
		elim := ""
		for _, c := range sortedKeys(tallies) {
			if elim == "" || tallies[c] < tallies[elim] ||
				(tallies[c] == tallies[elim] && c > elim) {
				elim = c
			}
		}
		// 计算淘汰者权重转移去向。
		transfers := map[string]int64{}
		for _, b := range ballots {
			cur := ""
			for _, r := range b.Ranking {
				if !eliminated[r] {
					cur = r
					break
				}
			}
			if cur != elim {
				continue
			}
			next := ""
			for _, r := range b.Ranking {
				if !eliminated[r] && r != elim {
					next = r
					break
				}
			}
			if next == "" {
				transfers[exhaustedKey] += b.Weight
			} else {
				transfers[next] += b.Weight
			}
		}
		rec.Eliminated = elim
		rec.Transfers = transfers
		res.Rounds = append(res.Rounds, rec)
		eliminated[elim] = true
	}
}

func sortedKeys(m map[string]int64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
