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
	case If:
		return p.ifStmt()
	case Return:
		p.consume(Return)
		expr := p.expression()
		return &StmtReturn{expr}
	case Continue:
		p.consume(Continue)
		return &StmtContinue{}
	case Match:
		return p.matchStmt()
	default:
		expr := p.expression()
		return &StmtExpr{expr}
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

func (p *Parser) ifStmt() Stmt {
	p.consume(If)
	condition := p.expression()
	body := p.block()
	return &StmtIf{condition, body}
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
	lhs := p.unary()

	switch p.token.Tag {
	case Equal, EqualEqual, Star, Plus, Greater, GreaterEqual, Minus:
		op := p.token.Tag
		p.advance()
		rhs := p.expression()
		return &ExprBinary{lhs, rhs, op}
	default:
		return lhs
	}
}

func (p *Parser) unary() Expr {
	lhs := p.primary()
	switch p.token.Tag {
	case LParen:
		p.consume(LParen)
		args := make([]Expr, 0)
		for {
			arg := p.expression()
			args = append(args, arg)
			if p.token.Tag == Comma {
				p.consume(Comma)
			} else {
				break
			}
		}
		p.consume(RParen)
		return &ExprFuncall{lhs, args}
	}
	return lhs
}

func (p *Parser) primary() Expr {
	switch p.token.Tag {
	case Str:
		return p.string()
	case Num:
		return p.number()
	case Identifier:
		return p.identifier()
	case LSquare:
		return p.array()
	case LParen:
		p.consume(LParen)
		expr := p.expression()
		p.consume(RParen)
		return expr
	default:
		panic(p.fmtError(fmt.Sprintf("expected a value but found %s", p.token.Tag)))
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

func (p *Parser) array() Expr {
	p.consume(LSquare)
	items := make([]Expr, 0)
	for p.token.Tag != RSquare {
		items = append(items, p.expression())
		if p.token.Tag == Comma {
			p.consume(Comma)
		}
	}
	p.consume(RSquare)
	return &ExprArray{items}
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
