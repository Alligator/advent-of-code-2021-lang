package lang

import (
	"fmt"
	"strconv"
	"strings"
)

type ValueTag uint8

const (
	ValNil ValueTag = iota
	ValStr
	ValNum
	ValArray
	ValNativeFn
)

func (t ValueTag) String() string {
	return []string{"nil", "string", "num", "array", "nativeFn"}[t]
}

type Value struct {
	Tag      ValueTag
	Str      *string
	Num      *int
	Array    *[]Value
	NativeFn func([]Value) Value
}

var Nil = Value{Tag: ValNil}

func (v Value) Repr() string {
	switch v.Tag {
	case ValNil:
		return "nil"
	case ValStr:
		return "'" + *v.Str + "'"
	case ValNum:
		return strconv.Itoa(*v.Num)
	case ValArray:
		var sb strings.Builder
		sb.WriteString("[ ")
		for _, val := range *v.Array {
			sb.WriteString(val.Repr())
			sb.WriteString(", ")
		}
		sb.WriteString("\b\b ]")
		return sb.String()
	default:
		return fmt.Sprintf("<unknown> %#v\n", v)
	}
}

func (v Value) String() string {
	switch v.Tag {
	case ValNil:
		return "nil"
	case ValStr:
		return *v.Str
	case ValNum:
		return strconv.Itoa(*v.Num)
	case ValArray:
		var sb strings.Builder
		sb.WriteString("[ ")
		for _, val := range *v.Array {
			sb.WriteString(val.Repr())
			sb.WriteString(", ")
		}
		sb.WriteString("\b\b ]")
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

func (v Value) CheckTagOrPanic(expectedTag ValueTag) {
	if v.Tag != expectedTag {
		panic(fmt.Errorf("expected a %s but found a %s", expectedTag.String(), v.Tag.String()))
	}
}

func (v Value) Compare(b Value) (bool, error) {
	switch {
	case v.Tag == ValNum && b.Tag == ValNum:
		return *v.Num == *b.Num, nil
	case v.Tag == ValStr && b.Tag == ValStr:
		return *v.Str == *b.Str, nil
	}
	return false, fmt.Errorf("cannot compare %s and %s", v.Tag.String(), b.Tag.String())
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

func NewEvaluator(prog *Program) Evaluator {
	env := Env{vars: make(map[string]Value)}
	ev := Evaluator{
		env:      &env,
		sections: make(map[string]Section),
	}

	ev.setEnv("print", Value{Tag: ValNativeFn, NativeFn: nativePrint})
	ev.setEnv("num", Value{Tag: ValNativeFn, NativeFn: nativeNum})
	ev.setEnv("read", Value{Tag: ValNativeFn, NativeFn: nativeRead})
	ev.setEnv("split", Value{Tag: ValNativeFn, NativeFn: nativeSplit})
	ev.setEnv("len", Value{Tag: ValNativeFn, NativeFn: nativeLen})

	ev.evalProgram(prog)
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

func (ev *Evaluator) ReadInput(input string) {
	lines := make([]Value, 0)

	for _, line := range strings.Split(input, "\n") {
		l := line
		lines = append(lines, Value{Tag: ValStr, Str: &l})
	}

	ev.setEnv("input", Value{Tag: ValStr, Str: &input})
	ev.setEnv("lines", Value{Tag: ValArray, Array: &lines})
}

func (ev *Evaluator) evalProgram(prog *Program) {
	// read all the sections
	for _, section := range prog.sections {
		name := section.getName()
		ev.sections[name] = section
	}
}

func (ev *Evaluator) EvalSection(name string) (retVal Value) {
	retVal = Nil

	defer func() {
		if r := recover(); r != nil {
			switch e := r.(type) {
			case Value:
				// unwind the stack
				env := ev.env
				for env.parent != nil {
					env = env.parent
				}
				ev.env = env
				ev.section = nil
				retVal = e
			default:
				panic(r)
			}
		}
	}()

	if ev.section != nil {
		panic("cannot nest sections")
	}

	section, preset := ev.sections[name]
	if !preset {
		panic(fmt.Errorf("couldn't find section %s", name))
	}
	switch node := section.(type) {
	case *SectionBlock:
		ev.section = &section
		ev.evalBlock(node.block)
		ev.section = nil
	case *SectionExpr:
		return ev.evalExpr(&node.expression)
	}

	return retVal
}

func (ev *Evaluator) HasSection(name string) bool {
	_, present := ev.sections[name]
	return present
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
		args := make([]Value, 0)
		for _, arg := range node.args {
			args = append(args, ev.evalExpr(&arg))
		}
		return fnVal.NativeFn(args)
	case *ExprBinary:
		return ev.evalBinaryExpr(node)
	case *ExprArray:
		items := make([]Value, 0)
		for _, itemExpr := range node.items {
			items = append(items, ev.evalExpr(&itemExpr))
		}
		return Value{Tag: ValArray, Array: &items}
	default:
		panic(fmt.Sprintf("unhandled expression type %#v\n", node))
	}
}

func (ev *Evaluator) evalBinaryExpr(expr *ExprBinary) Value {
	if expr.op == Equal {
		switch node := expr.lhs.(type) {
		case *ExprIdentifier:
			ident := node.identifier
			val := ev.evalExpr(&expr.rhs)
			ev.updateEnv(ident, val)
			return val
		case *ExprBinary:
			if node.op != LSquare {
				break
			}
			array := ev.evalExpr(&node.lhs)
			if array.Tag != ValArray {
				panic(fmt.Errorf("%v is not subscriptable", array.Tag))
			}
			index := ev.evalExpr(&node.rhs)
			if index.Tag != ValNum {
				panic("attempt to subscript with non-numeric value")
			}
			val := ev.evalExpr(&expr.rhs)
			(*array.Array)[*index.Num] = val
			return val
		}
		panic("lhs of assignment is not assignable")
	}

	lhs := ev.evalExpr(&expr.lhs)
	rhs := ev.evalExpr(&expr.rhs)

	switch expr.op {
	case Plus:
		if lhs.Tag != ValNum || rhs.Tag != ValNum {
			panic("+ is only supported for numbers")
		}
		result := *lhs.Num + *rhs.Num
		return Value{Tag: ValNum, Num: &result}
	case Minus:
		if lhs.Tag != ValNum || rhs.Tag != ValNum {
			panic("- is only supported for numbers")
		}
		result := *lhs.Num - *rhs.Num
		return Value{Tag: ValNum, Num: &result}
	case Star:
		if lhs.Tag != ValNum || rhs.Tag != ValNum {
			panic("* is only supported for numbers")
		}
		result := *lhs.Num * *rhs.Num
		return Value{Tag: ValNum, Num: &result}
	case Slash:
		if lhs.Tag != ValNum || rhs.Tag != ValNum {
			panic("/ is only supported for numbers")
		}
		result := *lhs.Num / *rhs.Num
		return Value{Tag: ValNum, Num: &result}
	case EqualEqual:
		result, err := lhs.Compare(rhs)
		if err != nil {
			panic(err)
		}
		num := 0
		if result {
			num = 1
		}
		return Value{Tag: ValNum, Num: &num}
	case Greater, GreaterEqual, Less:
		switch {
		case lhs.Tag == ValNum && rhs.Tag == ValNum:
			result := false
			switch expr.op {
			case Greater:
				result = *lhs.Num > *rhs.Num
			case GreaterEqual:
				result = *lhs.Num >= *rhs.Num
			case Less:
				result = *lhs.Num < *rhs.Num
			}
			num := 0
			if result {
				num = 1
			}
			return Value{Tag: ValNum, Num: &num}
		}
		panic(fmt.Errorf("cannot compare %v and %v", lhs.Tag, rhs.Tag))
	case LSquare:
		if lhs.Tag != ValArray || rhs.Tag != ValNum {
			panic(fmt.Errorf("cannot subscript a %v with a %v", lhs.Tag, rhs.Tag))
		}
		return (*lhs.Array)[*rhs.Num]
	default:
		panic(fmt.Errorf("unknown operator %s", expr.op))
	}
}

func (ev *Evaluator) evalStmt(stmt *Stmt) {
	switch node := (*stmt).(type) {
	case *StmtVar:
		ident := node.identifier
		val := ev.evalExpr(&node.value)
		ev.setEnv(ident, val)
	case *StmtFor:
		ev.forLoop(node)
	case *StmtExpr:
		ev.evalExpr(&node.expr)
	case *StmtIf:
		val := ev.evalExpr(&node.condition)
		if val.isTruthy() {
			ev.evalBlock(node.body)
		} else if len(node.elseBody) > 0 {
			ev.evalBlock(node.elseBody)
		}
	case *StmtReturn:
		val := ev.evalExpr(&node.value)
		panic(val) // control flow panic
	case *StmtContinue:
		panic(*node) // control flow panic
	case *StmtMatch:
		ev.match(node)
	default:
		panic(fmt.Sprintf("unhandled statement type %#v\n", node))
	}
}

func (ev *Evaluator) match(match *StmtMatch) {
	candidate := ev.evalExpr(&match.value)

MatchLoop:
	for _, c := range match.cases {
		switch pattern := c.cond.(type) {
		case *ExprArray:
			if candidate.Tag != ValArray {
				continue
			}

			vars := make(map[string]Value)
			for index, item := range pattern.items {
				if index >= len(*candidate.Array) {
					continue MatchLoop
				}

				switch itemNode := item.(type) {
				case *ExprIdentifier:
					vars[itemNode.identifier] = (*candidate.Array)[index]
				default:
					itemVal := ev.evalExpr(&item)
					result, err := (*candidate.Array)[index].Compare(itemVal)
					if err != nil {
						panic(err)
					}
					if !result {
						continue MatchLoop
					}
				}

			}

			// we found a match
			ev.pushEnv()
			for k, v := range vars {
				ev.env.vars[k] = v
			}

			for _, stmt := range c.body {
				ev.evalStmt(&stmt)
			}
			ev.popEnv()
			return
		default:
			panic(fmt.Errorf("unsupported match type %v", c.cond))
		}
	}
}

func (ev *Evaluator) forLoop(node *StmtFor) {
	val := ev.evalExpr(&node.value)
	if val.Tag != ValArray {
		panic("expected an array in for loop")
	}
	ev.pushEnv()
	for index, val := range *val.Array {
		ev.runForLoopBody(node, val, index)
	}
	ev.popEnv()
}

func (ev *Evaluator) runForLoopBody(node *StmtFor, val Value, index int) {
	// run the body of a for loop, handling continue statements
	defer catchContinue(ev, ev.env)

	ev.setEnv(node.identifier, val)
	if node.indexIdentifier != "" {
		ev.setEnv(node.indexIdentifier, Value{Tag: ValNum, Num: &index})
	}

	for _, stmt := range node.body {
		ev.evalStmt(&stmt)
	}
}

func catchContinue(ev *Evaluator, rootEnv *Env) {
	if r := recover(); r != nil {
		switch r.(type) {
		case StmtContinue:
			ev.env = rootEnv
			return
		default:
			panic(r)
		}
	}
}
