// Ported from: packages/core/src/processors/processors/skills.ts
package concreteprocessors

import (
	"encoding/json"
	"fmt"
	"strings"

	processors "github.com/brainlet/brainkit/agent-kit/core/processors"
)

// ---------------------------------------------------------------------------
// Stub types for unported dependencies
// ---------------------------------------------------------------------------

// Workspace is a stub for ../../workspace/workspace.Workspace.
// TODO: import from workspace package once ported.
type Workspace interface {
	Skills() WorkspaceSkills
}

// WorkspaceSkills is a stub for ../../workspace/skills.WorkspaceSkills.
// TODO: import from workspace package once ported.
type WorkspaceSkills interface {
	List() ([]SkillMeta, error)
	Get(name string) (*Skill, error)
	Has(name string) (bool, error)
	GetReference(skillName, referencePath string) (*string, error)
	GetScript(skillName, scriptPath string) (*string, error)
	GetAsset(skillName, assetPath string) ([]byte, error)
	ListReferences(skillName string) ([]string, error)
	ListScripts(skillName string) ([]string, error)
	ListAssets(skillName string) ([]string, error)
	Search(query string, opts *SkillSearchOpts) ([]SkillSearchResult, error)
	MaybeRefresh(opts *SkillRefreshOpts) error
}

// SkillMeta holds basic skill metadata.
type SkillMeta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	License     string `json:"license,omitempty"`
}

// Skill holds full skill data.
type Skill struct {
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	License      string     `json:"license,omitempty"`
	Path         string     `json:"path"`
	Source       SkillSource `json:"source"`
	Instructions string     `json:"instructions"`
}

// SkillSource identifies where a skill comes from.
type SkillSource struct {
	Type string `json:"type"`
}

// SkillSearchOpts holds options for skill search.
type SkillSearchOpts struct {
	TopK       int      `json:"topK,omitempty"`
	SkillNames []string `json:"skillNames,omitempty"`
}

// SkillSearchResult holds a single skill search result.
type SkillSearchResult struct {
	SkillName string     `json:"skillName"`
	Source    string     `json:"source"`
	Score    float64    `json:"score"`
	Content  string     `json:"content"`
	LineRange *LineRange `json:"lineRange,omitempty"`
}

// LineRange is a line range for content.
type LineRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// SkillRefreshOpts holds options for skill refresh.
type SkillRefreshOpts struct {
	RequestContext any `json:"requestContext,omitempty"`
}

// SkillFormat is the format for skill injection: "xml", "json", "markdown".
type SkillFormat string

const (
	SkillFormatXML      SkillFormat = "xml"
	SkillFormatJSON     SkillFormat = "json"
	SkillFormatMarkdown SkillFormat = "markdown"
)

// ---------------------------------------------------------------------------
// SkillsProcessorOptions
// ---------------------------------------------------------------------------

// SkillsProcessorOptions configures the SkillsProcessor.
type SkillsProcessorOptions struct {
	// Workspace instance containing skills.
	Workspace Workspace

	// Format for skill injection. Default: "xml".
	Format SkillFormat
}

// ---------------------------------------------------------------------------
// SkillsProcessor
// ---------------------------------------------------------------------------

// SkillsProcessor makes skills available to agents via tools and system message injection.
type SkillsProcessor struct {
	processors.BaseProcessor
	workspace       Workspace
	format          SkillFormat
	activatedSkills map[string]bool
}

// NewSkillsProcessor creates a new SkillsProcessor.
func NewSkillsProcessor(opts SkillsProcessorOptions) *SkillsProcessor {
	format := opts.Format
	if format == "" {
		format = SkillFormatXML
	}

	return &SkillsProcessor{
		BaseProcessor:   processors.NewBaseProcessor("skills-processor", "Skills Processor"),
		workspace:       opts.Workspace,
		format:          format,
		activatedSkills: make(map[string]bool),
	}
}

// ListSkills lists all skills available to this processor.
func (sp *SkillsProcessor) ListSkills() ([]SkillMeta, error) {
	skills := sp.workspace.Skills()
	if skills == nil {
		return nil, nil
	}
	return skills.List()
}

// ProcessInputStep injects available skills and provides skill tools.
func (sp *SkillsProcessor) ProcessInputStep(args processors.ProcessInputStepArgs) (*processors.ProcessInputStepResult, []processors.MastraDBMessage, error) {
	skills := sp.workspace.Skills()

	// Refresh skills on first step only.
	if args.StepNumber == 0 && skills != nil {
		_ = skills.MaybeRefresh(&SkillRefreshOpts{RequestContext: args.RequestContext})
	}

	var skillsList []SkillMeta
	hasSkills := false
	if skills != nil {
		var err error
		skillsList, err = skills.List()
		if err == nil && len(skillsList) > 0 {
			hasSkills = true
		}
	}

	// Build system messages.
	var systemMessages []processors.CoreMessageV4

	if hasSkills {
		availableSkillsMessage, err := sp.formatAvailableSkills()
		if err == nil && availableSkillsMessage != "" {
			systemMessages = append(systemMessages, processors.CoreMessageV4{
				Role:    "system",
				Content: availableSkillsMessage,
			})
		}

		systemMessages = append(systemMessages, processors.CoreMessageV4{
			Role: "system",
			Content: "IMPORTANT: Skills are NOT tools. Do not call skill names directly. " +
				"To use a skill, call the skill-activate tool with the skill name as the \"name\" parameter. " +
				"When a user asks about a topic covered by an available skill, activate it immediately without asking for permission.",
		})
	}

	if len(sp.activatedSkills) > 0 {
		activatedSkillsMessage, err := sp.formatActivatedSkills()
		if err == nil && activatedSkillsMessage != "" {
			systemMessages = append(systemMessages, processors.CoreMessageV4{
				Role:    "system",
				Content: activatedSkillsMessage,
			})
		}
	}

	// Build skill tools.
	// TODO: Once tools/createTool is ported, create actual tool objects.
	// For now, return the system messages and existing tools.
	result := &processors.ProcessInputStepResult{
		SystemMessages: systemMessages,
		Tools:          args.Tools,
	}

	return result, nil, nil
}

// ProcessInput is not implemented for this processor.
func (sp *SkillsProcessor) ProcessInput(args processors.ProcessInputArgs) ([]processors.MastraDBMessage, *processors.MessageList, *processors.ProcessInputResultWithSystemMessages, error) {
	return nil, nil, nil, nil
}

// ProcessOutputStream is not implemented for this processor.
func (sp *SkillsProcessor) ProcessOutputStream(args processors.ProcessOutputStreamArgs) (*processors.ChunkType, error) {
	return &args.Part, nil
}

// ProcessOutputResult is not implemented for this processor.
func (sp *SkillsProcessor) ProcessOutputResult(args processors.ProcessOutputResultArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	return nil, nil, nil
}

// ProcessOutputStep is not implemented for this processor.
func (sp *SkillsProcessor) ProcessOutputStep(args processors.ProcessOutputStepArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	return nil, nil, nil
}

// ActivateSkill activates a skill by name.
func (sp *SkillsProcessor) ActivateSkill(name string) {
	sp.activatedSkills[name] = true
}

// IsSkillActivated checks if a skill is activated.
func (sp *SkillsProcessor) IsSkillActivated(name string) bool {
	return sp.activatedSkills[name]
}

// ---------------------------------------------------------------------------
// Private helpers
// ---------------------------------------------------------------------------

// formatLocation returns the path to a skill's SKILL.md file.
func formatLocation(skill *Skill) string {
	return fmt.Sprintf("%s/SKILL.md", skill.Path)
}

// formatSourceType returns the skill's source type for display.
func formatSourceType(skill *Skill) string {
	return skill.Source.Type
}

// formatAvailableSkills formats available skills metadata based on configured format.
func (sp *SkillsProcessor) formatAvailableSkills() (string, error) {
	skills := sp.workspace.Skills()
	if skills == nil {
		return "", nil
	}

	skillsList, err := skills.List()
	if err != nil || len(skillsList) == 0 {
		return "", err
	}

	// Get full skill objects.
	var fullSkills []*Skill
	for _, meta := range skillsList {
		skill, err := skills.Get(meta.Name)
		if err == nil && skill != nil {
			fullSkills = append(fullSkills, skill)
		}
	}

	switch sp.format {
	case SkillFormatXML:
		var skillsXml []string
		for _, skill := range fullSkills {
			skillsXml = append(skillsXml, fmt.Sprintf(`  <skill>
    <name>%s</name>
    <description>%s</description>
    <location>%s</location>
    <source>%s</source>
  </skill>`,
				escapeXml(skill.Name),
				escapeXml(skill.Description),
				escapeXml(formatLocation(skill)),
				escapeXml(formatSourceType(skill))))
		}
		return fmt.Sprintf("<available_skills>\n%s\n</available_skills>", strings.Join(skillsXml, "\n")), nil

	case SkillFormatJSON:
		var items []map[string]string
		for _, skill := range fullSkills {
			items = append(items, map[string]string{
				"name":        skill.Name,
				"description": skill.Description,
				"location":    formatLocation(skill),
				"source":      formatSourceType(skill),
			})
		}
		jsonBytes, err := json.MarshalIndent(items, "", "  ")
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Available Skills:\n\n%s", string(jsonBytes)), nil

	case SkillFormatMarkdown:
		var skillsMd []string
		for _, skill := range fullSkills {
			skillsMd = append(skillsMd, fmt.Sprintf("- **%s** [%s] (%s): %s",
				skill.Name, formatSourceType(skill), formatLocation(skill), skill.Description))
		}
		return fmt.Sprintf("# Available Skills\n\n%s", strings.Join(skillsMd, "\n")), nil

	default:
		return "", nil
	}
}

// formatActivatedSkills formats activated skills based on configured format.
func (sp *SkillsProcessor) formatActivatedSkills() (string, error) {
	skills := sp.workspace.Skills()
	if skills == nil {
		return "", nil
	}

	var activatedSkillsList []*Skill
	for name := range sp.activatedSkills {
		skill, err := skills.Get(name)
		if err == nil && skill != nil {
			activatedSkillsList = append(activatedSkillsList, skill)
		}
	}

	if len(activatedSkillsList) == 0 {
		return "", nil
	}

	var skillInstructions []string

	switch sp.format {
	case SkillFormatXML:
		for _, skill := range activatedSkillsList {
			skillInstructions = append(skillInstructions, fmt.Sprintf(
				"# Skill: %s\nLocation: %s\nSource: %s\n\n%s",
				skill.Name, formatLocation(skill), formatSourceType(skill), skill.Instructions))
		}
		return fmt.Sprintf("<activated_skills>\n%s\n</activated_skills>",
			strings.Join(skillInstructions, "\n\n---\n\n")), nil

	case SkillFormatJSON, SkillFormatMarkdown:
		for _, skill := range activatedSkillsList {
			skillInstructions = append(skillInstructions, fmt.Sprintf(
				"# Skill: %s\n*Location: %s | Source: %s*\n\n%s",
				skill.Name, formatLocation(skill), formatSourceType(skill), skill.Instructions))
		}
		return fmt.Sprintf("# Activated Skills\n\n%s",
			strings.Join(skillInstructions, "\n\n---\n\n")), nil

	default:
		return "", nil
	}
}

// escapeXml escapes XML special characters.
func escapeXml(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
