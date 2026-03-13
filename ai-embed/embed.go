package aiembed

//go:generate go run ./cmd/compile-bundle

import (
	_ "embed"

	"fmt"

	"github.com/brainlet/brainkit/jsbridge"
)

//go:embed ai_sdk_bundle.js
var bundleSource string

//go:embed ai_sdk_bundle.bc
var bundleBytecode []byte

// LoadBundle evaluates the AI SDK bundle from precompiled bytecode.
// Falls back to source evaluation if bytecode is empty.
// After loading, globalThis.__ai_sdk is available with generateText, streamText, and createOpenAI.
func LoadBundle(b *jsbridge.Bridge) error {
	if len(bundleBytecode) > 0 {
		val, err := b.EvalBytecode(bundleBytecode)
		if err != nil {
			return fmt.Errorf("ai-embed: load bytecode: %w", err)
		}
		val.Free()
		return nil
	}

	val, err := b.Eval("ai-sdk-bundle.js", bundleSource)
	if err != nil {
		return fmt.Errorf("ai-embed: load bundle: %w", err)
	}
	val.Free()
	return nil
}

// LoadBundleFromSource forces evaluation from JavaScript source instead of bytecode.
// Useful for debugging or when bytecode may be stale.
func LoadBundleFromSource(b *jsbridge.Bridge) error {
	val, err := b.Eval("ai-sdk-bundle.js", bundleSource)
	if err != nil {
		return fmt.Errorf("ai-embed: load bundle: %w", err)
	}
	val.Free()
	return nil
}
