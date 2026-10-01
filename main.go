package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	inputPath := flag.String("input", "", "选举输入 JSON 文件路径")
	outDir := flag.String("out", "", "结果输出目录")
	flag.Parse()

	if *inputPath == "" || *outDir == "" {
		fmt.Fprintln(os.Stderr, "用法: ranked-ballot-counter --input 选举.json --out 输出目录")
		os.Exit(2)
	}

	in, err := parseElectionFile(*inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "输入错误: %v\n", err)
		os.Exit(1)
	}

	result := runElection(in)
	if err := writeOutputs(*outDir, result); err != nil {
		fmt.Fprintf(os.Stderr, "输出错误: %v\n", err)
		os.Exit(1)
	}

	if result.Winner != nil {
		fmt.Printf("计票完成，共 %d 轮，获胜者: %s\n", result.RoundsCount, *result.Winner)
	} else {
		fmt.Printf("计票完成，共 %d 轮，无人当选\n", result.RoundsCount)
	}
}
