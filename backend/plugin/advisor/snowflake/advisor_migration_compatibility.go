// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/snowsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/db"
	bbparser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
)

var (
	_ advisor.Advisor = (*MigrationCompatibilityAdvisor)(nil)
)

func init() {
	advisor.Register(db.Snowflake, advisor.SnowflakeMigrationCompatibility, &MigrationCompatibilityAdvisor{})
}

// MigrationCompatibilityAdvisor is the advisor checking for migration compatibility.
type MigrationCompatibilityAdvisor struct {
}

// Check checks for migration compatibility.
func (*MigrationCompatibilityAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	tree, ok := ctx.AST.(antlr.Tree)
	if !ok {
		return nil, errors.Errorf("failed to convert to Tree")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}

	listener := &migrationCompatibilityChecker{
		level:                              level,
		title:                              string(ctx.Rule.Type),
		currentDatabase:                    ctx.CurrentDatabase,
		normalizedNewCreateTableNameMap:    make(map[string]bool),
		normalizedNewCreateSchemaNameMap:   make(map[string]bool),
		normalizedNewCreateDatabaseNameMap: make(map[string]bool),
	}

	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	return listener.generateAdvice()
}

// migrationCompatibilityChecker is the listener for migration compatibility.
type migrationCompatibilityChecker struct {
	*parser.BaseSnowflakeParserListener

	level advisor.Status
	title string

	// normalizedLastCreateTableNameMap contain the last created table name in normalized format, e.g. "SNOWFLAKE.PUBLIC.TABLE", If there are IF NOT EXISTS, the value will be false.
	normalizedNewCreateTableNameMap map[string]bool
	// normalizedLastCreateSchemaNameMap contain the last created schema name in normalized format, e.g. "SNOWFLAKE.PUBLIC", If there are IF NOT EXISTS, the value will be false.
	normalizedNewCreateSchemaNameMap map[string]bool
	// normalizedNewCreateDatabaseNameMap contain the new created database name in normalized format, e.g. "SNOWFLAKE", If there are IF NOT EXISTS, the value will be false.
	normalizedNewCreateDatabaseNameMap map[string]bool

	// currentDatabase is the current database name.
	currentDatabase string

	adviceList []advisor.Advice
}

// generateAdvice returns the advices generated by the listener, the advices must not be empty.
func (l *migrationCompatibilityChecker) generateAdvice() ([]advisor.Advice, error) {
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

// EnterCreate_Table is called when production create_Table is entered.
func (l *migrationCompatibilityChecker) EnterCreate_table(ctx *parser.Create_tableContext) {
	normalizedFullTableName := bbparser.NormalizeObjectName(ctx.Object_name(), l.currentDatabase, "PUBLIC")
	l.normalizedNewCreateTableNameMap[normalizedFullTableName] = ctx.If_not_exists() == nil
}

// EnterCreate_table_as_select is called when production create_table_as_select is entered.
func (l *migrationCompatibilityChecker) EnterCreate_table_as_select(ctx *parser.Create_table_as_selectContext) {
	normalizedFullTableName := bbparser.NormalizeObjectName(ctx.Object_name(), l.currentDatabase, "PUBLIC")
	l.normalizedNewCreateTableNameMap[normalizedFullTableName] = ctx.If_not_exists() == nil
}

// EnterCreate_schema is called when production create_schema is entered.
func (l *migrationCompatibilityChecker) EnterCreate_schema(ctx *parser.Create_schemaContext) {
	normalizedFullSchemaName := bbparser.NormalizeSchemaName(ctx.Schema_name(), l.currentDatabase)
	l.normalizedNewCreateSchemaNameMap[normalizedFullSchemaName] = ctx.If_not_exists() == nil
}

// EnterCreate_database is called when production create_database is entered.
func (l *migrationCompatibilityChecker) EnterCreate_database(ctx *parser.Create_databaseContext) {
	normalizedFullDatabaseName := bbparser.NormalizeObjectNamePart(ctx.Id_())
	l.normalizedNewCreateDatabaseNameMap[normalizedFullDatabaseName] = ctx.If_not_exists() == nil
}

// EnterDrop_table is called when production drop_table is entered.
func (l *migrationCompatibilityChecker) EnterDrop_table(ctx *parser.Drop_tableContext) {
	normalizedFullDropTableName := bbparser.NormalizeObjectName(ctx.Object_name(), l.currentDatabase, "PUBLIC")
	mustNewCreate, ok := l.normalizedNewCreateTableNameMap[normalizedFullDropTableName]
	if ok && mustNewCreate {
		return
	}
	level := l.level
	if ok && !mustNewCreate {
		level = advisor.Warn
	}
	l.adviceList = append(l.adviceList, advisor.Advice{
		Status:  level,
		Code:    advisor.CompatibilityDropTable,
		Title:   l.title,
		Content: fmt.Sprintf("Drop table %q may cause incompatibility with the existing data and code", normalizedFullDropTableName),
		Line:    ctx.GetStart().GetLine(),
	})
}

// EnterDrop_schema is called when production drop_schema is entered.
func (l *migrationCompatibilityChecker) EnterDrop_schema(ctx *parser.Drop_schemaContext) {
	normalizedFullDropSchemaName := bbparser.NormalizeSchemaName(ctx.Schema_name(), l.currentDatabase)
	mustNewCreate, ok := l.normalizedNewCreateSchemaNameMap[normalizedFullDropSchemaName]
	if ok && mustNewCreate {
		return
	}
	level := l.level
	if ok && !mustNewCreate {
		level = advisor.Warn
	}
	l.adviceList = append(l.adviceList, advisor.Advice{
		Status:  level,
		Code:    advisor.CompatibilityDropSchema,
		Title:   l.title,
		Content: fmt.Sprintf("Drop schema %q may cause incompatibility with the existing data and code", normalizedFullDropSchemaName),
		Line:    ctx.GetStart().GetLine(),
	})
}

// EnterDrop_database is called when production drop_database is entered.
func (l *migrationCompatibilityChecker) EnterDrop_database(ctx *parser.Drop_databaseContext) {
	normalizedFullDropDatabaseName := bbparser.NormalizeObjectNamePart(ctx.Id_())
	mustNewCreate, ok := l.normalizedNewCreateDatabaseNameMap[normalizedFullDropDatabaseName]
	if ok && mustNewCreate {
		return
	}
	level := l.level
	if ok && !mustNewCreate {
		level = advisor.Warn
	}
	l.adviceList = append(l.adviceList, advisor.Advice{
		Status:  level,
		Code:    advisor.CompatibilityDropDatabase,
		Title:   l.title,
		Content: fmt.Sprintf("Drop database %q may cause incompatibility with the existing data and code", normalizedFullDropDatabaseName),
		Line:    ctx.GetStart().GetLine(),
	})
}

// EnterAlter_table is called when production alter_table is entered.
func (l *migrationCompatibilityChecker) EnterAlter_table(ctx *parser.Alter_tableContext) {
	tableColumnAction := ctx.Table_column_action()
	if tableColumnAction == nil {
		return
	}
	dropColumn := tableColumnAction.DROP(0)
	if dropColumn == nil {
		return
	}
	allColumnName := tableColumnAction.Column_list().AllColumn_name()
	normalizedAllColumnNames := make([]string, 0, len(allColumnName))
	for _, columnName := range allColumnName {
		normalizedAllColumnNames = append(normalizedAllColumnNames, fmt.Sprintf("%q", bbparser.NormalizeObjectNamePart(columnName.Id_())))
	}
	l.adviceList = append(l.adviceList, advisor.Advice{
		Status:  l.level,
		Code:    advisor.CompatibilityDropColumn,
		Title:   l.title,
		Content: fmt.Sprintf("Drop column %s may cause incompatibility with the existing data and code", strings.Join(normalizedAllColumnNames, ",")),
		Line:    ctx.GetStart().GetLine(),
	})
}