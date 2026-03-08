# gray-matter

Go port of [gray-matter](https://github.com/jonschlinkert/gray-matter).

Parses front-matter (YAML, JSON) from strings and files.

## Usage

```go
import "github.com/brainlet/brainkit/gray-matter"
```

### Parse from string

```go
input := `---
title: Hello World
date: 2024-01-15
---
This is the content.
`

file, err := graymatter.Parse(input, nil)
if err != nil {
    log.Fatal(err)
}

fmt.Println(file.Data["title"]) // "Hello World"
fmt.Println(file.Content)       // "This is the content."
fmt.Println(file.Language)      // "yaml"
```

### Parse with options

```go
file, err := graymatter.Parse(input, &graymatter.Options{
    Language: "json",
    Delimiters: "---",
})
```

### Parse JSON front-matter

```go
input := `---json
{"title": "Hello World", "count": 42}
---
Content here.
`

file, err := graymatter.Parse(input, nil)
// file.Data["title"] = "Hello World"
// file.Language = "json"
```

### Read from file

```go
file, err := graymatter.Read("post.md", nil)
if err != nil {
    log.Fatal(err)
}

fmt.Println(file.Data)
fmt.Println(file.Content)
```

### Test if input has front-matter

```go
hasFrontMatter := graymatter.Test("---yaml\ntitle: Hi\n---\nContent")
// returns true
```

### Detect language

```go
lang := graymatter.Language("---json\n{}\n---")
// returns "json"
```

### Stringify (create front-matter)

```go
data := map[string]any{
    "title": "Hello",
    "tags":  []string{"a", "b"},
}

output := graymatter.Stringify(data, "Content goes here", nil)
// ---
// title: Hello
// tags:
//   - a
//   - b
// ---
// Content goes here
```

## Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| Language | `string` | `"yaml"` | Front-matter language (yaml, json) |
| Delimiters | `string` or `[]string` | `"---"` | Opening/closing delimiters |
| Engines | `map[string]Engine` | built-in | Custom parser engines |
| Excerpt | `any` | nil | Enable excerpt extraction |
| ExcerptSeparator | `string` | `"\n"` | Separator for excerpts |
| Eval | `bool` | false | Enable JavaScript evaluation (not implemented) |
| Parser | `any` | nil | Custom parser function |

### Delimiter Options

```go
// Same delimiter for open and close
Delimiters: "---"

// Different open and close
Delimiters: []string{"---", "..."}

// Multiple delimiter pairs
Delimiters: []string{"---", "...", "===", "==="}
```

### Custom Engines

```go
customEngine := &graymatter.EngineWithStringify{
    ParseFunc: func(input string) (map[string]any, error) {
        // custom parsing logic
        return parseCustom(input)
    },
    StringifyFunc: func(data map[string]any) (string, error) {
        // custom stringify logic
        return stringifyCustom(data)
    },
}

file, err := graymatter.Parse(input, &graymatter.Options{
    Engines: map[string]graymatter.Engine{
        "toml": customEngine,
    },
})
```

## Language Detection

The language can be specified in three ways:

1. **Delimiter line**: `---json` or `---yaml`
2. **Options.Language**: Set explicitly in Options
3. **Default**: Falls back to "yaml"

## TS Source

`/Users/davidroman/Documents/code/clones/gray-matter/`

## Tests

```
go test ./gray-matter/...
go test -race ./gray-matter/...
```

## TS → Go Patterns

### Optional Parameters → Pointer

TS: `grayMatter(input, { language: 'json' })`
Go: `graymatter.Parse(input, &graymatter.Options{Language: "json"})`

### Delimiters

TS: `delimiters: '---'` or `delimiters: ['---', '...']`
Go: `Delimiters: "---"` or `Delimiters: []string{"---", "..."}`

### Engine Interface

TS engines can be functions or objects with parse/stringify methods.
Go uses an interface with Parse and Stringify methods.

### Custom Parser Functions

TS: `parser: (input) => YAML.parse(input)`
Go: Use custom Engine implementation with ParseFunc/StringifyFunc
