// Code generated by "stringer -type=ValueTag -linecomment"; DO NOT EDIT.

package lang

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[ValNil-0]
	_ = x[ValStr-1]
	_ = x[ValNum-2]
	_ = x[ValArray-3]
	_ = x[ValMap-4]
	_ = x[ValRange-5]
	_ = x[ValNativeFn-6]
	_ = x[ValFn-7]
}

const _ValueTag_name = "nilstringnumberarraymaprange<nativeFn><fn>"

var _ValueTag_index = [...]uint8{0, 3, 9, 15, 20, 23, 28, 38, 42}

func (i ValueTag) String() string {
	if i >= ValueTag(len(_ValueTag_index)-1) {
		return "ValueTag(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _ValueTag_name[_ValueTag_index[i]:_ValueTag_index[i+1]]
}
