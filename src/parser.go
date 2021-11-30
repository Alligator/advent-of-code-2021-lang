package lang

import (
	"fmt"
	"strconv"
)

type Parser struct {
	lex       *Lexer
	token     Token
	prevToken Token
}

func NewParser(lex *Lexer) Parser {
	eof := Token{EOF, 0, 0}
	return Parser{lex, eof, eof}
}

func (p *Parser) atEnd() bool {
	return p.token.Tag == EOF
}

func (p *Parser) fmtError(msg string) string {
	line, col := p.lex.GetLineAndCol(p.token)
	return fmt.Sprintf("parse error on line %d col %d: %s\n", line, col, msg)
}

func (p *Parser) advance() {
	p.prevToken = p.token
	p.token = p.lex.NextToken()
}

func (p *Parser) consume(expected TokenTag) {
	if p.token.Tag != expected {
		panic(p.fmtError(fmt.Sprintf("expected %s but saw %s", expected, p.token.Tag)))
	}
	p.advance()
}

func (p *Parser) section() Section {
	p.consume(Identifier)
	ident := p.lex.GetString(p.prevToken)
	p.consume(Colon)

	if p.token.Tag == LCurly {
		block := p.block()
		return &SectionBlock{ident, block}
	}

	expr := p.expression()
	return &SectionExpr{ident, expr}
}

func (p *Parser) block() []Stmt {
	p.consume(LCurly)
	stmts := make([]Stmt, 0)
	for p.token.Tag != RCurly {
		stmts = append(stmts, p.statement())
	}
	p.consume(RCurly)
	return stmts
}

func (p *Parser) statement() Stmt {
	switch p.token.Tag {
	case Var:
		return p.varDecl()
	case For:
		return p.forLoop()

	default:
		expr := p.expression()
		return &StmtExpr{expr}
		// case Identifier:
		// 	lhs := p.identifier()
		// 	switch p.token.Tag {
		// 	case LParen:
		// 		return p.funcall()
		// 	default:
		// 		return p.statement()
		// 	}
		// default:
		// 	panic(fmt.Sprintf("unexpected token %s\n", p.token.Tag.String()))
	}
}

func (p *Parser) varDecl() Stmt {
	p.consume(Var)
	p.consume(Identifier)
	ident := p.lex.GetString(p.prevToken)
	p.consume(Equal)
	expr := p.expression()
	return &StmtVar{ident, expr}
}

func (p *Parser) forLoop() Stmt {
	p.consume(For)
	p.consume(Identifier)
	ident := p.lex.GetString(p.prevToken)
	p.consume(In)
	val := p.expression()
	body := p.block()
	return &StmtFor{ident, val, body}
}

// func (p *Parser) funcall() Stmt {
// 	ident := p.lex.GetString(p.prevToken)
// 	p.consume(LParen)
// 	expr := p.expression()
// 	p.consume(RParen)
// 	return &StmtFuncall{ident, expr}
// }

func (p *Parser) expression() Expr {
	lhs := p.value()

	switch p.token.Tag {
	case Equal:
		fallthrough
	case Plus:
		op := p.token.Tag
		p.advance()
		rhs := p.expression()
		return &ExprBinary{lhs, rhs, op}
	case LParen:
		p.consume(LParen)
		expr := p.expression()
		p.consume(RParen)
		return &ExprFuncall{lhs, expr}
	default:
		return lhs
	}
}

func (p *Parser) value() Expr {
	switch p.token.Tag {
	case Str:
		return p.string()
	case Num:
		return p.number()
	case Identifier:
		return p.identifier()
	default:
		panic(p.fmtError("expected a value"))
	}
}

func (p *Parser) string() Expr {
	p.consume(Str)
	s := p.lex.GetString(p.prevToken)
	return &ExprString{s}
}

func (p *Parser) number() Expr {
	p.consume(Num)
	s := p.lex.GetString(p.prevToken)
	num, err := strconv.Atoi(s)
	if err != nil {
		panic(p.fmtError(err.Error()))
	}
	return &ExprNum{num}
}

func (p *Parser) identifier() Expr {
	p.consume(Identifier)
	ident := p.lex.GetString(p.prevToken)
	return &ExprIdentifier{ident}
}

func (p *Parser) Parse() Program {
	p.advance()
	sections := make([]Section, 0)
	for !p.atEnd() {
		section := p.section()
		sections = append(sections, section)
	}
	return Program{sections}
}
