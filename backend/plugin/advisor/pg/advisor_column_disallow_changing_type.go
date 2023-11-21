package pg

// Framework code is generated by the generator.

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/ashutoshojha5/bytebase/backend/plugin/advisor"
	"github.com/ashutoshojha5/bytebase/backend/plugin/parser/sql/ast"
	storepb "github.com/ashutoshojha5/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*ColumnDisallowChangingTypeAdvisor)(nil)
	_ ast.Visitor     = (*columnDisallowChangingTypeChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.PostgreSQLColumnDisallowChangingType, &ColumnDisallowChangingTypeAdvisor{})
}

// ColumnDisallowChangingTypeAdvisor is the advisor checking for disallow changing column type.
type ColumnDisallowChangingTypeAdvisor struct {
}

// Check checks for disallow changing column type.
func (*ColumnDisallowChangingTypeAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &columnDisallowChangingTypeChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}

	for _, stmt := range stmtList {
		checker.text = stmt.Text()
		checker.line = stmt.LastLine()
		ast.Walk(checker, stmt)
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

type columnDisallowChangingTypeChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	text       string
	line       int
	title      string
}

// Visit implements ast.Visitor interface.
func (checker *columnDisallowChangingTypeChecker) Visit(in ast.Node) ast.Visitor {
	code := advisor.Ok
	if _, ok := in.(*ast.AlterColumnTypeStmt); ok {
		code = advisor.ChangeColumnType
	}

	if code != advisor.Ok {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    code,
			Title:   checker.title,
			Content: fmt.Sprintf("The statement \"%s\" changes column type", checker.text),
			Line:    checker.line,
		})
	}

	return checker
}
