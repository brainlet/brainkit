package store

import (
	"context"
	_ "embed"
)

//go:embed sql/sqlite/schema.sql
var sqliteSchemaSQL string

//go:embed sql/postgres/schema.sql
var postgresSchemaSQL string

func sqliteSchema() (string, error) {
	return sqliteSchemaSQL, nil
}

func postgresSchema() (string, error) {
	return postgresSchemaSQL, nil
}

// ctx returns a background context for store operations.
// Store operations are internal and don't need caller-controlled cancellation.
func ctx() context.Context {
	return context.Background()
}
