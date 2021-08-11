package sq

import "database/sql"

type DBQualifiedName struct{ Schema, Name string }

type DBMetadataer interface {
	GetTables() (tables []DBQualifiedName, err error)
	GetColumns(table DBQualifiedName) (columns map[string]DBColumn, err error)
	GetIndices(table DBQualifiedName) (indices map[string]DBIndex, err error)
}

type DBColumn struct {
	TableSchema        string
	TableName          string
	ColumnName         string
	ColumnType         string
	NotNull            bool
	IsPrimaryKey       bool
	IsUnique           bool
	IsAutoincrement    bool
	ColumnDefault      sql.NullString
	ReferencesSchema   sql.NullString
	ReferencesTable    sql.NullString
	ReferencesColumn   sql.NullString
	ReferencesOnUpdate sql.NullString
	ReferencesOnDelete sql.NullString
}

type DBIndex struct {
	TableSchema string
	TableName   string
	IndexSchema string
	IndexName   string
	IndexType   string // BTREE | HASH | GIST | SPGIST | GIN | BRIN | FULLTEXT | SPATIAL
	IsUnique    bool
	IsPartial   bool
	Columns     []string
}
