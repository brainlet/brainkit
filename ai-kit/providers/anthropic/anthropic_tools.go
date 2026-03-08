// Ported from: packages/anthropic/src/anthropic-tools.ts
package anthropic

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// AnthropicTools is the registry of all Anthropic provider tool factories.
var AnthropicTools = struct {
	Bash20241022           func(opts providerutils.ProviderToolOptions[Bash20241022Input, interface{}]) providerutils.ProviderTool[Bash20241022Input, interface{}]
	Bash20250124           func(opts providerutils.ProviderToolOptions[Bash20250124Input, interface{}]) providerutils.ProviderTool[Bash20250124Input, interface{}]
	CodeExecution20250522  func(opts providerutils.ProviderToolOptions[CodeExecution20250522Input, CodeExecution20250522Output]) providerutils.ProviderTool[CodeExecution20250522Input, CodeExecution20250522Output]
	CodeExecution20250825  func(opts providerutils.ProviderToolOptions[CodeExecution20250825Input, CodeExecution20250825Output]) providerutils.ProviderTool[CodeExecution20250825Input, CodeExecution20250825Output]
	CodeExecution20260120  func(opts providerutils.ProviderToolOptions[CodeExecution20260120Input, CodeExecution20260120Output]) providerutils.ProviderTool[CodeExecution20260120Input, CodeExecution20260120Output]
	Computer20241022       func(opts providerutils.ProviderToolOptions[Computer20241022Input, interface{}]) providerutils.ProviderTool[Computer20241022Input, interface{}]
	Computer20250124       func(opts providerutils.ProviderToolOptions[Computer20250124Input, interface{}]) providerutils.ProviderTool[Computer20250124Input, interface{}]
	Computer20251124       func(opts providerutils.ProviderToolOptions[Computer20251124Input, interface{}]) providerutils.ProviderTool[Computer20251124Input, interface{}]
	Memory20250818         func(opts providerutils.ProviderToolOptions[Memory20250818Input, interface{}]) providerutils.ProviderTool[Memory20250818Input, interface{}]
	TextEditor20241022     func(opts providerutils.ProviderToolOptions[TextEditor20241022Input, interface{}]) providerutils.ProviderTool[TextEditor20241022Input, interface{}]
	TextEditor20250124     func(opts providerutils.ProviderToolOptions[TextEditor20250124Input, interface{}]) providerutils.ProviderTool[TextEditor20250124Input, interface{}]
	TextEditor20250429     func(opts providerutils.ProviderToolOptions[TextEditor20250429Input, interface{}]) providerutils.ProviderTool[TextEditor20250429Input, interface{}]
	TextEditor20250728     func(opts providerutils.ProviderToolOptions[TextEditor20250728Input, interface{}]) providerutils.ProviderTool[TextEditor20250728Input, interface{}]
	WebFetch20250910       func(opts providerutils.ProviderToolOptions[WebFetch20250910Input, WebFetch20250910Output]) providerutils.ProviderTool[WebFetch20250910Input, WebFetch20250910Output]
	WebFetch20260209       func(opts providerutils.ProviderToolOptions[WebFetch20260209Input, WebFetch20260209Output]) providerutils.ProviderTool[WebFetch20260209Input, WebFetch20260209Output]
	WebSearch20250305      func(opts providerutils.ProviderToolOptions[WebSearch20250305Input, WebSearch20250305Output]) providerutils.ProviderTool[WebSearch20250305Input, WebSearch20250305Output]
	WebSearch20260209      func(opts providerutils.ProviderToolOptions[WebSearch20260209Input, WebSearch20260209Output]) providerutils.ProviderTool[WebSearch20260209Input, WebSearch20260209Output]
	ToolSearchRegex20251119 func(opts providerutils.ProviderToolOptions[ToolSearchRegex20251119Input, ToolSearchRegex20251119Output]) providerutils.ProviderTool[ToolSearchRegex20251119Input, ToolSearchRegex20251119Output]
	ToolSearchBm2520251119  func(opts providerutils.ProviderToolOptions[ToolSearchBm2520251119Input, ToolSearchBm2520251119Output]) providerutils.ProviderTool[ToolSearchBm2520251119Input, ToolSearchBm2520251119Output]
}{
	Bash20241022:           Bash20241022,
	Bash20250124:           Bash20250124,
	CodeExecution20250522:  CodeExecution20250522,
	CodeExecution20250825:  CodeExecution20250825,
	CodeExecution20260120:  CodeExecution20260120,
	Computer20241022:       Computer20241022,
	Computer20250124:       Computer20250124,
	Computer20251124:       Computer20251124,
	Memory20250818:         Memory20250818,
	TextEditor20241022:     TextEditor20241022,
	TextEditor20250124:     TextEditor20250124,
	TextEditor20250429:     TextEditor20250429,
	TextEditor20250728:     TextEditor20250728,
	WebFetch20250910:       WebFetch20250910,
	WebFetch20260209:       WebFetch20260209,
	WebSearch20250305:      WebSearch20250305,
	WebSearch20260209:      WebSearch20260209,
	ToolSearchRegex20251119: ToolSearchRegex20251119,
	ToolSearchBm2520251119:  ToolSearchBm2520251119,
}
