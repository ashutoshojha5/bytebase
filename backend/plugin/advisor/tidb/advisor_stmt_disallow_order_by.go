package tidb

// Framework code is generated by the generator.

import (
	"fmt"

	"github.com/pingcap/tidb/parser/ast"
	"github.com/pkg/errors"

	"github.com/ashutoshojha5/bytebase/backend/plugin/advisor"
	storepb "github.com/ashutoshojha5/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*DisallowOrderByAdvisor)(nil)
	_ ast.Visitor     = (*disallowOrderByChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, advisor.MySQLDisallowOrderBy, &DisallowOrderByAdvisor{})
}

// DisallowOrderByAdvisor is the advisor checking for no ORDER BY clause in DELETE/UPDATE statements.
type DisallowOrderByAdvisor struct {
}

// Check checks for no ORDER BY clause in DELETE/UPDATE statements.
func (*DisallowOrderByAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]ast.StmtNode)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &disallowOrderByChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}

	for _, stmt := range stmtList {
		checker.text = stmt.Text()
		checker.line = stmt.OriginTextPosition()
		(stmt).Accept(checker)
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return checker.adviceList, nil
}

type disallowOrderByChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	text       string
	line       int
}

// Enter implements the ast.Visitor interface.
func (checker *disallowOrderByChecker) Enter(in ast.Node) (ast.Node, bool) {
	code := advisor.Ok
	switch node := in.(type) {
	case *ast.UpdateStmt:
		if node.Order != nil {
			code = advisor.UpdateUseOrderBy
		}
	case *ast.DeleteStmt:
		if node.Order != nil {
			code = advisor.DeleteUseOrderBy
		}
	}

	if code != advisor.Ok {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    code,
			Title:   checker.title,
			Content: fmt.Sprintf("ORDER BY clause is forbidden in DELETE and UPDATE statements, but \"%s\" uses", checker.text),
			Line:    checker.line,
		})
	}
	return in, false
}

// Leave implements the ast.Visitor interface.
func (*disallowOrderByChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
