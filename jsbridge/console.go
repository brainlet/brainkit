package jsbridge

import (
	"fmt"
	"io"
	"os"
	"sync"

	quickjs "github.com/buke/quickjs-go"
)

// ConsoleMessage is a captured console output entry.
type ConsoleMessage struct {
	Level   string
	Message string
}

// ConsolePolyfill provides globalThis.console with message capture.
type ConsolePolyfill struct {
	mu       sync.Mutex
	messages []ConsoleMessage
	stdout   io.Writer
	stderr   io.Writer
}

// ConsoleOption configures a ConsolePolyfill.
type ConsoleOption func(*ConsolePolyfill)

// ConsoleStdout sets the writer for log/info/debug output.
func ConsoleStdout(w io.Writer) ConsoleOption {
	return func(c *ConsolePolyfill) { c.stdout = w }
}

// ConsoleStderr sets the writer for warn/error output.
func ConsoleStderr(w io.Writer) ConsoleOption {
	return func(c *ConsolePolyfill) { c.stderr = w }
}

// Console creates a console polyfill.
func Console(opts ...ConsoleOption) *ConsolePolyfill {
	c := &ConsolePolyfill{stdout: os.Stdout, stderr: os.Stderr}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *ConsolePolyfill) Name() string { return "console" }

// Messages returns a copy of all captured console messages.
func (c *ConsolePolyfill) Messages() []ConsoleMessage {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]ConsoleMessage, len(c.messages))
	copy(out, c.messages)
	return out
}

func (c *ConsolePolyfill) capture(level, msg string) {
	c.mu.Lock()
	c.messages = append(c.messages, ConsoleMessage{Level: level, Message: msg})
	c.mu.Unlock()
}

func (c *ConsolePolyfill) Setup(ctx *quickjs.Context) error {
	reg := func(name, level string, w io.Writer) {
		ctx.Globals().Set(name, ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			msg := ""
			if len(args) > 0 {
				msg = args[0].ToString()
			}
			c.capture(level, msg)
			fmt.Fprintln(w, msg)
			return ctx.NewUndefined()
		}))
	}

	reg("__go_console_log", "log", c.stdout)
	reg("__go_console_warn", "warn", c.stderr)
	reg("__go_console_error", "error", c.stderr)
	reg("__go_console_info", "info", c.stdout)
	reg("__go_console_debug", "debug", c.stdout)

	return evalJS(ctx, `
globalThis.console = {
  log:   (...a) => __go_console_log(a.map(String).join(' ')),
  warn:  (...a) => __go_console_warn(a.map(String).join(' ')),
  error: (...a) => __go_console_error(a.map(String).join(' ')),
  info:  (...a) => __go_console_info(a.map(String).join(' ')),
  debug: (...a) => __go_console_debug(a.map(String).join(' ')),
};
`)
}
