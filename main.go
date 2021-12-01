package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	lang "github.com/alligator/advent-of-code-2021-lang/src"
)

var benchMode bool

func main() {
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

	l := lang.NewLexer(string(f))

	if *dbgLex {
		for {
			t := l.NextToken()
			fmt.Printf("%v\n", t)
			if t.Tag == lang.EOF {
				break
			}
		}
		return
	}

	p := lang.NewParser(&l)
	prog := p.Parse()

	if *dbgAst {
		lang.PrettyPrint(&prog)
		return
	}

	ev := lang.NewEvaluator(&prog)

	if *testMode {
		test(&ev)
	} else {
		run(&ev)
	}
}

func test(ev *lang.Evaluator) {
	testInput := ev.EvalSection("test")
	testInput.CheckTagOrPanic(lang.ValStr)
	ev.ReadInput(*testInput.Str)
	testSection(ev, "test_part1", "part1")
	testSection(ev, "test_part2", "part2")
}

func testSection(ev *lang.Evaluator, expectedSection string, actualSection string) {
	expected := ev.EvalSection(expectedSection)
	actual := evalSection(ev, actualSection)

	res, err := expected.Compare(actual)
	if err != nil {
		panic(err)
	}

	if res {
		fmt.Printf("\x1b[92m✓\x1b[0m %s\n", actualSection)
	} else {
		fmt.Printf("\x1b[91m✗\x1b[0m %s\n  expected %s\n       got %s\n", actualSection, expected.String(), actual.String())
	}
}

func run(ev *lang.Evaluator) {
	f := ev.EvalSection("file")
	if f.Tag != lang.ValStr {
		panic("file section must evaluate to a string")
	}
	ev.ReadInput(*f.Str)
	fmt.Printf("part1: %s\n", evalSection(ev, "part1").String())
	fmt.Printf("part2: %s\n", evalSection(ev, "part2").String())
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
