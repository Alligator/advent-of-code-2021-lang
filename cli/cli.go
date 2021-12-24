package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	lang "github.com/alligator/advent-of-code-2021-lang/src"
)

func Run() (exitCode int) {
	dbgLex := flag.Bool("debug-lex", false, "debug lexing")
	dbgAst := flag.Bool("debug-ast", false, "debug ast parsing")
	testMode := flag.Bool("t", false, "run tests")
	benchMode := flag.Bool("b", false, "benchmark")
	flag.Parse()

	filePath := flag.Arg(0)
	if filePath == "" {
		fmt.Fprintln(os.Stderr, "no file given!")
		return 1
	}

	f, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	src := strings.TrimSpace(string(f))
	defer handleErrors(&exitCode)

	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(lang.Error); ok {
				fmt.Fprintf(os.Stderr, "\n\x1b[91m%s on line %d\x1b[0m\n%s\n", e.Tag.String(), e.Line, e.Msg)
				exitCode = 1
				return
			}
			panic(r)
		}
	}()

	if *dbgLex {
		debugLex(src)
		return 0
	}

	l := lang.NewLexer(src)
	p := lang.NewParser(&l)
	prog := p.Parse()

	if *dbgAst {
		lang.PrettyPrint(&prog)
		return 0
	}

	ev := lang.NewEvaluator(&prog, &l)

	if *testMode {
		Test(&ev, *benchMode)
	} else {
		run(&ev, *benchMode)
	}

	return exitCode
}

func debugLex(src string) {
	l := lang.NewLexer(src)
	line := 0
	lines := strings.Split(src, "\n")
	for {
		t, err := l.NextToken()
		if err != nil {
			panic(err)
		}
		tline, _ := l.GetLineAndCol(t)
		if tline != line {
			line = tline
			if line > 1 {
				fmt.Print("\n")
			}
			fmt.Printf("line %2d: \"%s\"\n     %2d: ", line, lines[line-1], line)
		}

		if t.Len == 0 {
			fmt.Printf("%s ", t.Tag)
		} else {
			fmt.Printf("%s(%#v) ", t.Tag, l.GetString(t))
		}

		if t.Tag == lang.EOF {
			break
		}
	}
	fmt.Print("\n")
}

func Test(ev *lang.Evaluator, benchMode bool) bool {
	testInput, err := ev.EvalSection("test")
	if err != nil {
		panic(err)
	}
	testInput.CheckTagOrPanic(lang.ValStr)
	ev.ReadInput(*testInput.Str)

	oneOk := true
	twoOk := true
	oneOk = testSection(ev, "test_part1", "part1", benchMode)
	if ev.HasSection("part2") {
		twoOk = testSection(ev, "test_part2", "part2", benchMode)
	}

	return oneOk && twoOk
}

func testSection(ev *lang.Evaluator, expectedSection string, actualSection string, benchMode bool) bool {
	expected, err := ev.EvalSection(expectedSection)
	if err != nil {
		panic(err)
	}
	actual := evalSection(ev, actualSection, benchMode)

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

func run(ev *lang.Evaluator, benchMode bool) {
	f, err := ev.EvalSection("file")
	if err != nil {
		panic(err)
	}
	if f.Tag != lang.ValStr {
		panic("file section must evaluate to a string")
	}
	ev.ReadInput(*f.Str)
	fmt.Printf("part1: %s\n", evalSection(ev, "part1", benchMode).Repr())
	if ev.HasSection("part2") {
		fmt.Printf("part2: %s\n", evalSection(ev, "part2", benchMode).Repr())
	}
}

func evalSection(ev *lang.Evaluator, name string, benchMode bool) lang.Value {
	if benchMode {
		defer timeFunc(name)()
	}
	v, err := ev.EvalSection(name)
	if err != nil {
		panic(err)
	}
	return v
}

func timeFunc(name string) func() {
	start := time.Now()
	return func() {
		fmt.Printf("\x1b[93mbench:\x1b[0m %s took %0.3fs\n", name, time.Since(start).Seconds())
	}
}

func handleErrors(exitCode *int) {
	if r := recover(); r != nil {
		if e, ok := r.(lang.Error); ok {
			fmt.Fprintf(os.Stderr, "\n\x1b[91m%s on line %d\x1b[0m\n%s\n", e.Tag.String(), e.Line, e.Msg)
			code := 1
			exitCode = &code
			return
		}
		panic(r)
	}
}
