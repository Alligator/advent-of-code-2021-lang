package lang

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

type TokenTag uint8

const (
	EOF TokenTag = iota
	Identifier
	Colon
	Str
	Num
	LCurly
	RCurly
	LParen
	RParen
	Var
	Equal
	EqualEqual
	GreaterEqual
	For
	In
	Plus
	If
	Star
	Return
	Continue
)

func (t TokenTag) String() string {
	return []string{
		"EOF", "Identifier", "Colon", "Str", "Num", "LCurly", "RCurly", "LParen", "RParen", "Var",
		"Equal", "EqualEqual", "GreaterEqual", "For", "In", "Plus", "If", "Star", "Return", "Continue",
	}[t]
}

type Token struct {
	Tag TokenTag
	Pos int
	Len int
}

type Lexer struct {
	src  string
	pos  int
	line int
}

func NewLexer(src string) Lexer {
	return Lexer{
		src:  src,
		pos:  0,
		line: 1,
	}
}

func simpleToken(lex *Lexer, tag TokenTag) Token {
	return Token{tag, lex.pos, 0}
}

func stringToken(lex *Lexer, tag TokenTag, start int) Token {
	return Token{tag, start, lex.pos - start}
}

func (lex *Lexer) fmtError(msg string) string {
	return fmt.Sprintf("lex error on line %d: %s\n", lex.line, msg)
}

func (lex *Lexer) peek() rune {
	r, _ := utf8.DecodeRuneInString(lex.src[lex.pos:])
	return r
}

func (lex *Lexer) advance() rune {
	if lex.pos > 0 {
		prev, _ := utf8.DecodeRuneInString(lex.src[lex.pos-1:])
		if prev == '\n' {
			lex.line++
		}
	}
	r, size := utf8.DecodeRuneInString(lex.src[lex.pos:])
	lex.pos += size
	return r
}
func (lex *Lexer) consume(expected rune) rune {
	r, size := utf8.DecodeRuneInString(lex.src[lex.pos:])
	if r != expected {
		panic(lex.fmtError(fmt.Sprintf("expected %q but saw %q", expected, r)))
	}
	lex.pos += size
	return r
}

func (lex *Lexer) skipWhitespace() {
	for {
		switch lex.peek() {
		case ' ', '\n', '\r':
			lex.advance()
		default:
			return
		}
	}
}

func (lex *Lexer) identifier() Token {
	start := lex.pos
	for {
		c := lex.peek()
		if unicode.IsLetter(c) || unicode.IsNumber(c) || c == '_' {
			lex.advance()
		} else {
			break
		}
	}

	str := lex.src[start:lex.pos]
	switch str {
	case "var":
		return simpleToken(lex, Var)
	case "for":
		return simpleToken(lex, For)
	case "in":
		return simpleToken(lex, In)
	case "if":
		return simpleToken(lex, If)
	case "return":
		return simpleToken(lex, Return)
	case "continue":
		return simpleToken(lex, Continue)
	default:
		return stringToken(lex, Identifier, start)
	}
}

func (lex *Lexer) string() Token {
	start := lex.pos
	for lex.peek() != '\'' {
		lex.advance()
	}
	t := stringToken(lex, Str, start)
	lex.consume('\'')
	return t
}

func (lex *Lexer) number() Token {
	start := lex.pos
	for unicode.IsDigit(lex.peek()) {
		lex.advance()
	}
	return stringToken(lex, Num, start)
}

func (lex *Lexer) NextToken() Token {
	if lex.pos >= len(lex.src) {
		return simpleToken(lex, EOF)
	}

	lex.skipWhitespace()
	r := lex.peek()

	if unicode.IsLetter(r) {
		return lex.identifier()
	}

	if unicode.IsDigit(r) {
		return lex.number()
	}

	lex.advance()

	switch r {
	case '\'':
		return lex.string()
	case ':':
		return simpleToken(lex, Colon)
	case '{':
		return simpleToken(lex, LCurly)
	case '}':
		return simpleToken(lex, RCurly)
	case '(':
		return simpleToken(lex, LParen)
	case ')':
		return simpleToken(lex, RParen)
	case '*':
		return simpleToken(lex, Star)
	case '+':
		return simpleToken(lex, Plus)
	case '>':
		if lex.peek() == '=' {
			lex.advance()
			return simpleToken(lex, GreaterEqual)
		}
	case '=':
		if lex.peek() == '=' {
			lex.advance()
			return simpleToken(lex, EqualEqual)
		}
		return simpleToken(lex, Equal)
	}
	panic(lex.fmtError(fmt.Sprintf("unexpected character %q (%x)", r, r)))
}

func (lex *Lexer) GetString(token Token) string {
	return lex.src[token.Pos : token.Pos+token.Len]
}

func (lex *Lexer) GetLineAndCol(token Token) (int, int) {
	line := 1
	lineStart := 0
	for i, r := range lex.src {
		if r == '\n' {
			line++
			lineStart = i + 1
		}
		if i > token.Pos {
			break
		}
	}
	return line, token.Pos - lineStart
}
