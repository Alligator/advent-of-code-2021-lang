package lang

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func checkArgs(args []Value, tags ...ValueTag) {
	if len(args) != len(tags) {
		panic("arg count mismatch")
	}

	for index, tag := range tags {
		if args[index].Tag != tag {
			panic(fmt.Errorf("arg type mismatch: expected %s got %s", tag.String(), args[index].Tag.String()))
		}
	}
}

func nativePrint(args []Value) Value {
	for _, arg := range args {
		fmt.Print(arg.String() + " ")
	}
	fmt.Println()
	return Nil
}

func nativeNum(args []Value) Value {
	checkArgs(args, ValStr)
	i, err := strconv.Atoi(*args[0].Str)
	if err != nil {
		return Nil
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
