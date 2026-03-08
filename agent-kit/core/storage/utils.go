// Ported from: packages/core/src/storage/utils.ts
package storage

import (
	"encoding/json"
	"reflect"
	"regexp"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// StoreName – canonical store/provider names
// ---------------------------------------------------------------------------

// StoreName represents a canonical storage provider name.
// In TypeScript this is a branded union with (string & {}); in Go we use a
// plain string alias so callers can pass arbitrary values while the constants
// provide IDE autocomplete.
type StoreName = string

const (
	StoreNamePG           StoreName = "PG"
	StoreNameMSSQL        StoreName = "MSSQL"
	StoreNameLIBSQL       StoreName = "LIBSQL"
	StoreNameMONGODB      StoreName = "MONGODB"
	StoreNameCLICKHOUSE   StoreName = "CLICKHOUSE"
	StoreNameCLOUDFLARE   StoreName = "CLOUDFLARE"
	StoreNameCLOUDFLARED1 StoreName = "CLOUDFLARE_D1"
	StoreNameDYNAMODB     StoreName = "DYNAMODB"
	StoreNameLANCE        StoreName = "LANCE"
	StoreNameUPSTASH      StoreName = "UPSTASH"
	StoreNameASTRA        StoreName = "ASTRA"
	StoreNameCHROMA       StoreName = "CHROMA"
	StoreNameCOUCHBASE    StoreName = "COUCHBASE"
	StoreNameOPENSEARCH   StoreName = "OPENSEARCH"
	StoreNamePINECONE     StoreName = "PINECONE"
	StoreNameQDRANT       StoreName = "QDRANT"
	StoreNameS3           StoreName = "S3"
	StoreNameTURBOPUFFER  StoreName = "TURBOPUFFER"
	StoreNameVECTORIZE    StoreName = "VECTORIZE"
)

// ---------------------------------------------------------------------------
// ScoreRowData – stub type for evals dependency
// ---------------------------------------------------------------------------

// ScoreRowData is the canonical score record type from the evals package.
// TransformScoreRow returns this type after transforming raw storage rows.
// Note: The transform function returns map[string]any which callers
// must convert to evals.ScoreRowData via JSON marshal/unmarshal.
type ScoreRowData = map[string]any

// ---------------------------------------------------------------------------
// SafelyParseJSON
// ---------------------------------------------------------------------------

// SafelyParseJSON attempts to parse input into a Go value.
//   - If input is already a non-nil map/slice/struct, it is returned as-is.
//   - If input is nil, an empty map is returned.
//   - If input is a string, it is JSON-decoded; on failure the raw string is
//     returned.
//   - For anything else (number, bool, etc.) an empty map is returned.
func SafelyParseJSON(input any) any {
	if input == nil {
		return map[string]any{}
	}

	switch v := input.(type) {
	case map[string]any:
		return v
	case []any:
		return v
	case string:
		var parsed any
		if err := json.Unmarshal([]byte(v), &parsed); err != nil {
			return v // return raw string on parse failure
		}
		return parsed
	default:
		rv := reflect.ValueOf(input)
		if rv.Kind() == reflect.Map || rv.Kind() == reflect.Slice || rv.Kind() == reflect.Struct {
			return input
		}
		return map[string]any{}
	}
}

// ---------------------------------------------------------------------------
// TransformRow
// ---------------------------------------------------------------------------

// TransformRowOptions configures how storage rows are transformed.
type TransformRowOptions struct {
	// PreferredTimestampFields maps target field names to alternative source
	// field names. For example {"createdAt": "createdAtZ"} means: use the
	// value from "createdAtZ" if available, otherwise fall back to "createdAt".
	PreferredTimestampFields map[string]string

	// ConvertTimestamps controls whether timestamp strings are converted to
	// time.Time values. Default is false for backwards compatibility.
	ConvertTimestamps bool

	// NullValuePattern is a string value that should be treated as null
	// (e.g., "_null_" for ClickHouse).
	NullValuePattern string

	// FieldMappings maps target field names to source field names.
	// For example {"entity": "entityData"} for DynamoDB.
	FieldMappings map[string]string
}

// TransformRow applies schema-driven transformations to a raw storage row.
//   - 'jsonb' fields are parsed from JSON strings via SafelyParseJSON.
//   - 'timestamp' fields are optionally parsed into time.Time when
//     ConvertTimestamps is true.
//
// The tableName is looked up in TableSchemas to determine column types.
func TransformRow(row map[string]any, tableName TableName, opts *TransformRowOptions) map[string]any {
	if opts == nil {
		opts = &TransformRowOptions{}
	}

	preferredTimestampFields := opts.PreferredTimestampFields
	if preferredTimestampFields == nil {
		preferredTimestampFields = map[string]string{}
	}
	fieldMappings := opts.FieldMappings
	if fieldMappings == nil {
		fieldMappings = map[string]string{}
	}

	tableSchema, ok := TableSchemas[tableName]
	if !ok {
		return row
	}

	result := make(map[string]any, len(tableSchema))

	for key, columnSchema := range tableSchema {
		// Handle field mappings (e.g. entityData -> entity for DynamoDB)
		sourceKey := key
		if mapped, exists := fieldMappings[key]; exists {
			sourceKey = mapped
		}

		value, exists := row[sourceKey]
		if !exists {
			continue
		}

		// Handle preferred timestamp sources
		if altKey, hasAlt := preferredTimestampFields[key]; hasAlt {
			if altVal, altExists := row[altKey]; altExists {
				value = altVal
			}
		}

		// Skip nil values
		if value == nil {
			continue
		}

		// Skip null-pattern values (e.g. ClickHouse's "_null_")
		if opts.NullValuePattern != "" {
			if strVal, ok := value.(string); ok && strVal == opts.NullValuePattern {
				continue
			}
		}

		// Transform based on column type
		switch columnSchema.Type {
		case ColTypeJSONB:
			if s, ok := value.(string); ok {
				result[key] = SafelyParseJSON(s)
			} else {
				result[key] = value
			}
		case ColTypeTimestamp:
			if opts.ConvertTimestamps {
				if s, ok := value.(string); ok {
					if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
						result[key] = t
					} else if t, err := time.Parse(time.RFC3339, s); err == nil {
						result[key] = t
					} else {
						result[key] = value
					}
				} else {
					result[key] = value
				}
			} else {
				result[key] = value
			}
		default:
			result[key] = value
		}
	}

	return result
}

// TransformScoreRow is a convenience wrapper around TransformRow for the
// scores table (TableScorers).
func TransformScoreRow(row map[string]any, opts *TransformRowOptions) ScoreRowData {
	return TransformRow(row, TableScorers, opts)
}

// ---------------------------------------------------------------------------
// toUpperSnakeCase
// ---------------------------------------------------------------------------

var (
	camelBoundaryRe       = regexp.MustCompile(`([a-z])([A-Z])`)
	upperSequenceBoundary = regexp.MustCompile(`([A-Z])([A-Z][a-z])`)
	nonAlphanumericRe     = regexp.MustCompile(`[^A-Z0-9]+`)
)

// toUpperSnakeCase converts a string to UPPER_SNAKE_CASE, preserving word
// boundaries from camelCase, PascalCase, kebab-case, etc.
func toUpperSnakeCase(s string) string {
	// Insert underscore before uppercase letters following lowercase letters
	result := camelBoundaryRe.ReplaceAllString(s, "${1}_${2}")
	// Insert underscore before uppercase letters followed by lowercase
	result = upperSequenceBoundary.ReplaceAllString(result, "${1}_${2}")
	// Convert to uppercase
	result = strings.ToUpper(result)
	// Replace non-alphanumeric with underscore
	result = nonAlphanumericRe.ReplaceAllString(result, "_")
	// Trim leading/trailing underscores
	result = strings.TrimFunc(result, func(r rune) bool {
		return r == '_'
	})
	return result
}

// ---------------------------------------------------------------------------
// Error ID helpers
// ---------------------------------------------------------------------------

// StoreErrorType distinguishes storage from vector operations in error IDs.
type StoreErrorType string

const (
	StoreErrorTypeStorage StoreErrorType = "storage"
	StoreErrorTypeVector  StoreErrorType = "vector"
)

// CreateStoreErrorID generates a standardised error ID for storage/vector
// operations.
//
// Formats:
//   - Storage: MASTRA_STORAGE_{STORE}_{OPERATION}_{STATUS}
//   - Vector:  MASTRA_VECTOR_{STORE}_{OPERATION}_{STATUS}
//
// Inputs are auto-normalised to UPPER_SNAKE_CASE.
func CreateStoreErrorID(errType StoreErrorType, store StoreName, operation, status string) string {
	normalizedStore := toUpperSnakeCase(store)
	normalizedOperation := toUpperSnakeCase(operation)
	normalizedStatus := toUpperSnakeCase(status)
	typePrefix := "STORAGE"
	if errType == StoreErrorTypeVector {
		typePrefix = "VECTOR"
	}
	return "MASTRA_" + typePrefix + "_" + normalizedStore + "_" + normalizedOperation + "_" + normalizedStatus
}

// CreateStorageErrorID is a convenience wrapper for storage-type error IDs.
func CreateStorageErrorID(store StoreName, operation, status string) string {
	return CreateStoreErrorID(StoreErrorTypeStorage, store, operation, status)
}

// CreateVectorErrorID is a convenience wrapper for vector-type error IDs.
func CreateVectorErrorID(store StoreName, operation, status string) string {
	return CreateStoreErrorID(StoreErrorTypeVector, store, operation, status)
}

// ---------------------------------------------------------------------------
// SQL type helpers
// ---------------------------------------------------------------------------

// GetSQLType maps a StorageColumnType to the corresponding SQL type keyword.
func GetSQLType(colType StorageColumnType) string {
	switch colType {
	case ColTypeText:
		return "TEXT"
	case ColTypeTimestamp:
		return "TIMESTAMP"
	case ColTypeFloat:
		return "FLOAT"
	case ColTypeInteger:
		return "INTEGER"
	case ColTypeBigint:
		return "BIGINT"
	case ColTypeJSONB:
		return "JSONB"
	case ColTypeBoolean:
		return "BOOLEAN"
	default:
		return "TEXT"
	}
}

// GetDefaultValue returns a SQL DEFAULT clause for the given column type.
func GetDefaultValue(colType StorageColumnType) string {
	switch colType {
	case ColTypeText, ColTypeUUID:
		return "DEFAULT ''"
	case ColTypeTimestamp:
		return "DEFAULT '1970-01-01 00:00:00'"
	case ColTypeInteger, ColTypeBigint, ColTypeFloat:
		return "DEFAULT 0"
	case ColTypeJSONB:
		return "DEFAULT '{}'"
	case ColTypeBoolean:
		return "DEFAULT FALSE"
	default:
		return "DEFAULT ''"
	}
}

// ---------------------------------------------------------------------------
// Date helpers
// ---------------------------------------------------------------------------

// EnsureTime converts a string or time.Time value to *time.Time.
// Returns nil if the input is nil or cannot be parsed.
func EnsureTime(v any) *time.Time {
	if v == nil {
		return nil
	}
	switch d := v.(type) {
	case time.Time:
		return &d
	case *time.Time:
		return d
	case string:
		if d == "" {
			return nil
		}
		if t, err := time.Parse(time.RFC3339Nano, d); err == nil {
			return &t
		}
		if t, err := time.Parse(time.RFC3339, d); err == nil {
			return &t
		}
		return nil
	default:
		return nil
	}
}

// SerializeDate converts a time value to an ISO-8601 string, or returns nil.
func SerializeDate(v any) *string {
	t := EnsureTime(v)
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339Nano)
	return &s
}

// ---------------------------------------------------------------------------
// DateRangeFilter – in-memory date filtering
// ---------------------------------------------------------------------------

// DateRangeFilter describes inclusive/exclusive date bounds for filtering.
type DateRangeFilter struct {
	Start          *time.Time
	End            *time.Time
	StartExclusive bool
	EndExclusive   bool
}

// FilterByDateRange filters items by a date range. getCreatedAt extracts
// the createdAt timestamp from each item. If dateRange is nil, items are
// returned unchanged.
func FilterByDateRange[T any](items []T, getCreatedAt func(T) time.Time, dateRange *DateRangeFilter) []T {
	if dateRange == nil {
		return items
	}

	result := items

	if dateRange.Start != nil {
		startTime := dateRange.Start.UnixNano()
		filtered := make([]T, 0, len(result))
		for _, item := range result {
			itemTime := getCreatedAt(item).UnixNano()
			if dateRange.StartExclusive {
				if itemTime > startTime {
					filtered = append(filtered, item)
				}
			} else {
				if itemTime >= startTime {
					filtered = append(filtered, item)
				}
			}
		}
		result = filtered
	}

	if dateRange.End != nil {
		endTime := dateRange.End.UnixNano()
		filtered := make([]T, 0, len(result))
		for _, item := range result {
			itemTime := getCreatedAt(item).UnixNano()
			if dateRange.EndExclusive {
				if itemTime < endTime {
					filtered = append(filtered, item)
				}
			} else {
				if itemTime <= endTime {
					filtered = append(filtered, item)
				}
			}
		}
		result = filtered
	}

	return result
}

// ---------------------------------------------------------------------------
// JSONValueEquals – deep equality for JSON-compatible values
// ---------------------------------------------------------------------------

// JSONValueEquals performs a deep equality comparison on JSON-compatible Go
// values (nil, bool, float64, string, []any, map[string]any, time.Time).
func JSONValueEquals(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Handle time.Time
	if ta, ok := a.(time.Time); ok {
		if tb, ok := b.(time.Time); ok {
			return ta.Equal(tb)
		}
		return false
	}
	if _, ok := b.(time.Time); ok {
		return false // b is Time but a was not
	}

	va := reflect.ValueOf(a)
	vb := reflect.ValueOf(b)

	if va.Type() != vb.Type() {
		return false
	}

	switch va.Kind() {
	case reflect.Slice:
		if va.Len() != vb.Len() {
			return false
		}
		for i := 0; i < va.Len(); i++ {
			if !JSONValueEquals(va.Index(i).Interface(), vb.Index(i).Interface()) {
				return false
			}
		}
		return true
	case reflect.Map:
		if va.Len() != vb.Len() {
			return false
		}
		for _, key := range va.MapKeys() {
			bVal := vb.MapIndex(key)
			if !bVal.IsValid() {
				return false
			}
			if !JSONValueEquals(va.MapIndex(key).Interface(), bVal.Interface()) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(a, b)
	}
}

