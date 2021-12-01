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

type ExprBinary struct {
	lhs Expr
	rhs Expr
	op  TokenTag
}

type ExprFuncall struct {
	identifier Expr
	arg        Expr
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

type StmtContinue struct{}

// impelement type guards
func (*SectionExpr) sectionNode()  {}
func (*SectionBlock) sectionNode() {}

func (*ExprString) exprNode()     {}
func (*ExprIdentifier) exprNode() {}
func (*ExprNum) exprNode()        {}
func (*ExprFuncall) exprNode()    {}
func (*ExprBinary) exprNode()     {}

func (*StmtExpr) stmtNode()     {}
func (*StmtVar) stmtNode()      {}
func (*StmtFor) stmtNode()      {}
func (*StmtIf) stmtNode()       {}
func (*StmtReturn) stmtNode()   {}
func (*StmtContinue) stmtNode() {}

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
		ap.printIndented("SectionBlock")
		ap.printIndented("label:", node.label)
		ap.depth++
		for _, stmt := range node.block {
			ap.printStmt(&stmt)
		}
		ap.depth--
		ap.depth--
	case *SectionExpr:
		ap.depth++
		ap.printIndented("SectionExpr")
		ap.printIndented("label:", node.label)
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
	ap.printIndented("Stmt")
	ap.depth++
	switch node := (*stmt).(type) {
	case *StmtVar:
		ap.printIndented("StmtVar")
		ap.printIndented("ident:", node.identifier)
		ap.printExpr(&node.value)
	case *StmtFor:
		ap.printIndented("StmtFor")
		ap.printIndented("ident:", node.identifier)
		ap.printExpr(&node.value)
		for _, stmt := range node.body {
			ap.printStmt(&stmt)
		}
	case *StmtExpr:
		ap.printIndented("StmtExpr")
		ap.printExpr(&node.expr)
	case *StmtIf:
		ap.printIndented("StmtIf")
		ap.printExpr(&node.condition)
		for _, stmt := range node.body {
			ap.printStmt(&stmt)
		}
	default:
		ap.printIndented("UNKNOWN", fmt.Sprintf("%#v", node))
	}
	ap.depth--
}

func (ap *AstPrinter) printExpr(expr *Expr) {
	ap.printIndented("Expr")
	ap.depth++
	switch node := (*expr).(type) {
	case *ExprString:
		ap.printIndented("ExprString", "'"+node.str+"'")
	case *ExprIdentifier:
		ap.printIndented("ExprIdentifier", node.identifier)
	case *ExprNum:
		ap.printIndented("ExprNum", node.num)
	case *ExprBinary:
		ap.printIndented("ExprBinary")
		ap.printExpr(&node.lhs)
		ap.printIndented("op:", node.op)
		ap.printExpr(&node.rhs)
	case *ExprFuncall:
		ap.printIndented("ExprFuncall")
		ap.printIndented("func:", node.identifier)
		ap.printExpr(&node.arg)
	default:
		ap.printIndented("UNKNOWN", fmt.Sprintf("%#v", node))
	}
	ap.depth--
}

func PrettyPrint(prog *Program) {
	ap := AstPrinter{0}
	ap.printProgram(prog)
}
