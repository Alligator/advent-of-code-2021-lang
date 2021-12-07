package lang

type LexError struct {
	Msg  string
	Line int
}

func (e *LexError) Error() string { return e.Msg }

type ParseError struct {
	Msg  string
	Line int
}

func (e *ParseError) String() string { return e.Msg }

type RuntimeError struct {
	Msg  string
	Line int
}

func (e *RuntimeError) String() string { return e.Msg }
