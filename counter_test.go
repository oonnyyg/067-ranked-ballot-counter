package main

import (
	"strings"
	"testing"
)

func runCase(t *testing.T, cands []string, bs []ballot) *Result {
	t.Helper()
	return runElection(&electionInput{candidates: cands, valid: bs})
}

func TestImmediateMajority(t *testing.T) {
	res := runCase(t, []string{"A", "B"}, []ballot{
		{ranking: []string{"A"}, weight: 7},
		{ranking: []string{"B"}, weight: 3},
	})
	if res.Winner == nil || *res.Winner != "A" {
		t.Fatalf("期望 A 首轮当选, got %+v", res.Winner)
	}
	if res.RoundsCount != 1 {
		t.Fatalf("期望 1 轮, got %d", res.RoundsCount)
	}
	if res.TotalValidWeight != 10 {
		t.Fatalf("总权重应为 10, got %d", res.TotalValidWeight)
	}
}

func TestStrictMajorityRequiresMoreThanHalf(t *testing.T) {
	// A=5 B=5，未耗尽 10，无人严格过半；淘汰字典序较大的 B，第二轮 A 获全部 10 票。
	res := runCase(t, []string{"A", "B"}, []ballot{
		{ranking: []string{"A"}, weight: 5},
		{ranking: []string{"B"}, weight: 5},
	})
	if res.RoundsCount != 2 {
		t.Fatalf("期望 2 轮, got %d", res.RoundsCount)
	}
	r1 := res.Rounds[0]
	if r1.Winner != "" || r1.Eliminated != "B" {
		t.Fatalf("首轮应淘汰 B, got winner=%q elim=%q", r1.Winner, r1.Eliminated)
	}
	if res.Winner == nil || *res.Winner != "A" {
		t.Fatalf("次轮 A 应当选, got %+v", res.Winner)
	}
}

func TestTransferAndExhaustion(t *testing.T) {
	// A=4, B=3, C=3；最低票并列 B/C，字典序较大的 C 被淘汰。
	// C 的 3 票：2 票转 B，1 票只列 C 而耗尽。
	res := runCase(t, []string{"A", "B", "C"}, []ballot{
		{ranking: []string{"A"}, weight: 4},
		{ranking: []string{"B"}, weight: 3},
		{ranking: []string{"C", "B"}, weight: 2},
		{ranking: []string{"C"}, weight: 1},
	})
	if res.Rounds[0].Eliminated != "C" {
		t.Fatalf("应淘汰 C, got %q", res.Rounds[0].Eliminated)
	}
	tr := res.Rounds[0].Transfers
	if tr.EliminatedVotes != 3 || tr.ToCandidates["B"] != 2 || tr.ToExhausted != 1 {
		t.Fatalf("转移记录错误: %+v", tr)
	}
	r2 := res.Rounds[1]
	if r2.ExhaustedWeight != 1 || r2.NonExhaustedWeight != 9 || r2.Tallies["B"] != 5 {
		t.Fatalf("第二轮数值错误: %+v", r2)
	}
	if res.Winner == nil || *res.Winner != "B" {
		t.Fatalf("B 应以 5/9 严格过半当选, got %+v", res.Winner)
	}
}

func TestEmptyRankingExhaustedFromRoundOne(t *testing.T) {
	res := runCase(t, []string{"A", "B"}, []ballot{
		{ranking: []string{}, weight: 4},
		{ranking: []string{"A"}, weight: 3},
		{ranking: []string{"B"}, weight: 3},
	})
	r1 := res.Rounds[0]
	if r1.ExhaustedWeight != 4 || r1.NonExhaustedWeight != 6 {
		t.Fatalf("空排序应首轮耗尽: %+v", r1)
	}
}

func TestAllExhaustedNoWinner(t *testing.T) {
	res := runCase(t, []string{"A", "B"}, []ballot{
		{ranking: []string{}, weight: 5},
	})
	if res.Winner != nil {
		t.Fatalf("无人可投应无人当选, got %v", *res.Winner)
	}
	if res.Rounds[0].NonExhaustedWeight != 0 {
		t.Fatalf("未耗尽应为 0")
	}
}

func TestTieEliminatesLexicographicallyGreatest(t *testing.T) {
	// 三人各 1 票，排序均无后续候选：淘汰字节序最大者 zeta，
	// 其票耗尽；下一轮剩余两人仍各 1 票且未耗尽=2，淘汰 zeta 之外较大者 beta，
	// 再下一轮 alpha 获 1 票、未耗尽 1，1 > 0 当选。
	res := runCase(t, []string{"alpha", "beta", "zeta"}, []ballot{
		{ranking: []string{"alpha"}, weight: 1},
		{ranking: []string{"beta"}, weight: 1},
		{ranking: []string{"zeta"}, weight: 1},
	})
	if res.Rounds[0].Eliminated != "zeta" || res.Rounds[1].Eliminated != "beta" {
		t.Fatalf("并列淘汰顺序错误: %q %q", res.Rounds[0].Eliminated, res.Rounds[1].Eliminated)
	}
	if res.Winner == nil || *res.Winner != "alpha" {
		t.Fatalf("alpha 最终应当选, got %+v", res.Winner)
	}
	if res.Rounds[1].ExhaustedWeight != 1 {
		t.Fatalf("zeta 的票应计入累计耗尽")
	}
}

func TestWeightValidation(t *testing.T) {
	cases := map[string]string{
		"0":   "不是正整数",
		"-3":  "不是正整数",
		"1.5": "不是整数",
		"01":  "含前导零",
		"1e3": "不是整数",
	}
	for in, want := range cases {
		if _, reason := parsePositiveInt64(in); !strings.Contains(reason, want) {
			t.Errorf("parsePositiveInt64(%q) 原因=%q, 期望包含 %q", in, reason, want)
		}
	}
	if v, reason := parsePositiveInt64("42"); reason != "" || v != 42 {
		t.Errorf("42 应合法, got %d %q", v, reason)
	}
}
