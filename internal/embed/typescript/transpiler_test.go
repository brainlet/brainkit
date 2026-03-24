package typescript

import (
	"strings"
	"testing"
)

func TestTranspileTS(t *testing.T) {
	source := `import { generateText } from "ai";
import { model, output } from "kit";

interface Config {
  prompt: string;
  temperature?: number;
}

const cfg: Config = { prompt: "Hello" };
const result = await generateText({
  model: model("openai", "gpt-4o-mini"),
  prompt: cfg.prompt,
});
output({ text: result.text });`

	js, err := TranspileTS(source, "test.ts")
	if err != nil {
		t.Fatalf("TranspileTS: %v", err)
	}

	// Types stripped
	if strings.Contains(js, "interface Config") {
		t.Errorf("interface not stripped:\n%s", js)
	}
	if strings.Contains(js, ": Config") {
		t.Errorf("type annotation not stripped:\n%s", js)
	}
	// Code preserved
	if !strings.Contains(js, "generateText") {
		t.Errorf("generateText lost:\n%s", js)
	}
	if !strings.Contains(js, `"ai"`) {
		t.Errorf("ai import lost:\n%s", js)
	}
	t.Logf("Output:\n%s", js)
}
