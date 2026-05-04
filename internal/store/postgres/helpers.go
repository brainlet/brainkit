package postgres

import (
	"context"
	_ "embed"
)

//go:embed schema.sql
var postgresSchemaSQL string

func postgresSchema() (string, error) {
	return postgresSchemaSQL, nil
}

// ctx returns a background context for store operations.
// Store operations are internal and don't need caller-controlled cancellation.
func ctx() context.Context {
	return context.Background()
}
