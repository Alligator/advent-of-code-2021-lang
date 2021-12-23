package lang

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

func checkArgs(args []Value, tags ...ValueTag) {
	if len(args) != len(tags) {
		panic(E(RuntimeError, "arity mismatch", 0))
	}

	for index, tag := range tags {
		if args[index].Tag != tag {
			msg := fmt.Sprintf("arg type mismatch: expected %s got %s", tag.String(), args[index].Tag.String())
			panic(E(RuntimeError, msg, 0))
		}
	}
}

func nativePrint(args []Value) Value {
	for idx, arg := range args {
		if idx > 0 {
			fmt.Print(" " + arg.String())
		} else {
			fmt.Print(arg.String())
		}
	}
	return NilValue
}

func nativePrintLn(args []Value) Value {
	v := nativePrint(args)
	fmt.Println()
	return v
}

func nativeNum(args []Value) Value {
	checkArgs(args, ValStr)
	i, err := strconv.Atoi(strings.TrimSpace(*args[0].Str))
	if err != nil {
		return NilValue
	}
	return Value{Tag: ValNum, Num: &i}
}

func nativeRead(args []Value) Value {
	checkArgs(args, ValStr)
	f, err := os.ReadFile(*args[0].Str)
	if err != nil {
		panic(err)
	}
	s := string(f)
	return Value{Tag: ValStr, Str: &s}
}

func nativeSplit(args []Value) Value {
	checkArgs(args, ValStr, ValStr)
	sp := strings.Split(*args[0].Str, *args[1].Str)
	arr := make([]Value, 0)
	for _, s := range sp {
		p := s
		arr = append(arr, Value{Tag: ValStr, Str: &p})
	}
	return Value{Tag: ValArray, Array: &arr}
}

func nativeLen(args []Value) Value {
	l := 0
	switch args[0].Tag {
	case ValArray:
		l = len(*args[0].Array)
	case ValStr:
		l = len(*args[0].Str)
	}
	return Value{Tag: ValNum, Num: &l}
}

func nativePush(args []Value) Value {
	if len(args) < 2 {
		panic("arg count mismatch")
	}

	if args[0].Tag != ValArray {
		panic("can only push to an array")
	}

	array := *args[0].Array
	array = append(array, args[1])
	return Value{Tag: ValArray, Array: &array}
}

func nativeSlice(args []Value) Value {
	checkArgs(args, ValArray, ValNum, ValNum)
	array := *args[0].Array
	from := *args[1].Num
	to := *args[2].Num

	if from < 0 || from > len(array)-1 || to < 0 || to > len(array)-1 {
		panic(E(RuntimeError, "invalid index", 0))
	}
	slice := array[from:to]
	return Value{Tag: ValArray, Array: &slice}
}

func nativeDelete(args []Value) Value {
	checkArgs(args, ValArray, ValNum)

	newArray := make([]Value, 0)
	for index, val := range *args[0].Array {
		if index == *args[1].Num {
			continue
		}
		newArray = append(newArray, val)
	}
	return Value{Tag: ValArray, Array: &newArray}
}

func nativeRange(args []Value) Value {
	checkArgs(args, ValNum, ValNum)
	from := *args[0].Num
	to := *args[1].Num
	step := 1
	if to < from {
		step = -1
	}
	r := Range{from, to, step}
	return Value{Tag: ValRange, Range: &r}
}

func nativeRangeI(args []Value) Value {
	checkArgs(args, ValNum, ValNum)
	from := *args[0].Num
	to := *args[1].Num
	step := 1
	if to < from {
		step = -1
	}
	to += step
	r := Range{from, to, step}
	return Value{Tag: ValRange, Range: &r}
}

func nativeSort(args []Value) Value {
	checkArgs(args, ValArray)
	arr := *args[0].Array
	dest := make([]Value, len(arr))
	copy(dest, arr)
	sort.Slice(dest, func(a int, b int) bool {
		if dest[a].Tag == ValNum {
			valA := dest[a].Num
			valB := dest[b].Num
			return *valA < *valB
		}

		if dest[a].Tag == ValStr {
			valA := dest[a].Str
			valB := dest[b].Str
			return *valA < *valB
		}

		return false
	})
	return Value{Tag: ValArray, Array: &dest}
}

func nativeUpper(args []Value) Value {
	checkArgs(args, ValStr)
	str := *args[0].Str
	ustr := strings.ToUpper(str)
	return Value{Tag: ValStr, Str: &ustr}
}
