package pg

// Framework code is generated by the generator.

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/ashutoshojha5/bytebase/backend/plugin/advisor"
	"github.com/ashutoshojha5/bytebase/backend/plugin/parser/sql/ast"
	storepb "github.com/ashutoshojha5/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*InsertDisallowOrderByRandAdvisor)(nil)
	_ ast.Visitor     = (*insertDisallowOrderByRandChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.PostgreSQLInsertDisallowOrderByRand, &InsertDisallowOrderByRandAdvisor{})
}

// InsertDisallowOrderByRandAdvisor is the advisor checking for to disallow order by rand in INSERT statements.
type InsertDisallowOrderByRandAdvisor struct {
}

// Check checks for to disallow order by rand in INSERT statements.
func (*InsertDisallowOrderByRandAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker := &insertDisallowOrderByRandChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}

	for _, stmt := range stmtList {
		checker.text = advisor.NormalizeStatement(stmt.Text())
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

type insertDisallowOrderByRandChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	text       string
}

// Visit implements ast.Visitor interface.
func (checker *insertDisallowOrderByRandChecker) Visit(in ast.Node) ast.Visitor {
	code := advisor.Ok
	if insert, ok := in.(*ast.InsertStmt); ok && insert.Select != nil {
		for _, item := range insert.Select.OrderByClause {
			if strings.Contains(item.Expression.Text(), "random()") ||
				strings.Contains(item.Expression.Text(), "random_between()") {
				code = advisor.InsertUseOrderByRand
				break
			}
		}
	}

	if code != advisor.Ok {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.level,
			Code:    code,
			Title:   checker.title,
			Content: fmt.Sprintf("The INSERT statement uses ORDER BY random() or random_between(), related statement \"%s\"", checker.text),
			Line:    in.LastLine(),
		})
	}

	return checker
}
