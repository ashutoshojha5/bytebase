package tsql

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ashutoshojha5/bytebase/backend/plugin/parser/base"
)

func TestExtractMSSQLNormalizedResourceListFromSelectStatement(t *testing.T) {
	tests := []struct {
		statement string
		want      []base.SchemaResource
	}{
		{
			statement: `SELECT * FROM t1;SELECT * FROM t2;`,
			want: []base.SchemaResource{
				{
					Database: "MyDB",
					Schema:   "dbo",
					Table:    "t1",
				},
				{
					Database: "MyDB",
					Schema:   "dbo",
					Table:    "t2",
				},
			},
		},
		{
			statement: `SELECT * FROM myTable;SELECT * FROM MyTable;`,
			want: []base.SchemaResource{
				{
					Database: "MyDB",
					Schema:   "dbo",
					Table:    "mytable",
				},
			},
		},
		{
			statement: `SELECT * FROM TableOne JOIN TableTwo ON TableOne.ID = TableTwo.ID;`,
			want: []base.SchemaResource{
				{
					Database: "MyDB",
					Schema:   "dbo",
					Table:    "tableone",
				},
				{
					Database: "MyDB",
					Schema:   "dbo",
					Table:    "tabletwo",
				},
			},
		},
		{
			statement: `SELECT * FROM DatabaseA.dbo.[TableOne] JOIN DatabaseB.dbo.TableTwo ON DatabaseA.dbo.TableOne.ID = DatabaseB.dbo.TableTwo.ID;`,
			want: []base.SchemaResource{
				{
					Database: "databasea",
					Schema:   "dbo",
					Table:    "tableone",
				},
				{
					Database: "databaseb",
					Schema:   "dbo",
					Table:    "tabletwo",
				},
			},
		},
		{
			statement: `SELECT (SELECT MAX(b) FROM t1 WHERE c > 0);`,
			want: []base.SchemaResource{
				{
					Database: "MyDB",
					Schema:   "dbo",
					Table:    "t1",
				},
			},
		},
	}

	for _, test := range tests {
		res, err := ExtractResourceList("MyDB", "dbo", test.statement)
		require.NoError(t, err)
		require.Equal(t, test.want, res, "for statement: %v", test.statement)
	}
}
