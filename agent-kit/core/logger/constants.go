// Ported from: packages/core/src/logger/constants.ts
package logger

// RegisteredLogger identifies a named logger category within the system.
type RegisteredLogger string

const (
	RegisteredLoggerAgent       RegisteredLogger = "AGENT"
	RegisteredLoggerObservability RegisteredLogger = "OBSERVABILITY"
	RegisteredLoggerAuth        RegisteredLogger = "AUTH"
	RegisteredLoggerNetwork     RegisteredLogger = "NETWORK"
	RegisteredLoggerWorkflow    RegisteredLogger = "WORKFLOW"
	RegisteredLoggerLLM         RegisteredLogger = "LLM"
	RegisteredLoggerTTS         RegisteredLogger = "TTS"
	RegisteredLoggerVoice       RegisteredLogger = "VOICE"
	RegisteredLoggerVector      RegisteredLogger = "VECTOR"
	RegisteredLoggerBundler     RegisteredLogger = "BUNDLER"
	RegisteredLoggerDeployer    RegisteredLogger = "DEPLOYER"
	RegisteredLoggerMemory      RegisteredLogger = "MEMORY"
	RegisteredLoggerStorage     RegisteredLogger = "STORAGE"
	RegisteredLoggerEmbeddings  RegisteredLogger = "EMBEDDINGS"
	RegisteredLoggerMCPServer   RegisteredLogger = "MCP_SERVER"
	RegisteredLoggerServerCache RegisteredLogger = "SERVER_CACHE"
	RegisteredLoggerServer      RegisteredLogger = "SERVER"
	RegisteredLoggerWorkspace   RegisteredLogger = "WORKSPACE"
)

// LogLevel represents the severity level for log messages.
// Uses int values for ordering so level comparisons use <=.
type LogLevel int

const (
	LogLevelDebug  LogLevel = iota // "debug"
	LogLevelInfo                   // "info"
	LogLevelWarn                   // "warn"
	LogLevelError                  // "error"
	LogLevelNone                   // "silent"
)

// logLevelNames maps LogLevel values to their string representations.
var logLevelNames = map[LogLevel]string{
	LogLevelDebug: "debug",
	LogLevelInfo:  "info",
	LogLevelWarn:  "warn",
	LogLevelError: "error",
	LogLevelNone:  "silent",
}

// String returns the string representation of a LogLevel.
func (l LogLevel) String() string {
	if name, ok := logLevelNames[l]; ok {
		return name
	}
	return "unknown"
}

// ParseLogLevel converts a string to a LogLevel.
// Returns LogLevelError if the string is not recognized (matching TS default).
func ParseLogLevel(s string) LogLevel {
	switch s {
	case "debug":
		return LogLevelDebug
	case "info":
		return LogLevelInfo
	case "warn":
		return LogLevelWarn
	case "error":
		return LogLevelError
	case "silent":
		return LogLevelNone
	default:
		return LogLevelError
	}
}
