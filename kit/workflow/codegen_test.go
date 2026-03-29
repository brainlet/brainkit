package workflow

import (
	"strings"
	"testing"
)

func TestCodegen_BuiltinDeclarations(t *testing.T) {
	reg := NewHostFunctionRegistry()
	files := GenerateASDeclarations(reg)

	// brainkit.d.ts should always exist
	brainkit, ok := files["brainkit.d.ts"]
	if !ok {
		t.Fatal("expected brainkit.d.ts")
	}
	if !strings.Contains(brainkit, "function step(name: string): void") {
		t.Fatal("expected step declaration")
	}
	if !strings.Contains(brainkit, "function waitForEvent") {
		t.Fatal("expected waitForEvent declaration")
	}
	if !strings.Contains(brainkit, "function complete") {
		t.Fatal("expected complete declaration")
	}

	// ai.d.ts should always exist
	ai, ok := files["ai.d.ts"]
	if !ok {
		t.Fatal("expected ai.d.ts")
	}
	if !strings.Contains(ai, "function generate(prompt: string): string") {
		t.Fatal("expected generate declaration")
	}
}

func TestCodegen_PluginDeclarations(t *testing.T) {
	reg := NewHostFunctionRegistry()

	// Register plugin host functions
	reg.Register(HostFunctionDef{
		Module:      "telegram",
		Name:        "send",
		Description: "Send a message to a Telegram chat",
		Params:      []HostParam{{Name: "chatId", Type: "i64"}, {Name: "text", Type: "string"}},
		Returns:     "void",
		PluginName:  "telegram-gateway",
	})
	reg.Register(HostFunctionDef{
		Module:      "telegram",
		Name:        "send_photo",
		Description: "Send a photo",
		Params:      []HostParam{{Name: "chatId", Type: "i64"}, {Name: "url", Type: "string"}},
		Returns:     "void",
		PluginName:  "telegram-gateway",
	})
	reg.Register(HostFunctionDef{
		Module:      "db",
		Name:        "query",
		Description: "Execute SQL query",
		Params:      []HostParam{{Name: "sql", Type: "string"}},
		Returns:     "string",
		PluginName:  "postgres-driver",
	})

	files := GenerateASDeclarations(reg)

	// telegram.d.ts
	tg, ok := files["telegram.d.ts"]
	if !ok {
		t.Fatal("expected telegram.d.ts")
	}
	if !strings.Contains(tg, `declare module "telegram"`) {
		t.Fatal("expected telegram module declaration")
	}
	if !strings.Contains(tg, "function send(chatId: i64, text: string): void") {
		t.Fatalf("expected send declaration, got:\n%s", tg)
	}
	if !strings.Contains(tg, "function send_photo") {
		t.Fatal("expected send_photo declaration")
	}
	if !strings.Contains(tg, "Send a message") {
		t.Fatal("expected JSDoc comment")
	}

	// db.d.ts
	db, ok := files["db.d.ts"]
	if !ok {
		t.Fatal("expected db.d.ts")
	}
	if !strings.Contains(db, "function query(sql: string): string") {
		t.Fatalf("expected query declaration, got:\n%s", db)
	}
}

func TestCodegen_EmptyRegistry(t *testing.T) {
	reg := NewHostFunctionRegistry()
	files := GenerateASDeclarations(reg)

	// Should still have brainkit.d.ts and ai.d.ts
	if _, ok := files["brainkit.d.ts"]; !ok {
		t.Fatal("expected brainkit.d.ts even with empty registry")
	}
	if _, ok := files["ai.d.ts"]; !ok {
		t.Fatal("expected ai.d.ts even with empty registry")
	}
	// Should NOT have any plugin modules
	if len(files) != 2 {
		t.Fatalf("expected 2 files (brainkit + ai), got %d", len(files))
	}
}
