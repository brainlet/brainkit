package testing

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// Expectation provides assertion methods on a value.
type Expectation struct {
	value any
	not   bool
}

// Expect creates a new expectation.
func Expect(value any) *Expectation {
	return &Expectation{value: value}
}

// Not negates the next assertion.
func (e *Expectation) Not() *Expectation {
	return &Expectation{value: e.value, not: !e.not}
}

func (e *Expectation) check(pass bool, msg string) error {
	if e.not {
		pass = !pass
		if !pass {
			return fmt.Errorf("expected NOT: %s", msg)
		}
		return nil
	}
	if !pass {
		return fmt.Errorf("%s", msg)
	}
	return nil
}

// ToBe checks strict equality.
func (e *Expectation) ToBe(expected any) error {
	pass := fmt.Sprintf("%v", e.value) == fmt.Sprintf("%v", expected)
	return e.check(pass, fmt.Sprintf("expected %v to be %v", e.value, expected))
}

// ToEqual checks deep equality.
func (e *Expectation) ToEqual(expected any) error {
	pass := reflect.DeepEqual(e.value, expected)
	return e.check(pass, fmt.Sprintf("expected %v to equal %v", e.value, expected))
}

// ToContain checks string contains or array includes.
func (e *Expectation) ToContain(substring string) error {
	str := fmt.Sprintf("%v", e.value)
	pass := strings.Contains(str, substring)
	return e.check(pass, fmt.Sprintf("expected %q to contain %q", str, substring))
}

// ToMatch checks regex match.
func (e *Expectation) ToMatch(pattern string) error {
	str := fmt.Sprintf("%v", e.value)
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex %q: %w", pattern, err)
	}
	pass := re.MatchString(str)
	return e.check(pass, fmt.Sprintf("expected %q to match %q", str, pattern))
}

// ToBeTruthy checks value is truthy.
func (e *Expectation) ToBeTruthy() error {
	pass := isTruthy(e.value)
	return e.check(pass, fmt.Sprintf("expected %v to be truthy", e.value))
}

// ToBeFalsy checks value is falsy.
func (e *Expectation) ToBeFalsy() error {
	pass := !isTruthy(e.value)
	return e.check(pass, fmt.Sprintf("expected %v to be falsy", e.value))
}

// ToBeDefined checks value is not nil/undefined.
func (e *Expectation) ToBeDefined() error {
	pass := e.value != nil
	return e.check(pass, "expected value to be defined")
}

// ToBeGreaterThan checks numeric comparison.
func (e *Expectation) ToBeGreaterThan(n float64) error {
	v := toFloat(e.value)
	pass := v > n
	return e.check(pass, fmt.Sprintf("expected %v to be greater than %v", e.value, n))
}

// ToBeLessThan checks numeric comparison.
func (e *Expectation) ToBeLessThan(n float64) error {
	v := toFloat(e.value)
	pass := v < n
	return e.check(pass, fmt.Sprintf("expected %v to be less than %v", e.value, n))
}

// ToHaveLength checks length of string, array, or map.
func (e *Expectation) ToHaveLength(n int) error {
	length := getLength(e.value)
	pass := length == n
	return e.check(pass, fmt.Sprintf("expected length %d, got %d", n, length))
}

// ToBeNull checks value is nil/null.
func (e *Expectation) ToBeNull() error {
	pass := e.value == nil
	return e.check(pass, fmt.Sprintf("expected %v to be null", e.value))
}

// ToHaveProperty checks that a map has a specific key.
func (e *Expectation) ToHaveProperty(key string) error {
	m, ok := e.value.(map[string]any)
	if !ok {
		return e.check(false, fmt.Sprintf("expected object with property %q, got %T", key, e.value))
	}
	_, has := m[key]
	return e.check(has, fmt.Sprintf("expected object to have property %q", key))
}

// ToThrow checks that a function panics when called.
// If message is provided, also checks that the panic message contains it.
func (e *Expectation) ToThrow(message ...string) error {
	fn, ok := e.value.(func())
	if !ok {
		return fmt.Errorf("toThrow: expected a func(), got %T", e.value)
	}
	var panicked bool
	var panicMsg string
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				panicMsg = fmt.Sprintf("%v", r)
			}
		}()
		fn()
	}()
	if err := e.check(panicked, "expected function to throw/panic"); err != nil {
		return err
	}
	if len(message) > 0 && message[0] != "" {
		if !strings.Contains(panicMsg, message[0]) {
			return e.check(false, fmt.Sprintf("expected panic message to contain %q, got %q", message[0], panicMsg))
		}
	}
	return nil
}

func isTruthy(v any) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val != ""
	case int:
		return val != 0
	case float64:
		return val != 0
	case json.RawMessage:
		return len(val) > 0 && string(val) != "null"
	default:
		return true
	}
}

func toFloat(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case json.Number:
		f, _ := val.Float64()
		return f
	default:
		return 0
	}
}

func getLength(v any) int {
	switch val := v.(type) {
	case string:
		return len(val)
	case []any:
		return len(val)
	case map[string]any:
		return len(val)
	default:
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Map || rv.Kind() == reflect.String {
			return rv.Len()
		}
		return 0
	}
}
