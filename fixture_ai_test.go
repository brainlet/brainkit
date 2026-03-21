//go:build integration

package brainkit

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestFixture_TS_AIGenerate(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/ai-generate.js")

	result, err := kit.EvalModule(context.Background(), "ai-generate.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Text     string `json:"text"`
		HasUsage bool   `json:"hasUsage"`
	}
	json.Unmarshal([]byte(result), &out)

	if !strings.Contains(strings.ToUpper(out.Text), "DIRECT") {
		t.Errorf("text = %q", out.Text)
	}
	t.Logf("fixture ai-generate: %q", out.Text)
}

func TestFixture_TS_AIGenerateObject(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/ai-generate-object.js")

	result, err := kit.EvalModule(context.Background(), "ai-generate-object.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Object       map[string]interface{} `json:"object"`
		HasName      bool                   `json:"hasName"`
		HasAge       bool                   `json:"hasAge"`
		HasHobbies   bool                   `json:"hasHobbies"`
		HasUsage     bool                   `json:"hasUsage"`
		FinishReason string                 `json:"finishReason"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.HasName {
		t.Errorf("object missing name: %v", out.Object)
	}
	if !out.HasAge {
		t.Errorf("object missing age: %v", out.Object)
	}
	if !out.HasHobbies {
		t.Errorf("object missing hobbies: %v", out.Object)
	}
	if !out.HasUsage {
		t.Error("expected usage")
	}
	t.Logf("fixture ai-generate-object: %v finish=%s", out.Object, out.FinishReason)
}

func TestFixture_TS_AIStream(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/ai-stream.js")

	result, err := kit.EvalModule(context.Background(), "ai-stream.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Text              string `json:"text"`
		Chunks            int    `json:"chunks"`
		HasRealTimeTokens bool   `json:"hasRealTimeTokens"`
	}
	json.Unmarshal([]byte(result), &out)

	if out.Text == "" {
		t.Error("expected non-empty text")
	}
	if !out.HasRealTimeTokens {
		t.Error("expected real-time token chunks")
	}
	t.Logf("fixture ai-stream: %d chunks, text=%q", out.Chunks, out.Text)
}

func TestFixture_TS_AIEmbed(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/ai-embed.js")

	result, err := kit.EvalModule(context.Background(), "ai-embed.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Single struct {
			Dimensions int  `json:"dimensions"`
			HasValues  bool `json:"hasValues"`
		} `json:"single"`
		Multi struct {
			Count      int  `json:"count"`
			Dimensions int  `json:"dimensions"`
			AllVectors bool `json:"allVectors"`
		} `json:"multi"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.Single.HasValues || out.Single.Dimensions == 0 {
		t.Errorf("single embed failed: dims=%d hasValues=%v", out.Single.Dimensions, out.Single.HasValues)
	}
	if out.Multi.Count != 3 || !out.Multi.AllVectors {
		t.Errorf("multi embed failed: count=%d allVectors=%v", out.Multi.Count, out.Multi.AllVectors)
	}
	t.Logf("ai-embed: single=%d dims, multi=%d vectors × %d dims", out.Single.Dimensions, out.Multi.Count, out.Multi.Dimensions)
}
