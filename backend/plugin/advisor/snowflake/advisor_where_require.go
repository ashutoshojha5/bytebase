// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/snowsql-parser"
	"github.com/pkg/errors"

	"github.com/ashutoshojha5/bytebase/backend/plugin/advisor"
	storepb "github.com/ashutoshojha5/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*WhereRequireAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, advisor.SnowflakeWhereRequirement, &WhereRequireAdvisor{})
}

// WhereRequireAdvisor is the advisor checking for WHERE clause requirement.
type WhereRequireAdvisor struct {
}

// Check checks for WHERE clause requirement.
func (*WhereRequireAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	tree, ok := ctx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	listener := &whereRequireChecker{
		level: level,
		title: string(ctx.Rule.Type),
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.generateAdvice()
}

// whereRequireChecker is the listener for WHERE clause requirement.
type whereRequireChecker struct {
	*parser.BaseSnowflakeParserListener

	level advisor.Status
	title string

	adviceList []advisor.Advice
}

// generateAdvice returns the advices generated by the listener, the advices must not be empty.
func (l *whereRequireChecker) generateAdvice() ([]advisor.Advice, error) {
	if len(l.adviceList) == 0 {
		l.adviceList = append(l.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return l.adviceList, nil
}

// EnterUpdate_statement is called when production update_statement is entered.
func (l *whereRequireChecker) EnterUpdate_statement(ctx *parser.Update_statementContext) {
	if ctx.WHERE() == nil {
		l.adviceList = append(l.adviceList, advisor.Advice{
			Status:  l.level,
			Code:    advisor.StatementNoWhere,
			Title:   l.title,
			Content: "WHERE clause is required for UPDATE statement.",
			Line:    ctx.GetStart().GetLine(),
		})
	}
}

// EnterDelete_statement is called when production delete_statement is entered.
func (l *whereRequireChecker) EnterDelete_statement(ctx *parser.Delete_statementContext) {
	if ctx.WHERE() == nil {
		l.adviceList = append(l.adviceList, advisor.Advice{
			Status:  l.level,
			Code:    advisor.StatementNoWhere,
			Title:   l.title,
			Content: "WHERE clause is required for DELETE statement.",
			Line:    ctx.GetStart().GetLine(),
		})
	}
}

// EnterQuery_statement is called when production query_statement is entered.
func (l *whereRequireChecker) EnterQuery_statement(ctx *parser.Query_statementContext) {
	if ctx.Select_statement() == nil {
		return
	}
	if ctx.Select_statement().Select_optional_clauses() == nil || ctx.Select_statement().Select_optional_clauses().Where_clause() == nil {
		l.adviceList = append(l.adviceList, advisor.Advice{
			Status:  l.level,
			Code:    advisor.StatementNoWhere,
			Title:   l.title,
			Content: "WHERE clause is required for SELECT statement.",
			Line:    ctx.GetStart().GetLine(),
		})
	}
}
