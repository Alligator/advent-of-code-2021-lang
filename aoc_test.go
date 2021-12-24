package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	lang "github.com/alligator/advent-of-code-2021-lang/src"
	cli "github.com/alligator/advent-of-code-2021-lang/cli"
)

func TestFiles(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			log.Fatal(r)
		}
	}()

	files, err := filepath.Glob("tests/*.aoc")
	if err != nil {
		panic(err)
	}

	for _, fileName := range files {
		f, err := os.ReadFile(fileName)
		if err != nil {
			panic(err)
		}
		l := lang.NewLexer(strings.TrimSpace(string(f)))
		p := lang.NewParser(&l)
		prog := p.Parse()
		ev := lang.NewEvaluator(&prog, &l)
		result := cli.Test(&ev, false)
		if !result {
			t.Error(fileName)
		}
	}
}
