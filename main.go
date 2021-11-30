package main

import (
	"flag"
	"fmt"
	"os"

	lang "github.com/alligator/advent-of-code-2021-lang/src"
)

func main() {
	dbgLex := flag.Bool("debug-lex", false, "debug lexing")
	dbgAst := flag.Bool("debug-ast", false, "debug ast parsing")
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

	lang.Eval(&prog)
}
