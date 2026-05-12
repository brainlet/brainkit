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

type Result = { text: string };

const cfg: Config = { prompt: "Hello" };
const result: Result = { text: cfg.prompt };
output({ text: result.text });`

	js, err := TranspileTS(source, "test.ts")
	if err != nil {
		t.Fatalf("TranspileTS: %v", err)
	}

	if strings.Contains(js, "interface Config") {
		t.Errorf("interface not stripped:\n%s", js)
	}
	if strings.Contains(js, "type Result") {
		t.Errorf("type alias not stripped:\n%s", js)
	}
	if strings.Contains(js, ": Config") || strings.Contains(js, ": Result") {
		t.Errorf("type annotation not stripped:\n%s", js)
	}
	if !strings.Contains(js, "generateText") {
		t.Errorf("generateText lost:\n%s", js)
	}
	if !strings.Contains(js, `"ai"`) {
		t.Errorf("ai import lost:\n%s", js)
	}
}
