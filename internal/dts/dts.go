package dts

import _ "embed"

// Type definitions embedded for CLI scaffolding and project templates.
//
// Keeping these in a runtime-free package lets the public brainkit package
// expose scaffolding assets without importing the optional QuickJS runtime.

//go:embed runtime/kit.d.ts
var Kit string

//go:embed runtime/ai.d.ts
var AI string

//go:embed runtime/agent.d.ts
var Agent string

//go:embed runtime/brainkit.d.ts
var Brainkit string

//go:embed runtime/globals.d.ts
var Globals string
