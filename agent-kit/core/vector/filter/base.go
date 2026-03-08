// Ported from: packages/core/src/vector/filter/base.ts
package filter

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Operator constants
// ---------------------------------------------------------------------------

// Basic operators
const (
	OpEq = "$eq" // Matches values equal to specified value
	OpNe = "$ne" // Matches values not equal
)

// Numeric operators
const (
	OpGt  = "$gt"  // Greater than
	OpGte = "$gte" // Greater than or equal
	OpLt  = "$lt"  // Less than
	OpLte = "$lte" // Less than or equal
)

// Logical operators
const (
	OpAnd = "$and" // Joins query clauses with logical AND
	OpNot = "$not" // Inverts the effect of a query expression
	OpNor = "$nor" // Joins query clauses with logical NOR
	OpOr  = "$or"  // Joins query clauses with logical OR
)

// Array operators
const (
	OpAll       = "$all"       // Matches arrays containing all elements
	OpIn        = "$in"        // Matches any value in array
	OpNin       = "$nin"       // Matches none of the values in array
	OpElemMatch = "$elemMatch" // Matches array field with element matching all criteria
)

// Element operator
const (
	OpExists = "$exists" // Matches documents that have the specified field
)

// Regex operators
const (
	OpRegex   = "$regex"   // Regular expression match
	OpOptions = "$options" // Regex options
)

// ---------------------------------------------------------------------------
// Operator slices (static arrays from TS)
// ---------------------------------------------------------------------------

var (
	BasicOperators   = []string{OpEq, OpNe}
	NumericOperators = []string{OpGt, OpGte, OpLt, OpLte}
	ArrayOperators   = []string{OpIn, OpNin, OpAll, OpElemMatch}
	LogicalOperators = []string{OpAnd, OpOr, OpNot, OpNor}
	ElementOperators = []string{OpExists}
	RegexOperators   = []string{OpRegex, OpOptions}
)

// DefaultOperators provides the standard set of supported operators,
// mirroring BaseFilterTranslator.DEFAULT_OPERATORS in TS.
var DefaultOperators = OperatorSupport{
	Logical: LogicalOperators,
	Basic:   BasicOperators,
	Numeric: NumericOperators,
	Array:   ArrayOperators,
	Element: ElementOperators,
	Regex:   RegexOperators,
}

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// FilterValue represents a leaf value in a filter expression.
// Corresponds to TS: string | number | boolean | Date | null | undefined | EmptyObject
type FilterValue = any

// VectorFieldValue is a field value that can be a single FilterValue or a slice.
// Corresponds to TS: FilterValue | FilterValue[]
type VectorFieldValue = any

// VectorFilter is the top-level filter type passed to translators.
// In Go this is a map[string]any (or nil), mirroring the TS union of
// FilterCondition | null | undefined.
type VectorFilter = map[string]any

// OperatorCondition represents {$op: value} objects within a field.
type OperatorCondition = map[string]any

// FieldCondition represents {field: OperatorCondition | FieldValue}.
type FieldCondition = map[string]any

// LogicalCondition represents {$and: [...]} / {$or: [...]} / {$not: {...}} etc.
type LogicalCondition = map[string]any

// OperatorSupport declares which operator categories a translator supports.
type OperatorSupport struct {
	Logical []string `json:"logical,omitempty"`
	Array   []string `json:"array,omitempty"`
	Basic   []string `json:"basic,omitempty"`
	Numeric []string `json:"numeric,omitempty"`
	Element []string `json:"element,omitempty"`
	Regex   []string `json:"regex,omitempty"`
	Custom  []string `json:"custom,omitempty"`
}

// AllOperators returns every operator string across all categories.
func (os OperatorSupport) AllOperators() []string {
	var all []string
	all = append(all, os.Logical...)
	all = append(all, os.Array...)
	all = append(all, os.Basic...)
	all = append(all, os.Numeric...)
	all = append(all, os.Element...)
	all = append(all, os.Regex...)
	all = append(all, os.Custom...)
	return all
}

// ValidationResult holds the outcome of ValidateFilter.
type ValidationResult struct {
	Supported bool
	Messages  []string
}

// ---------------------------------------------------------------------------
// Error message factories
// ---------------------------------------------------------------------------

func ErrUnsupportedOperator(op string) string {
	return fmt.Sprintf("Unsupported operator: %s", op)
}

func ErrInvalidLogicalOperatorLocation(op, path string) string {
	return fmt.Sprintf("Logical operator %s cannot be used at field level: %s", op, path)
}

const ErrNotRequiresObject = "$not operator requires an object"
const ErrNotCannotBeEmpty = "$not operator cannot be empty"

func ErrInvalidLogicalOperatorContent(path string) string {
	return fmt.Sprintf("Logical operators must contain field conditions, not direct operators: %s", path)
}

func ErrInvalidTopLevelOperator(op string) string {
	return fmt.Sprintf("Invalid top-level operator: %s", op)
}

const ErrElemMatchRequiresObject = "$elemMatch requires an object with conditions"

// ---------------------------------------------------------------------------
// FilterTranslator interface (abstract class → Go interface)
// ---------------------------------------------------------------------------

// FilterTranslator is the interface that concrete filter translators must
// implement. It corresponds to the abstract BaseFilterTranslator class in TS.
type FilterTranslator interface {
	// Translate converts a VectorFilter into the target representation.
	Translate(filter VectorFilter) (any, error)

	// GetSupportedOperators returns the operators this translator supports.
	// Concrete types may override to restrict the set.
	GetSupportedOperators() OperatorSupport
}

// ---------------------------------------------------------------------------
// BaseFilterTranslator — shared helper methods
// ---------------------------------------------------------------------------

// BaseFilterTranslator provides shared helper methods that any concrete
// translator can embed. It mirrors the non-abstract members of the TS class.
type BaseFilterTranslator struct{}

// IsOperator returns true when the key starts with '$'.
func (b *BaseFilterTranslator) IsOperator(key string) bool {
	return strings.HasPrefix(key, "$")
}

// IsLogicalOperator checks membership in LogicalOperators.
func (b *BaseFilterTranslator) IsLogicalOperator(key string) bool {
	return contains(LogicalOperators, key)
}

// IsBasicOperator checks membership in BasicOperators.
func (b *BaseFilterTranslator) IsBasicOperator(key string) bool {
	return contains(BasicOperators, key)
}

// IsNumericOperator checks membership in NumericOperators.
func (b *BaseFilterTranslator) IsNumericOperator(key string) bool {
	return contains(NumericOperators, key)
}

// IsArrayOperator checks membership in ArrayOperators.
func (b *BaseFilterTranslator) IsArrayOperator(key string) bool {
	return contains(ArrayOperators, key)
}

// IsElementOperator checks membership in ElementOperators.
func (b *BaseFilterTranslator) IsElementOperator(key string) bool {
	return contains(ElementOperators, key)
}

// IsRegexOperator checks membership in RegexOperators.
func (b *BaseFilterTranslator) IsRegexOperator(key string) bool {
	return contains(RegexOperators, key)
}

// IsFieldOperator returns true for operators that are NOT logical.
func (b *BaseFilterTranslator) IsFieldOperator(key string) bool {
	return b.IsOperator(key) && !b.IsLogicalOperator(key)
}

// IsCustomOperator checks whether key is in the Custom slice of the given support.
func (b *BaseFilterTranslator) IsCustomOperator(key string, support OperatorSupport) bool {
	return contains(support.Custom, key)
}

// GetSupportedOperators returns the default operator set.
// Concrete translators embedding BaseFilterTranslator can override this.
func (b *BaseFilterTranslator) GetSupportedOperators() OperatorSupport {
	return DefaultOperators
}

// IsValidOperator checks whether key appears in any category of the given support.
func (b *BaseFilterTranslator) IsValidOperator(key string, support OperatorSupport) bool {
	return contains(support.AllOperators(), key)
}

// ---------------------------------------------------------------------------
// Value helpers
// ---------------------------------------------------------------------------

// NormalizeComparisonValue normalises a value for comparison operators.
// time.Time → ISO-8601 string, negative-zero → 0.
func (b *BaseFilterTranslator) NormalizeComparisonValue(value any) any {
	if t, ok := value.(time.Time); ok {
		return t.UTC().Format(time.RFC3339Nano)
	}
	if f, ok := toFloat64(value); ok {
		if math.IsInf(f, 0) || math.IsNaN(f) {
			return value
		}
		if f == 0 && math.Signbit(f) {
			return 0
		}
	}
	return value
}

// NormalizeArrayValues normalises every element in the slice.
func (b *BaseFilterTranslator) NormalizeArrayValues(values []any) []any {
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = b.NormalizeComparisonValue(v)
	}
	return out
}

// SimulateAllOperator builds an $and filter that simulates $all using $in.
// Useful for vector stores that don't support $all natively.
func (b *BaseFilterTranslator) SimulateAllOperator(field string, values []any) VectorFilter {
	clauses := make([]any, len(values))
	for i, v := range values {
		clauses[i] = map[string]any{
			field: map[string]any{
				OpIn: []any{b.NormalizeComparisonValue(v)},
			},
		}
	}
	return VectorFilter{OpAnd: clauses}
}

// IsPrimitive returns true for nil, string, numeric types, and bool.
func (b *BaseFilterTranslator) IsPrimitive(value any) bool {
	if value == nil {
		return true
	}
	switch value.(type) {
	case string, bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return true
	}
	return false
}

// IsEmpty returns true for nil or an empty map.
func (b *BaseFilterTranslator) IsEmpty(obj any) bool {
	if obj == nil {
		return true
	}
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Map && v.Len() == 0 {
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// Filter validation
// ---------------------------------------------------------------------------

// ValidateFilter validates the filter and returns an error if unsupported
// constructs are found.
func (b *BaseFilterTranslator) ValidateFilter(filter VectorFilter, support OperatorSupport) error {
	result := b.validateFilterSupport(filter, "", support)
	if !result.Supported {
		return fmt.Errorf("%s", strings.Join(result.Messages, ", "))
	}
	return nil
}

// validateFilterSupport recursively validates a filter node.
func (b *BaseFilterTranslator) validateFilterSupport(node any, path string, support OperatorSupport) ValidationResult {
	// Handle primitives and empty values
	if b.IsPrimitive(node) || b.IsEmpty(node) {
		return ValidationResult{Supported: true}
	}

	// Handle slices (arrays in TS)
	if slice, ok := toSlice(node); ok {
		var messages []string
		supported := true
		for _, item := range slice {
			r := b.validateFilterSupport(item, path, support)
			if !r.Supported {
				supported = false
				messages = append(messages, r.Messages...)
			}
		}
		return ValidationResult{Supported: supported, Messages: messages}
	}

	// Process map entries (object in TS)
	nodeMap, ok := toMap(node)
	if !ok {
		return ValidationResult{Supported: true}
	}

	var messages []string
	isSupported := true

	for key, value := range nodeMap {
		newPath := key
		if path != "" {
			newPath = path + "." + key
		}

		if b.IsOperator(key) {
			if !b.IsValidOperator(key, support) {
				isSupported = false
				messages = append(messages, ErrUnsupportedOperator(key))
				continue
			}

			// Non-logical operators at top level are invalid
			if path == "" && !b.IsLogicalOperator(key) {
				isSupported = false
				messages = append(messages, ErrInvalidTopLevelOperator(key))
				continue
			}

			// $elemMatch requires an object (map), not an array or primitive
			if key == OpElemMatch {
				if _, isMap := toMap(value); !isMap {
					isSupported = false
					messages = append(messages, ErrElemMatchRequiresObject)
					continue
				}
			}

			// Special validation for logical operators
			if b.IsLogicalOperator(key) {
				if key == OpNot {
					if _, isSlice := toSlice(value); isSlice {
						isSupported = false
						messages = append(messages, ErrNotRequiresObject)
						continue
					}
					if _, isMap := toMap(value); !isMap {
						isSupported = false
						messages = append(messages, ErrNotRequiresObject)
						continue
					}
					if b.IsEmpty(value) {
						isSupported = false
						messages = append(messages, ErrNotCannotBeEmpty)
						continue
					}
					// $not can be used at field level or top level
					continue
				}

				// Other logical operators can only be at top level or nested
				// inside another logical operator.
				if path != "" {
					parts := strings.Split(path, ".")
					parent := parts[len(parts)-1]
					if !b.IsLogicalOperator(parent) {
						isSupported = false
						messages = append(messages, ErrInvalidLogicalOperatorLocation(key, newPath))
						continue
					}
				}

				if arr, isSlice := toSlice(value); isSlice {
					hasDirectOperators := false
					for _, item := range arr {
						itemMap, isMap := toMap(item)
						if !isMap {
							continue
						}
						if len(itemMap) != 1 {
							continue
						}
						for k := range itemMap {
							if b.IsFieldOperator(k) {
								hasDirectOperators = true
							}
						}
					}
					if hasDirectOperators {
						isSupported = false
						messages = append(messages, ErrInvalidLogicalOperatorContent(newPath))
						continue
					}
				}
			}
		}

		// Recursively validate nested value
		nested := b.validateFilterSupport(value, newPath, support)
		if !nested.Supported {
			isSupported = false
			messages = append(messages, nested.Messages...)
		}
	}

	return ValidationResult{Supported: isSupported, Messages: messages}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// contains checks if a string is present in a slice.
func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// toFloat64 attempts to convert numeric types to float64.
func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	}
	return 0, false
}

// toSlice attempts to interpret v as []any.
func toSlice(v any) ([]any, bool) {
	if v == nil {
		return nil, false
	}
	if s, ok := v.([]any); ok {
		return s, true
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Slice {
		out := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out[i] = rv.Index(i).Interface()
		}
		return out, true
	}
	return nil, false
}

// toMap attempts to interpret v as map[string]any.
func toMap(v any) (map[string]any, bool) {
	if v == nil {
		return nil, false
	}
	if m, ok := v.(map[string]any); ok {
		return m, true
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String {
		out := make(map[string]any, rv.Len())
		for _, k := range rv.MapKeys() {
			out[k.String()] = rv.MapIndex(k).Interface()
		}
		return out, true
	}
	return nil, false
}
