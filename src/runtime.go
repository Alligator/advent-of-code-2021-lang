package lang

import (
	"fmt"
	"os"
	"strconv"
)

func nativePrint(arg Value) Value {
	fmt.Println(arg.String())
	return Nil
}

func nativeNum(arg Value) Value {
	if arg.Tag != ValStr {
		return Nil
	}
	i, err := strconv.Atoi(*arg.Str)
	if err != nil {
		return Nil
	}
	return Value{Tag: ValNum, Num: &i}
}

func nativeRead(arg Value) Value {
	if arg.Tag != ValStr {
		return Nil
	}

	f, err := os.ReadFile(*arg.Str)
	if err != nil {
		panic(err)
	}
	s := string(f)
	return Value{Tag: ValStr, Str: &s}
}
