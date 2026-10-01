package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTallyBasicElimination(t *testing.T) {
	cands := []string{"A", "B", "C"}
	ballots := []Ballot{
		{Weight: 5, Ranking: []string{"A"}},
		{Weight: 4, Ranking: []string{"B", "C"}},
		{Weight: 2, Ranking: []string{"C"}},
	}
	res := RunTally(cands, ballots)
	if len(res.Rounds) != 2 {
		t.Fatalf("期望 2 轮，实际 %d", len(res.Rounds))
	}
	r1 := res.Rounds[0]
	if r1.Tallies["A"] != 5 || r1.Tallies["B"] != 4 || r1.Tallies["C"] != 2 {
		t.Fatalf("首轮得票错误: %+v", r1.Tallies)
	}
	if r1.Eliminated != "C" {
		t.Fatalf("首轮应淘汰 C，实际 %q", r1.Eliminated)
	}
	if r1.Transfers["EXHAUSTED"] != 2 {
		t.Fatalf("C 的 2 票应转入耗尽池: %+v", r1.Transfers)
	}
	if !res.Elected || res.Winner != "A" {
		t.Fatalf("A 应在第二轮当选: %+v", res)
	}
	if res.Rounds[1].ActiveWeight != 9 || res.Rounds[1].ExhaustedWeight != 2 {
		t.Fatalf("第二轮权重错误: %+v", res.Rounds[1])
	}
}

func TestTallyTieBreaksLexicographicMax(t *testing.T) {
	cands := []string{"A", "B", "C"}
	ballots := []Ballot{
		{Weight: 3, Ranking: []string{"A"}},
		{Weight: 2, Ranking: []string{"B"}},
		{Weight: 2, Ranking: []string{"C"}},
	}
	res := RunTally(cands, ballots)
	// B、C 并列最低，淘汰字典序最大者 C。
	if res.Rounds[0].Eliminated != "C" {
		t.Fatalf("并列时应淘汰字典序最大者 C，实际 %q", res.Rounds[0].Eliminated)
	}
}

func TestTallyEmptyRankingExhausted(t *testing.T) {
	cands := []string{"A"}
	ballots := []Ballot{
		{Weight: 1, Ranking: []string{}},
	}
	res := RunTally(cands, ballots)
	if res.Elected {
		t.Fatalf("全部耗尽时不应有人当选")
	}
	if res.Rounds[0].ActiveWeight != 0 || res.Rounds[0].ExhaustedWeight != 1 {
		t.Fatalf("空排序票应从首轮起耗尽: %+v", res.Rounds[0])
	}
}

func TestValidateBallots(t *testing.T) {
	in := &Input{
		Candidates: []string{"A", "B"},
		Ballots: []ballotJSON{
			{Weight: "2", Ranking: []string{"A"}},
			{Weight: "0", Ranking: []string{"A"}},
			{Weight: "1", Ranking: []string{"A", "A"}},
			{Weight: "1", Ranking: []string{"Z"}},
			{Weight: "1.5", Ranking: []string{"A"}},
		},
	}
	valid, rejected := ValidateBallots(in)
	if len(valid) != 1 || valid[0].Weight != 2 {
		t.Fatalf("有效票错误: %+v", valid)
	}
	if len(rejected) != 4 {
		t.Fatalf("应有 4 张无效票: %+v", rejected)
	}
	if rejected[0].Index != 1 || rejected[1].Index != 2 || rejected[2].Index != 3 || rejected[3].Index != 4 {
		t.Fatalf("无效票原始序号错误: %+v", rejected)
	}
}

func TestLoadInputStructuralErrors(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte(`{"candidates":[],"ballots":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadInput(bad); err == nil {
		t.Fatal("空候选列表应报结构错误")
	}
	dup := filepath.Join(dir, "dup.json")
	if err := os.WriteFile(dup, []byte(`{"candidates":["A","A"],"ballots":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadInput(dup); err == nil {
		t.Fatal("重复候选应报结构错误")
	}
}

func TestDeterministicOutput(t *testing.T) {
	cands := []string{"B", "A", "C"}
	ballots := []Ballot{
		{Weight: 5, Ranking: []string{"C", "A"}},
		{Weight: 3, Ranking: []string{"A"}},
		{Weight: 3, Ranking: []string{"B"}},
	}
	r1 := RunTally(cands, ballots)
	r2 := RunTally(cands, ballots)
	if len(r1.Rounds) != len(r2.Rounds) || r1.Winner != r2.Winner {
		t.Fatal("重复运行结果不一致")
	}
}
