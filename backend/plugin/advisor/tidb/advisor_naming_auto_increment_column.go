package tidb

// Framework code is generated by the generator.

import (
	"fmt"
	"regexp"

	"github.com/pingcap/tidb/parser/ast"
	"github.com/pkg/errors"

	"github.com/ashutoshojha5/bytebase/backend/plugin/advisor"
	storepb "github.com/ashutoshojha5/bytebase/proto/generated-go/store"
)

var (
	_ advisor.Advisor = (*NamingAutoIncrementColumnAdvisor)(nil)
	_ ast.Visitor     = (*namingAutoIncrementColumnChecker)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, advisor.MySQLNamingAutoIncrementColumnConvention, &NamingAutoIncrementColumnAdvisor{})
}

// NamingAutoIncrementColumnAdvisor is the advisor checking for auto-increment naming convention.
type NamingAutoIncrementColumnAdvisor struct {
}

// Check checks for auto-increment naming convention.
func (*NamingAutoIncrementColumnAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	stmtList, ok := ctx.AST.([]ast.StmtNode)
	if !ok {
		return nil, errors.Errorf("failed to convert to StmtNode")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	format, maxLength, err := advisor.UnmarshalNamingRulePayloadAsRegexp(ctx.Rule.Payload)
	if err != nil {
		return nil, err
	}
	checker := &namingAutoIncrementColumnChecker{
		level:     level,
		title:     string(ctx.Rule.Type),
		format:    format,
		maxLength: maxLength,
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

type namingAutoIncrementColumnChecker struct {
	adviceList []advisor.Advice
	level      advisor.Status
	title      string
	text       string
	line       int
	format     *regexp.Regexp
	maxLength  int
}

// Enter implements the ast.Visitor interface.
func (checker *namingAutoIncrementColumnChecker) Enter(in ast.Node) (ast.Node, bool) {
	type columnData struct {
		name string
		line int
	}
	var columnList []columnData
	var tableName string
	switch node := in.(type) {
	// CREATE TABLE
	case *ast.CreateTableStmt:
		tableName = node.Table.Name.O
		for _, column := range node.Cols {
			if isAutoIncrement(column) {
				columnList = append(columnList, columnData{
					name: column.Name.Name.O,
					line: column.OriginTextPosition(),
				})
			}
		}
	// ALTER TABLE
	case *ast.AlterTableStmt:
		tableName = node.Table.Name.O
		for _, spec := range node.Specs {
			switch spec.Tp {
			// ADD COLUMNS
			case ast.AlterTableAddColumns:
				for _, column := range spec.NewColumns {
					if isAutoIncrement(column) {
						columnList = append(columnList, columnData{
							name: column.Name.Name.O,
							line: in.OriginTextPosition(),
						})
					}
				}
			// CHANGE COLUMN/MODIFY COLUMN
			case ast.AlterTableChangeColumn, ast.AlterTableModifyColumn:
				if isAutoIncrement(spec.NewColumns[0]) {
					columnList = append(columnList, columnData{
						name: spec.NewColumns[0].Name.Name.O,
						line: in.OriginTextPosition(),
					})
				}
			}
		}
	}

	for _, column := range columnList {
		if !checker.format.MatchString(column.name) {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.NamingAutoIncrementColumnConventionMismatch,
				Title:   checker.title,
				Content: fmt.Sprintf("`%s`.`%s` mismatches auto_increment column naming convention, naming format should be %q", tableName, column.name, checker.format),
				Line:    column.line,
			})
		}
		if checker.maxLength > 0 && len(column.name) > checker.maxLength {
			checker.adviceList = append(checker.adviceList, advisor.Advice{
				Status:  checker.level,
				Code:    advisor.NamingAutoIncrementColumnConventionMismatch,
				Title:   checker.title,
				Content: fmt.Sprintf("`%s`.`%s` mismatches auto_increment column naming convention, its length should be within %d characters", tableName, column.name, checker.maxLength),
				Line:    column.line,
			})
		}
	}

	return in, false
}

// Leave implements the ast.Visitor interface.
func (*namingAutoIncrementColumnChecker) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}

func isAutoIncrement(column *ast.ColumnDef) bool {
	for _, option := range column.Options {
		if option.Tp == ast.ColumnOptionAutoIncrement {
			return true
		}
	}
	return false
}
