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

func (v Value) isTruthy() bool {
	switch v.Tag {
	case ValNum:
		return *v.Num != 0
	}
	return false
}

func compare(a Value, b Value) (bool, error) {
	switch {
	case a.Tag == ValNum && b.Tag == ValNum:
		return *a.Num == *b.Num, nil
	}
	return false, fmt.Errorf("cannot compare %v and %v", a.Tag, b.Tag)
}

type Env struct {
	parent *Env
	vars   map[string]Value
}

type Evaluator struct {
	sections map[string]Section
	env      *Env
	section  *Section
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

func (ev *Evaluator) find(name string) (*Value, bool) {
	env := ev.env
	for env != nil {
		val, present := env.vars[name]
		if present {
			return &val, true
		}
		env = env.parent
	}
	return &Nil, false
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
	defer ev.handleSectionReturn()

	if ev.section != nil {
		panic("cannot nest sections")
	}

	section := ev.sections[name]
	switch node := section.(type) {
	case *SectionBlock:
		ev.section = &section
		ev.evalBlock(node.block)
		ev.section = nil
	case *SectionExpr:
		return ev.evalExpr(&node.expression)
	}

	return Nil
}

func (ev *Evaluator) handleSectionReturn() {
	if r := recover(); r != nil {
		switch e := r.(type) {
		case Value:
			// unwind the stack
			env := ev.env
			for env.parent != nil {
				env = env.parent
			}
			ev.env = env
			fmt.Printf("%s returned %s\n", (*ev.section).getName(), e.String())
			ev.section = nil
		default:
			panic(r)
		}
	}
}

func (ev *Evaluator) evalBlock(block []Stmt) {
	ev.pushEnv()
	for _, stmt := range block {
		ev.evalStmt(&stmt)
	}
	ev.popEnv()
}

func (ev *Evaluator) evalExpr(expr *Expr) Value {
	switch node := (*expr).(type) {
	case *ExprString:
		return Value{Tag: ValStr, Str: &node.str}
	case *ExprNum:
		return Value{Tag: ValNum, Num: &node.num}
	case *ExprIdentifier:
		v, ok := ev.find(node.identifier)
		if !ok {
			panic(fmt.Sprintf("unknown variable %s", node.identifier))
		}
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
		case Star:
			lhs := ev.evalExpr(&node.lhs)
			rhs := ev.evalExpr(&node.rhs)
			if lhs.Tag != ValNum || rhs.Tag != ValNum {
				panic("* is only supported for numbers")
			}
			result := *lhs.Num * *rhs.Num
			return Value{Tag: ValNum, Num: &result}
		case EqualEqual:
			lhs := ev.evalExpr(&node.lhs)
			rhs := ev.evalExpr(&node.rhs)
			result, err := compare(lhs, rhs)
			if err != nil {
				panic(err)
			}
			num := 0
			if result {
				num = 1
			}
			return Value{Tag: ValNum, Num: &num}
		default:
			panic(fmt.Sprintf("unknown operator %s\n", node.op))
		}
	default:
		panic(fmt.Sprintf("unhandled expression type %#v\n", node))
	}
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
			ev.evalBlock(node.body)
		}
		ev.popEnv()
	case *StmtExpr:
		ev.evalExpr(&node.expr)
	case *StmtIf:
		val := ev.evalExpr(&node.condition)
		if val.isTruthy() {
			ev.evalBlock(node.body)
		}
	case *StmtReturn:
		val := ev.evalExpr(&node.value)
		panic(val) // control flow panic
	default:
		panic(fmt.Sprintf("unhandled statement type %#v\n", node))
	}
}

func Eval(prog *Program) {
	ev := newEvaluator()
	ev.evalProgram(prog)
}
