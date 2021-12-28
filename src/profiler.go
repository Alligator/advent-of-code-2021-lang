package lang

import (
	"fmt"
	"time"
)

func (ev *Evaluator) PrintProfile() {
  ev.printProfile(ev.prog, 0)
}

func (ev *Evaluator) printProfile(node Node, depth int) {
  var duration time.Duration
  for _, evt := range ev.profileEvents {
    if evt.node == node {
      duration += evt.end.Sub(evt.start)
    }
  }

  nextDepth := depth
  if duration > 0 {
    tok := node.Token()
    line := 0
    if tok != nil {
      line, _ = ev.lex.GetLineAndCol(*node.Token())
    }
    fmt.Printf("%8dms %*s%s:%d\n", duration.Milliseconds(), depth * 2, "", node.Name(), line)
    nextDepth++
  }

  switch n := node.(type) {
  case *Program:
    for _, stmt := range n.Stmts {
      ev.printProfile(stmt, nextDepth)
    }
  case *StmtBlock:
    for _, stmt := range n.Body {
      ev.printProfile(stmt, nextDepth)
    }
  case *StmtExpr:
    ev.printProfile(n.Expr, nextDepth)
  case *StmtVar:
    ev.printProfile(n.Value, nextDepth)
  case *ExprFunc:
    ev.printProfile(n.Body, nextDepth)
  case *ExprBinary:
    ev.printProfile(n.Lhs, nextDepth)
    ev.printProfile(n.Rhs, nextDepth)
  case *ExprUnary:
    ev.printProfile(n.Lhs, nextDepth)
  }
}
