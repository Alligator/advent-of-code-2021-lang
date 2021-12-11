package lang

import (
	"fmt"
	"strconv"
	"strings"
)

type ValueTag uint8

//go:generate stringer -type=ValueTag -linecomment
const (
	ValNil      ValueTag = iota // nil
	ValStr                      // string
	ValNum                      // number
	ValArray                    // array
	ValMap                      // map
	ValRange                    // range
	ValNativeFn                 // <nativeFn>
	ValFn                       // <fn>
)

type Value struct {
	Tag      ValueTag
	Str      *string
	Num      *int
	Array    *[]Value
	Map      *map[string]Value
	Range    *Range
	NativeFn func([]Value) Value
	Fn       *ExprFunc
}

var NilValue = Value{Tag: ValNil}
var zero = 0
var ZeroValue = Value{Tag: ValNum, Num: &zero}

type Range struct {
	current int
	end     int
	step    int
}

func (r *Range) next() {
	if !r.done() {
		r.current += r.step
	}
}

func (r *Range) done() bool {
	return r.current == r.end
}

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
	case ValMap:
		var sb strings.Builder
		sb.WriteString("{")
		empty := true
		for k, v := range *v.Map {
			sb.WriteString(k)
			sb.WriteString(": ")
			sb.WriteString(v.Repr())
			sb.WriteString(", ")
			empty = false
		}
		if empty {
			sb.WriteString("}")
		} else {
			sb.WriteString("\b\b}") // backspace over the last comma
		}
		return sb.String()
	default:
		return fmt.Sprintf("<%s>\n", v.Tag.String())
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
	default:
		return v.Repr()
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

func (v Value) getKey(key Value) (Value, bool) {
tagSwitch:
	switch v.Tag {
	case ValArray:
		if key.Tag == ValNum {
			return (*v.Array)[*key.Num], true
		}
	case ValMap:
		var keyStr string

		switch key.Tag {
		case ValNum:
			keyStr = strconv.Itoa(*key.Num)
		case ValStr:
			keyStr = *key.Str
		default:
			break tagSwitch
		}

		return (*v.Map)[keyStr], true
	}
	return NilValue, false
}

func (v Value) setKey(key Value, val Value) bool {
tagSwitch:
	switch v.Tag {
	case ValArray:
		if key.Tag == ValNum {
			(*v.Array)[*key.Num] = val
			return true
		}
	case ValMap:
		var keyStr string

		switch key.Tag {
		case ValNum:
			keyStr = strconv.Itoa(*key.Num)
		case ValStr:
			keyStr = *key.Str
		default:
			break tagSwitch
		}

		(*v.Map)[keyStr] = val
		return true
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
	sections map[string]*StmtSection
	env      *Env
	section  *StmtSection
	lex      *Lexer
}

func NewEvaluator(prog *Program, lex *Lexer) Evaluator {
	env := Env{vars: make(map[string]Value)}
	ev := Evaluator{
		env:      &env,
		sections: make(map[string]*StmtSection),
		lex:      lex,
	}

	ev.setEnv("print", Value{Tag: ValNativeFn, NativeFn: nativePrint})
	ev.setEnv("println", Value{Tag: ValNativeFn, NativeFn: nativePrintLn})
	ev.setEnv("num", Value{Tag: ValNativeFn, NativeFn: nativeNum})
	ev.setEnv("read", Value{Tag: ValNativeFn, NativeFn: nativeRead})
	ev.setEnv("split", Value{Tag: ValNativeFn, NativeFn: nativeSplit})
	ev.setEnv("len", Value{Tag: ValNativeFn, NativeFn: nativeLen})
	ev.setEnv("push", Value{Tag: ValNativeFn, NativeFn: nativePush})
	ev.setEnv("delete", Value{Tag: ValNativeFn, NativeFn: nativeDelete})
	ev.setEnv("range", Value{Tag: ValNativeFn, NativeFn: nativeRange})
	ev.setEnv("rangei", Value{Tag: ValNativeFn, NativeFn: nativeRangeI})

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
	for _, section := range prog.Stmts {
		if stmt, ok := section.(*StmtSection); ok {
			name := stmt.Label
			ev.sections[name] = stmt
		} else {
			ev.evalStmt(&section)
		}
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

	ev.section = section
	retVal = ev.evalStmt(&section.Body)
	ev.section = nil

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
		for _, stmt := range b.Body {
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
		return Value{Tag: ValStr, Str: &node.Str}
	case *ExprNum:
		return Value{Tag: ValNum, Num: &node.Num}
	case *ExprNil:
		return NilValue
	case *ExprIdentifier:
		v, ok := ev.find(node.Identifier)
		if !ok {
			panic(ev.fmtError(node, "unknown variable %s", node.Identifier))
		}
		return *v
	case *ExprFuncall:
		fnVal := ev.evalExpr(&node.Identifier)

		args := make([]Value, 0)
		for _, arg := range node.Args {
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
		ev.setEnv(node.Identifier, fnVal)
		return fnVal
	case *ExprBinary:
		return ev.evalBinaryExpr(node)
	case *ExprArray:
		items := make([]Value, 0)
		for _, itemExpr := range node.Items {
			items = append(items, ev.evalExpr(&itemExpr))
		}
		return Value{Tag: ValArray, Array: &items}
	case *ExprMap:
		items := make(map[string]Value)
		for _, item := range node.Items {
			val := ev.evalExpr(&item.Value)
			items[item.Key] = val
		}
		return Value{Tag: ValMap, Map: &items}
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

	if len(fn.Args) != len(args) {
		panic(ev.fmtError(fn, "arity mismatch: %s expects %d arguments", fn.Identifier, len(fn.Args)))
	}

	ev.pushEnv()

	for index, ident := range fn.Args {
		ev.setEnv(ident, args[index])
	}

	b := fn.Body.(*StmtBlock)
	for _, stmt := range b.Body {
		ev.evalStmt(&stmt)
	}

	ev.popEnv()
	return retVal
}

func (ev *Evaluator) evalBinaryExpr(expr *ExprBinary) Value {
	if expr.op.Tag == Equal {
		return ev.evalAssignment(expr)
	}

	lhs := ev.evalExpr(&expr.Lhs)
	rhs := ev.evalExpr(&expr.Rhs)

	switch expr.op.Tag {
	case Plus, Minus, Star, Slash, Percent:
		// coerce nils to 0
		if lhs.Tag == ValNil {
			lhs = ZeroValue
		}
		if rhs.Tag == ValNil {
			rhs = ZeroValue
		}

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
		case Percent:
			result = *lhs.Num % *rhs.Num
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
		val, ok := lhs.getKey(rhs)
		if !ok {
			panic(fmt.Errorf("cannot subscript a %v with a %v", lhs.Tag, rhs.Tag))
		}
		return val
	default:
		panic(ev.fmtError(expr, "unknown operator %s", expr.op.Tag.String()))
	}
}

func (ev *Evaluator) evalAssignment(expr *ExprBinary) Value {
	switch node := expr.Lhs.(type) {

	case *ExprIdentifier:
		ident := node.Identifier
		val := ev.evalExpr(&expr.Rhs)
		ev.updateEnv(ident, val)
		return val

	case *ExprBinary:
		if node.op.Tag != LSquare {
			break
		}

		lhs := ev.evalExpr(&node.Lhs)
		key := ev.evalExpr(&node.Rhs)
		val := ev.evalExpr(&expr.Rhs)

		ok := lhs.setKey(key, val)
		if !ok {
			panic(ev.fmtError(node, "%v is not subscriptable", lhs.Tag))
		}
		return val
	}
	panic(ev.fmtError(expr, "left hand side of assignment is not assignable"))
}

func (ev *Evaluator) evalStmt(stmt *Stmt) Value {
	switch node := (*stmt).(type) {
	case *StmtVar:
		ident := node.Identifier
		val := ev.evalExpr(&node.Value)
		ev.setEnv(ident, val)
	case *StmtFor:
		ev.forLoop(node)
	case *StmtExpr:
		return ev.evalExpr(&node.Expr)
	case *StmtIf:
		val := ev.evalExpr(&node.Condition)
		if val.isTruthy() {
			ev.evalBlock(node.Body)
		} else if node.ElseBody != nil {
			ev.evalStmt(&node.ElseBody)
		}
	case *StmtReturn:
		val := ev.evalExpr(&node.Value)
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
	return NilValue
}

func (ev *Evaluator) match(match *StmtMatch) {
	candidate := ev.evalExpr(&match.Value)

MatchLoop:
	for _, c := range match.Cases {
		switch pattern := c.Cond.(type) {
		case *ExprArray:
			if candidate.Tag != ValArray {
				continue
			}

			vars := make(map[string]Value)
			for index, item := range pattern.Items {
				if index >= len(*candidate.Array) {
					continue MatchLoop
				}

				switch itemNode := item.(type) {
				case *ExprIdentifier:
					vars[itemNode.Identifier] = (*candidate.Array)[index]
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

			b := c.Body.(*StmtBlock)
			for _, stmt := range b.Body {
				ev.evalStmt(&stmt)
			}
			ev.popEnv()
			return
		default:
			panic(ev.fmtError(c.Cond, "unsupported match type %v", c.Cond))
		}
	}
}

func (ev *Evaluator) forLoop(node *StmtFor) {
	val := ev.evalExpr(&node.Value)
	switch val.Tag {
	case ValArray:
		ev.pushEnv()
		for index, item := range *val.Array {
			i := index
			stop := ev.runForLoopBody(node, item, Value{Tag: ValNum, Num: &i})
			if stop {
				break
			}
		}
		ev.popEnv()
	case ValRange:
		rng := val.Range
		ev.pushEnv()
		for !rng.done() {
			i := rng.current
			stop := ev.runForLoopBody(node, Value{Tag: ValNum, Num: &i}, Value{Tag: ValNum, Num: &i})
			if stop {
				break
			}
			rng.next()
		}
		ev.popEnv()
	case ValMap:
		mp := val.Map
		ev.pushEnv()
		for key, val := range *mp {
			s := key
			ev.runForLoopBody(node, Value{Tag: ValStr, Str: &s}, val)
		}
		ev.popEnv()
	default:
		panic(ev.fmtError(node, "%s is not iterable", val.Tag.String()))
	}
}

func (ev *Evaluator) runForLoopBody(node *StmtFor, val Value, index Value) (stop bool) {
	stop = false
	defer catchContinueOrBreak(ev, ev.env, &stop)

	ev.setEnv(node.Identifier, val)
	if node.IndexIdentifier != "" {
		ev.setEnv(node.IndexIdentifier, index)
	}

	b := node.body.(*StmtBlock)
	for _, stmt := range b.Body {
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
