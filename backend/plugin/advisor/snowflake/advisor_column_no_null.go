// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/snowsql-parser"
	"github.com/pkg/errors"

	snowsqlparser "github.com/ashutoshojha5/bytebase/backend/plugin/parser/snowflake"

	"github.com/ashutoshojha5/bytebase/backend/plugin/advisor"
	storepb "github.com/ashutoshojha5/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*ColumnNoNullAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, advisor.SnowflakeColumnNoNull, &ColumnNoNullAdvisor{})
}

// ColumnNoNullAdvisor is the advisor checking for column no NULL value.
type ColumnNoNullAdvisor struct {
}

// Check checks for column no NULL value.
func (*ColumnNoNullAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	tree, ok := ctx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	listener := &columnNoNullChecker{
		level:                    level,
		title:                    string(ctx.Rule.Type),
		currentOriginalTableName: "",
		columnNullable:           make(map[string]int),
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.generateAdvice()
}

// columnNoNullChecker is the listener for column no NULL value.
type columnNoNullChecker struct {
	*parser.BaseSnowflakeParserListener

	level advisor.Status
	title string

	adviceList []advisor.Advice

	// currentOriginalTableName is the original table name of the current table.
	currentOriginalTableName string
	// columnNullable is a map from normalized column name to the line number causing the column to be nullable.
	columnNullable map[string]int
}

// generateAdvice returns the advices generated by the listener, the advices must not be empty.
func (l *columnNoNullChecker) generateAdvice() ([]advisor.Advice, error) {
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

// EnterCreate_table is called when production create_table is entered.
func (l *columnNoNullChecker) EnterCreate_table(ctx *parser.Create_tableContext) {
	l.currentOriginalTableName = ctx.Object_name().GetText()
}

// ExitCreate_table is called when production create_table is exited.
func (l *columnNoNullChecker) ExitCreate_table(*parser.Create_tableContext) {
	for normalizedColumnName, columnNullableLine := range l.columnNullable {
		l.adviceList = append(l.adviceList, advisor.Advice{
			Status:  l.level,
			Code:    advisor.ColumnCannotNull,
			Title:   l.title,
			Content: fmt.Sprintf("Column %s is nullable, which is not allowed.", normalizedColumnName),
			Line:    columnNullableLine,
		})
	}
	l.currentOriginalTableName = ""
	l.columnNullable = make(map[string]int)
}

// EnterFull_col_decl is called when production full_col_decl is entered.
func (l *columnNoNullChecker) EnterFull_col_decl(ctx *parser.Full_col_declContext) {
	if l.currentOriginalTableName == "" {
		return
	}
	normalizedOriginalColumnID := snowsqlparser.NormalizeSnowSQLObjectNamePart(ctx.Col_decl().Column_name().Id_())
	l.columnNullable[normalizedOriginalColumnID] = ctx.GetStart().GetLine()
	for _, nullNotNull := range ctx.AllNull_not_null() {
		if nullNotNull.NOT() != nil {
			delete(l.columnNullable, normalizedOriginalColumnID)
			break
		}
	}
	for _, constraint := range ctx.AllInline_constraint() {
		if constraint.PRIMARY() != nil {
			delete(l.columnNullable, normalizedOriginalColumnID)
			break
		}
	}
}

// EnterOut_of_line_constraint is called when production out_of_line_constraint is entered.
func (l *columnNoNullChecker) EnterOut_of_line_constraint(ctx *parser.Out_of_line_constraintContext) {
	if l.currentOriginalTableName == "" {
		return
	}
	if ctx.PRIMARY() != nil {
		for _, columnListInParentheses := range ctx.AllColumn_list_in_parentheses() {
			for _, column := range columnListInParentheses.Column_list().AllColumn_name() {
				normalizedOriginalColumnID := snowsqlparser.NormalizeSnowSQLObjectNamePart(column.Id_())
				delete(l.columnNullable, normalizedOriginalColumnID)
			}
		}
	}
}

// EnterAlter_table is called when production alter_table is entered.
func (l *columnNoNullChecker) EnterAlter_table(ctx *parser.Alter_tableContext) {
	l.currentOriginalTableName = ctx.Object_name(0).GetText()
}

// ExitAlter_table is called when production alter_table is exited.
func (l *columnNoNullChecker) ExitAlter_table(*parser.Alter_tableContext) {
	for normalizedColumnName, columnNullableLine := range l.columnNullable {
		l.adviceList = append(l.adviceList, advisor.Advice{
			Status:  l.level,
			Code:    advisor.ColumnCannotNull,
			Title:   l.title,
			Content: fmt.Sprintf("Column %s is nullable, which is not allowed.", normalizedColumnName),
			Line:    columnNullableLine,
		})
	}
	l.currentOriginalTableName = ""
	l.columnNullable = make(map[string]int)
}

// EnterTable_column_action is called when production alter_table_add_constraint is entered.
func (l *columnNoNullChecker) EnterTable_column_action(ctx *parser.Table_column_actionContext) {
	if l.currentOriginalTableName == "" {
		return
	}
	if ctx.ADD() != nil {
		normalizedNewColumnName := snowsqlparser.NormalizeSnowSQLObjectNamePart(ctx.Column_name(0).Id_())
		inlineConstraintHasPK := ctx.Inline_constraint() != nil && ctx.Inline_constraint().PRIMARY() != nil
		inlineConstraintHasNotNull := ctx.Inline_constraint() != nil && (ctx.Inline_constraint().Null_not_null() != nil && ctx.Inline_constraint().Null_not_null().NOT() != nil)
		hasNotNull := ctx.Null_not_null() != nil && ctx.Null_not_null().NOT() != nil

		if !(inlineConstraintHasPK || inlineConstraintHasNotNull || hasNotNull) {
			l.columnNullable[normalizedNewColumnName] = ctx.GetStart().GetLine()
		}
		return
	}
	if ctx.Alter_modify() != nil {
		normalizedOriginalColumnName := snowsqlparser.NormalizeSnowSQLObjectNamePart(ctx.Column_name(0).Id_())
		if len(ctx.AllDROP()) == 1 && ctx.NOT() != nil && ctx.NULL_() != nil {
			l.columnNullable[normalizedOriginalColumnName] = ctx.GetStart().GetLine()
		}
		return
	}
}

// EnterAlter_table_alter_column is called when production alter_table_alter_column is entered.
func (l *columnNoNullChecker) EnterAlter_table_alter_column(ctx *parser.Alter_table_alter_columnContext) {
	l.currentOriginalTableName = ctx.Object_name().GetText()
}

// ExitAlter_table_alter_column is called when production alter_table_alter_column is exited.
func (l *columnNoNullChecker) ExitAlter_table_alter_column(*parser.Alter_table_alter_columnContext) {
	for normalizedColumnName, columnNullableLine := range l.columnNullable {
		l.adviceList = append(l.adviceList, advisor.Advice{
			Status:  l.level,
			Code:    advisor.ColumnCannotNull,
			Title:   l.title,
			Content: fmt.Sprintf("After dropping NOT NULL of column %s, it will be nullable, which is not allowed.", normalizedColumnName),
			Line:    columnNullableLine,
		})
	}
	l.currentOriginalTableName = ""
	l.columnNullable = make(map[string]int)
}

// EnterAlter_column_decl is called when production alter_column_decl is entered.
func (l *columnNoNullChecker) EnterAlter_column_decl(ctx *parser.Alter_column_declContext) {
	if l.currentOriginalTableName == "" {
		return
	}
	normalizedNewColumnName := snowsqlparser.NormalizeSnowSQLObjectNamePart(ctx.Column_name().Id_())
	if ctx.Alter_column_opts().DROP() != nil && ctx.Alter_column_opts().NOT() != nil && ctx.Alter_column_opts().NULL_() != nil {
		l.columnNullable[normalizedNewColumnName] = ctx.GetStart().GetLine()
	}
}
