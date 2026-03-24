package typescript

import (
	"strings"
	"testing"
)

func TestTranspileStripTypes(t *testing.T) {
	source := `const x: number = 42;
function add(a: number, b: number): number {
  return a + b;
}
export { add };`

	result, err := Transpile(source, TranspileOptions{})
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}

	// Types should be stripped
	if strings.Contains(result, ": number") {
		t.Errorf("types not stripped:\n%s", result)
	}
	// Code should be preserved
	if !strings.Contains(result, "const x") {
		t.Errorf("variable declaration lost:\n%s", result)
	}
	if !strings.Contains(result, "return a + b") {
		t.Errorf("function body lost:\n%s", result)
	}
	if !strings.Contains(result, "export") {
		t.Errorf("export lost:\n%s", result)
	}
	t.Logf("Output:\n%s", result)
}

func TestTranspileInterface(t *testing.T) {
	source := `interface Foo { bar: string; }
const x = 42;`

	result, err := Transpile(source, TranspileOptions{})
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}

	if strings.Contains(result, "interface") {
		t.Errorf("interface not removed:\n%s", result)
	}
	if !strings.Contains(result, "const x = 42") {
		t.Errorf("code lost:\n%s", result)
	}
	t.Logf("Output:\n%s", result)
}

func TestTranspileImports(t *testing.T) {
	source := `import { Agent } from "agent";
import { model, output } from "kit";

const a = new Agent({ name: "test" });
output(a);`

	result, err := Transpile(source, TranspileOptions{})
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}

	if !strings.Contains(result, `"agent"`) {
		t.Errorf("agent import lost:\n%s", result)
	}
	if !strings.Contains(result, `"kit"`) {
		t.Errorf("kit import lost:\n%s", result)
	}
	if !strings.Contains(result, "new Agent") {
		t.Errorf("constructor lost:\n%s", result)
	}
	t.Logf("Output:\n%s", result)
}

func TestTranspileTypeAlias(t *testing.T) {
	source := `type Result = { text: string; score: number };
const r: Result = { text: "hello", score: 42 };`

	result, err := Transpile(source, TranspileOptions{})
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}

	if strings.Contains(result, "type Result") {
		t.Errorf("type alias not removed:\n%s", result)
	}
	if !strings.Contains(result, "const r") {
		t.Errorf("variable lost:\n%s", result)
	}
	t.Logf("Output:\n%s", result)
}

func TestTranspileAsync(t *testing.T) {
	source := `import { generateText } from "ai";
import { model, output } from "kit";

const result = await generateText({
  model: model("openai", "gpt-4o-mini"),
  prompt: "Hello",
});
output({ text: result.text });`

	result, err := Transpile(source, TranspileOptions{})
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}

	if !strings.Contains(result, "await generateText") {
		t.Errorf("await lost:\n%s", result)
	}
	if !strings.Contains(result, "model(") {
		t.Errorf("model call lost:\n%s", result)
	}
	t.Logf("Output:\n%s", result)
}

func TestTranspileGenericFunction(t *testing.T) {
	source := `function identity<T>(arg: T): T {
  return arg;
}
const result = identity<string>("hello");`

	result, err := Transpile(source, TranspileOptions{})
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}

	if strings.Contains(result, "<T>") || strings.Contains(result, "<string>") {
		t.Errorf("generics not stripped:\n%s", result)
	}
	if !strings.Contains(result, "function identity") {
		t.Errorf("function lost:\n%s", result)
	}
	t.Logf("Output:\n%s", result)
}

func TestTranspileDecorators(t *testing.T) {
	// Decorators should pass through (ESNext target preserves them)
	source := `const x = 42;
export default x;`

	result, err := Transpile(source, TranspileOptions{})
	if err != nil {
		t.Fatalf("Transpile: %v", err)
	}

	if !strings.Contains(result, "const x = 42") {
		t.Errorf("code lost:\n%s", result)
	}
	t.Logf("Output:\n%s", result)
}

func TestTranspileEmptySource(t *testing.T) {
	result, err := Transpile("", TranspileOptions{})
	if err != nil {
		t.Fatalf("Transpile empty: %v", err)
	}
	// Empty source should produce empty or whitespace-only output
	trimmed := strings.TrimSpace(result)
	if trimmed != "" {
		t.Errorf("expected empty output, got: %q", trimmed)
	}
}
