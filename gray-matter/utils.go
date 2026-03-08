package graymatter

import (
	"fmt"
	"reflect"
	"strings"
)

func stripBOM(input string) string {
	return strings.TrimPrefix(input, "\ufeff")
}

func excerptFile(file *File, opts Options) error {
	if file.Data == nil {
		file.Data = map[string]any{}
	}

	if isCallable(opts.Excerpt) {
		return callExcerpt(opts.Excerpt, file, opts)
	}

	sep := excerptSeparatorFromData(file.Data)
	if sep == "" {
		sep = opts.ExcerptSeparator
	}
	if sep == "" && (opts.Excerpt == nil || excerptDisabled(opts.Excerpt)) {
		return nil
	}

	delimiter := ""
	if str, ok := opts.Excerpt.(string); ok {
		delimiter = str
	} else if sep != "" {
		delimiter = sep
	} else {
		delimiter = NormalizeDelimiters(opts.Delimiters)[0]
	}

	if delimiter == "" {
		return nil
	}

	if idx := strings.Index(file.Content, delimiter); idx != -1 {
		file.Excerpt = file.Content[:idx]
	}
	return nil
}

func excerptDisabled(value any) bool {
	flag, ok := value.(bool)
	return ok && !flag
}

func excerptSeparatorFromData(data any) string {
	m, ok := data.(map[string]any)
	if !ok {
		return ""
	}

	value, ok := m["excerpt_separator"]
	if !ok {
		return ""
	}

	str, ok := value.(string)
	if !ok {
		return ""
	}
	return str
}

func toMap(data any) map[string]any {
	if data == nil {
		return nil
	}
	m, ok := data.(map[string]any)
	if !ok {
		return nil
	}
	return m
}

func isCallable(fn any) bool {
	if fn == nil {
		return false
	}
	return reflect.TypeOf(fn).Kind() == reflect.Func
}

func callParser(parser any, input string, opts Options) (any, error) {
	switch fn := parser.(type) {
	case func(string, Options) (any, error):
		return fn(input, opts)
	case func(string, Options) any:
		return fn(input, opts), nil
	case func(string) (any, error):
		return fn(input)
	case func(string) any:
		return fn(input), nil
	}

	values, err := callDynamic(parser, input, opts)
	if err != nil {
		return nil, err
	}

	switch len(values) {
	case 1:
		return values[0].Interface(), nil
	case 2:
		if !values[1].IsNil() {
			return values[0].Interface(), values[1].Interface().(error)
		}
		return values[0].Interface(), nil
	default:
		return nil, fmt.Errorf("graymatter: unsupported parser signature")
	}
}

func callExcerpt(callback any, file *File, opts Options) error {
	switch fn := callback.(type) {
	case func(*File, Options):
		fn(file, opts)
		return nil
	case func(*File):
		fn(file)
		return nil
	case func(File, Options):
		fn(*file, opts)
		return nil
	case func(File):
		fn(*file)
		return nil
	}

	_, err := callDynamic(callback, file, opts)
	return err
}

func callSection(callback any, section *Section, sections []Section) error {
	switch fn := callback.(type) {
	case func(*Section, []Section):
		fn(section, sections)
		return nil
	case func(*Section):
		fn(section)
		return nil
	case func(Section, []Section):
		fn(*section, sections)
		return nil
	case func(Section):
		fn(*section)
		return nil
	}

	_, err := callDynamic(callback, section, sections)
	return err
}

func callDynamic(fn any, args ...any) ([]reflect.Value, error) {
	value := reflect.ValueOf(fn)
	if !value.IsValid() || value.Kind() != reflect.Func {
		return nil, fmt.Errorf("graymatter: expected function, got %T", fn)
	}

	typ := value.Type()
	if typ.NumIn() > len(args) {
		return nil, fmt.Errorf("graymatter: expected at least %d arguments, got %d", typ.NumIn(), len(args))
	}

	inputs := make([]reflect.Value, typ.NumIn())
	for i := 0; i < typ.NumIn(); i++ {
		arg := reflect.ValueOf(args[i])
		paramType := typ.In(i)

		if !arg.IsValid() {
			inputs[i] = reflect.Zero(paramType)
			continue
		}

		if arg.Type().AssignableTo(paramType) {
			inputs[i] = arg
			continue
		}
		if arg.Type().ConvertibleTo(paramType) {
			inputs[i] = arg.Convert(paramType)
			continue
		}
		return nil, fmt.Errorf("graymatter: cannot use %s as %s", arg.Type(), paramType)
	}

	results := value.Call(inputs)
	if len(results) > 0 {
		last := results[len(results)-1]
		errType := reflect.TypeOf((*error)(nil)).Elem()
		if last.IsValid() && last.Type().Implements(errType) && len(results) == 1 && !last.IsNil() {
			return nil, last.Interface().(error)
		}
	}
	return results, nil
}
