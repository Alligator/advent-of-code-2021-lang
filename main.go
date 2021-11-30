package main

import (
	"os"

	lang "github.com/alligator/advent-of-code-2021-lang/src"
)

func main() {
	f, err := os.ReadFile("test.aoc")
	if err != nil {
		panic(err)
	}

	l := lang.NewLexer(string(f))
	p := lang.NewParser(&l)
	prog := p.Parse()
	// lang.PrettyPrint(&prog)
	lang.Eval(&prog)

	// for {
	// 	t := l.NextToken()
	// 	fmt.Printf("%v\n", t)
	// 	if t.Tag == lang.EOF {
	// 		break
	// 	}
	// }
}
