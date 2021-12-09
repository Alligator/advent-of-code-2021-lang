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
	ValFn
)

func (t ValueTag) String() string {
	return []string{"nil", "string", "num", "array", "nativeFn", "fn"}[t]
}

type Value struct {
	Tag      ValueTag
	Str      *string
	Num      *int
	Array    *[]Value
	NativeFn func([]Value) Value
	Fn       *ExprFunc
}

var NilValue = Value{Tag: ValNil}

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
		sb.WriteString("[")
		for index, val := range *v.Array {
			sb.WriteString(val.Repr())
			if index < len(*v.Array)-1 {
				sb.WriteString(", ")
			}
		}
		sb.WriteString("]")
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
		return v.Repr()
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

func (v Value) negate() Value {
	switch v.Tag {
	case ValNum:
		num := *v.Num
		num = num - 1
		if num < 0 {
			num = -num
		}
		return Value{Tag: ValNum, Num: &num}
	}
	return NilValue
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
	case v.Tag == ValNil && b.Tag == ValNil:
		return true, nil
	case v.Tag == ValNil && b.Tag != ValNil, v.Tag != ValNil && b.Tag == ValNil:
		return false, nil
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
	lex      *Lexer
}

func NewEvaluator(prog *Program, lex *Lexer) Evaluator {
	env := Env{vars: make(map[string]Value)}
	ev := Evaluator{
		env:      &env,
		sections: make(map[string]Section),
		lex:      lex,
	}

	ev.setEnv("print", Value{Tag: ValNativeFn, NativeFn: nativePrint})
	ev.setEnv("num", Value{Tag: ValNativeFn, NativeFn: nativeNum})
	ev.setEnv("read", Value{Tag: ValNativeFn, NativeFn: nativeRead})
	ev.setEnv("split", Value{Tag: ValNativeFn, NativeFn: nativeSplit})
	ev.setEnv("len", Value{Tag: ValNativeFn, NativeFn: nativeLen})
	ev.setEnv("push", Value{Tag: ValNativeFn, NativeFn: nativePush})

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
	return &NilValue, false
}

func (ev *Evaluator) fmtError(node Node, format string, args ...interface{}) RuntimeError {
	line, _ := ev.lex.GetLineAndCol(*node.Token())
	msg := fmt.Sprintf(format, args...)
	return RuntimeError{msg, line}
}

func (ev *Evaluator) ReadInput(input string) {
	lines := make([]Value, 0)

	for _, line := range strings.Split(strings.TrimSpace(input), "\n") {
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
	retVal = NilValue

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

func (ev *Evaluator) evalBlock(block Stmt) {
	switch b := block.(type) {
	case *StmtBlock:
		ev.pushEnv()
		for _, stmt := range b.body {
			ev.evalStmt(&stmt)
		}
		ev.popEnv()
	default:
		panic(ev.fmtError(block, "expected a block but found %v", b))
	}
}

func (ev *Evaluator) evalExpr(expr *Expr) Value {
	switch node := (*expr).(type) {
	case *ExprString:
		return Value{Tag: ValStr, Str: &node.str}
	case *ExprNum:
		return Value{Tag: ValNum, Num: &node.num}
	case *ExprNil:
		return NilValue
	case *ExprIdentifier:
		v, ok := ev.find(node.identifier)
		if !ok {
			panic(ev.fmtError(node, "unknown variable %s", node.identifier))
		}
		return *v
	case *ExprFuncall:
		fnVal := ev.evalExpr(&node.identifier)

		args := make([]Value, 0)
		for _, arg := range node.args {
			args = append(args, ev.evalExpr(&arg))
		}

		switch fnVal.Tag {
		case ValNativeFn:
			return fnVal.NativeFn(args)
		case ValFn:
			return ev.fn(fnVal, args)
		}

		panic(ev.fmtError(node, "attempted to call non function"))
	case *ExprFunc:
		fnVal := Value{Tag: ValFn, Fn: node}
		ev.setEnv(node.identifier, fnVal)
		return fnVal
	case *ExprBinary:
		return ev.evalBinaryExpr(node)
	case *ExprArray:
		items := make([]Value, 0)
		for _, itemExpr := range node.items {
			items = append(items, ev.evalExpr(&itemExpr))
		}
		return Value{Tag: ValArray, Array: &items}
	default:
		panic(ev.fmtError(node, "unhandled expression type %#v\n", node))
	}
}

func (ev *Evaluator) fn(fnVal Value, args []Value) (retVal Value) {
	retVal = NilValue

	defer func() {
		if r := recover(); r != nil {
			switch e := r.(type) {
			case Value:
				retVal = e
				ev.popEnv()
			default:
				panic(r)
			}
		}
	}()

	fn := fnVal.Fn

	if len(fn.args) != len(args) {
		panic(ev.fmtError(fn, "arity mismatch: %s expects %d arguments", fn.identifier, len(fn.args)))
	}

	ev.pushEnv()

	for index, ident := range fn.args {
		ev.setEnv(ident, args[index])
	}

	b := fn.body.(*StmtBlock)
	for _, stmt := range b.body {
		ev.evalStmt(&stmt)
	}

	ev.popEnv()
	return retVal
}

func (ev *Evaluator) evalBinaryExpr(expr *ExprBinary) Value {
	if expr.op.Tag == Equal {
		switch node := expr.lhs.(type) {
		case *ExprIdentifier:
			ident := node.identifier
			val := ev.evalExpr(&expr.rhs)
			ev.updateEnv(ident, val)
			return val
		case *ExprBinary:
			if node.op.Tag != LSquare {
				break
			}
			array := ev.evalExpr(&node.lhs)
			if array.Tag != ValArray {
				panic(ev.fmtError(node, "%v is not subscriptable", array.Tag))
			}
			index := ev.evalExpr(&node.rhs)
			if index.Tag != ValNum {
				panic(ev.fmtError(node, "attempt to subscript with non-numeric value"))
			}
			val := ev.evalExpr(&expr.rhs)
			(*array.Array)[*index.Num] = val
			return val
		}
		panic(ev.fmtError(expr, "lhs of assignment is not assignable"))
	}

	lhs := ev.evalExpr(&expr.lhs)
	rhs := ev.evalExpr(&expr.rhs)

	switch expr.op.Tag {
	case Plus, Minus, Star, Slash:
		if lhs.Tag != ValNum || rhs.Tag != ValNum {
			panic(ev.fmtError(expr, "operator only supported for numbers"))
		}

		var result int
		switch expr.op.Tag {
		case Plus:
			result = *lhs.Num + *rhs.Num
		case Minus:
			result = *lhs.Num - *rhs.Num
		case Star:
			result = *lhs.Num * *rhs.Num
		case Slash:
			result = *lhs.Num / *rhs.Num
		}

		return Value{Tag: ValNum, Num: &result}
	case EqualEqual, BangEqual:
		result, err := lhs.Compare(rhs)
		if err != nil {
			panic(ev.fmtError(expr, err.Error()))
		}
		num := 0
		if result {
			num = 1
		}
		val := Value{Tag: ValNum, Num: &num}
		if expr.op.Tag == BangEqual {
			return val.negate()
		}
		return val
	case Greater, GreaterEqual, Less:
		switch {
		case lhs.Tag == ValNum && rhs.Tag == ValNum:
			result := false
			switch expr.op.Tag {
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
		panic(ev.fmtError(expr, "cannot compare %v and %v", lhs.Tag, rhs.Tag))
	case LSquare:
		switch {
		case lhs.Tag == ValArray && rhs.Tag == ValNum:
			return (*lhs.Array)[*rhs.Num]
		case lhs.Tag == ValStr && rhs.Tag == ValNum:
			b := string((*lhs.Str)[*rhs.Num])
			return Value{Tag: ValStr, Str: &b}
		}
		panic(fmt.Errorf("cannot subscript a %v with a %v", lhs.Tag, rhs.Tag))
	default:
		panic(ev.fmtError(expr, "unknown operator %s", expr.op.Tag.String()))
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
		} else if node.elseBody != nil {
			ev.evalStmt(&node.elseBody)
		}
	case *StmtReturn:
		val := ev.evalExpr(&node.value)
		panic(val) // control flow panic
	case *StmtContinue:
		panic(*node) // control flow panic
	case *StmtBreak:
		panic(*node) // control flow panic
	case *StmtMatch:
		ev.match(node)
	case *StmtBlock:
		ev.evalBlock(node)
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

			b := c.body.(*StmtBlock)
			for _, stmt := range b.body {
				ev.evalStmt(&stmt)
			}
			ev.popEnv()
			return
		default:
			panic(ev.fmtError(c.cond, "unsupported match type %v", c.cond))
		}
	}
}

func (ev *Evaluator) forLoop(node *StmtFor) {
	val := ev.evalExpr(&node.value)
	if val.Tag != ValArray {
		panic(ev.fmtError(node, "only arrays can be used in for loops"))
	}
	ev.pushEnv()
	for index, val := range *val.Array {
		stop := ev.runForLoopBody(node, val, index)
		if stop {
			break
		}
	}
	ev.popEnv()
}

func (ev *Evaluator) runForLoopBody(node *StmtFor, val Value, index int) (stop bool) {
	stop = false
	defer catchContinueOrBreak(ev, ev.env, &stop)

	ev.setEnv(node.identifier, val)
	if node.indexIdentifier != "" {
		ev.setEnv(node.indexIdentifier, Value{Tag: ValNum, Num: &index})
	}

	b := node.body.(*StmtBlock)
	for _, stmt := range b.body {
		ev.evalStmt(&stmt)
	}

	return stop
}

func catchContinueOrBreak(ev *Evaluator, rootEnv *Env, stop *bool) {
	if r := recover(); r != nil {
		switch r.(type) {
		case StmtContinue:
			ev.env = rootEnv
			*stop = false
			return
		case StmtBreak:
			ev.env = rootEnv
			*stop = true
			return
		default:
			panic(r)
		}
	}
}
