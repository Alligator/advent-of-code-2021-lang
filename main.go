package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	lang "github.com/alligator/advent-of-code-2021-lang/src"
)

var benchMode bool

func main() {
	defer func() {
		if r := recover(); r != nil {
			switch e := r.(type) {
			case lang.LexError:
				fmt.Fprintf(os.Stderr, "\n\x1b[91mtokenization error on line %d\x1b[0m\n%s\n", e.Line, e.Msg)
			case lang.ParseError:
				fmt.Fprintf(os.Stderr, "\n\x1b[91mparse error on line %d\x1b[0m\n%s\n", e.Line, e.Msg)
			case lang.RuntimeError:
				fmt.Fprintf(os.Stderr, "\n\x1b[91mruntime error on line %d\x1b[0m\n%s\n", e.Line, e.Msg)
			default:
				panic(r)
			}
			os.Exit(1)
		}
	}()

	dbgLex := flag.Bool("debug-lex", false, "debug lexing")
	dbgAst := flag.Bool("debug-ast", false, "debug ast parsing")
	testMode := flag.Bool("t", false, "run tests")
	flag.BoolVar(&benchMode, "b", false, "benchmark")
	flag.Parse()

	filePath := flag.Arg(0)
	if filePath == "" {
		fmt.Fprintln(os.Stderr, "no file given!")
		return
	}

	f, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	l := lang.NewLexer(strings.TrimSpace(string(f)))

	if *dbgLex {
		line := 0
		lines := strings.Split(strings.TrimSpace(string(f)), "\n")
		for {
			t := l.NextToken()
			tline, tcol := l.GetLineAndCol(t)
			if tline != line {
				line = tline
				if line > 1 {
					fmt.Print("\n\n")
				}
				fmt.Printf("%s\n", lines[line-1])
			}
			fmt.Printf("(%v %d %d) ", t.Tag, tline, tcol)
			if t.Tag == lang.EOF {
				break
			}
		}
		fmt.Print("\n")
		return
	}

	p := lang.NewParser(&l)
	prog := p.Parse()

	if *dbgAst {
		lang.PrettyPrint(&prog)
		return
	}

	ev := lang.NewEvaluator(&prog, &l)

	if *testMode {
		test(&ev)
	} else {
		run(&ev)
	}
}

func test(ev *lang.Evaluator) bool {
	testInput := ev.EvalSection("test")
	testInput.CheckTagOrPanic(lang.ValStr)
	ev.ReadInput(*testInput.Str)

	oneOk := true
	twoOk := true
	oneOk = testSection(ev, "test_part1", "part1")
	if ev.HasSection("part2") {
		twoOk = testSection(ev, "test_part2", "part2")
	}

	return oneOk && twoOk
}

func testSection(ev *lang.Evaluator, expectedSection string, actualSection string) bool {
	expected := ev.EvalSection(expectedSection)
	actual := evalSection(ev, actualSection)

	res, err := expected.Compare(actual)
	if err != nil {
		panic(err)
	}

	if res {
		fmt.Printf("\x1b[92m✓\x1b[0m %s\n", actualSection)
	} else {
		fmt.Printf("\x1b[91m✗\x1b[0m %s\n  expected %s\n       got %s\n", actualSection, expected.Repr(), actual.Repr())
	}

	return res
}

func run(ev *lang.Evaluator) {
	f := ev.EvalSection("file")
	if f.Tag != lang.ValStr {
		panic("file section must evaluate to a string")
	}
	ev.ReadInput(*f.Str)
	fmt.Printf("part1: %s\n", evalSection(ev, "part1").Repr())
	if ev.HasSection("part2") {
		fmt.Printf("part2: %s\n", evalSection(ev, "part2").Repr())
	}
}

func evalSection(ev *lang.Evaluator, name string) lang.Value {
	if benchMode {
		defer timeFunc(name)()
	}
	return ev.EvalSection(name)
}

func timeFunc(name string) func() {
	start := time.Now()
	return func() {
		fmt.Printf("\x1b[93mbench:\x1b[0m %s took %v\n", name, time.Since(start))
	}
}
