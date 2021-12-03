package lang

import "fmt"

type Program struct {
	sections []Section
}

type Section interface {
	getName() string
	sectionNode() // type guard
}

type SectionExpr struct {
	label      string
	expression Expr
}

func (node *SectionExpr) getName() string { return node.label }

type SectionBlock struct {
	label string
	block []Stmt
}

func (node *SectionBlock) getName() string { return node.label }

type Expr interface {
	exprNode() // type guard
}

type ExprString struct {
	str string
}

type ExprIdentifier struct {
	identifier string
}

type ExprNum struct {
	num int
}

type ExprArray struct {
	items []Expr
}

type ExprBinary struct {
	lhs Expr
	rhs Expr
	op  TokenTag
}

type ExprFuncall struct {
	identifier Expr
	args       []Expr
}

type Stmt interface {
	stmtNode()
}

type StmtExpr struct {
	expr Expr
}

type StmtVar struct {
	identifier string
	value      Expr
}

type StmtFor struct {
	identifier string
	value      Expr
	body       []Stmt
}

type StmtIf struct {
	condition Expr
	body      []Stmt
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
	body []Stmt
}

type StmtContinue struct{}

// impelement type guards
func (*SectionExpr) sectionNode()  {}
func (*SectionBlock) sectionNode() {}

func (*ExprString) exprNode()     {}
func (*ExprIdentifier) exprNode() {}
func (*ExprNum) exprNode()        {}
func (*ExprArray) exprNode()      {}
func (*ExprFuncall) exprNode()    {}
func (*ExprBinary) exprNode()     {}

func (*StmtExpr) stmtNode()     {}
func (*StmtVar) stmtNode()      {}
func (*StmtFor) stmtNode()      {}
func (*StmtIf) stmtNode()       {}
func (*StmtReturn) stmtNode()   {}
func (*StmtContinue) stmtNode() {}
func (*StmtMatch) stmtNode()    {}

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
		ap.depth++
		for _, stmt := range node.block {
			ap.printStmt(&stmt)
		}
		ap.depth--
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
		for _, stmt := range node.body {
			ap.printStmt(&stmt)
		}
		ap.depth--
	case *StmtExpr:
		ap.printIndented("StmtExpr")
		ap.depth++
		ap.printExpr(&node.expr)
		ap.depth--
	case *StmtIf:
		ap.printIndented("StmtIf")
		ap.printExpr(&node.condition)
		for _, stmt := range node.body {
			ap.printStmt(&stmt)
		}
	case *StmtMatch:
		ap.printIndented("StmtMatch")
		ap.depth++
		ap.printExpr(&node.value)
		for _, c := range node.cases {
			ap.printIndented("MatchCase")
			ap.depth++
			ap.printExpr(&c.cond)
			for _, s := range c.body {
				ap.printStmt(&s)
			}
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
		ap.printIndented("ExprBinary", node.op.String())
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
	default:
		ap.printIndented("UNKNOWN", fmt.Sprintf("%#v", node))
	}
}

func PrettyPrint(prog *Program) {
	ap := AstPrinter{0}
	ap.printProgram(prog)
}
