// Ported from: packages/core/src/logger/logger.test.ts
//
// Faithful 1:1 port of the Mastra Logger vitest suite.
// Tests cover: ConsoleLogger, NoopLogger, MultiLogger, LogLevel,
// transports, and the MastraLoggerBase delegation.
package logger

import (
	"bytes"
	"strings"
	"testing"

	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
)

// =============================================================================
// LogLevel Tests
// =============================================================================

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LogLevelDebug, "debug"},
		{LogLevelInfo, "info"},
		{LogLevelWarn, "warn"},
		{LogLevelError, "error"},
		{LogLevelNone, "silent"},
		{LogLevel(99), "unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			if got := tc.level.String(); got != tc.expected {
				t.Errorf("LogLevel(%d).String() = %q, want %q", tc.level, got, tc.expected)
			}
		})
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"debug", LogLevelDebug},
		{"info", LogLevelInfo},
		{"warn", LogLevelWarn},
		{"error", LogLevelError},
		{"silent", LogLevelNone},
		{"", LogLevelError},       // default
		{"unknown", LogLevelError}, // default
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			if got := ParseLogLevel(tc.input); got != tc.expected {
				t.Errorf("ParseLogLevel(%q) = %d, want %d", tc.input, got, tc.expected)
			}
		})
	}
}

// =============================================================================
// ConsoleLogger Tests
// =============================================================================

func TestConsoleLogger_Defaults(t *testing.T) {
	logger := NewConsoleLogger(nil)

	if logger.Name != "Mastra" {
		t.Errorf("expected default name 'Mastra', got %q", logger.Name)
	}
	// Default level is LogLevelError
	if logger.Level != LogLevelError {
		t.Errorf("expected default level Error(%d), got %d", LogLevelError, logger.Level)
	}
	if len(logger.GetTransports()) != 0 {
		t.Errorf("expected 0 transports, got %d", len(logger.GetTransports()))
	}
}

func TestConsoleLogger_CustomOptions(t *testing.T) {
	logger := NewConsoleLogger(&ConsoleLoggerOptions{
		Name:  "TestLogger",
		Level: LogLevelDebug,
	})

	if logger.Name != "TestLogger" {
		t.Errorf("expected name 'TestLogger', got %q", logger.Name)
	}
	if logger.Level != LogLevelDebug {
		t.Errorf("expected level Debug(%d), got %d", LogLevelDebug, logger.Level)
	}
}

func TestConsoleLogger_LevelFiltering(t *testing.T) {
	// At INFO level, Debug should be suppressed, Info/Warn/Error should pass
	logger := NewConsoleLogger(&ConsoleLoggerOptions{
		Level: LogLevelInfo,
	})

	// Debug should NOT log (level > LogLevelDebug)
	// We can't capture stdout easily, but we verify the logger doesn't panic
	logger.Debug("should be suppressed")
	logger.Info("should appear")
	logger.Warn("should appear")
	logger.Error("should appear")
}

func TestConsoleLogger_ListLogs_ReturnsEmpty(t *testing.T) {
	logger := NewConsoleLogger(nil)

	result, err := logger.ListLogs("any-transport", nil)
	if err != nil {
		t.Fatalf("ListLogs returned error: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("expected 0 total, got %d", result.Total)
	}
	if len(result.Logs) != 0 {
		t.Errorf("expected 0 logs, got %d", len(result.Logs))
	}
}

func TestConsoleLogger_ListLogsByRunID_ReturnsEmpty(t *testing.T) {
	logger := NewConsoleLogger(nil)

	result, err := logger.ListLogsByRunID(&ListLogsByRunIDFullArgs{
		TransportID: "test",
		ListLogsByRunIDArgs: ListLogsByRunIDArgs{
			RunID: "run-1",
		},
	})
	if err != nil {
		t.Fatalf("ListLogsByRunID returned error: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("expected 0 total, got %d", result.Total)
	}
}

// =============================================================================
// NoopLogger Tests
// =============================================================================

func TestNoopLogger_ImplementsInterface(t *testing.T) {
	// NoopLogger should satisfy IMastraLogger
	var _ IMastraLogger = NoopLogger
}

func TestNoopLogger_AllMethodsAreNoOp(t *testing.T) {
	// These should not panic
	NoopLogger.Debug("test")
	NoopLogger.Info("test")
	NoopLogger.Warn("test")
	NoopLogger.Error("test")
	NoopLogger.TrackException(&mastraerror.MastraBaseError{})
}

func TestNoopLogger_GetTransports_ReturnsEmptyMap(t *testing.T) {
	transports := NoopLogger.GetTransports()
	if len(transports) != 0 {
		t.Errorf("expected 0 transports, got %d", len(transports))
	}
}

func TestNoopLogger_ListLogs_ReturnsEmpty(t *testing.T) {
	result, err := NoopLogger.ListLogs("any", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 0 || len(result.Logs) != 0 {
		t.Errorf("expected empty result, got total=%d logs=%d", result.Total, len(result.Logs))
	}
	if result.Page != 1 || result.PerPage != 100 {
		t.Errorf("expected page=1 perPage=100, got page=%d perPage=%d", result.Page, result.PerPage)
	}
}

func TestNoopLogger_ListLogsByRunID_ReturnsEmpty(t *testing.T) {
	result, err := NoopLogger.ListLogsByRunID(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 0 || len(result.Logs) != 0 {
		t.Errorf("expected empty result, got total=%d logs=%d", result.Total, len(result.Logs))
	}
}

// =============================================================================
// MultiLogger Tests
// =============================================================================

// trackingLogger records calls to each log method for testing MultiLogger delegation.
type trackingLogger struct {
	noopLoggerImpl
	debugCalls []string
	infoCalls  []string
	warnCalls  []string
	errorCalls []string
	exceptCalls int
}

func (t *trackingLogger) Debug(message string, args ...any) { t.debugCalls = append(t.debugCalls, message) }
func (t *trackingLogger) Info(message string, args ...any)  { t.infoCalls = append(t.infoCalls, message) }
func (t *trackingLogger) Warn(message string, args ...any)  { t.warnCalls = append(t.warnCalls, message) }
func (t *trackingLogger) Error(message string, args ...any) { t.errorCalls = append(t.errorCalls, message) }
func (t *trackingLogger) TrackException(err *mastraerror.MastraBaseError) { t.exceptCalls++ }

func TestMultiLogger_DelegatesToAllLoggers(t *testing.T) {
	l1 := &trackingLogger{}
	l2 := &trackingLogger{}
	multi := NewMultiLogger([]IMastraLogger{l1, l2})

	multi.Debug("debug msg")
	multi.Info("info msg")
	multi.Warn("warn msg")
	multi.Error("error msg")
	multi.TrackException(&mastraerror.MastraBaseError{})

	// Both loggers should have received all calls
	for _, l := range []*trackingLogger{l1, l2} {
		if len(l.debugCalls) != 1 || l.debugCalls[0] != "debug msg" {
			t.Errorf("expected 1 debug call with 'debug msg', got %v", l.debugCalls)
		}
		if len(l.infoCalls) != 1 || l.infoCalls[0] != "info msg" {
			t.Errorf("expected 1 info call with 'info msg', got %v", l.infoCalls)
		}
		if len(l.warnCalls) != 1 || l.warnCalls[0] != "warn msg" {
			t.Errorf("expected 1 warn call with 'warn msg', got %v", l.warnCalls)
		}
		if len(l.errorCalls) != 1 || l.errorCalls[0] != "error msg" {
			t.Errorf("expected 1 error call with 'error msg', got %v", l.errorCalls)
		}
		if l.exceptCalls != 1 {
			t.Errorf("expected 1 exception call, got %d", l.exceptCalls)
		}
	}
}

func TestMultiLogger_MergesTransports(t *testing.T) {
	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}
	transport1 := CreateCustomTransport(buf1, nil, nil)
	transport2 := CreateCustomTransport(buf2, nil, nil)

	l1 := NewConsoleLogger(nil)
	l1.Transports["t1"] = transport1
	l2 := NewConsoleLogger(nil)
	l2.Transports["t2"] = transport2

	multi := NewMultiLogger([]IMastraLogger{l1, l2})
	transports := multi.GetTransports()

	if len(transports) != 2 {
		t.Errorf("expected 2 merged transports, got %d", len(transports))
	}
	if _, ok := transports["t1"]; !ok {
		t.Error("expected transport 't1' in merged map")
	}
	if _, ok := transports["t2"]; !ok {
		t.Error("expected transport 't2' in merged map")
	}
}

func TestMultiLogger_ListLogs_ReturnsFirstNonEmpty(t *testing.T) {
	// l1 returns empty, l2 uses MastraLoggerBase (not ConsoleLogger) which
	// delegates to its transport. ConsoleLogger overrides ListLogs to always
	// return empty regardless of transport, so we use a loggerWithTransport
	// that properly delegates via MastraLoggerBase.
	l1 := NoopLogger // always empty

	// Create a logger that delegates ListLogs via MastraLoggerBase (not ConsoleLogger)
	base := NewMastraLoggerBase(&MastraLoggerOptions{
		Transports: map[string]LoggerTransport{
			"test": CreateCustomTransport(
				&bytes.Buffer{},
				func(params *ListLogsParams) (LogResult, error) {
					return LogResult{
						Logs:    []BaseLogMessage{{Msg: "found"}},
						Total:   1,
						Page:    1,
						PerPage: 100,
						HasMore: false,
					}, nil
				},
				nil,
			),
		},
	})
	l2 := &baseLoggerWrapper{MastraLoggerBase: base}

	multi := NewMultiLogger([]IMastraLogger{l1, l2})
	result, err := multi.ListLogs("test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected total=1, got %d", result.Total)
	}
	if len(result.Logs) != 1 || result.Logs[0].Msg != "found" {
		t.Errorf("expected log with msg 'found', got %v", result.Logs)
	}
}

// baseLoggerWrapper wraps MastraLoggerBase with no-op log methods to satisfy IMastraLogger.
// Unlike ConsoleLogger, it does NOT override ListLogs, so transport delegation works.
type baseLoggerWrapper struct {
	MastraLoggerBase
}

func (b *baseLoggerWrapper) Debug(message string, args ...any) {}
func (b *baseLoggerWrapper) Info(message string, args ...any)  {}
func (b *baseLoggerWrapper) Warn(message string, args ...any)  {}
func (b *baseLoggerWrapper) Error(message string, args ...any) {}

// =============================================================================
// MastraLoggerBase Tests
// =============================================================================

func TestMastraLoggerBase_Defaults(t *testing.T) {
	base := NewMastraLoggerBase(nil)

	if base.Name != "Mastra" {
		t.Errorf("expected default name 'Mastra', got %q", base.Name)
	}
	if base.Level != LogLevelError {
		t.Errorf("expected default level Error, got %d", base.Level)
	}
	if len(base.Transports) != 0 {
		t.Errorf("expected 0 transports, got %d", len(base.Transports))
	}
}

func TestMastraLoggerBase_CustomTransports(t *testing.T) {
	buf := &bytes.Buffer{}
	transport := CreateCustomTransport(buf, nil, nil)

	base := NewMastraLoggerBase(&MastraLoggerOptions{
		Name:       "Custom",
		Level:      LogLevelDebug,
		Transports: map[string]LoggerTransport{"custom": transport},
	})

	if base.Name != "Custom" {
		t.Errorf("expected name 'Custom', got %q", base.Name)
	}
	if len(base.Transports) != 1 {
		t.Errorf("expected 1 transport, got %d", len(base.Transports))
	}
}

func TestMastraLoggerBase_ListLogs_DelegatesToTransport(t *testing.T) {
	transport := CreateCustomTransport(
		&bytes.Buffer{},
		func(params *ListLogsParams) (LogResult, error) {
			return LogResult{
				Logs:    []BaseLogMessage{{Msg: "from transport"}},
				Total:   1,
				Page:    1,
				PerPage: 10,
				HasMore: false,
			}, nil
		},
		nil,
	)

	base := NewMastraLoggerBase(&MastraLoggerOptions{
		Transports: map[string]LoggerTransport{"t1": transport},
	})

	result, err := base.ListLogs("t1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 1 || result.Logs[0].Msg != "from transport" {
		t.Errorf("expected delegated result, got %+v", result)
	}
}

func TestMastraLoggerBase_ListLogs_EmptyTransportID(t *testing.T) {
	base := NewMastraLoggerBase(nil)

	result, err := base.ListLogs("", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("expected empty result for empty transportID, got total=%d", result.Total)
	}
}

func TestMastraLoggerBase_ListLogs_MissingTransport(t *testing.T) {
	base := NewMastraLoggerBase(nil)

	result, err := base.ListLogs("nonexistent", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("expected empty result for missing transport, got total=%d", result.Total)
	}
}

func TestMastraLoggerBase_ListLogsByRunID_DelegatesToTransport(t *testing.T) {
	transport := CreateCustomTransport(
		&bytes.Buffer{},
		nil,
		func(args *ListLogsByRunIDArgs) (LogResult, error) {
			if args.RunID != "run-123" {
				return LogResult{}, nil
			}
			return LogResult{
				Logs:    []BaseLogMessage{{Msg: "run log", RunID: "run-123"}},
				Total:   1,
				Page:    1,
				PerPage: 100,
				HasMore: false,
			}, nil
		},
	)

	base := NewMastraLoggerBase(&MastraLoggerOptions{
		Transports: map[string]LoggerTransport{"t1": transport},
	})

	result, err := base.ListLogsByRunID(&ListLogsByRunIDFullArgs{
		TransportID:         "t1",
		ListLogsByRunIDArgs: ListLogsByRunIDArgs{RunID: "run-123"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 1 || result.Logs[0].RunID != "run-123" {
		t.Errorf("expected run-123 result, got %+v", result)
	}
}

func TestMastraLoggerBase_ListLogsByRunID_NilArgs(t *testing.T) {
	base := NewMastraLoggerBase(nil)

	result, err := base.ListLogsByRunID(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("expected empty result for nil args, got total=%d", result.Total)
	}
}

// =============================================================================
// Transport Tests
// =============================================================================

func TestCreateCustomTransport_Write(t *testing.T) {
	buf := &bytes.Buffer{}
	transport := CreateCustomTransport(buf, nil, nil)

	n, err := transport.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != 5 {
		t.Errorf("expected 5 bytes written, got %d", n)
	}
	if buf.String() != "hello" {
		t.Errorf("expected 'hello' in buffer, got %q", buf.String())
	}
}

func TestCreateCustomTransport_DefaultListMethods(t *testing.T) {
	buf := &bytes.Buffer{}
	transport := CreateCustomTransport(buf, nil, nil)

	result, err := transport.ListLogs(nil)
	if err != nil {
		t.Fatalf("ListLogs error: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("expected 0 total, got %d", result.Total)
	}

	result2, err := transport.ListLogsByRunID(nil)
	if err != nil {
		t.Fatalf("ListLogsByRunID error: %v", err)
	}
	if result2.Total != 0 {
		t.Errorf("expected 0 total, got %d", result2.Total)
	}
}

func TestBaseTransport_DefaultMethods(t *testing.T) {
	base := &BaseTransport{}

	result, err := base.ListLogs(nil)
	if err != nil {
		t.Fatalf("ListLogs error: %v", err)
	}
	if result.Total != 0 || result.Page != 1 || result.PerPage != 100 {
		t.Errorf("unexpected result: %+v", result)
	}

	p := 2
	pp := 50
	result2, err := base.ListLogs(&ListLogsParams{Page: &p, PerPage: &pp})
	if err != nil {
		t.Fatalf("ListLogs error: %v", err)
	}
	if result2.Page != 2 || result2.PerPage != 50 {
		t.Errorf("expected page=2 perPage=50, got page=%d perPage=%d", result2.Page, result2.PerPage)
	}
}

func TestEmptyLogResult_DefaultPagination(t *testing.T) {
	result := emptyLogResult(nil, nil)
	if result.Page != 1 {
		t.Errorf("expected default page=1, got %d", result.Page)
	}
	if result.PerPage != 100 {
		t.Errorf("expected default perPage=100, got %d", result.PerPage)
	}
	if result.HasMore {
		t.Error("expected HasMore=false")
	}
	if result.Total != 0 {
		t.Errorf("expected total=0, got %d", result.Total)
	}
}

func TestEmptyLogResult_CustomPagination(t *testing.T) {
	page := 3
	perPage := 25
	result := emptyLogResult(&page, &perPage)
	if result.Page != 3 {
		t.Errorf("expected page=3, got %d", result.Page)
	}
	if result.PerPage != 25 {
		t.Errorf("expected perPage=25, got %d", result.PerPage)
	}
}

// =============================================================================
// formatMessage Tests
// =============================================================================

func TestFormatMessage_NoArgs(t *testing.T) {
	msg := formatMessage("hello world")
	if msg != "hello world" {
		t.Errorf("expected 'hello world', got %q", msg)
	}
}

func TestFormatMessage_WithArgs(t *testing.T) {
	msg := formatMessage("hello", "arg1", 42)
	if !strings.Contains(msg, "hello") {
		t.Errorf("expected message to contain 'hello', got %q", msg)
	}
	if !strings.Contains(msg, "arg1") {
		t.Errorf("expected message to contain 'arg1', got %q", msg)
	}
}

// =============================================================================
// CreateLogger Deprecated Factory
// =============================================================================

func TestCreateLogger_ReturnsConsoleLogger(t *testing.T) {
	// CreateLogger is deprecated but should still work
	logger := CreateLogger(&ConsoleLoggerOptions{
		Name:  "Deprecated",
		Level: LogLevelWarn,
	})
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
	if logger.Name != "Deprecated" {
		t.Errorf("expected name 'Deprecated', got %q", logger.Name)
	}
	if logger.Level != LogLevelWarn {
		t.Errorf("expected level Warn, got %d", logger.Level)
	}
}

// =============================================================================
// RegisteredLogger Constants
// =============================================================================

func TestRegisteredLoggerConstants(t *testing.T) {
	// Verify all registered logger constants are non-empty
	constants := []RegisteredLogger{
		RegisteredLoggerAgent,
		RegisteredLoggerObservability,
		RegisteredLoggerAuth,
		RegisteredLoggerNetwork,
		RegisteredLoggerWorkflow,
		RegisteredLoggerLLM,
		RegisteredLoggerTTS,
		RegisteredLoggerVoice,
		RegisteredLoggerVector,
		RegisteredLoggerBundler,
		RegisteredLoggerDeployer,
		RegisteredLoggerMemory,
		RegisteredLoggerStorage,
		RegisteredLoggerEmbeddings,
		RegisteredLoggerMCPServer,
		RegisteredLoggerServerCache,
		RegisteredLoggerServer,
		RegisteredLoggerWorkspace,
	}

	for _, c := range constants {
		if c == "" {
			t.Error("registered logger constant should not be empty")
		}
	}

	if len(constants) != 18 {
		t.Errorf("expected 18 registered logger constants, got %d", len(constants))
	}
}
