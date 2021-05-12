package sq

import (
	"bytes"
	"testing"

	"github.com/bokwoon95/pagemanager/testutil"
)

func Test_CTE(t *testing.T) {
	type TT struct {
		q       Query
		name    string
		alias1  string
		alias2  string
		columns []string
	}

	assertCTE := func(t *testing.T, tt TT) {
		is := testutil.New(t)
		buf1, buf2 := &bytes.Buffer{}, &bytes.Buffer{}
		var args1, args2 []interface{}

		// cte1 "fields"
		cte1 := NewCTE(tt.q, tt.name, tt.alias1, tt.columns)
		is.Equal(tt.q, cte1.GetQuery())
		is.Equal(tt.name, cte1.GetName())
		is.Equal(tt.alias1, cte1.GetAlias())
		is.Equal(tt.columns, cte1.GetColumns())
		is.Equal(false, cte1.IsRecursive())
		// cte1.AppendSQL
		err := cte1.AppendSQL("", buf1, &args1, make(map[string]int))
		is.NoErr(err)
		is.Equal(tt.name, buf1.String())

		// cte2 "fields"
		cte2 := cte1.As(tt.alias2)
		is.Equal(tt.q, cte2.GetQuery())
		is.Equal(tt.name, cte2.GetName())
		is.Equal(tt.alias2, cte2.GetAlias())
		is.Equal(tt.columns, cte2.GetColumns())
		is.Equal(false, cte2.IsRecursive())
		// cte2.AppendSQL
		err = cte2.AppendSQL("", buf2, &args2, make(map[string]int))
		is.NoErr(err)
		is.Equal(tt.name, buf2.String())

		if len(tt.columns) == 0 {
			return
		}
		column := tt.columns[0]
		buf1.Reset()
		buf2.Reset()
		args1 = args1[:0]
		args2 = args2[:0]

		// cte1 column
		err = cte1[column].AppendSQLExclude("", buf1, &args1, make(map[string]int), nil)
		is.NoErr(err)
		prefix1 := tt.name
		if tt.alias1 != "" {
			prefix1 = tt.alias1
		}
		is.Equal(prefix1+"."+column, buf1.String())
		is.Equal(0, len(args1))
		// cte2 column
		err = cte2[column].AppendSQLExclude("", buf2, &args2, make(map[string]int), nil)
		is.NoErr(err)
		prefix2 := tt.name
		if tt.alias2 != "" {
			prefix2 = tt.alias2
		}
		is.Equal(prefix2+"."+column, buf2.String())
		is.Equal(0, len(args2))
	}
	t.Run("cte1 unaliased, cte2 aliased", func(t *testing.T) {
		tt := TT{
			q:       querylite{readQuery: `SELECT column_1 FROM tbl WHERE column_2 = column_3`},
			name:    "cte",
			alias1:  "",
			alias2:  "C2",
			columns: []string{"column_1"},
		}
		assertCTE(t, tt)
	})
	t.Run("cte1 aliased, cte2 aliased", func(t *testing.T) {
		tt := TT{
			q:       querylite{readQuery: `SELECT column_1 FROM tbl WHERE column_2 = column_3`},
			name:    "cte",
			alias1:  "C1",
			alias2:  "C2",
			columns: []string{"column_1"},
		}
		assertCTE(t, tt)
	})
}

func Test_CTEs(t *testing.T) {
	type TT struct {
		ctes       CTEs
		fromTable  Table
		joinTables []JoinTable
		wantQuery  string
		wantArgs   []interface{}
	}

	assertCTEs := func(t *testing.T, tt TT) {
		is := testutil.New(t)
		buf := &bytes.Buffer{}
		var args []interface{}
		_ = tt.ctes.AppendCTEs("", buf, &args, make(map[string]int), tt.fromTable, tt.joinTables)
		is.Equal(tt.wantQuery, buf.String())
		is.Equal(tt.wantArgs, args)
	}
	t.Run("empty", func(t *testing.T) {
		var tt TT
		tt.ctes = CTEs{}
		tt.wantQuery = ""
		assertCTEs(t, tt)
		is := testutil.New(t)
		cte := CTE{}
		is.Equal(nil, cte.GetQuery())
		is.Equal("", cte.GetName())
		is.Equal("", cte.GetAlias())
		is.Equal([]string(nil), cte.GetColumns())
	})
	t.Run("basic", func(t *testing.T) {
		var tt TT
		cte1 := NewCTE(
			querylite{fields: fieldliterals("column_1"), readQuery: "FROM tbl WHERE column_2 = column_3"}, "cte1", "", []string{"column_1"},
		)
		cte2 := NewCTE(
			querylite{fields: fieldliterals("column_4"), readQuery: "FROM tbl WHERE column_5 = column_6"}, "cte2", "", []string{"column_4"},
		)
		tt.ctes = CTEs{cte1, cte2}
		tt.wantQuery = "WITH cte1 (column_1) AS" +
			" (SELECT column_1 FROM tbl WHERE column_2 = column_3)," +
			" cte2 (column_4) AS" +
			" (SELECT column_4 FROM tbl WHERE column_5 = column_6)" +
			" "
		assertCTEs(t, tt)
	})
	t.Run("recursive", func(t *testing.T) {
		var tt TT
		cte1 := NewCTE(
			querylite{fields: fieldliterals("column_1"), readQuery: "FROM tbl WHERE column_2 = column_3"}, "cte1", "c1", []string{"column_1"},
		)
		cte2 := RecursiveCTE("tens", "n")
		cte2 = cte2.
			Initial(querylite{fields: fieldliterals("10")}).
			UnionAll(querylite{fields: fieldliterals("tens.n"), readQuery: "FROM tens WHERE tens.n + 10 <= 100"})
		cte3 := NewCTE(nil, "cte3", "", nil)
		cte4 := RecursiveCTE("tens_v2")
		cte4 = cte4.
			Initial(querylite{fields: Fields{Fieldf("10").As("n")}}).
			Union(querylite{fields: fieldliterals("tens_v2.n"), readQuery: "FROM tens_v2 WHERE tens_v2.n + 10 <= 100"})
		tt.ctes = CTEs{cte1, cte2, cte3.As("C3"), cte4}
		tt.joinTables = []JoinTable{Join(cte1)}
		tt.wantQuery = "WITH RECURSIVE cte1 (column_1) AS" +
			" (SELECT column_1 FROM tbl WHERE column_2 = column_3)," +
			" tens (n) AS" +
			" (SELECT 10 UNION ALL SELECT tens.n FROM tens WHERE tens.n + 10 <= 100)," +
			" cte3 AS (NULL)," +
			" tens_v2 AS" +
			" (SELECT 10 AS n UNION SELECT tens_v2.n FROM tens_v2 WHERE tens_v2.n + 10 <= 100)" +
			" "
		assertCTEs(t, tt)
	})
	t.Run("calling Initial/Union/UnionAll on a non-recursive CTE is a no-op", func(t *testing.T) {
		var tt TT
		cte1 := NewCTE(
			querylite{fields: fieldliterals("column_1"), readQuery: "FROM tbl WHERE column_2 = column_3"}, "cte1", "c1", []string{"column_1"},
		)
		cte1 = cte1.Initial(nil).Union(nil)
		cte1 = cte1.Initial(nil).UnionAll(nil)
		tt.ctes = CTEs{cte1}
		tt.wantQuery = "WITH cte1 (column_1) AS (SELECT column_1 FROM tbl WHERE column_2 = column_3) "
		assertCTEs(t, tt)
	})
}
