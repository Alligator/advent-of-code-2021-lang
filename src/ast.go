package lang

import "fmt"

// interfaces
type (
	Node interface {
		Token() *Token
	}
	Section interface {
		Node
		getName() string
		sectionNode() // type guard
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
	sections []Section
}

func (p *Program) Pos() int { return 0 }

//
// sections
//
type SectionExpr struct {
	label      string
	expression Expr
	labelToken *Token
}

type SectionBlock struct {
	label        string
	block        Stmt
	openingToken *Token
}

func (node *SectionExpr) Token() *Token  { return node.labelToken }
func (node *SectionBlock) Token() *Token { return node.openingToken }

func (node *SectionExpr) getName() string  { return node.label }
func (node *SectionBlock) getName() string { return node.label }

func (*SectionExpr) sectionNode()  {}
func (*SectionBlock) sectionNode() {}

//
// expressions
//
type ExprString struct {
	str   string
	token *Token
}

type ExprIdentifier struct {
	identifier string
	token      *Token
}

type ExprNum struct {
	num   int
	token *Token
}

type ExprArray struct {
	items        []Expr
	openingToken *Token
}

type ExprBinary struct {
	lhs Expr
	rhs Expr
	op  *Token
}

type ExprFuncall struct {
	identifier      Expr
	args            []Expr
	identifierToken *Token
}

type ExprFunc struct {
	identifier      string
	args            []string
	body            Stmt
	identifierToken *Token
}

func (e *ExprString) Token() *Token     { return e.token }
func (e *ExprIdentifier) Token() *Token { return e.token }
func (e *ExprNum) Token() *Token        { return e.token }
func (e *ExprArray) Token() *Token      { return e.openingToken }
func (e *ExprBinary) Token() *Token     { return e.op }
func (e *ExprFuncall) Token() *Token    { return e.identifierToken }
func (e *ExprFunc) Token() *Token       { return e.identifierToken }

func (*ExprString) exprNode()     {}
func (*ExprIdentifier) exprNode() {}
func (*ExprNum) exprNode()        {}
func (*ExprArray) exprNode()      {}
func (*ExprBinary) exprNode()     {}
func (*ExprFuncall) exprNode()    {}
func (*ExprFunc) exprNode()       {}

//
// statements
//
type StmtExpr struct {
	expr Expr
}

type StmtBlock struct {
	body         []Stmt
	openingToken *Token
}

type StmtVar struct {
	identifier      string
	value           Expr
	identifierToken *Token
}

type StmtFor struct {
	identifier      string
	indexIdentifier string
	value           Expr
	body            Stmt
}

type StmtIf struct {
	condition Expr
	body      Stmt
	elseBody  Stmt
}

type StmtReturn struct {
	value Expr
}

type StmtMatch struct {
	value Expr
	cases []MatchCase
}

type MatchCase struct {
	cond Expr
	body Stmt
}

type StmtContinue struct {
	token *Token
}

type StmtBreak struct {
	token *Token
}

func (s *StmtExpr) Token() *Token     { return s.expr.Token() }
func (s StmtBlock) Token() *Token     { return s.openingToken }
func (s *StmtVar) Token() *Token      { return s.identifierToken }
func (s *StmtFor) Token() *Token      { return s.value.Token() }
func (s *StmtIf) Token() *Token       { return s.condition.Token() }
func (s *StmtReturn) Token() *Token   { return s.value.Token() }
func (s *StmtMatch) Token() *Token    { return s.value.Token() }
func (s *StmtContinue) Token() *Token { return s.token }
func (s *StmtBreak) Token() *Token    { return s.token }

func (*StmtExpr) stmtNode()     {}
func (*StmtBlock) stmtNode()    {}
func (*StmtVar) stmtNode()      {}
func (*StmtFor) stmtNode()      {}
func (*StmtIf) stmtNode()       {}
func (*StmtReturn) stmtNode()   {}
func (*StmtMatch) stmtNode()    {}
func (*StmtContinue) stmtNode() {}
func (*StmtBreak) stmtNode()    {}

type AstPrinter struct {
	depth uint8
}

func (ap *AstPrinter) printIndented(strs ...interface{}) {
	fmt.Printf("%*s", ap.depth*2, "")
	fmt.Println(strs...)
}

func (ap *AstPrinter) printProgram(prog *Program) {
	ap.printIndented("Program")
	ap.depth++
	for _, section := range prog.sections {
		ap.printSection(&section)
	}
	ap.depth--
}

func (ap *AstPrinter) printSection(section *Section) {
	ap.printIndented("Section")
	ap.depth++
	switch node := (*section).(type) {
	case *SectionBlock:
		ap.depth++
		ap.printIndented("SectionBlock", node.label)
		ap.printStmt(&node.block)
		ap.depth--
	case *SectionExpr:
		ap.depth++
		ap.printIndented("SectionExpr", node.label)
		ap.depth++
		ap.printExpr(&node.expression)
		ap.depth--
		ap.depth--
	default:
		ap.printIndented("UNKNOWN", fmt.Sprintf("%#v", node))
	}
	ap.depth--
}

func (ap *AstPrinter) printStmt(stmt *Stmt) {
	switch node := (*stmt).(type) {
	case *StmtVar:
		ap.printIndented("StmtVar", node.identifier)
		ap.depth++
		ap.printExpr(&node.value)
		ap.depth--
	case *StmtFor:
		ap.printIndented("StmtFor", node.identifier)
		ap.depth++
		ap.printExpr(&node.value)
		ap.printStmt(&node.body)
		ap.depth--
	case *StmtExpr:
		ap.printIndented("StmtExpr")
		ap.depth++
		ap.printExpr(&node.expr)
		ap.depth--
	case *StmtBlock:
		ap.printIndented("StmtBlockr")
		ap.depth++
		for _, stmt := range node.body {
			ap.printStmt(&stmt)
		}
		ap.depth--
	case *StmtIf:
		ap.printIndented("StmtIf")
		ap.depth++
		ap.printExpr(&node.condition)
		ap.printStmt(&node.body)
		if node.elseBody != nil {
			ap.printStmt(&node.elseBody)
		}
		ap.depth--
	case *StmtMatch:
		ap.printIndented("StmtMatch")
		ap.depth++
		ap.printExpr(&node.value)
		for _, c := range node.cases {
			ap.printIndented("MatchCase")
			ap.depth++
			ap.printExpr(&c.cond)
			ap.printStmt(&c.body)
			ap.depth--
		}
		ap.depth--
	default:
		ap.printIndented("UNKNOWN", fmt.Sprintf("%#v", node))
	}
}

func (ap *AstPrinter) printExpr(expr *Expr) {
	switch node := (*expr).(type) {
	case *ExprString:
		ap.printIndented("ExprString", "\""+node.str+"\"")
	case *ExprIdentifier:
		ap.printIndented("ExprIdentifier", node.identifier)
	case *ExprNum:
		ap.printIndented("ExprNum", node.num)
	case *ExprBinary:
		ap.printIndented("ExprBinary", node.op.Tag.String())
		ap.depth++
		ap.printExpr(&node.lhs)
		ap.printExpr(&node.rhs)
		ap.depth--
	case *ExprFuncall:
		ap.printIndented("ExprFuncall", node.identifier)
		ap.depth++
		for _, arg := range node.args {
			ap.printExpr(&arg)
		}
		ap.depth--
	case *ExprArray:
		ap.printIndented("Array")
		ap.depth++
		for _, v := range node.items {
			ap.printExpr(&v)
		}
		ap.depth--
	case *ExprFunc:
		ap.printIndented("ExprFunc")
		ap.depth++
		for _, a := range node.args {
			ap.printIndented("arg", a)
		}
		ap.printStmt(&node.body)
		ap.depth--
	default:
		ap.printIndented("UNKNOWN", fmt.Sprintf("%#v", node))
	}
}

func PrettyPrint(prog *Program) {
	ap := AstPrinter{0}
	ap.printProgram(prog)
}
