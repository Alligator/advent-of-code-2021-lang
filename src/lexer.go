package lang

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

type TokenTag uint8

//go:generate stringer -type=TokenTag -linecomment
const (
	EOF TokenTag = iota
	Identifier
	Str
	Num
	Colon        // :
	LCurly       // {
	RCurly       // }
	LParen       // (
	RParen       // )
	LSquare      // [
	RSquare      // ]
	Equal        // =
	EqualEqual   // ==
	BangEqual    // !=
	Greater      // >
	GreaterEqual // >=
	Less         // <
	LessEqual    // <=
	Plus         // +
	Star         // *
	Comma        // ,
	Minus        // -
	Slash        // /
	Percent      // %
	AmpAmp       // &&
	PipePipe     // ||
	Var          // var
	For          // for
	In           // in
	If           // if
	Return       // return
	Continue     // continue
	Match        // match
	Else         // else
	Break        // break
	Fn           // fn
	Nil          // nil
)

type Token struct {
	Tag TokenTag
	Pos int
	Len int
}

type Lexer struct {
	src        string
	pos        int
	line       int
	tokenStart int
}

func NewLexer(src string) Lexer {
	return Lexer{
		src:        src,
		pos:        0,
		line:       1,
		tokenStart: 0,
	}
}

func simpleToken(lex *Lexer, tag TokenTag) Token {
	return Token{tag, lex.tokenStart, 0}
}

func stringToken(lex *Lexer, tag TokenTag, start int) Token {
	return Token{tag, start, lex.pos - start}
}

func (lex *Lexer) fmtError(msg string, args ...interface{}) Error {
	formattedMsg := fmt.Sprintf(msg, args...)
	return E(LexError, formattedMsg, lex.line)
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
func (lex *Lexer) consume(expected rune) error {
	r, size := utf8.DecodeRuneInString(lex.src[lex.pos:])
	if r != expected {
		return lex.fmtError("expected %q but saw %q", expected, r)
	}
	lex.pos += size
	return nil
}

func (lex *Lexer) skipWhitespace() {
	for {
		switch lex.peek() {
		case ' ', '\n', '\r':
			lex.advance()
		case '#':
			for lex.peek() != '\n' {
				lex.advance()
				if lex.pos >= len(lex.src) {
					return
				}
			}
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
	case "break":
		return simpleToken(lex, Break)
	case "match":
		return simpleToken(lex, Match)
	case "else":
		return simpleToken(lex, Else)
	case "fn":
		return simpleToken(lex, Fn)
	case "nil":
		return simpleToken(lex, Nil)
	default:
		return stringToken(lex, Identifier, start)
	}
}

func (lex *Lexer) string() (Token, error) {
	for lex.peek() != '\'' {
		lex.advance()
	}
	t := stringToken(lex, Str, lex.tokenStart+1)
	err := lex.consume('\'')
	if err != nil {
		return Token{}, err
	}
	return t, nil
}

func (lex *Lexer) number() Token {
	for unicode.IsDigit(lex.peek()) {
		lex.advance()
	}
	return stringToken(lex, Num, lex.tokenStart)
}

func (lex *Lexer) NextToken() (retToken Token, err error) {
	if lex.pos >= len(lex.src) {
		return simpleToken(lex, EOF), nil
	}

	lex.skipWhitespace()
	r := lex.peek()
	lex.tokenStart = lex.pos

	if unicode.IsLetter(r) {
		return lex.identifier(), nil
	}

	if unicode.IsDigit(r) {
		return lex.number(), nil
	}

	lex.advance()

	switch r {
	case '\'':
		token, err := lex.string()
		if err != nil {
			return Token{}, err
		}
		return token, nil
	case ':':
		return simpleToken(lex, Colon), nil
	case '{':
		return simpleToken(lex, LCurly), nil
	case '}':
		return simpleToken(lex, RCurly), nil
	case '(':
		return simpleToken(lex, LParen), nil
	case ')':
		return simpleToken(lex, RParen), nil
	case '*':
		return simpleToken(lex, Star), nil
	case '+':
		return simpleToken(lex, Plus), nil
	case '%':
		return simpleToken(lex, Percent), nil
	case ',':
		return simpleToken(lex, Comma), nil
	case '[':
		return simpleToken(lex, LSquare), nil
	case ']':
		return simpleToken(lex, RSquare), nil
	case '-':
		return simpleToken(lex, Minus), nil
	case '/':
		return simpleToken(lex, Slash), nil
	case '<':
		if lex.peek() == '=' {
			lex.advance()
			return simpleToken(lex, LessEqual), nil
		}
		return simpleToken(lex, Less), nil
	case '>':
		if lex.peek() == '=' {
			lex.advance()
			return simpleToken(lex, GreaterEqual), nil
		}
		return simpleToken(lex, Greater), nil
	case '=':
		if lex.peek() == '=' {
			lex.advance()
			return simpleToken(lex, EqualEqual), nil
		}
		return simpleToken(lex, Equal), nil
	case '!':
		if lex.peek() == '=' {
			lex.advance()
			return simpleToken(lex, BangEqual), nil
		}
	case '&':
		if lex.peek() == '&' {
			lex.advance()
			return simpleToken(lex, AmpAmp), nil
		}
	case '|':
		if lex.peek() == '|' {
			lex.advance()
			return simpleToken(lex, PipePipe), nil
		}
	}
	return retToken, lex.fmtError("unexpected character %q (%x)", r, r)
}

func (lex *Lexer) GetString(token Token) string {
	return lex.src[token.Pos : token.Pos+token.Len]
}

func (lex *Lexer) GetLineAndCol(token Token) (int, int) {
	line := 1
	lineStart := 0
	for i, r := range lex.src {
		if i >= token.Pos {
			break
		}
		if r == '\n' {
			line++
			lineStart = i + 1
		}
	}
	return line, token.Pos - lineStart
}
