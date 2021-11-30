package lang

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type ValueTag uint8

const (
	ValNil ValueTag = iota
	ValStr
	ValNum
	ValObj
	ValNativeFn
)

type Value struct {
	Tag      ValueTag
	Str      *string
	Num      *int
	Obj      *map[string]Value
	NativeFn func(Value) Value
}

var Nil = Value{Tag: ValNil}

func (v Value) String() string {
	switch v.Tag {
	case ValNil:
		return "nil"
	case ValStr:
		return "'" + *v.Str + "'"
	case ValNum:
		return strconv.Itoa(*v.Num)
	case ValObj:
		var sb strings.Builder
		sb.WriteString("{ ")
		for key, val := range *v.Obj {
			sb.WriteString(key)
			sb.WriteString(": ")
			sb.WriteString(val.String())
			sb.WriteString(", ")
		}
		sb.WriteString("\b\b }")
		return sb.String()
	default:
		return fmt.Sprintf("<unknown> %#v\n", v)
	}
}

type Env struct {
	parent *Env
	vars   map[string]Value
}

type Evaluator struct {
	sections map[string]Section
	env      *Env
}

func newEvaluator() Evaluator {
	env := Env{vars: make(map[string]Value)}
	ev := Evaluator{
		env:      &env,
		sections: make(map[string]Section),
	}

	ev.setEnv("print", Value{Tag: ValNativeFn, NativeFn: nativePrint})
	ev.setEnv("num", Value{Tag: ValNativeFn, NativeFn: nativeNum})
	return ev
}

func (ev *Evaluator) pushEnv() {
	env := ev.env
	newEnv := Env{vars: make(map[string]Value)}
	newEnv.parent = env
	ev.env = &newEnv
}

func (ev *Evaluator) popEnv() {
	if ev.env.parent == nil {
		panic("attempted to pop last env")
	}
	ev.env = ev.env.parent
}

func (ev *Evaluator) setEnv(name string, val Value) {
	ev.env.vars[name] = val
}

func (ev *Evaluator) updateEnv(name string, val Value) {
	env := ev.env
	for env != nil {
		_, present := env.vars[name]
		if present {
			env.vars[name] = val
			return
		}
		env = env.parent
	}
}

func (ev *Evaluator) find(name string) *Value {
	env := ev.env
	for env != nil {
		val, present := env.vars[name]
		if present {
			return &val
		}
		env = env.parent
	}
	return &Nil
}

func (ev *Evaluator) readInputFile() {
	fileName := ev.evalSection("file")
	if fileName.Tag != ValStr {
		panic("file section must be a string")
	}

	f, err := os.ReadFile(*fileName.Str)
	if err != nil {
		panic(err)
	}

	file := string(f)
	mapOfLines := make(map[string]Value)

	for index, line := range strings.Split(file, "\n") {
		l := line
		mapOfLines[strconv.Itoa(index)] = Value{Tag: ValStr, Str: &l}
	}

	ev.setEnv("input", Value{Tag: ValStr, Str: &file})
	ev.setEnv("lines", Value{Tag: ValObj, Obj: &mapOfLines})
}

func (ev *Evaluator) evalProgram(prog *Program) {
	// read all the sections
	for _, section := range prog.sections {
		name := section.getName()
		ev.sections[name] = section
	}

	// read the file
	ev.readInputFile()
	ev.evalSection("part1")
	ev.evalSection("part2")
}

func (ev *Evaluator) evalSection(name string) Value {
	section := ev.sections[name]
	switch node := section.(type) {
	case *SectionBlock:
		ev.pushEnv()
		for _, stmt := range node.block {
			ev.evalStmt(&stmt)
		}
		ev.popEnv()
	case *SectionExpr:
		return ev.evalExpr(&node.expression)
	}

	return Nil
}

func (ev *Evaluator) evalExpr(expr *Expr) Value {
	switch node := (*expr).(type) {
	case *ExprString:
		return Value{Tag: ValStr, Str: &node.str}
	case *ExprNum:
		return Value{Tag: ValNum, Num: &node.num}
	case *ExprIdentifier:
		v := ev.find(node.identifier)
		return *v
	case *ExprFuncall:
		fnVal := ev.evalExpr(&node.identifier)
		if fnVal.Tag != ValNativeFn {
			fmt.Printf("%#v\n", fnVal)
			panic("attempted to call non function")
		}
		arg := ev.evalExpr(&node.arg)
		return fnVal.NativeFn(arg)
	case *ExprBinary:
		switch node.op {
		case Equal:
			lhs, ok := node.lhs.(*ExprIdentifier)
			if !ok {
				panic("lhs of an expression must be an identifier")
			}
			ident := lhs.identifier
			val := ev.evalExpr(&node.rhs)
			ev.updateEnv(ident, val)
			return val
		case Plus:
			lhs := ev.evalExpr(&node.lhs)
			rhs := ev.evalExpr(&node.rhs)
			if lhs.Tag != ValNum || rhs.Tag != ValNum {
				panic("+ is only supported for numbers")
			}
			result := *lhs.Num + *rhs.Num
			return Value{Tag: ValNum, Num: &result}
		default:
			panic(fmt.Sprintf("unknown operator %s\n", node.op))
		}
	default:
		panic(fmt.Sprintf("unhandled expression type %#v\n", node))
	}
	// return Nil
}

func (ev *Evaluator) evalStmt(stmt *Stmt) {
	switch node := (*stmt).(type) {
	case *StmtVar:
		ident := node.identifier
		val := ev.evalExpr(&node.value)
		ev.setEnv(ident, val)
	case *StmtFor:
		ident := node.identifier
		val := ev.evalExpr(&node.value)
		if val.Tag != ValObj {
			panic("expected a obj in for loop")
		}
		ev.pushEnv()
		for _, val := range *val.Obj {
			ev.setEnv(ident, val)
			for _, stmt := range node.body {
				ev.evalStmt(&stmt)
			}
		}
		ev.popEnv()
	case *StmtExpr:
		ev.evalExpr(&node.expr)
	default:
		panic(fmt.Sprintf("unhandled statement type %#v\n", node))
	}
}

func Eval(prog *Program) {
	ev := newEvaluator()
	ev.evalProgram(prog)
}
