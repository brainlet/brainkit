# slugify

Go port of [slugify](https://github.com/sindresorhus/slugify).

## Usage

```go
import "github.com/brainlet/brainkit/slugify"

// Basic usage
slugify.Slugify("Hello World") // "hello-world"

// Custom replacement character
slugify.Slugify("Hello World", slugify.String("_")) // "hello_world"

// Lowercase
slugify.Slugify("Hello World", slugify.Bool(true)) // "hello-world"

// Strict mode (alphanumeric only)
slugify.Slugify("foo_bar!baz", &slugify.Options{Strict: slugify.Bool(true)}) // "foobarbaz"

// Locale-specific transliteration
slugify.Slugify("Ä Ö Ü", slugify.String("de")) // "AE-OE-UE"

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
| Locale | `*string` | `nil` | Language code (de, fr, es, etc.) |
| Trim | `*bool` | `true` | Trim leading/trailing separators |
| Remove | `*regexp.Regexp` | nil | Custom regex to remove |

## TS Source

`/Users/davidroman/Documents/code/clones/slugify/`

## Tests

```
go test ./slugify/...
go test -race ./slugify/...
```

## TS → Go Patterns

### Optional Parameters → Pointer Fields

TS: `options.replacement?: string`
Go: `Replacement *string` — use `slugify.String("custom")` helper

### undefined → nil

TS uses `undefined` for omitted options. Go uses nil pointer values.

### Extend → Global Mutex

TS: `slugify.extend({...})` — modifies global charMap
Go: `Extend(map[string]string)` — thread-safe with sync.RWMutex

### Regex Options

TS: `{remove: /regex/g}` — native RegExp
Go: `Remove *regexp.Regexp` — use `regexp.MustCompile()` or pass compiled regex
