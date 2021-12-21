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
	Fn       *Closure
}

type Closure struct {
	fn  *ExprFunc
	env *Env
}

var NilValue = Value{Tag: ValNil}
var zero = 0
var ZeroValue = Value{Tag: ValNum, Num: &zero}

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
