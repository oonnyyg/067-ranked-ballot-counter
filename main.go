package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
)

func main() {
	input := flag.String("input", "", "选举输入 JSON 文件路径")
	out := flag.String("out", "", "输出目录")
	flag.Parse()
	if *input == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "用法: go run . --input 选举.json --out 输出目录")
		os.Exit(2)
	}
	if err := run(*input, *out); err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
}

func run(inputPath, outDir string) error {
	in, err := LoadInput(inputPath)
	if err != nil {
		return err // 全局结构错误：退出非零，不生成结果
	}
	valid, rejected := ValidateBallots(in)
	res := RunTally(in.Candidates, valid)
	cands := append([]string(nil), in.Candidates...)
	sort.Strings(cands)
	if rejected == nil {
		rejected = []RejectedBallot{}
	}
	rep := &Report{
		Candidates:       cands,
		ValidBallots:     res.ValidBallots,
		InvalidBallots:   len(rejected),
		Rejected:         rejected,
		TotalValidWeight: res.TotalWeight,
		Elected:          res.Elected,
		Winner:           res.Winner,
		Rounds:           res.Rounds,
	}
	if err := WriteOutputs(outDir, rep); err != nil {
		return err
	}
	if res.Elected {
		fmt.Printf("计票完成：%s 当选，共 %d 轮，结果已写入 %s\n", res.Winner, len(res.Rounds), outDir)
	} else {
		fmt.Printf("计票完成：无人当选（未耗尽权重为零），结果已写入 %s\n", outDir)
	}
	return nil
}
