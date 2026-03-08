package graymatter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/dop251/goja"
)

var jsRetryRE = regexp.MustCompile(`(?i)(unexpected|identifier)`)

type javascriptEngine struct{}

func (javascriptEngine) Parse(input string) (any, error) {
	return parseJavaScript(input, true)
}

func (javascriptEngine) Stringify(data any) (string, error) {
	return "", fmt.Errorf("stringifying JavaScript is not supported")
}

func parseJavaScript(input string, wrap bool) (any, error) {
	vm := goja.New()
	source := strings.TrimSpace(input)
	if wrap {
		source = "(function() {\nreturn " + source + ";\n}());"
	}

	value, err := vm.RunString(source)
	if err != nil {
		if wrap && jsRetryRE.MatchString(err.Error()) {
			return parseJavaScript(input, false)
		}
		return nil, fmt.Errorf("%w", err)
	}

	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return map[string]any{}, nil
	}

	return exportJSValue(vm, value), nil
}

func exportJSValue(vm *goja.Runtime, value goja.Value) any {
	return exportJSValueWithSeen(vm, value, map[*goja.Object]bool{})
}

func exportJSValueWithSeen(vm *goja.Runtime, value goja.Value, seen map[*goja.Object]bool) any {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil
	}

	if _, ok := goja.AssertFunction(value); ok {
		return JSFunction{runtime: vm, value: value}
	}

	exported := value.Export()
	switch exported.(type) {
	case nil, bool, string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return exported
	}

	object := value.ToObject(vm)
	if object == nil {
		return exported
	}
	if seen[object] {
		return exported
	}
	seen[object] = true
	defer delete(seen, object)

	if object.ClassName() == "Array" {
		length := int(object.Get("length").ToInteger())
		items := make([]any, 0, length)
		for i := 0; i < length; i++ {
			items = append(items, exportJSValueWithSeen(vm, object.Get(fmt.Sprintf("%d", i)), seen))
		}
		return items
	}

	keys := object.Keys()
	if len(keys) == 0 {
		return exported
	}

	result := make(map[string]any, len(keys))
	for _, key := range keys {
		result[key] = exportJSValueWithSeen(vm, object.Get(key), seen)
	}
	return result
}
