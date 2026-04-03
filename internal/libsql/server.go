// Package libsql provides an embedded HTTP server that speaks the LibSQL/Hrana
// pipeline protocol, backed by a local SQLite file via modernc.org/sqlite (pure Go).
//
// This bridges the gap between @libsql/client's HTTP mode running in QuickJS and
// local persistent storage. The JS code does:
//
//	new LibSQLStore({ url: "http://127.0.0.1:<port>" })
//
// This server handles: POST /v2/pipeline → execute SQL → return Hrana results.
//
// Why: @libsql/client's file: mode uses a native N-API addon that can't run in
// QuickJS. The HTTP mode uses the Hrana protocol over fetch — which works perfectly
// in QuickJS via our fetch polyfill. This server provides the HTTP endpoint backed
// by a real SQLite file.
package libsql

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

// Server is an embedded LibSQL HTTP bridge backed by a local SQLite file.
type Server struct {
	db       *sql.DB
	listener net.Listener
	srv      *http.Server
	url      string
	logger   func(format string, args ...any)

	// DB access lock — RWMutex allows concurrent readers, serializes writers
	dbMu sync.RWMutex
	// baton-based connections for transactions
	mu      sync.Mutex
	batons  map[string]*sql.Tx
	batonID int
	// SQL cache for store_sql/close_sql
	sqlCache map[int]string
}

// Option configures a Server.
type Option func(*Server)

// WithLogger sets a custom logger. Default is silent (no logging).
func WithLogger(fn func(format string, args ...any)) Option {
	return func(s *Server) { s.logger = fn }
}

// NewServer creates and starts an embedded LibSQL HTTP server.
// dbPath is the path to the SQLite file (created if it doesn't exist).
// Use ":memory:" for an in-memory database.
func NewServer(dbPath string, opts ...Option) (*Server, error) {
	if dbPath != ":memory:" {
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("libsql: creating db directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("libsql: opening sqlite: %w", err)
	}

	// Enable WAL mode + busy timeout for better concurrent access
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("libsql: setting WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("libsql: setting busy timeout: %w", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("libsql: listening: %w", err)
	}

	s := &Server{
		db:       db,
		listener: listener,
		url:      fmt.Sprintf("http://127.0.0.1:%d", listener.Addr().(*net.TCPAddr).Port),
		batons:   make(map[string]*sql.Tx),
		sqlCache: make(map[int]string),
		logger:   func(string, ...any) {}, // silent by default
	}

	for _, o := range opts {
		o(s)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v2/pipeline", s.handlePipeline)
	mux.HandleFunc("/v3/pipeline", s.handlePipeline)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("brainkit-libsql-bridge 0.1.0"))
	})

	s.srv = &http.Server{Handler: mux}

	go func() {
		if err := s.srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			s.logger("libsql bridge error: %v", err)
		}
	}()

	s.logger("LibSQL bridge listening on %s (db: %s)", s.url, dbPath)
	return s, nil
}

// URL returns the HTTP URL for connecting from JS.
func (s *Server) URL() string { return s.url }

// Close shuts down the server and closes the database.
func (s *Server) Close() error {
	s.mu.Lock()
	for _, tx := range s.batons {
		tx.Rollback()
	}
	s.batons = nil
	s.mu.Unlock()

	s.srv.Shutdown(context.Background())
	return s.db.Close()
}

// --- Hrana/Pipeline Protocol Types ---
//
// These match the Hrana v2/v3 pipeline protocol as consumed by @libsql/hrana-client.
// The client-side decoder (shared/json_decode.js) is strict:
//   - StmtResult.affected_row_count MUST be a number (not omitted)
//   - Col.name and Col.decltype use stringOpt (null OK, undefined NOT OK for name)
//   - Value.type MUST be a string
//   - Value.value MUST be present for text/integer types (even if empty string)
//   - Value.value MUST be absent for null type

type pipelineRequest struct {
	Baton    *string         `json:"baton"`
	Requests []pipelineEntry `json:"requests"`
}

type pipelineEntry struct {
	Type  string        `json:"type"`
	Stmt  *stmtRequest  `json:"stmt,omitempty"`
	Batch *batchRequest `json:"batch,omitempty"`
	SQL   *string       `json:"sql,omitempty"`   // for sequence
	SQLId *int          `json:"sql_id,omitempty"` // for store_sql, close_sql
}

type batchRequest struct {
	Steps []batchStep `json:"steps"`
}

type batchStep struct {
	Stmt      stmtRequest    `json:"stmt"`
	Condition *batchCondition `json:"condition,omitempty"`
}

type batchCondition struct {
	Type  string          `json:"type"`
	Step  *int            `json:"step,omitempty"`
	Cond  *batchCondition `json:"cond,omitempty"`
	Conds []batchCondition `json:"conds,omitempty"`
}

type stmtRequest struct {
	SQL       string     `json:"sql"`
	SQLId     *int       `json:"sql_id,omitempty"`
	Args      []hValue   `json:"args,omitempty"`
	NamedArgs []namedArg `json:"named_args,omitempty"`
	WantRows  *bool      `json:"want_rows,omitempty"`
}

type namedArg struct {
	Name  string `json:"name"`
	Value hValue `json:"value"`
}

// hValue represents a Hrana protocol value with custom JSON marshaling.
// The Hrana client is strict about field presence:
//   - type "null": only {"type":"null"} — no "value" field
//   - type "text": {"type":"text","value":"..."} — value MUST be present even for ""
//   - type "integer": {"type":"integer","value":"123"} — value as string
//   - type "float": {"type":"float","value":1.5} — value as number
//   - type "blob": {"type":"blob","base64":"..."} — base64 encoded
type hValue struct {
	Type     string
	Value    string // for text, integer
	FloatVal float64
	Base64   string // for blob
}

func (v hValue) MarshalJSON() ([]byte, error) {
	switch v.Type {
	case "null":
		return []byte(`{"type":"null"}`), nil
	case "text":
		// Value MUST always be present, even for empty string
		b, _ := json.Marshal(v.Value)
		return []byte(`{"type":"text","value":` + string(b) + `}`), nil
	case "integer":
		return []byte(`{"type":"integer","value":"` + v.Value + `"}`), nil
	case "float":
		return []byte(`{"type":"float","value":` + strconv.FormatFloat(v.FloatVal, 'g', -1, 64) + `}`), nil
	case "blob":
		return []byte(`{"type":"blob","base64":"` + v.Base64 + `"}`), nil
	default:
		return []byte(`{"type":"null"}`), nil
	}
}

func (v *hValue) UnmarshalJSON(data []byte) error {
	var raw struct {
		Type   string          `json:"type"`
		Value  json.RawMessage `json:"value,omitempty"`
		Base64 string          `json:"base64,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	v.Type = raw.Type
	v.Base64 = raw.Base64
	if raw.Value != nil {
		// Could be string or number depending on type
		var s string
		if err := json.Unmarshal(raw.Value, &s); err == nil {
			v.Value = s
		} else {
			var f float64
			if err := json.Unmarshal(raw.Value, &f); err == nil {
				v.FloatVal = f
				v.Value = string(raw.Value)
			} else {
				v.Value = string(raw.Value)
			}
		}
	}
	return nil
}

type pipelineResponse struct {
	Baton   *string          `json:"baton"`
	BaseURL *string          `json:"base_url"`
	Results []pipelineResult `json:"results"`
}

type pipelineResult struct {
	Type     string          `json:"type"`
	Response json.RawMessage `json:"response,omitempty"`
	Error    *execError      `json:"error,omitempty"`
}

func okResult(resp any) pipelineResult {
	data, _ := json.Marshal(resp)
	return pipelineResult{Type: "ok", Response: data}
}

func errResult(msg string) pipelineResult {
	return pipelineResult{Type: "error", Error: &execError{Message: msg}}
}

type execResponse struct {
	Type          string       `json:"type"`
	Result        *execResult  `json:"result,omitempty"`
	IsAutocommit  *bool        `json:"is_autocommit,omitempty"`
}

type batchResponse struct {
	StepResults []*execResult `json:"step_results"`
	StepErrors  []*execError  `json:"step_errors"`
}

// batchExecResponse is used for batch type responses — it needs a different result shape.
type batchExecResponse struct {
	Type   string         `json:"type"`
	Result *batchResponse `json:"result"`
}

type execResult struct {
	Cols             []colDef   `json:"cols"`
	Rows             [][]hValue `json:"rows"`
	AffectedRowCount int64      `json:"affected_row_count"`
	LastInsertRowid  *string    `json:"last_insert_rowid"`
	ReplicationIndex *int64     `json:"replication_index"`
}

type colDef struct {
	Name     *string `json:"name"`
	Decltype *string `json:"decltype"`
}

type execError struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// --- Pipeline Classification ---

// pipelineNeedsWrite returns true if the pipeline contains any write operations.
// Read-only pipelines can run concurrently under RWMutex.
func (s *Server) pipelineNeedsWrite(req *pipelineRequest) bool {
	if req.Baton != nil && *req.Baton != "" {
		return true
	}
	for _, entry := range req.Requests {
		switch entry.Type {
		case "batch", "sequence":
			return true // may contain writes
		case "execute":
			if entry.Stmt == nil {
				continue
			}
			sqlText := entry.Stmt.SQL
			if sqlText == "" && entry.Stmt.SQLId != nil {
				s.mu.Lock()
				if cached, ok := s.sqlCache[*entry.Stmt.SQLId]; ok {
					sqlText = cached
				}
				s.mu.Unlock()
			}
			if sqlText != "" && !isQuery(strings.TrimSpace(sqlText)) {
				return true
			}
		case "store_sql", "close_sql", "close":
			return true // metadata operations
		}
	}
	return false
}

// --- Pipeline Handler ---

func (s *Server) handlePipeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read error: "+err.Error(), http.StatusBadRequest)
		return
	}

	var req pipelineRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}

	if s.pipelineNeedsWrite(&req) {
		s.dbMu.Lock()
		defer s.dbMu.Unlock()
	} else {
		s.dbMu.RLock()
		defer s.dbMu.RUnlock()
	}

	// Resolve baton (transaction context)
	var tx *sql.Tx
	if req.Baton != nil && *req.Baton != "" {
		s.mu.Lock()
		tx = s.batons[*req.Baton]
		s.mu.Unlock()
	}

	resp := pipelineResponse{
		Results: make([]pipelineResult, 0, len(req.Requests)),
	}

	for _, entry := range req.Requests {
		switch entry.Type {
		case "execute":
			if entry.Stmt == nil {
				resp.Results = append(resp.Results, errResult("missing stmt"))
				continue
			}
			result, newTx, err := s.executeStmt(entry.Stmt, tx)
			if err != nil {
				resp.Results = append(resp.Results, errResult(err.Error()))
			} else {
				resp.Results = append(resp.Results, okResult(execResponse{Type: "execute", Result: result}))
			}
			if newTx != nil {
				tx = newTx
			}

		case "batch":
			if entry.Batch == nil {
				resp.Results = append(resp.Results, errResult("missing batch"))
				continue
			}
			batchResp := s.executeBatch(entry.Batch, tx)
			resp.Results = append(resp.Results, okResult(batchExecResponse{Type: "batch", Result: batchResp}))

		case "sequence":
			// sequence executes multiple semicolon-separated statements
			if entry.SQL != nil {
				_, err := s.db.Exec(*entry.SQL)
				if err != nil {
					resp.Results = append(resp.Results, errResult(err.Error()))
				} else {
					resp.Results = append(resp.Results, okResult(execResponse{Type: "sequence"}))
				}
			} else {
				resp.Results = append(resp.Results, okResult(execResponse{Type: "sequence"}))
			}

		case "get_autocommit":
			isAuto := tx == nil
			resp.Results = append(resp.Results, okResult(execResponse{Type: "get_autocommit", IsAutocommit: &isAuto}))

		case "store_sql":
			if entry.SQLId != nil && entry.SQL != nil {
				s.sqlCache[*entry.SQLId] = *entry.SQL
			}
			resp.Results = append(resp.Results, okResult(execResponse{Type: "store_sql"}))

		case "close_sql":
			if entry.SQLId != nil {
				delete(s.sqlCache, *entry.SQLId)
			}
			resp.Results = append(resp.Results, okResult(execResponse{Type: "close_sql"}))

		case "close":
			if tx != nil {
				s.mu.Lock()
				for k, v := range s.batons {
					if v == tx {
						delete(s.batons, k)
					}
				}
				s.mu.Unlock()
				tx = nil
			}
			resp.Results = append(resp.Results, okResult(execResponse{Type: "close"}))

		default:
			resp.Results = append(resp.Results, errResult("unsupported request type: "+entry.Type))
		}
	}

	// If there's an active transaction, create/update baton
	if tx != nil {
		s.mu.Lock()
		s.batonID++
		baton := fmt.Sprintf("baton-%d", s.batonID)
		s.batons[baton] = tx
		s.mu.Unlock()
		resp.Baton = &baton
	}

	w.Header().Set("Content-Type", "application/json")
	respBytes, _ := json.Marshal(resp)
	// Log first SQL in the pipeline for debugging
	for _, entry := range req.Requests {
		if entry.Stmt != nil {
			sql := entry.Stmt.SQL
			if len(sql) > 100 {
				sql = sql[:100] + "..."
			}
			s.logger("[pipeline] %s → %d bytes", sql, len(respBytes))
			break
		}
		if entry.Batch != nil && len(entry.Batch.Steps) > 0 {
			sql := entry.Batch.Steps[0].Stmt.SQL
			if len(sql) > 100 {
				sql = sql[:100] + "..."
			}
			s.logger("[pipeline:batch] %s (+ %d more steps)", sql, len(entry.Batch.Steps)-1)
			break
		}
	}
	w.Write(respBytes)
}

// --- SQL Execution ---

func (s *Server) executeStmt(stmt *stmtRequest, tx *sql.Tx) (*execResult, *sql.Tx, error) {
	// Resolve sql_id → cached SQL if no inline sql
	if stmt.SQL == "" && stmt.SQLId != nil {
		if cached, ok := s.sqlCache[*stmt.SQLId]; ok {
			stmt.SQL = cached
		} else {
			return nil, nil, fmt.Errorf("sql_id %d not found in cache", *stmt.SQLId)
		}
	}

	args, err := buildArgs(stmt)
	if err != nil {
		return nil, nil, err
	}

	trimmed := strings.TrimSpace(stmt.SQL)
	if len(trimmed) >= 5 {
		switch {
		case isCmd(trimmed, "BEGIN"):
			newTx, err := s.db.Begin()
			if err != nil {
				return nil, nil, err
			}
			return emptyResult(), newTx, nil

		case isCmd(trimmed, "COMMIT") || isCmd(trimmed, "END"):
			if tx != nil {
				if err := tx.Commit(); err != nil {
					return nil, nil, err
				}
			}
			return emptyResult(), nil, nil

		case isCmd(trimmed, "ROLLBACK"):
			if tx != nil {
				tx.Rollback()
			}
			return emptyResult(), nil, nil
		}
	}

	// Choose execution context
	var execer interface {
		ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
		QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	}
	if tx != nil {
		execer = tx
	} else {
		execer = s.db
	}

	if isQuery(trimmed) {
		return s.executeQuery(execer, stmt.SQL, args)
	}

	return s.executeExec(execer, stmt.SQL, args)
}

func (s *Server) executeQuery(execer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}, query string, args []any) (*execResult, *sql.Tx, error) {
	rows, err := execer.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}
	colTypes, _ := rows.ColumnTypes()

	colDefs := make([]colDef, len(cols))
	for i, c := range cols {
		name := c
		colDefs[i] = colDef{Name: &name}
		if i < len(colTypes) {
			dt := colTypes[i].DatabaseTypeName()
			if dt != "" {
				colDefs[i].Decltype = &dt
			}
		}
	}

	var resultRows [][]hValue
	for rows.Next() {
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, nil, err
		}

		row := make([]hValue, len(cols))
		for i, v := range values {
			row[i] = goToHValue(v)
		}
		resultRows = append(resultRows, row)
	}

	if resultRows == nil {
		resultRows = [][]hValue{}
	}

	return &execResult{
		Cols:             colDefs,
		Rows:             resultRows,
		AffectedRowCount: 0,
	}, nil, nil
}

func (s *Server) executeExec(execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}, query string, args []any) (*execResult, *sql.Tx, error) {
	result, err := execer.ExecContext(context.Background(), query, args...)
	if err != nil {
		return nil, nil, err
	}

	res := &execResult{
		Cols: []colDef{},
		Rows: [][]hValue{},
	}

	// sql.Result can be a non-nil interface wrapping a nil pointer (modernc.org/sqlite
	// does this for some DDL statements). Calling methods on it panics. Guard with recover.
	func() {
		defer func() { recover() }()
		res.AffectedRowCount, _ = result.RowsAffected()
		lastID, _ := result.LastInsertId()
		s := fmt.Sprintf("%d", lastID)
		res.LastInsertRowid = &s
	}()

	return res, nil, nil
}

func (s *Server) executeBatch(batch *batchRequest, tx *sql.Tx) *batchResponse {
	n := len(batch.Steps)
	stepResults := make([]*execResult, n)
	stepErrors := make([]*execError, n)

	// Batch steps may contain BEGIN/COMMIT — track the batch-local transaction.
	batchTx := tx
	for i, step := range batch.Steps {
		// Evaluate condition
		if step.Condition != nil && !s.evalCondition(step.Condition, stepResults, stepErrors) {
			continue
		}

		result, newTx, err := s.executeStmt(&step.Stmt, batchTx)
		if err != nil {
			stepErrors[i] = &execError{Message: err.Error()}
		} else {
			stepResults[i] = result
		}
		if newTx != nil {
			batchTx = newTx
		}
		// If COMMIT/ROLLBACK was executed, the tx is done
		trimmed := strings.TrimSpace(step.Stmt.SQL)
		if isCmd(trimmed, "COMMIT") || isCmd(trimmed, "END") || isCmd(trimmed, "ROLLBACK") {
			batchTx = tx // revert to the pipeline-level tx (or nil)
		}
	}

	return &batchResponse{StepResults: stepResults, StepErrors: stepErrors}
}

func (s *Server) evalCondition(cond *batchCondition, results []*execResult, errors []*execError) bool {
	switch cond.Type {
	case "ok":
		if cond.Step != nil && *cond.Step < len(results) {
			return results[*cond.Step] != nil
		}
		return false
	case "error":
		if cond.Step != nil && *cond.Step < len(errors) {
			return errors[*cond.Step] != nil
		}
		return false
	case "not":
		if cond.Cond != nil {
			return !s.evalCondition(cond.Cond, results, errors)
		}
		return true
	case "and":
		for i := range cond.Conds {
			if !s.evalCondition(&cond.Conds[i], results, errors) {
				return false
			}
		}
		return true
	case "or":
		for i := range cond.Conds {
			if s.evalCondition(&cond.Conds[i], results, errors) {
				return true
			}
		}
		return false
	case "is_autocommit":
		return true // we're always autocommit unless in a baton transaction
	default:
		return true
	}
}

func emptyResult() *execResult {
	return &execResult{
		Cols:             []colDef{},
		Rows:             [][]hValue{},
		AffectedRowCount: 0,
	}
}

// --- Value Conversion ---

func buildArgs(stmt *stmtRequest) ([]any, error) {
	var args []any
	for _, a := range stmt.Args {
		args = append(args, hValueToGo(a))
	}
	for _, na := range stmt.NamedArgs {
		args = append(args, sql.Named(na.Name, hValueToGo(na.Value)))
	}
	return args, nil
}

func hValueToGo(v hValue) any {
	switch v.Type {
	case "null":
		return nil
	case "integer":
		i, _ := strconv.ParseInt(v.Value, 10, 64)
		return i
	case "float":
		f, _ := strconv.ParseFloat(v.Value, 64)
		return f
	case "text":
		return v.Value
	case "blob":
		if v.Base64 != "" {
			data, _ := base64.StdEncoding.DecodeString(v.Base64)
			return data
		}
		return []byte(v.Value)
	default:
		return v.Value
	}
}

func goToHValue(v any) hValue {
	if v == nil {
		return hValue{Type: "null"}
	}
	switch val := v.(type) {
	case int64:
		return hValue{Type: "integer", Value: fmt.Sprintf("%d", val)}
	case int:
		return hValue{Type: "integer", Value: fmt.Sprintf("%d", val)}
	case float64:
		return hValue{Type: "float", FloatVal: val}
	case string:
		return hValue{Type: "text", Value: val}
	case []byte:
		return hValue{Type: "blob", Base64: base64.StdEncoding.EncodeToString(val)}
	case bool:
		if val {
			return hValue{Type: "integer", Value: "1"}
		}
		return hValue{Type: "integer", Value: "0"}
	default:
		return hValue{Type: "text", Value: fmt.Sprintf("%v", val)}
	}
}

// --- SQL Classification ---

func isQuery(sql string) bool {
	for i := 0; i < len(sql); i++ {
		c := sql[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			continue
		}
		upper := strings.ToUpper(sql[i:])
		return strings.HasPrefix(upper, "SELECT") ||
			strings.HasPrefix(upper, "PRAGMA") ||
			strings.HasPrefix(upper, "EXPLAIN") ||
			strings.HasPrefix(upper, "WITH")
	}
	return false
}

func isCmd(s, cmd string) bool {
	if len(s) < len(cmd) {
		return false
	}
	if !strings.EqualFold(s[:len(cmd)], cmd) {
		return false
	}
	if len(s) == len(cmd) {
		return true
	}
	next := s[len(cmd)]
	return next == ' ' || next == '\t' || next == '\n' || next == '\r' || next == ';' || next == '('
}
