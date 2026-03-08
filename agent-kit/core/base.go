// Ported from: packages/core/src/base.ts
package core

import (
	"fmt"

	"github.com/brainlet/brainkit/agent-kit/core/logger"
)

// MastraBaseOptions holds the constructor options for MastraBase.
type MastraBaseOptions struct {
	Component logger.RegisteredLogger
	Name      string
	RawConfig map[string]any
}

// MastraBase is the base type for all Mastra primitives.
// It provides a logger, a component identifier, and optional raw configuration.
type MastraBase struct {
	Component logger.RegisteredLogger
	Name      string
	logger    logger.IMastraLogger
	rawConfig map[string]any
}

// NewMastraBase creates a new MastraBase with the given options.
func NewMastraBase(opts MastraBaseOptions) *MastraBase {
	component := opts.Component
	if component == "" {
		component = logger.RegisteredLoggerLLM
	}

	name := opts.Name

	loggerName := fmt.Sprintf("%s - %s", component, name)

	return &MastraBase{
		Component: component,
		Name:      name,
		logger:    logger.NewConsoleLogger(&logger.ConsoleLoggerOptions{Name: loggerName}),
		rawConfig: opts.RawConfig,
	}
}

// Logger returns the current logger instance.
func (b *MastraBase) Logger() logger.IMastraLogger {
	return b.logger
}

// ToRawConfig returns the raw storage configuration this primitive was created from,
// or nil if it was created from code.
func (b *MastraBase) ToRawConfig() map[string]any {
	return b.rawConfig
}

// SetRawConfig sets the raw storage configuration for this primitive.
func (b *MastraBase) SetRawConfig(rawConfig map[string]any) {
	b.rawConfig = rawConfig
}

// SetLogger sets the logger for the primitive.
func (b *MastraBase) SetLogger(l logger.IMastraLogger) {
	b.logger = l

	if b.Component != logger.RegisteredLoggerLLM {
		b.logger.Debug(fmt.Sprintf("Logger updated [component=%s] [name=%s]", b.Component, b.Name))
	}
}
