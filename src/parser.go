package lang

import (
	"fmt"
	"strconv"
	"strings"
)

type Parser struct {
	lex       *Lexer
	token     Token
	prevToken Token
}

type Precedence uint8

const (
	PrecNone Precedence = iota
	PrecAssign
	PrecLogical
	PrecCompare
	PrecSum
	PrecProduct
	PrecHighest
)

func NewParser(lex *Lexer) Parser {
	eof := Token{EOF, 0, 0}
	return Parser{lex, eof, eof}
}

func (p *Parser) atEnd() bool {
	return p.token.Tag == EOF
}

func (p *Parser) fmtError(msg string, args ...interface{}) ParseError {
	line, _ := p.lex.GetLineAndCol(p.token)
	formattedMsg := fmt.Sprintf(msg, args...)
	return ParseError{formattedMsg, line}
}

func (p *Parser) advance() {
	p.prevToken = p.token
	p.token = p.lex.NextToken()
}

func (p *Parser) consume(expected ...TokenTag) Token {
	for _, tag := range expected {
		if p.token.Tag == tag {
			p.advance()
			return p.prevToken
		}
	}

	if len(expected) > 1 {
		tags := make([]string, len(expected))
		for index, tag := range expected {
			tags[index] = tag.String()
		}
		panic(p.fmtError("expected one of %s but saw %s", strings.Join(tags, ", "), p.token.Tag))
	} else {
		panic(p.fmtError("expected %s but saw %s", expected[0], p.token.Tag))
	}
}

func (p *Parser) section() Stmt {
	p.consume(Identifier)
	ident := p.lex.GetString(p.prevToken)
	identToken := &p.prevToken
	p.consume(Colon)

	if p.token.Tag == LCurly {
		block := p.block()
		return &StmtSection{ident, block, &p.token}
	}

	expr := p.expression()
	stmtExpr := StmtExpr{expr}
	return &StmtSection{ident, &stmtExpr, identToken}
}

func (p *Parser) block() Stmt {
	p.consume(LCurly)
	openingToken := p.prevToken
	stmts := make([]Stmt, 0)
	for p.token.Tag != RCurly {
		stmts = append(stmts, p.statement())
	}
	p.consume(RCurly)
	return &StmtBlock{stmts, &openingToken}
}

func (p *Parser) statement() Stmt {
	switch p.token.Tag {
	case Var:
		return p.varDecl()
	case For:
		return p.forLoop()
	case If:
		return p.ifStmt()
	case Return:
		p.consume(Return)
		expr := p.expression()
		return &StmtReturn{expr}
	case Continue:
		p.consume(Continue)
		return &StmtContinue{}
	case Break:
		p.consume(Break)
		return &StmtBreak{}
	case Match:
		return p.matchStmt()
	case LCurly:
		return p.block()
	default:
		expr := p.expression()
		return &StmtExpr{expr}
	}
}

func (p *Parser) varDecl() Stmt {
	p.consume(Var)
	p.consume(Identifier)
	ident := p.lex.GetString(p.prevToken)
	identToken := &p.prevToken
	p.consume(Equal)
	expr := p.expression()
	return &StmtVar{ident, expr, identToken}
}

func (p *Parser) forLoop() Stmt {
	p.consume(For)
	p.consume(Identifier)
	ident := p.lex.GetString(p.prevToken)
	indexIdent := ""
	if p.token.Tag == Comma {
		p.consume(Comma)
		p.consume(Identifier)
		indexIdent = p.lex.GetString(p.prevToken)
	}
	p.consume(In)
	val := p.expression()
	body := p.block()
	return &StmtFor{ident, indexIdent, val, body}
}

func (p *Parser) ifStmt() Stmt {
	p.consume(If)
	condition := p.expression()
	body := p.block()
	var elseBody Stmt = nil
	if p.token.Tag == Else {
		p.consume(Else)
		if p.token.Tag == If {
			elseBody = p.ifStmt()
		} else {
			elseBody = p.block()
		}
	}
	return &StmtIf{condition, body, elseBody}
}

func (p *Parser) matchStmt() Stmt {
	p.consume(Match)
	val := p.expression()
	p.consume(LCurly)
	cases := make([]MatchCase, 0)
	for p.token.Tag != RCurly {
		cond := p.expression()
		p.consume(Colon)
		body := p.block()
		cases = append(cases, MatchCase{cond, body})
	}
	p.consume(RCurly)
	return &StmtMatch{val, cases}
}

func (p *Parser) expression() Expr {
	return p.expressionWithPrec(PrecNone)
}

func (p *Parser) expressionWithPrec(prec Precedence) Expr {
	if prec == PrecHighest {
		return p.unary()
	}

	lhs := p.expressionWithPrec(prec + 1)

	for {
		op := p.token
		opLevel := PrecNone
		switch op.Tag {
		case Equal:
			opLevel = PrecAssign
		case AmpAmp:
			opLevel = PrecLogical
		case EqualEqual, Greater, GreaterEqual, Less, BangEqual:
			opLevel = PrecCompare
		case Plus, Minus:
			opLevel = PrecSum
		case Star, Slash, Percent:
			opLevel = PrecProduct
		default:
			return lhs
		}

		if opLevel >= prec {
			p.advance()
			rhs := p.expressionWithPrec(prec + 1)
			lhs = &ExprBinary{lhs, rhs, &op}
		} else {
			return lhs
		}
	}
}

func (p *Parser) unary() Expr {
	identToken := p.token
	lhs := p.primary()
	switch p.token.Tag {
	case LParen:
		p.consume(LParen)
		args := make([]Expr, 0)
		for p.token.Tag != RParen {
			arg := p.expression()
			args = append(args, arg)
			if p.token.Tag == Comma {
				p.consume(Comma)
			} else {
				break
			}
		}
		p.consume(RParen)
		return &ExprFuncall{lhs, args, &identToken}
	case LSquare:
		opToken := p.token
		p.consume(LSquare)
		index := p.expression()
		p.consume(RSquare)
		return &ExprBinary{lhs, index, &opToken}
	}
	return lhs
}

func (p *Parser) primary() Expr {
	switch p.token.Tag {
	case Str:
		return p.string()
	case Num:
		return p.number()
	case Nil:
		p.consume(Nil)
		return &ExprNil{&p.prevToken}
	case Identifier:
		return p.identifier()
	case LSquare:
		return p.array()
	case LCurly:
		return p.hashMap()
	case LParen:
		p.consume(LParen)
		expr := p.expression()
		p.consume(RParen)
		return expr
	case Fn:
		return p.fn()
	default:
		panic(p.fmtError("expected a value but found %s", p.token.Tag))
	}
}

func (p *Parser) string() Expr {
	p.consume(Str)
	s := p.lex.GetString(p.prevToken)
	return &ExprString{s, &p.prevToken}
}

func (p *Parser) number() Expr {
	p.consume(Num)
	s := p.lex.GetString(p.prevToken)
	num, err := strconv.Atoi(s)
	if err != nil {
		panic(p.fmtError(err.Error()))
	}
	return &ExprNum{num, &p.prevToken}
}

func (p *Parser) identifier() Expr {
	p.consume(Identifier)
	ident := p.lex.GetString(p.prevToken)
	return &ExprIdentifier{ident, &p.prevToken}
}

func (p *Parser) array() Expr {
	p.consume(LSquare)
	openingToken := &p.prevToken
	items := make([]Expr, 0)
	for p.token.Tag != RSquare {
		items = append(items, p.expression())
		if p.token.Tag == Comma {
			p.consume(Comma)
		}
	}
	p.consume(RSquare)
	return &ExprArray{items, openingToken}
}

func (p *Parser) hashMap() Expr {
	openingToken := p.consume(LCurly)
	items := make([]ExprMapItem, 0)
	for p.token.Tag != RCurly {
		ident := p.consume(Identifier, Num)
		p.consume(Colon)
		val := p.expression()
		item := ExprMapItem{Key: p.lex.GetString(ident), Value: val}
		items = append(items, item)
		if p.token.Tag == Comma {
			p.consume(Comma)
		}
	}
	p.consume(RCurly)
	return &ExprMap{items, &openingToken}
}

func (p *Parser) fn() Expr {
	p.consume(Fn)
	openingToken := p.prevToken

	ident := "<anonymous>"
	if p.token.Tag == Identifier {
		p.consume(Identifier)
		ident = p.lex.GetString(p.prevToken)
	}

	p.consume(LParen)

	args := make([]string, 0)
	for p.token.Tag != RParen {
		p.consume(Identifier)
		args = append(args, p.lex.GetString(p.prevToken))
		if p.token.Tag == Comma {
			p.consume(Comma)
		}
	}

	p.consume(RParen)

	body := p.block()
	return &ExprFunc{
		Identifier:   ident,
		Args:         args,
		Body:         body,
		openingToken: &openingToken,
	}
}

func (p *Parser) Parse() Program {
	p.advance()
	sections := make([]Stmt, 0)
	for !p.atEnd() {
		switch p.token.Tag {
		case Identifier:
			section := p.section()
			sections = append(sections, section)
		case Fn:
			fn := p.fn()
			sections = append(sections, &StmtExpr{fn})
		default:
			// let consume panic
			p.consume(Identifier, Fn)
		}
	}
	return Program{sections}
}
