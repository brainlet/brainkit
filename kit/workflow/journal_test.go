package workflow

import (
	"encoding/json"
	"testing"
)

func TestJournal_MarkStep(t *testing.T) {
	j := NewJournal("wf-1", "run-1")

	j.MarkStep("enrich")
	j.MarkStep("analyze")
	j.MarkStep("persist")

	entries := j.Entries()
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].StepName != "enrich" {
		t.Fatalf("expected step 'enrich', got %q", entries[0].StepName)
	}
	if entries[1].StepName != "analyze" {
		t.Fatalf("expected step 'analyze', got %q", entries[1].StepName)
	}
	if entries[2].StepName != "persist" {
		t.Fatalf("expected step 'persist', got %q", entries[2].StepName)
	}
	// First two steps should be completed (closed by the next MarkStep)
	if entries[0].Status != "completed" {
		t.Fatalf("step 0 status: expected 'completed', got %q", entries[0].Status)
	}
	if entries[1].Status != "completed" {
		t.Fatalf("step 1 status: expected 'completed', got %q", entries[1].Status)
	}
	// Last step still pending
	if entries[2].Status != "pending" {
		t.Fatalf("step 2 status: expected 'pending', got %q", entries[2].Status)
	}
}

func TestJournal_RecordCall(t *testing.T) {
	j := NewJournal("wf-1", "run-1")
	j.MarkStep("fetch")

	args, _ := json.Marshal(map[string]string{"url": "https://example.com"})
	result, _ := json.Marshal(map[string]string{"status": "ok"})
	j.RecordCall("http", "get", args, result, nil, 100)

	entries := j.Entries()
	if len(entries[0].Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(entries[0].Calls))
	}
	call := entries[0].Calls[0]
	if call.Function != "http.get" {
		t.Fatalf("expected function 'http.get', got %q", call.Function)
	}
}

func TestJournal_Replay(t *testing.T) {
	// Simulate a previous run's journal
	prevEntries := []JournalEntry{
		{
			StepName:  "fetch",
			StepIndex: 0,
			Status:    "completed",
			Calls: []HostCallRecord{
				{
					Function: "http.get",
					Args:     json.RawMessage(`{"url":"https://example.com"}`),
					Result:   json.RawMessage(`{"data":"cached"}`),
				},
			},
		},
		{
			StepName:  "analyze",
			StepIndex: 1,
			Status:    "completed",
			Calls: []HostCallRecord{
				{
					Function: "ai.generate",
					Args:     json.RawMessage(`{"prompt":"analyze"}`),
					Result:   json.RawMessage(`"AI result"`),
				},
			},
		},
	}

	j := NewJournalFromEntries("wf-1", "run-1", prevEntries)
	if !j.IsReplaying() {
		t.Fatal("expected replaying=true")
	}

	// Replay step 1
	j.MarkStep("fetch")
	result, ok := j.GetRecordedResult("http", "get", nil)
	if !ok {
		t.Fatal("expected recorded result for http.get")
	}
	if string(result) != `{"data":"cached"}` {
		t.Fatalf("expected cached result, got %s", result)
	}

	// Replay step 2
	j.MarkStep("analyze")
	result, ok = j.GetRecordedResult("ai", "generate", nil)
	if !ok {
		t.Fatal("expected recorded result for ai.generate")
	}
	if string(result) != `"AI result"` {
		t.Fatalf("expected AI result, got %s", result)
	}

	// Step 3: new territory — no more replay
	j.MarkStep("persist")
	if j.IsReplaying() {
		t.Fatal("expected replaying=false after catching up")
	}
	_, ok = j.GetRecordedResult("db", "query", nil)
	if ok {
		t.Fatal("expected no recorded result for new step")
	}
}

func TestJournal_MarkCompleted(t *testing.T) {
	j := NewJournal("wf-1", "run-1")
	j.MarkStep("only-step")
	j.MarkCompleted()

	entries := j.Entries()
	if entries[0].Status != "completed" {
		t.Fatalf("expected 'completed', got %q", entries[0].Status)
	}
	if entries[0].CompletedAt == nil {
		t.Fatal("expected CompletedAt to be set")
	}
}

func TestJournal_MarkFailed(t *testing.T) {
	j := NewJournal("wf-1", "run-1")
	j.MarkStep("failing-step")
	j.MarkFailed("something broke")

	entries := j.Entries()
	if entries[0].Status != "failed" {
		t.Fatalf("expected 'failed', got %q", entries[0].Status)
	}
	if entries[0].Error != "something broke" {
		t.Fatalf("expected error msg, got %q", entries[0].Error)
	}
}

func TestHostFunctionRegistry(t *testing.T) {
	reg := NewHostFunctionRegistry()

	reg.Register(HostFunctionDef{
		Module: "telegram", Name: "send",
		Description: "Send message",
		Params:      []HostParam{{Name: "chatId", Type: "i64"}, {Name: "text", Type: "string"}},
		Returns:     "void",
	})
	reg.Register(HostFunctionDef{
		Module: "telegram", Name: "send_photo",
		Description: "Send photo",
		Params:      []HostParam{{Name: "chatId", Type: "i64"}, {Name: "url", Type: "string"}},
		Returns:     "void",
	})
	reg.Register(HostFunctionDef{
		Module: "db", Name: "query",
		Description: "Execute SQL",
		Params:      []HostParam{{Name: "sql", Type: "string"}},
		Returns:     "string",
	})

	// List modules
	modules := reg.ListModules()
	if len(modules) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(modules))
	}

	// List functions in module
	tgFuncs := reg.ListFunctions("telegram")
	if len(tgFuncs) != 2 {
		t.Fatalf("expected 2 telegram functions, got %d", len(tgFuncs))
	}

	// Get specific
	def := reg.Get("db", "query")
	if def == nil {
		t.Fatal("expected db.query")
	}
	if def.Returns != "string" {
		t.Fatalf("expected returns 'string', got %q", def.Returns)
	}

	// Get missing
	if reg.Get("db", "nonexistent") != nil {
		t.Fatal("expected nil for missing function")
	}

	// All
	all := reg.All()
	if len(all) != 3 {
		t.Fatalf("expected 3 total, got %d", len(all))
	}

	// Unregister
	reg.Unregister("telegram", "send_photo")
	tgFuncs = reg.ListFunctions("telegram")
	if len(tgFuncs) != 1 {
		t.Fatalf("expected 1 after unregister, got %d", len(tgFuncs))
	}

	// Unregister module
	reg.UnregisterModule("telegram")
	if len(reg.ListModules()) != 1 {
		t.Fatalf("expected 1 module after unregister, got %d", len(reg.ListModules()))
	}
}
