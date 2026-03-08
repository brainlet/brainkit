// Ported from: packages/provider-utils/src/schema.ts
package providerutils

// ValidationResult represents the result of schema validation.
type ValidationResult[T any] struct {
	Success bool
	Value   T
	Error   error
}

// Schema represents a validated JSON schema.
type Schema[T any] struct {
	// Validate optionally validates that the structure of a value matches this schema.
	Validate func(value interface{}) (*ValidationResult[T], error)
	// JSONSchema is the JSON Schema for the schema.
	JSONSchema interface{}
}

// FlexibleSchema represents any schema type. In Go, we use a simplified version
// since we don't have Zod or StandardSchema. This accepts either a Schema directly
// or a function that returns one (LazySchema).
type FlexibleSchema[T any] struct {
	schema     *Schema[T]
	lazySchema func() *Schema[T]
}

// NewFlexibleSchema creates a FlexibleSchema from a Schema.
func NewFlexibleSchema[T any](s *Schema[T]) FlexibleSchema[T] {
	return FlexibleSchema[T]{schema: s}
}

// NewLazyFlexibleSchema creates a FlexibleSchema from a lazy initializer.
func NewLazyFlexibleSchema[T any](fn func() *Schema[T]) FlexibleSchema[T] {
	return FlexibleSchema[T]{lazySchema: fn}
}

// AsSchema resolves the FlexibleSchema to a concrete Schema.
func (fs FlexibleSchema[T]) AsSchema() *Schema[T] {
	if fs.schema != nil {
		return fs.schema
	}
	if fs.lazySchema != nil {
		return fs.lazySchema()
	}
	return &Schema[T]{
		JSONSchema: map[string]interface{}{
			"properties":           map[string]interface{}{},
			"additionalProperties": false,
		},
	}
}

// JSONSchema creates a Schema using a JSON Schema definition.
func JSONSchemaCreate[T any](jsonSchema interface{}, validate func(value interface{}) (*ValidationResult[T], error)) *Schema[T] {
	return &Schema[T]{
		JSONSchema: jsonSchema,
		Validate:   validate,
	}
}

// LazySchema creates a deferred schema that is only constructed when first accessed.
func LazySchema[T any](createSchema func() *Schema[T]) func() *Schema[T] {
	var cached *Schema[T]
	return func() *Schema[T] {
		if cached == nil {
			cached = createSchema()
		}
		return cached
	}
}
