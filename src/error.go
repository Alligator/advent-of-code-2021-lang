package lang

type ErrorTag uint8

const (
	LexError ErrorTag = iota
	ParseError
	RuntimeError
)

func (et ErrorTag) String() string {
	switch et {
	case LexError:
		return "syntax error"
	case ParseError:
		return "parse error"
	case RuntimeError:
		return "runtime error"
	}
	return "unknown error"
}

type Error struct {
	Tag  ErrorTag
	Msg  string
	Line int
}

func (e Error) Error() string { return e.Msg }

func E(tag ErrorTag, msg string, line int) Error {
	return Error{tag, msg, line}
}
