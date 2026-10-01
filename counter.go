package main

import "sort"

// vote 是一张有效选票在计票过程中的内部表示。
type vote struct {
	ranking []string
	weight  int64
	cursor  int // 下一个待尝试的排序位置
}

// runElection 按确定的排序复选规则逐轮计票，返回完整结果记录。
func runElection(in *electionInput) *Result {
	votes := make([]*vote, len(in.valid))
	var totalWeight int64
	for i, b := range in.valid {
		votes[i] = &vote{ranking: b.ranking, weight: b.weight, cursor: 0}
		totalWeight += b.weight
	}

	remaining := make(map[string]bool, len(in.candidates))
	for _, c := range in.candidates {
		remaining[c] = true
	}

	var exhausted int64
	var rounds []RoundRecord
	var winner *string

	// 空排序（或首轮即无可投候选）的选票从第一轮起计入耗尽。
	for _, v := range votes {
		if v.currentChoice(remaining) == "" {
			exhausted += v.weight
		}
	}

	for roundNo := 1; ; roundNo++ {
		tallies := make(map[string]int64, len(remaining))
		for c := range remaining {
			tallies[c] = 0
		}

		var nonExhausted int64
		for _, v := range votes {
			name := v.currentChoice(remaining)
			if name == "" {
				continue
			}
			tallies[name] += v.weight
			nonExhausted += v.weight
		}

		record := RoundRecord{
			Round:               roundNo,
			RemainingCandidates: sortedKeys(remaining),
			Tallies:             tallies,
			NonExhaustedWeight:  nonExhausted,
			ExhaustedWeight:     exhausted,
		}

		if nonExhausted == 0 {
			rounds = append(rounds, record)
			winner = nil
			break
		}

		if w := majorityWinner(tallies, nonExhausted); w != "" {
			record.Winner = w
			rounds = append(rounds, record)
			wname := w
			winner = &wname
			break
		}

		eliminated := eliminateOne(tallies)
		record.Eliminated = eliminated

		transfer := &RoundTransfer{
			Round:           roundNo,
			Eliminated:      eliminated,
			EliminatedVotes: tallies[eliminated],
			ToCandidates:    map[string]int64{},
		}
		delete(remaining, eliminated)
		for c := range remaining {
			transfer.ToCandidates[c] = 0
		}

		// 淘汰后按原排序转移本轮投给淘汰者的选票；排序本身不被改写。
		for _, v := range votes {
			if v.cursor >= len(v.ranking) || v.ranking[v.cursor] != eliminated {
				continue
			}
			v.cursor++
			dest := v.currentChoice(remaining)
			if dest == "" {
				transfer.ToExhausted += v.weight
				exhausted += v.weight
			} else {
				transfer.ToCandidates[dest] += v.weight
			}
		}
		record.Transfers = transfer
		rounds = append(rounds, record)
		if len(remaining) == 0 {
			break
		}
	}

	res := &Result{
		Winner:           winner,
		RoundsCount:      len(rounds),
		ValidBallots:     len(in.valid),
		InvalidBallots:   len(in.invalid),
		TotalValidWeight: totalWeight,
		InvalidDetails:   in.invalid,
		Rounds:           rounds,
	}
	return res
}

// currentChoice 从游标起返回第一个仍在选的候选；无人可投返回空串并把游标推到末尾。
func (v *vote) currentChoice(alive map[string]bool) string {
	for v.cursor < len(v.ranking) {
		name := v.ranking[v.cursor]
		if alive[name] {
			return name
		}
		v.cursor++
	}
	return ""
}

// majorityWinner 返回得票严格大于未耗尽权重一半的候选（按标识排序保证确定）。
func majorityWinner(tallies map[string]int64, nonExhausted int64) string {
	threshold := nonExhausted / 2 // 严格大于一半，即 votes > floor(nonExhausted/2)
	names := sortedKeys(tallies)
	for _, name := range names {
		if tallies[name] > threshold {
			return name
		}
	}
	return ""
}

// eliminateOne 淘汰得票最少者；最低票并列时淘汰字节字典序最大的标识。
func eliminateOne(tallies map[string]int64) string {
	names := sortedKeys(tallies)
	worst := names[0]
	for _, name := range names[1:] {
		if tallies[name] < tallies[worst] || (tallies[name] == tallies[worst] && name > worst) {
			worst = name
		}
	}
	return worst
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
