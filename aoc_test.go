package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	lang "github.com/alligator/advent-of-code-2021-lang/src"
)

func TestFiles(t *testing.T) {
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
		result := test(&ev)
		if !result {
			t.Error(fileName)
		}
	}
}
