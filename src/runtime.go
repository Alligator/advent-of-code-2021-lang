package lang

import (
	"fmt"
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
		panic(err)
	}
	return Value{Tag: ValNum, Num: &i}
}
