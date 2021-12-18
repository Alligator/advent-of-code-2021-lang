package lang

import (
	"fmt"
	"strconv"
	"strings"
)

type rule struct {
	prec   Precedence
	prefix func(*Parser) Expr
	infix  func(*Parser, Expr) Expr
}

type Parser struct {
	lex       *Lexer
	token     Token
	prevToken Token
	rules     map[TokenTag]rule
}

type Precedence uint8

const (
	PrecNone Precedence = iota
	PrecAssign
	PrecLogical
	PrecCompare
	PrecSum
	PrecProduct
	PrecCall
	PrecHighest
)

func NewParser(lex *Lexer) Parser {
	eof := Token{EOF, 0, 0}
	p := Parser{
		lex:       lex,
		token:     eof,
		prevToken: eof,
	}
	p.makeRules()
	return p
}

func (p *Parser) makeRules() {
	rules := map[TokenTag]rule{
		Str:          {PrecNone, str, nil},
		Num:          {PrecNone, number, nil},
		Nil:          {PrecNone, nilExpr, nil},
		Identifier:   {PrecNone, identifier, nil},
		LSquare:      {PrecCall, array, subscript},
		LParen:       {PrecCall, group, call},
		LCurly:       {PrecNone, hashMap, nil},
		Fn:           {PrecNone, fn, nil},
		Equal:        {PrecAssign, nil, binary},
		AmpAmp:       {PrecLogical, nil, binary},
		EqualEqual:   {PrecCompare, nil, binary},
		Greater:      {PrecCompare, nil, binary},
		GreaterEqual: {PrecCompare, nil, binary},
		Less:         {PrecCompare, nil, binary},
		BangEqual:    {PrecCompare, nil, binary},
		Plus:         {PrecSum, nil, binary},
		Minus:        {PrecSum, nil, binary},
		Star:         {PrecProduct, nil, binary},
		Slash:        {PrecProduct, nil, binary},
		Percent:      {PrecProduct, nil, binary},
	}

	p.rules = rules
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
	identToken := p.prevToken
	p.consume(Colon)

	if p.token.Tag == LCurly {
		block := p.block()
		return &StmtSection{ident, block, p.token}
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
	return &StmtBlock{stmts, openingToken}
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
	identToken := p.prevToken
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
	return p.expressionWithPrec(PrecAssign)
}

func (p *Parser) expressionWithPrec(prec Precedence) Expr {
	prefixRule := p.rules[p.token.Tag]
	if prefixRule.prefix == nil {
		panic(p.fmtError("unexpected %s", p.token.Tag.String()))
	}

	// p.advance()
	lhs := prefixRule.prefix(p)

	for prec <= p.rules[p.token.Tag].prec {
		infixRule := p.rules[p.token.Tag]
		if infixRule.infix == nil {
			panic(p.fmtError("unknown operator %s", p.token.Tag.String()))
		}
		lhs = infixRule.infix(p, lhs)
	}
	return lhs
}

func binary(p *Parser, lhs Expr) Expr {
	p.advance()
	op := p.prevToken
	rhs := p.expressionWithPrec(p.rules[op.Tag].prec)
	return &ExprBinary{lhs, rhs, op}
}

func str(p *Parser) Expr {
	p.consume(Str)
	s := p.lex.GetString(p.prevToken)
	return &ExprString{s, p.prevToken}
}

func number(p *Parser) Expr {
	p.consume(Num)
	s := p.lex.GetString(p.prevToken)
	num, err := strconv.Atoi(s)
	if err != nil {
		panic(p.fmtError(err.Error()))
	}
	return &ExprNum{num, p.prevToken}
}

func nilExpr(p *Parser) Expr {
	p.consume(Nil)
	return &ExprNil{p.prevToken}
}

func identifier(p *Parser) Expr {
	p.consume(Identifier)
	ident := p.lex.GetString(p.prevToken)
	return &ExprIdentifier{ident, p.prevToken}
}

func array(p *Parser) Expr {
	p.consume(LSquare)
	openingToken := p.prevToken
	items := make([]Expr, 0)
	for p.token.Tag != RSquare {
		items = append(items, p.expression())
		if p.token.Tag != Comma {
			break
		}
		p.consume(Comma)
	}
	p.consume(RSquare)
	return &ExprArray{items, openingToken}
}

func hashMap(p *Parser) Expr {
	openingToken := p.consume(LCurly)
	items := make([]ExprMapItem, 0)
	for p.token.Tag != RCurly {
		ident := p.consume(Identifier, Num)
		p.consume(Colon)
		val := p.expression()
		item := ExprMapItem{Key: p.lex.GetString(ident), Value: val}
		items = append(items, item)
		if p.token.Tag != Comma {
			break
		}
		p.consume(Comma)
	}
	p.consume(RCurly)
	return &ExprMap{items, openingToken}
}

func group(p *Parser) Expr {
	p.consume(LParen)
	expr := p.expression()
	p.consume(RParen)
	return expr
}

func fn(p *Parser) Expr {
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
		if p.token.Tag != Comma {
			break
		}
		p.consume(Comma)
	}

	p.consume(RParen)

	body := p.block()
	return &ExprFunc{
		Identifier:   ident,
		Args:         args,
		Body:         body,
		openingToken: openingToken,
	}
}

func call(p *Parser, lhs Expr) Expr {
	p.consume(LParen)
	args := make([]Expr, 0)
	for p.token.Tag != RParen {
		arg := p.expression()
		args = append(args, arg)
		if p.token.Tag != Comma {
			break
		}
		p.consume(Comma)
	}
	p.consume(RParen)
	return &ExprFuncall{lhs, args, *lhs.Token()}
}

func subscript(p *Parser, lhs Expr) Expr {
	opToken := p.token
	p.consume(LSquare)
	index := p.expression()
	p.consume(RSquare)
	return &ExprBinary{lhs, index, opToken}
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
			fn := fn(p)
			sections = append(sections, &StmtExpr{fn})
		default:
			// let consume panic
			p.consume(Identifier, Fn)
		}
	}
	return Program{sections}
}
