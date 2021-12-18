package lang

import (
	"go/ast"
)

// interfaces
type (
	Node interface {
		Token() *Token
	}
	Expr interface {
		Node
		exprNode() // type guard
	}
	Stmt interface {
		Node
		stmtNode()
	}
)

//
// program
//
type Program struct {
	Stmts []Stmt // either StmtSection or StmtExpr -> ExprFunc
}

func (p *Program) Pos() int      { return 0 }
func (p *Program) Token() *Token { return nil }

//
// expressions
//
type ExprString struct {
	Str   string
	token Token
}

type ExprIdentifier struct {
	Identifier string
	token      Token
}

type ExprNum struct {
	Num   int
	token Token
}

type ExprNil struct {
	token Token
}

type ExprArray struct {
	Items        []Expr
	openingToken Token
}

type ExprMap struct {
	Items        []ExprMapItem
	openingtoken Token
}

type ExprMapItem struct {
	Key   string
	Value Expr
}

type ExprBinary struct {
	Lhs Expr
	Rhs Expr
	Op  Token
}

type ExprFuncall struct {
	Identifier      Expr
	Args            []Expr
	identifierToken Token
}

type ExprFunc struct {
	Identifier   string
	Args         []string
	Body         Stmt
	openingToken Token
}

func (e *ExprString) Token() *Token     { return &e.token }
func (e *ExprIdentifier) Token() *Token { return &e.token }
func (e *ExprNum) Token() *Token        { return &e.token }
func (e *ExprNil) Token() *Token        { return &e.token }
func (e *ExprArray) Token() *Token      { return &e.openingToken }
func (e *ExprMap) Token() *Token        { return &e.openingtoken }
func (e *ExprBinary) Token() *Token     { return &e.Op }
func (e *ExprFuncall) Token() *Token    { return &e.identifierToken }
func (e *ExprFunc) Token() *Token       { return &e.openingToken }

func (*ExprString) exprNode()     {}
func (*ExprIdentifier) exprNode() {}
func (*ExprNum) exprNode()        {}
func (*ExprNil) exprNode()        {}
func (*ExprArray) exprNode()      {}
func (*ExprMap) exprNode()        {}
func (*ExprBinary) exprNode()     {}
func (*ExprFuncall) exprNode()    {}
func (*ExprFunc) exprNode()       {}

//
// statements
//
type StmtExpr struct {
	Expr Expr
}

type StmtBlock struct {
	Body         []Stmt
	openingToken Token
}

type StmtVar struct {
	Identifier      string
	Value           Expr
	identifierToken Token
}

type StmtFor struct {
	Identifier      string
	IndexIdentifier string
	Value           Expr
	body            Stmt
}

type StmtIf struct {
	Condition Expr
	Body      Stmt
	ElseBody  Stmt
}

type StmtReturn struct {
	Value Expr
}

type StmtMatch struct {
	Value Expr
	Cases []MatchCase
}

type MatchCase struct {
	Cond Expr
	Body Stmt
}

type StmtContinue struct {
	token Token
}

type StmtBreak struct {
	token Token
}

type StmtSection struct {
	Label      string
	Body       Stmt
	labelToken Token
}

func (s *StmtExpr) Token() *Token     { return s.Expr.Token() }
func (s *StmtBlock) Token() *Token    { return &s.openingToken }
func (s *StmtVar) Token() *Token      { return &s.identifierToken }
func (s *StmtFor) Token() *Token      { return s.Value.Token() }
func (s *StmtIf) Token() *Token       { return s.Condition.Token() }
func (s *StmtReturn) Token() *Token   { return s.Value.Token() }
func (s *StmtMatch) Token() *Token    { return s.Value.Token() }
func (s *StmtContinue) Token() *Token { return &s.token }
func (s *StmtBreak) Token() *Token    { return &s.token }
func (s *StmtSection) Token() *Token  { return &s.labelToken }

func (*StmtExpr) stmtNode()     {}
func (*StmtBlock) stmtNode()    {}
func (*StmtVar) stmtNode()      {}
func (*StmtFor) stmtNode()      {}
func (*StmtIf) stmtNode()       {}
func (*StmtReturn) stmtNode()   {}
func (*StmtMatch) stmtNode()    {}
func (*StmtContinue) stmtNode() {}
func (*StmtBreak) stmtNode()    {}
func (*StmtSection) stmtNode()  {}

func PrettyPrint(prog *Program) {
	ast.Print(nil, prog)
}
