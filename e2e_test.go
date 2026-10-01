package main

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const sampleElection = `{
  "candidates": ["A", "B", "C"],
  "ballots": [
    {"weight": 4, "ranking": ["A"]},
    {"weight": 3, "ranking": ["B", "A"]},
    {"weight": 2, "ranking": ["C"]},
    {"weight": 1, "ranking": ["C", "A"]},
    {"weight": 0, "ranking": ["A"]},
    {"weight": 2, "ranking": ["A", "A"]},
    {"weight": 2, "ranking": ["X"]},
    {"weight": 2, "ranking": []}
  ]
}`

func TestEndToEndInvalidBallotsAndOutputs(t *testing.T) {
	dir := t.TempDir()
	inPath := filepath.Join(dir, "election.json")
	if err := os.WriteFile(inPath, []byte(sampleElection), 0o644); err != nil {
		t.Fatal(err)
	}

	in, err := parseElectionFile(inPath)
	if err != nil {
		t.Fatalf("解析不应失败: %v", err)
	}
	if len(in.valid) != 5 {
		t.Fatalf("有效票应为 5, got %d", len(in.valid))
	}
	if len(in.invalid) != 3 {
		t.Fatalf("无效票应为 3, got %+v", in.invalid)
	}
	for _, ib := range in.invalid {
		if ib.Reason == "" {
			t.Fatalf("无效票必须给出原因: %+v", ib)
		}
	}

	res := runElection(in)
	outDir := filepath.Join(dir, "out")
	if err := writeOutputs(outDir, res); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(outDir, "result.json"))
	if err != nil {
		t.Fatal(err)
	}
	var loaded Result
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("result.json 非法: %v", err)
	}
	if loaded.ValidBallots != 5 || loaded.InvalidBallots != 3 || loaded.TotalValidWeight != 12 {
		t.Fatalf("汇总数字错误: %+v", loaded)
	}

	cf, err := os.Open(filepath.Join(outDir, "rounds.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer cf.Close()
	rows, err := csv.NewReader(cf).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != loaded.RoundsCount+1 {
		t.Fatalf("CSV 行数应为轮数+1, got %d", len(rows))
	}
}

func TestGlobalStructuralErrors(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"not json":     "{",
		"缺 candidates": `{"ballots": []}`,
		"候选为空":         `{"candidates": [], "ballots": []}`,
		"候选重复":         `{"candidates": ["A", "A"], "ballots": []}`,
		"候选非 ASCII":    `{"candidates": ["好"], "ballots": []}`,
		"缺 ballots":    `{"candidates": ["A"]}`,
		"选票缺 weight":   `{"candidates": ["A"], "ballots": [{"ranking": []}]}`,
		"选票缺 ranking":  `{"candidates": ["A"], "ballots": [{"weight": 1}]}`,
		"ranking 类型错误": `{"candidates": ["A"], "ballots": [{"weight": 1, "ranking": "A"}]}`,
	}
	for name, content := range cases {
		p := filepath.Join(dir, "x.json")
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := parseElectionFile(p); err == nil {
			t.Errorf("%s: 期望结构错误", name)
		}
	}
}
