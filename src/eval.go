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

// control flow errors
type returnValue struct {
	value Value
}
type breakError struct{}
type continueError struct{}

func (r returnValue) Error() string   { return "" }
func (b breakError) Error() string    { return "" }
func (c continueError) Error() string { return "" }

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

func (v Value) getKey(key Value) (Value, error) {
tagSwitch:
	switch v.Tag {
	case ValArray:
		if key.Tag == ValNum {
			index := *key.Num
			array := *v.Array
			if index >= len(array) || index < 0 {
				return NilValue, fmt.Errorf("index %d out of range", index)
			}
			return (*v.Array)[*key.Num], nil
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

		return (*v.Map)[keyStr], nil
	case ValStr:
		if key.Tag == ValNum {
			index := *key.Num
			str := *v.Str
			if index >= len(str) {
				return NilValue, fmt.Errorf("index %d out of range", index)
			}
			s := string((*v.Str)[*key.Num])
			return Value{Tag: ValStr, Str: &s}, nil
		}
	}
	return NilValue, fmt.Errorf("cannot subscript a %v with a %v", v.Tag, key.Tag)
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

type stackFrame struct {
	callSite Node
	fn       *Value
	parent   *stackFrame
}

type Evaluator struct {
	sections map[string]*StmtSection
	env      *Env
	section  *StmtSection
	lex      *Lexer
	stackTop *stackFrame
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
	ev.setEnv("slice", Value{Tag: ValNativeFn, NativeFn: nativeSlice})
	ev.setEnv("delete", Value{Tag: ValNativeFn, NativeFn: nativeDelete})
	ev.setEnv("range", Value{Tag: ValNativeFn, NativeFn: nativeRange})
	ev.setEnv("rangei", Value{Tag: ValNativeFn, NativeFn: nativeRangeI})
	ev.setEnv("sort", Value{Tag: ValNativeFn, NativeFn: nativeSort})

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
	lines := make([]string, 0)
	frame := ev.stackTop
	for frame != nil {
		frame_line, _ := ev.lex.GetLineAndCol(*frame.callSite.Token())
		lines = append(lines, fmt.Sprintf("  line %d", frame_line))
		frame = frame.parent
	}
	msg := fmt.Sprintf(format, args...)
	msg = fmt.Sprintf("%s\n%s", strings.Join(lines, "\n"), msg)
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

func (ev *Evaluator) evalProgram(prog *Program) error {
	// read all the sections
	for _, section := range prog.Stmts {
		if stmt, ok := section.(*StmtSection); ok {
			name := stmt.Label
			ev.sections[name] = stmt
		} else {
			_, err := ev.evalStmt(&section)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (ev *Evaluator) EvalSection(name string) (Value, error) {
	if ev.section != nil {
		panic("cannot nest sections")
	}

	section, preset := ev.sections[name]
	if !preset {
		panic(fmt.Errorf("couldn't find section %s", name))
	}

	ev.section = section
	defer func() { ev.section = nil }()

	v, err := ev.evalStmt(&section.Body)
	if r, ok := err.(returnValue); ok {
		return r.value, nil
	}
	if err != nil {
		return NilValue, err
	}

	return v, nil
}

func (ev *Evaluator) HasSection(name string) bool {
	_, present := ev.sections[name]
	return present
}

func (ev *Evaluator) evalBlock(block Stmt) error {
	switch b := block.(type) {
	case *StmtBlock:
		ev.pushEnv()
		defer func() { ev.popEnv() }()
		for _, stmt := range b.Body {
			_, err := ev.evalStmt(&stmt)
			if err != nil {
				return err
			}
		}
	default:
		panic(ev.fmtError(block, "expected a block but found %v", b))
	}
	return nil
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
			defer func() {
				if r := recover(); r != nil {
					if e, ok := r.(RuntimeError); ok {
						// patch the line number, native functions don't know it
						line, _ := ev.lex.GetLineAndCol(node.identifierToken)
						e.Line = line
						panic(e)
					}
					panic(r)
				}
			}()
			return fnVal.NativeFn(args)
		case ValFn:
			v, err := ev.fn(node, fnVal, args)
			if err != nil {
				panic(err) // FIXME
			}
			return v
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

func (ev *Evaluator) fn(node Node, fnVal Value, args []Value) (Value, error) {
	fn := fnVal.Fn
	frame := stackFrame{
		callSite: node,
		fn:       &fnVal,
		parent:   ev.stackTop,
	}
	ev.stackTop = &frame

	if len(fn.Args) != len(args) {
		panic(ev.fmtError(fn, "arity mismatch: %s expects %d arguments", fn.Identifier, len(fn.Args)))
	}

	ev.pushEnv()
	defer func() {
		ev.popEnv()
		ev.stackTop = frame.parent
	}()

	for index, ident := range fn.Args {
		ev.setEnv(ident, args[index])
	}

	b := fn.Body.(*StmtBlock)
	for _, stmt := range b.Body {
		_, err := ev.evalStmt(&stmt)
		if r, ok := err.(returnValue); ok {
			return r.value, nil
		}
		if err != nil {
			return NilValue, err
		}
	}

	return NilValue, nil
}

func (ev *Evaluator) evalBinaryExpr(expr *ExprBinary) Value {
	if expr.Op.Tag == Equal {
		return ev.evalAssignment(expr)
	}

	lhs := ev.evalExpr(&expr.Lhs)
	rhs := ev.evalExpr(&expr.Rhs)

	switch expr.Op.Tag {
	case Plus:
		// coerce nils to 0
		if lhs.Tag == ValNil {
			lhs = ZeroValue
		}
		if rhs.Tag == ValNil {
			rhs = ZeroValue
		}

		switch {
		case lhs.Tag == ValNum && rhs.Tag == ValNum:
			result := *lhs.Num + *rhs.Num
			return Value{Tag: ValNum, Num: &result}
		case lhs.Tag == ValStr || rhs.Tag == ValStr:
			// coerce everything to string
			result := lhs.String() + rhs.String()
			return Value{Tag: ValStr, Str: &result}
		}
		panic(ev.fmtError(expr, "operator only supported for numbers and strings"))
	case Minus, Star, Slash, Percent:
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
		switch expr.Op.Tag {
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
		if expr.Op.Tag == BangEqual {
			return val.negate()
		}
		return val
	case Greater, GreaterEqual, Less, LessEqual:
		switch {
		case lhs.Tag == ValNum && rhs.Tag == ValNum:
			result := false
			switch expr.Op.Tag {
			case Greater:
				result = *lhs.Num > *rhs.Num
			case GreaterEqual:
				result = *lhs.Num >= *rhs.Num
			case Less:
				result = *lhs.Num < *rhs.Num
			case LessEqual:
				result = *lhs.Num <= *rhs.Num
			}
			num := 0
			if result {
				num = 1
			}
			return Value{Tag: ValNum, Num: &num}
		}
		panic(ev.fmtError(expr, "cannot compare %v and %v", lhs.Tag, rhs.Tag))
	case AmpAmp:
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

		lhs_truthy := lhs.isTruthy()
		rhs_truthy := rhs.isTruthy()
		result := lhs_truthy && rhs_truthy
		num_result := 0
		if result {
			num_result = 1
		}

		return Value{Tag: ValNum, Num: &num_result}
	case LSquare:
		val, err := lhs.getKey(rhs)
		if err != nil {
			panic(ev.fmtError(expr, "%s", err))
		}
		return val
	default:
		panic(ev.fmtError(expr, "unknown operator %s", expr.Op.Tag.String()))
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
		if node.Op.Tag != LSquare {
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

func (ev *Evaluator) evalStmt(stmt *Stmt) (Value, error) {
	switch node := (*stmt).(type) {
	case *StmtVar:
		ident := node.Identifier
		val := ev.evalExpr(&node.Value)
		ev.setEnv(ident, val)
	case *StmtFor:
		err := ev.forLoop(node)
		if err != nil {
			return NilValue, err
		}
	case *StmtExpr:
		return ev.evalExpr(&node.Expr), nil
	case *StmtIf:
		val := ev.evalExpr(&node.Condition)
		if val.isTruthy() {
			err := ev.evalBlock(node.Body)
			if err != nil {
				return NilValue, err
			}
		} else if node.ElseBody != nil {
			_, err := ev.evalStmt(&node.ElseBody)
			if err != nil {
				return NilValue, err
			}
		}
	case *StmtReturn:
		val := ev.evalExpr(&node.Value)
		return NilValue, returnValue{val}
	case *StmtContinue:
		return NilValue, continueError{}
	case *StmtBreak:
		return NilValue, breakError{}
	case *StmtMatch:
		err := ev.match(node)
		if err != nil {
			return NilValue, err
		}
	case *StmtBlock:
		err := ev.evalBlock(node)
		if err != nil {
			return NilValue, err
		}
	default:
		panic(fmt.Sprintf("unhandled statement type %#v\n", node))
	}
	return NilValue, nil
}

func (ev *Evaluator) match(match *StmtMatch) error {
	candidate := ev.evalExpr(&match.Value)

MatchLoop:
	for _, c := range match.Cases {
		switch pattern := c.Cond.(type) {
		case *ExprString:
			if candidate.Tag != ValStr {
				continue
			}
			if *candidate.Str == pattern.Str {
				b := c.Body.(*StmtBlock)
				ev.evalBlock(b)
				return nil
			}
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
			defer func() { ev.popEnv() }()
			for k, v := range vars {
				ev.env.vars[k] = v
			}

			b := c.Body.(*StmtBlock)
			for _, stmt := range b.Body {
				_, err := ev.evalStmt(&stmt)
				if err != nil {
					return err
				}
			}
			return nil
		default:
			panic(ev.fmtError(c.Cond, "unsupported match type %v", c.Cond.Token().Tag.String()))
		}
	}
	return nil
}

func (ev *Evaluator) forLoop(node *StmtFor) error {
	if node.Value == nil {
		// infinite loop
		ev.pushEnv()
		defer func() { ev.popEnv() }()
		for {
			stop, err := ev.runForLoopBody(node, NilValue, NilValue)
			if err != nil {
				return err
			}
			if stop {
				return nil
			}
		}
	}

	val := ev.evalExpr(&node.Value)
	switch val.Tag {
	case ValArray:
		ev.pushEnv()
		defer func() { ev.popEnv() }()
		for index, item := range *val.Array {
			i := index
			stop, err := ev.runForLoopBody(node, item, Value{Tag: ValNum, Num: &i})
			if err != nil {
				return err
			}
			if stop {
				break
			}
		}
	case ValRange:
		rng := val.Range
		ev.pushEnv()
		defer func() { ev.popEnv() }()
		for !rng.done() {
			i := rng.current
			stop, err := ev.runForLoopBody(node, Value{Tag: ValNum, Num: &i}, Value{Tag: ValNum, Num: &i})
			if err != nil {
				return err
			}
			if stop {
				break
			}
			rng.next()
		}
	case ValMap:
		mp := val.Map
		ev.pushEnv()
		defer func() { ev.popEnv() }()
		for key, val := range *mp {
			s := key
			ev.runForLoopBody(node, Value{Tag: ValStr, Str: &s}, val)
		}
	default:
		panic(ev.fmtError(node, "%s is not iterable", val.Tag.String()))
	}
	return nil
}

func (ev *Evaluator) runForLoopBody(node *StmtFor, val Value, index Value) (bool, error) {
	if node.Identifier != "" {
		ev.setEnv(node.Identifier, val)
	}
	if node.IndexIdentifier != "" {
		ev.setEnv(node.IndexIdentifier, index)
	}

	b := node.body.(*StmtBlock)
	for _, stmt := range b.Body {
		_, err := ev.evalStmt(&stmt)
		if err != nil {
			switch err.(type) {
			case breakError:
				return true, nil
			case continueError:
				return false, nil
			default:
				return true, err
			}
		}
	}

	return false, nil
}
