package libsql

import (
	"bytes"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"
)

func TestServerStartStop(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	srv, err := NewServer(dbPath, WithLogger(t.Logf))
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	if srv.URL() == "" {
		t.Fatal("URL should not be empty")
	}

	resp, err := http.Get(srv.URL() + "/health")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("health check: got %d, want 200", resp.StatusCode)
	}
}

func TestInMemoryMode(t *testing.T) {
	srv, err := NewServer(":memory:", WithLogger(t.Logf))
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	result := doPipeline(t, srv.URL(), pipelineRequest{
		Requests: []pipelineEntry{
			{Type: "execute", Stmt: &stmtRequest{SQL: "CREATE TABLE t (id INTEGER PRIMARY KEY, name TEXT)"}},
			{Type: "execute", Stmt: &stmtRequest{
				SQL:  "INSERT INTO t (name) VALUES (?)",
				Args: []hValue{{Type: "text", Value: "test"}},
			}},
			{Type: "execute", Stmt: &stmtRequest{SQL: "SELECT id, name FROM t"}},
			{Type: "close"},
		},
	})

	assertAllOK(t, result, 4)

	r := getExecResult(t, result, 2)
	if len(r.Cols) != 2 {
		t.Fatalf("expected 2 cols, got %d", len(r.Cols))
	}
	if len(r.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(r.Rows))
	}
	if r.Rows[0][0].Type != "integer" || r.Rows[0][0].Value != "1" {
		t.Errorf("row[0][0]: got %+v, want integer 1", r.Rows[0][0])
	}
	if r.Rows[0][1].Type != "text" || r.Rows[0][1].Value != "test" {
		t.Errorf("row[0][1]: got %+v, want text 'test'", r.Rows[0][1])
	}
}

func TestNullValues(t *testing.T) {
	srv, err := NewServer(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	doPipeline(t, srv.URL(), pipelineRequest{
		Requests: []pipelineEntry{
			{Type: "execute", Stmt: &stmtRequest{SQL: "CREATE TABLE t (a TEXT, b TEXT)"}},
			{Type: "execute", Stmt: &stmtRequest{
				SQL:  "INSERT INTO t VALUES (?, ?)",
				Args: []hValue{{Type: "text", Value: "hello"}, {Type: "null"}},
			}},
			{Type: "close"},
		},
	})

	result := doPipeline(t, srv.URL(), pipelineRequest{
		Requests: []pipelineEntry{
			{Type: "execute", Stmt: &stmtRequest{SQL: "SELECT a, b FROM t"}},
			{Type: "close"},
		},
	})

	row := getExecResult(t, result, 0).Rows[0]
	if row[0].Type != "text" || row[0].Value != "hello" {
		t.Errorf("col a: got %+v, want text 'hello'", row[0])
	}
	if row[1].Type != "null" {
		t.Errorf("col b: got %+v, want null", row[1])
	}
}

func TestEmptyStringValue(t *testing.T) {
	srv, err := NewServer(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	doPipeline(t, srv.URL(), pipelineRequest{
		Requests: []pipelineEntry{
			{Type: "execute", Stmt: &stmtRequest{SQL: "CREATE TABLE t (val TEXT)"}},
			{Type: "execute", Stmt: &stmtRequest{
				SQL:  "INSERT INTO t VALUES (?)",
				Args: []hValue{{Type: "text", Value: ""}},
			}},
			{Type: "close"},
		},
	})

	result := doPipeline(t, srv.URL(), pipelineRequest{
		Requests: []pipelineEntry{
			{Type: "execute", Stmt: &stmtRequest{SQL: "SELECT val FROM t"}},
			{Type: "close"},
		},
	})

	row := getExecResult(t, result, 0).Rows[0]
	if row[0].Type != "text" {
		t.Errorf("expected text type, got %s", row[0].Type)
	}

	// Verify the JSON output has "value":"" and not omitted
	respBytes := doPipelineRaw(t, srv.URL(), pipelineRequest{
		Requests: []pipelineEntry{
			{Type: "execute", Stmt: &stmtRequest{SQL: "SELECT val FROM t"}},
			{Type: "close"},
		},
	})

	if !bytes.Contains(respBytes, []byte(`"value":""`)) {
		t.Errorf("JSON should contain \"value\":\"\" for empty string, got: %s", string(respBytes))
	}
}

func TestNamedArgs(t *testing.T) {
	srv, err := NewServer(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	doPipeline(t, srv.URL(), pipelineRequest{
		Requests: []pipelineEntry{
			{Type: "execute", Stmt: &stmtRequest{SQL: "CREATE TABLE kv (key TEXT, value TEXT)"}},
			{Type: "execute", Stmt: &stmtRequest{
				SQL: "INSERT INTO kv (key, value) VALUES (:key, :value)",
				NamedArgs: []namedArg{
					{Name: "key", Value: hValue{Type: "text", Value: "name"}},
					{Name: "value", Value: hValue{Type: "text", Value: "David"}},
				},
			}},
			{Type: "close"},
		},
	})

	result := doPipeline(t, srv.URL(), pipelineRequest{
		Requests: []pipelineEntry{
			{Type: "execute", Stmt: &stmtRequest{
				SQL:  "SELECT value FROM kv WHERE key = ?",
				Args: []hValue{{Type: "text", Value: "name"}},
			}},
			{Type: "close"},
		},
	})

	r := getExecResult(t, result, 0)
	if len(r.Rows) != 1 {
		t.Fatal("expected 1 row")
	}
	if r.Rows[0][0].Value != "David" {
		t.Errorf("expected 'David', got %q", r.Rows[0][0].Value)
	}
}

func TestTransaction(t *testing.T) {
	srv, err := NewServer(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	doPipeline(t, srv.URL(), pipelineRequest{
		Requests: []pipelineEntry{
			{Type: "execute", Stmt: &stmtRequest{SQL: "CREATE TABLE counter (n INTEGER)"}},
			{Type: "execute", Stmt: &stmtRequest{SQL: "INSERT INTO counter VALUES (0)"}},
			{Type: "close"},
		},
	})

	result := doPipeline(t, srv.URL(), pipelineRequest{
		Requests: []pipelineEntry{
			{Type: "execute", Stmt: &stmtRequest{SQL: "BEGIN"}},
			{Type: "execute", Stmt: &stmtRequest{SQL: "UPDATE counter SET n = n + 1"}},
			{Type: "execute", Stmt: &stmtRequest{SQL: "UPDATE counter SET n = n + 1"}},
			{Type: "execute", Stmt: &stmtRequest{SQL: "COMMIT"}},
			{Type: "close"},
		},
	})
	assertAllOK(t, result, 5)

	result = doPipeline(t, srv.URL(), pipelineRequest{
		Requests: []pipelineEntry{
			{Type: "execute", Stmt: &stmtRequest{SQL: "SELECT n FROM counter"}},
			{Type: "close"},
		},
	})

	r := getExecResult(t, result, 0)
	if r.Rows[0][0].Value != "2" {
		t.Errorf("counter should be 2, got %s", r.Rows[0][0].Value)
	}
}

func TestPragmaTableInfo(t *testing.T) {
	srv, err := NewServer(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	doPipeline(t, srv.URL(), pipelineRequest{
		Requests: []pipelineEntry{
			{Type: "execute", Stmt: &stmtRequest{SQL: "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, age INTEGER, bio TEXT DEFAULT '')"}},
			{Type: "close"},
		},
	})

	result := doPipeline(t, srv.URL(), pipelineRequest{
		Requests: []pipelineEntry{
			{Type: "execute", Stmt: &stmtRequest{SQL: `PRAGMA table_info("users")`}},
			{Type: "close"},
		},
	})

	r := getExecResult(t, result, 0)
	if len(r.Rows) != 4 {
		t.Fatalf("expected 4 columns in PRAGMA table_info, got %d", len(r.Rows))
	}
	if len(r.Cols) != 6 {
		t.Fatalf("expected 6 cols from PRAGMA table_info, got %d", len(r.Cols))
	}

	if r.Rows[1][1].Value != "name" {
		t.Errorf("row 1 name should be 'name', got %q", r.Rows[1][1].Value)
	}

	// Check dflt_value for bio — should be text type (empty string default)
	bioDefault := r.Rows[3][4]
	if bioDefault.Type != "text" {
		t.Errorf("bio default type should be 'text', got %q", bioDefault.Type)
	}

	respBytes := doPipelineRaw(t, srv.URL(), pipelineRequest{
		Requests: []pipelineEntry{
			{Type: "execute", Stmt: &stmtRequest{SQL: `PRAGMA table_info("users")`}},
			{Type: "close"},
		},
	})
	t.Logf("PRAGMA table_info response: %s", string(respBytes))
}

func TestBatonPersistence(t *testing.T) {
	srv, err := NewServer(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	doPipeline(t, srv.URL(), pipelineRequest{
		Requests: []pipelineEntry{
			{Type: "execute", Stmt: &stmtRequest{SQL: "CREATE TABLE t (id INTEGER)"}},
			{Type: "close"},
		},
	})

	result := doPipeline(t, srv.URL(), pipelineRequest{
		Requests: []pipelineEntry{
			{Type: "execute", Stmt: &stmtRequest{SQL: "BEGIN"}},
			{Type: "execute", Stmt: &stmtRequest{SQL: "INSERT INTO t VALUES (1)"}},
		},
	})

	if result.Baton == nil {
		t.Fatal("expected baton for active transaction")
	}
	baton := *result.Baton

	result = doPipeline(t, srv.URL(), pipelineRequest{
		Baton: &baton,
		Requests: []pipelineEntry{
			{Type: "execute", Stmt: &stmtRequest{SQL: "INSERT INTO t VALUES (2)"}},
			{Type: "execute", Stmt: &stmtRequest{SQL: "COMMIT"}},
			{Type: "close"},
		},
	})
	assertAllOK(t, result, 3)

	result = doPipeline(t, srv.URL(), pipelineRequest{
		Requests: []pipelineEntry{
			{Type: "execute", Stmt: &stmtRequest{SQL: "SELECT COUNT(*) FROM t"}},
			{Type: "close"},
		},
	})
	r := getExecResult(t, result, 0)
	if r.Rows[0][0].Value != "2" {
		t.Errorf("expected 2 rows, got %s", r.Rows[0][0].Value)
	}
}

func TestAffectedRowCountAlwaysPresent(t *testing.T) {
	srv, err := NewServer(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	doPipeline(t, srv.URL(), pipelineRequest{
		Requests: []pipelineEntry{
			{Type: "execute", Stmt: &stmtRequest{SQL: "CREATE TABLE t (id INTEGER)"}},
			{Type: "close"},
		},
	})

	respBytes := doPipelineRaw(t, srv.URL(), pipelineRequest{
		Requests: []pipelineEntry{
			{Type: "execute", Stmt: &stmtRequest{SQL: "SELECT * FROM t"}},
			{Type: "close"},
		},
	})

	if !bytes.Contains(respBytes, []byte(`"affected_row_count"`)) {
		t.Errorf("affected_row_count missing from response: %s", string(respBytes))
	}
}

func TestFilePersistence(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "persist.db")

	srv1, err := NewServer(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	doPipeline(t, srv1.URL(), pipelineRequest{
		Requests: []pipelineEntry{
			{Type: "execute", Stmt: &stmtRequest{SQL: "CREATE TABLE t (val TEXT)"}},
			{Type: "execute", Stmt: &stmtRequest{
				SQL:  "INSERT INTO t VALUES (?)",
				Args: []hValue{{Type: "text", Value: "persisted"}},
			}},
			{Type: "close"},
		},
	})
	srv1.Close()

	srv2, err := NewServer(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer srv2.Close()

	result := doPipeline(t, srv2.URL(), pipelineRequest{
		Requests: []pipelineEntry{
			{Type: "execute", Stmt: &stmtRequest{SQL: "SELECT val FROM t"}},
			{Type: "close"},
		},
	})

	r := getExecResult(t, result, 0)
	if len(r.Rows) != 1 {
		t.Fatal("expected 1 row after persistence")
	}
	if r.Rows[0][0].Value != "persisted" {
		t.Errorf("expected 'persisted', got %q", r.Rows[0][0].Value)
	}
}

func TestBatchRequest(t *testing.T) {
	srv, err := NewServer(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	// Setup table
	doPipeline(t, srv.URL(), pipelineRequest{
		Requests: []pipelineEntry{
			{Type: "execute", Stmt: &stmtRequest{SQL: "CREATE TABLE t (id INTEGER PRIMARY KEY, name TEXT)"}},
			{Type: "close"},
		},
	})

	// Batch insert + select
	step0 := 0
	result := doPipeline(t, srv.URL(), pipelineRequest{
		Requests: []pipelineEntry{
			{Type: "batch", Batch: &batchRequest{
				Steps: []batchStep{
					{Stmt: stmtRequest{SQL: "INSERT INTO t (name) VALUES ('Alice')"}},
					{
						Stmt:      stmtRequest{SQL: "INSERT INTO t (name) VALUES ('Bob')"},
						Condition: &batchCondition{Type: "ok", Step: &step0},
					},
					{
						Stmt:      stmtRequest{SQL: "SELECT COUNT(*) FROM t"},
						Condition: &batchCondition{Type: "ok", Step: &step0},
					},
				},
			}},
			{Type: "close"},
		},
	})

	if len(result.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result.Results))
	}
	if result.Results[0].Type != "ok" {
		t.Fatalf("batch result should be ok, got %s", result.Results[0].Type)
	}

	// Decode batch response
	var batchResp batchExecResponse
	if err := json.Unmarshal(result.Results[0].Response, &batchResp); err != nil {
		t.Fatal(err)
	}

	if len(batchResp.Result.StepResults) != 3 {
		t.Fatalf("expected 3 step results, got %d", len(batchResp.Result.StepResults))
	}

	// Step 2 (SELECT COUNT) should have 1 row
	countResult := batchResp.Result.StepResults[2]
	if countResult == nil {
		t.Fatal("step 2 result should not be nil")
	}
	if len(countResult.Rows) != 1 {
		t.Fatalf("expected 1 row from COUNT, got %d", len(countResult.Rows))
	}
	if countResult.Rows[0][0].Value != "2" {
		t.Errorf("expected count=2, got %s", countResult.Rows[0][0].Value)
	}
}

// --- Helpers ---

func getExecResult(t *testing.T, resp pipelineResponse, i int) *execResult {
	t.Helper()
	if i >= len(resp.Results) {
		t.Fatalf("result index %d out of range (have %d)", i, len(resp.Results))
	}
	if resp.Results[i].Type != "ok" {
		t.Fatalf("result %d is not ok: %s %v", i, resp.Results[i].Type, resp.Results[i].Error)
	}
	var er execResponse
	if err := json.Unmarshal(resp.Results[i].Response, &er); err != nil {
		t.Fatalf("decode response %d: %v", i, err)
	}
	return er.Result
}

func assertAllOK(t *testing.T, resp pipelineResponse, n int) {
	t.Helper()
	if len(resp.Results) != n {
		t.Fatalf("expected %d results, got %d", n, len(resp.Results))
	}
	for i, r := range resp.Results {
		if r.Type != "ok" {
			t.Fatalf("result %d: type=%s error=%v", i, r.Type, r.Error)
		}
	}
}

func doPipeline(t *testing.T, url string, req pipelineRequest) pipelineResponse {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(url+"/v2/pipeline", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("pipeline returned %d", resp.StatusCode)
	}
	var result pipelineResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	return result
}

func doPipelineRaw(t *testing.T, url string, req pipelineRequest) []byte {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(url+"/v2/pipeline", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)
	return buf.Bytes()
}
