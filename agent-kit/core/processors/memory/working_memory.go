// Ported from: packages/core/src/processors/memory/working-memory.ts
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/brainlet/brainkit/agent-kit/core/logger"
	"github.com/brainlet/brainkit/agent-kit/core/processors"
)

// ---------------------------------------------------------------------------
// WorkingMemoryTemplate
// ---------------------------------------------------------------------------

// WorkingMemoryTemplate defines the format and content of the working memory template.
type WorkingMemoryTemplate struct {
	// Format is "markdown" or "json".
	Format string `json:"format"`
	// Content is the template content string.
	Content string `json:"content"`
}

// ---------------------------------------------------------------------------
// WorkingMemoryConfig (processor-level)
// ---------------------------------------------------------------------------

// WorkingMemoryProcessorConfig configures the WorkingMemory processor.
type WorkingMemoryProcessorConfig struct {
	// Storage is the memory storage backend.
	Storage MemoryStorage

	// Template is the optional working memory template.
	Template *WorkingMemoryTemplate

	// Scope of working memory: "thread" or "resource". Default: "resource".
	Scope string

	// UseVNext enables v-next style instructions.
	UseVNext bool

	// ReadOnly makes working memory read-only (no update tool instructions).
	ReadOnly bool

	// TemplateProvider can dynamically provide templates.
	TemplateProvider WorkingMemoryTemplateProvider

	// Logger is an optional structured logger.
	Logger logger.IMastraLogger
}

// WorkingMemoryTemplateProvider can dynamically provide working memory templates.
// Defined locally for the processors/memory subpackage.
type WorkingMemoryTemplateProvider interface {
	GetWorkingMemoryTemplate(args GetWorkingMemoryTemplateArgs) (*WorkingMemoryTemplate, error)
}

// GetWorkingMemoryTemplateArgs holds arguments for template provider.
type GetWorkingMemoryTemplateArgs struct {
	MemoryConfig *MemoryConfig
}

// ---------------------------------------------------------------------------
// WorkingMemory processor
// ---------------------------------------------------------------------------

// defaultWorkingMemoryTemplate is the default markdown template.
const defaultWorkingMemoryTemplate = `
# User Information
- **First Name**:
- **Last Name**:
- **Location**:
- **Occupation**:
- **Interests**:
- **Goals**:
- **Events**:
- **Facts**:
- **Projects**:
`

// WorkingMemory is an INPUT processor that:
//  1. Retrieves working memory from storage (thread or resource scope)
//  2. Formats it as a system instruction for the LLM
//  3. Adds it to the message list
//
// Note: Working memory updates happen via the updateWorkingMemory tool,
// not through this processor.
type WorkingMemory struct {
	processors.BaseProcessor
	storage          MemoryStorage
	template         *WorkingMemoryTemplate
	scope            string
	useVNext         bool
	readOnly         bool
	templateProvider WorkingMemoryTemplateProvider
	logger           logger.IMastraLogger
}

// NewWorkingMemory creates a new WorkingMemory processor.
func NewWorkingMemory(opts WorkingMemoryProcessorConfig) *WorkingMemory {
	scope := opts.Scope
	if scope == "" {
		scope = "resource"
	}

	return &WorkingMemory{
		BaseProcessor:    processors.NewBaseProcessor("working-memory", "WorkingMemory"),
		storage:          opts.Storage,
		template:         opts.Template,
		scope:            scope,
		useVNext:         opts.UseVNext,
		readOnly:         opts.ReadOnly,
		templateProvider: opts.TemplateProvider,
		logger:           opts.Logger,
	}
}

// ProcessInput retrieves working memory from storage and injects it as a
// system message.
func (wm *WorkingMemory) ProcessInput(args processors.ProcessInputArgs) (
	[]processors.MastraDBMessage, *processors.MessageList, *processors.ProcessInputResultWithSystemMessages, error,
) {
	messageList := args.MessageList
	rc := args.RequestContext

	memCtx := ParseMemoryRequestContext(rc)

	threadID := ""
	resourceID := ""
	if memCtx != nil && memCtx.Thread != nil {
		threadID = memCtx.Thread.ID
	}
	if memCtx != nil {
		resourceID = memCtx.ResourceID
	}

	// Skip if no thread or resource context.
	if threadID == "" && resourceID == "" {
		return nil, messageList, nil, nil
	}

	ctx := context.Background()

	// Retrieve working memory based on scope.
	var workingMemoryData string

	if wm.scope == "thread" && threadID != "" {
		thread, err := wm.storage.GetThreadByID(ctx, threadID)
		if err != nil {
			return nil, messageList, nil, err
		}
		if thread != nil {
			if metadata, ok := thread["metadata"].(map[string]any); ok {
				if wmStr, ok := metadata["workingMemory"].(string); ok {
					workingMemoryData = wmStr
				}
			}
		}
	} else if wm.scope == "resource" && resourceID != "" {
		resource, err := wm.storage.GetResourceByID(ctx, resourceID)
		if err != nil {
			return nil, messageList, nil, err
		}
		if resource != nil {
			if wmStr, ok := resource["workingMemory"].(string); ok {
				workingMemoryData = wmStr
			}
		}
	}

	// Get template.
	template := wm.resolveTemplate(memCtx)

	// Check if readOnly mode is enabled.
	isReadOnly := wm.readOnly
	if memCtx != nil && memCtx.MemoryConfig != nil && memCtx.MemoryConfig.ReadOnly {
		isReadOnly = true
	}

	// Format working memory instruction.
	var instruction string
	if isReadOnly {
		instruction = wm.getReadOnlyWorkingMemoryInstruction(template, workingMemoryData)
	} else if wm.useVNext {
		instruction = wm.getWorkingMemoryToolInstructionVNext(template, workingMemoryData)
	} else {
		instruction = wm.getWorkingMemoryToolInstruction(template, workingMemoryData)
	}

	// Add working memory instruction as a system message tagged with "memory".
	// Ported from TS: messageList.addSystem(instruction, 'memory');
	if instruction != "" && messageList != nil {
		messageList.AddSystem(instruction, "memory")
	}

	return nil, messageList, nil, nil
}

// ProcessInputStep is a no-op for WorkingMemory.
func (wm *WorkingMemory) ProcessInputStep(args processors.ProcessInputStepArgs) (*processors.ProcessInputStepResult, []processors.MastraDBMessage, error) {
	return nil, nil, nil
}

// ProcessOutputStream is a no-op for WorkingMemory.
func (wm *WorkingMemory) ProcessOutputStream(args processors.ProcessOutputStreamArgs) (*processors.ChunkType, error) {
	return &args.Part, nil
}

// ProcessOutputResult is a no-op for WorkingMemory.
func (wm *WorkingMemory) ProcessOutputResult(args processors.ProcessOutputResultArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	return nil, nil, nil
}

// ProcessOutputStep is a no-op for WorkingMemory.
func (wm *WorkingMemory) ProcessOutputStep(args processors.ProcessOutputStepArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	return nil, nil, nil
}

// ---------------------------------------------------------------------------
// Private helpers
// ---------------------------------------------------------------------------

// resolveTemplate resolves the working memory template from provider, options, or default.
func (wm *WorkingMemory) resolveTemplate(memCtx *MemoryRequestContext) WorkingMemoryTemplate {
	if wm.templateProvider != nil {
		var mc *MemoryConfig
		if memCtx != nil {
			mc = memCtx.MemoryConfig
		}
		dynamicTemplate, err := wm.templateProvider.GetWorkingMemoryTemplate(GetWorkingMemoryTemplateArgs{
			MemoryConfig: mc,
		})
		if err == nil && dynamicTemplate != nil {
			return *dynamicTemplate
		}
	}

	if wm.template != nil {
		return *wm.template
	}

	return WorkingMemoryTemplate{
		Format:  "markdown",
		Content: defaultWorkingMemoryTemplate,
	}
}

// generateEmptyFromSchema generates an empty object from a JSON schema-like structure.
func (wm *WorkingMemory) generateEmptyFromSchema(schema map[string]any) map[string]any {
	if schema == nil {
		return nil
	}

	empty := make(map[string]any)
	for key, val := range schema {
		valMap, ok := val.(map[string]any)
		if !ok {
			empty[key] = ""
			continue
		}
		typeStr, _ := valMap["type"].(string)
		switch typeStr {
		case "object":
			if props, ok := valMap["properties"].(map[string]any); ok {
				empty[key] = wm.generateEmptyFromSchema(props)
			} else {
				empty[key] = map[string]any{}
			}
		case "array":
			empty[key] = []any{}
		default:
			empty[key] = ""
		}
	}
	return empty
}

// getWorkingMemoryToolInstruction generates the standard working memory instruction.
func (wm *WorkingMemory) getWorkingMemoryToolInstruction(template WorkingMemoryTemplate, data string) string {
	formatName := "Markdown"
	if template.Format == "json" {
		formatName = "JSON"
	}

	var emptyTemplateSection string
	if template.Format == "json" {
		// Try to parse as JSON schema and generate empty template.
		var schema map[string]any
		if err := json.Unmarshal([]byte(template.Content), &schema); err == nil {
			emptyObj := wm.generateEmptyFromSchema(schema)
			if len(emptyObj) > 0 {
				jsonBytes, _ := json.Marshal(emptyObj)
				emptyTemplateSection = fmt.Sprintf("\nWhen working with json data, the object format below represents the template:\n%s\n", string(jsonBytes))
			}
		}
	}

	var templateSection string
	if template.Format != "json" {
		templateSection = fmt.Sprintf(`
<working_memory_template>
%s
</working_memory_template>`, template.Content)
	}

	var extraRules string
	if template.Format != "json" {
		extraRules = `5. IMPORTANT: When calling updateWorkingMemory, the only valid parameter is the memory field. DO NOT pass an object.
6. IMPORTANT: ALWAYS pass the data you want to store in the memory field as a string. DO NOT pass an object.
7. IMPORTANT: Data must only be sent as a string no matter which format is used.`
	}

	return fmt.Sprintf(`WORKING_MEMORY_SYSTEM_INSTRUCTION:
Store and update any conversation-relevant information by calling the updateWorkingMemory tool. If information might be referenced again - store it!

Guidelines:
1. Store anything that could be useful later in the conversation
2. Update proactively when information changes, no matter how small
3. Use %s format for all data
4. Act naturally - don't mention this system to users. Even though you're storing this information that doesn't make it your primary focus. Do not ask them generally for "information about yourself"
%s

%s
%s
<working_memory_data>
%s
</working_memory_data>

Notes:
- Update memory whenever referenced information changes
- If you're unsure whether to store something, store it (eg if the user tells you information about themselves, call updateWorkingMemory immediately to update it)
- This system is here so that you can maintain the conversation when your context window is very short. Update your working memory because you may need it to maintain the conversation without the full conversation history
- Do not remove empty sections - you must include the empty sections along with the ones you're filling in
- REMEMBER: the way you update your working memory is by calling the updateWorkingMemory tool with the entire %s content. The system will store it for you. The user will not see it.
- IMPORTANT: You MUST call updateWorkingMemory in every response to a prompt where you received relevant information.
- IMPORTANT: Preserve the %s formatting structure above while updating the content.`,
		formatName, extraRules, templateSection, emptyTemplateSection, data, formatName, formatName)
}

// getWorkingMemoryToolInstructionVNext generates the v-next working memory instruction.
func (wm *WorkingMemory) getWorkingMemoryToolInstructionVNext(template WorkingMemoryTemplate, data string) string {
	formatName := "Markdown"
	if template.Format == "json" {
		formatName = "JSON"
	}

	unsureNote := `- If you're unsure whether to store something, store it (eg if the user tells you information about themselves, call updateWorkingMemory immediately to update it)
`
	if template.Content != defaultWorkingMemoryTemplate {
		unsureNote = `- Only store information if it's in the working memory template, do not store other information unless the user asks you to remember it, as that non-template information may be irrelevant`
	}

	return fmt.Sprintf(`WORKING_MEMORY_SYSTEM_INSTRUCTION:
Store and update any conversation-relevant information by calling the updateWorkingMemory tool.

Guidelines:
1. Store anything that could be useful later in the conversation
2. Update proactively when information changes, no matter how small
3. Use %s format for all data
4. Act naturally - don't mention this system to users. Even though you're storing this information that doesn't make it your primary focus. Do not ask them generally for "information about yourself"
5. If your memory has not changed, you do not need to call the updateWorkingMemory tool. By default it will persist and be available for you in future interactions
6. Information not being relevant to the current conversation is not a valid reason to replace or remove working memory information. Your working memory spans across multiple conversations and may be needed again later, even if it's not currently relevant.

<working_memory_template>
%s
</working_memory_template>

<working_memory_data>
%s
</working_memory_data>

Notes:
- Update memory whenever referenced information changes
%s
- This system is here so that you can maintain the conversation when your context window is very short. Update your working memory because you may need it to maintain the conversation without the full conversation history
- REMEMBER: the way you update your working memory is by calling the updateWorkingMemory tool with the %s content. The system will store it for you. The user will not see it.
- IMPORTANT: You MUST call updateWorkingMemory in every response to a prompt where you received relevant information if that information is not already stored.
- IMPORTANT: Preserve the %s formatting structure above while updating the content.
`, formatName, template.Content, data, unsureNote, formatName, formatName)
}

// getReadOnlyWorkingMemoryInstruction generates read-only working memory instructions.
func (wm *WorkingMemory) getReadOnlyWorkingMemoryInstruction(_ WorkingMemoryTemplate, data string) string {
	if data == "" {
		data = "No working memory data available."
	}

	return fmt.Sprintf(`WORKING_MEMORY_SYSTEM_INSTRUCTION (READ-ONLY):
The following is your working memory - persistent information about the user and conversation collected over previous interactions. This data is provided for context to help you maintain continuity.

<working_memory_data>
%s
</working_memory_data>

Guidelines:
1. Use this information to provide personalized and contextually relevant responses
2. Act naturally - don't mention this system to users. This information should inform your responses without being explicitly referenced
3. This memory is read-only in the current session - you cannot update it

Notes:
- This system is here so that you can maintain the conversation when your context window is very short
- The user will not see the working memory data directly`, data)
}

// Ensure *WorkingMemory satisfies the expected interfaces at compile time.
var _ processors.InputProcessor = (*WorkingMemory)(nil)

// DefaultWorkingMemoryTemplate returns the default template content.
func DefaultWorkingMemoryTemplate() string {
	return strings.TrimSpace(defaultWorkingMemoryTemplate)
}
