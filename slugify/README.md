# slugify

Go port of [simov/slugify](https://github.com/simov/slugify).

## Usage

```go
import "github.com/brainlet/brainkit/slugify"

// Basic usage
slugify.Slugify("some string") // "some-string"

// JS-compatible replacement shorthand
slugify.Slugify("some string", "_") // "some_string"

// Lowercase
slugify.Slugify("Foo bAr baZ", slugify.Options{Lower: slugify.Bool(true)}) // "foo-bar-baz"

// Strict mode (alphanumeric only)
slugify.Slugify("foo_bar. -@-baz!", slugify.Options{Strict: slugify.Bool(true)}) // "foobar-baz"

// Locale-specific transliteration
slugify.Slugify("Ä Ö Ü", slugify.Options{Locale: slugify.String("de")}) // "AE-OE-UE"

// Extend with custom characters
slugify.Extend(map[string]string{"☢": "radioactive"})
slugify.Slugify("unicode ♥ is ☢") // "unicode-love-is-radioactive"
```

## Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| Replacement | `*string` | `"-"` | Character to replace spaces |
| Lower | `*bool` | `false` | Convert to lowercase |
| Strict | `*bool` | `false` | Strip special characters |
| Locale | `*string` | `nil` | Language code (`de`, `fr`, `es`, etc.) |
| Trim | `*bool` | `true` | Trim leading/trailing separators |
| Remove | `*regexp.Regexp` | default JS regex | Custom regex to remove |

`Slugify` accepts either:

- no second argument
- a `string` replacement shorthand, matching the JS API
- `slugify.Options` or `*slugify.Options`

## TS Source

[simov/slugify](https://github.com/simov/slugify)

## Tests

```
go test ./slugify/...
go test -race ./slugify/...
```

## TS → Go Patterns

- Optional parameters become pointer fields in `Options`.
- `undefined` becomes `nil`.
- `extend()` maps to `Extend(map[string]string)`.
- JS `string.normalize()` is mirrored with Go NFC normalization.
- The built-in whitespace/default-remove behavior follows the JS implementation, including Unicode whitespace handling.
