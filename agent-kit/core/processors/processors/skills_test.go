// Ported from: packages/core/src/processors/processors/skills.test.ts
package concreteprocessors

import (
	"testing"
)

// mockWorkspaceSkills is a mock for the WorkspaceSkills interface.
type mockWorkspaceSkills struct {
	skills     []SkillMeta
	skillMap   map[string]*Skill
	listErr    error
	getErr     error
}

func (m *mockWorkspaceSkills) List() ([]SkillMeta, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.skills, nil
}

func (m *mockWorkspaceSkills) Get(name string) (*Skill, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if s, ok := m.skillMap[name]; ok {
		return s, nil
	}
	return nil, nil
}

func (m *mockWorkspaceSkills) Has(name string) (bool, error) {
	_, ok := m.skillMap[name]
	return ok, nil
}

func (m *mockWorkspaceSkills) GetReference(skillName, referencePath string) (*string, error) {
	ref := "mock reference content"
	return &ref, nil
}

func (m *mockWorkspaceSkills) GetScript(skillName, scriptPath string) (*string, error) {
	s := "mock script content"
	return &s, nil
}

func (m *mockWorkspaceSkills) GetAsset(skillName, assetPath string) ([]byte, error) {
	return []byte("mock asset"), nil
}

func (m *mockWorkspaceSkills) ListReferences(skillName string) ([]string, error) {
	return []string{"ref1.md"}, nil
}

func (m *mockWorkspaceSkills) ListScripts(skillName string) ([]string, error) {
	return []string{"script1.sh"}, nil
}

func (m *mockWorkspaceSkills) ListAssets(skillName string) ([]string, error) {
	return []string{"asset1.png"}, nil
}

func (m *mockWorkspaceSkills) Search(query string, opts *SkillSearchOpts) ([]SkillSearchResult, error) {
	return nil, nil
}

func (m *mockWorkspaceSkills) MaybeRefresh(opts *SkillRefreshOpts) error {
	return nil
}

// mockWorkspace is a mock for the Workspace interface.
type mockWorkspace struct {
	skills WorkspaceSkills
}

func (m *mockWorkspace) Skills() WorkspaceSkills {
	return m.skills
}

func TestSkillsProcessor(t *testing.T) {
	t.Run("constructor", func(t *testing.T) {
		t.Run("should create processor with workspace", func(t *testing.T) {
			ws := &mockWorkspace{
				skills: &mockWorkspaceSkills{
					skills: []SkillMeta{
						{Name: "test-skill", Description: "A test skill"},
					},
					skillMap: map[string]*Skill{},
				},
			}
			sp := NewSkillsProcessor(SkillsProcessorOptions{Workspace: ws})
			if sp == nil {
				t.Fatal("expected non-nil SkillsProcessor")
			}
			if sp.ID() != "skills-processor" {
				t.Fatalf("expected id 'skills-processor', got '%s'", sp.ID())
			}
		})
	})

	t.Run("listSkills", func(t *testing.T) {
		t.Run("should list available skills from workspace", func(t *testing.T) {
			ws := &mockWorkspace{
				skills: &mockWorkspaceSkills{
					skills: []SkillMeta{
						{Name: "skill-1", Description: "First skill"},
						{Name: "skill-2", Description: "Second skill"},
					},
					skillMap: map[string]*Skill{},
				},
			}
			sp := NewSkillsProcessor(SkillsProcessorOptions{Workspace: ws})
			skills, err := sp.ListSkills()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(skills) != 2 {
				t.Fatalf("expected 2 skills, got %d", len(skills))
			}
		})
	})

	t.Run("processInputStep", func(t *testing.T) {
		t.Run("should inject system messages with available skills in XML format", func(t *testing.T) {
			t.Skip("not yet implemented: processInputStep system message injection requires full integration")
		})

		t.Run("should inject system messages with available skills in JSON format", func(t *testing.T) {
			t.Skip("not yet implemented: processInputStep system message injection requires full integration")
		})

		t.Run("should inject system messages with available skills in markdown format", func(t *testing.T) {
			t.Skip("not yet implemented: processInputStep system message injection requires full integration")
		})

		t.Run("should handle no skills configured", func(t *testing.T) {
			t.Skip("not yet implemented: processInputStep system message injection requires full integration")
		})

		t.Run("should preserve existing tools", func(t *testing.T) {
			t.Skip("not yet implemented: processInputStep system message injection requires full integration")
		})

		t.Run("should pass request context through", func(t *testing.T) {
			t.Skip("not yet implemented: processInputStep system message injection requires full integration")
		})
	})

	t.Run("skill-activate tool", func(t *testing.T) {
		t.Run("should activate a skill by name", func(t *testing.T) {
			t.Skip("not yet implemented: skill-activate tool requires createTool porting")
		})
	})

	t.Run("skill-search tool", func(t *testing.T) {
		t.Run("should search for skills by query", func(t *testing.T) {
			t.Skip("not yet implemented: skill-search tool requires createTool porting")
		})
	})

	t.Run("skill resource tools", func(t *testing.T) {
		t.Run("should read skill reference", func(t *testing.T) {
			t.Skip("not yet implemented: skill-read-reference tool requires createTool porting")
		})

		t.Run("should read skill script", func(t *testing.T) {
			t.Skip("not yet implemented: skill-read-script tool requires createTool porting")
		})

		t.Run("should read skill asset", func(t *testing.T) {
			t.Skip("not yet implemented: skill-read-asset tool requires createTool porting")
		})
	})

	t.Run("activated skills injection", func(t *testing.T) {
		t.Run("should inject activated skill instructions", func(t *testing.T) {
			t.Skip("not yet implemented: requires full processInputStep integration")
		})
	})

	t.Run("no skills configured", func(t *testing.T) {
		t.Run("should handle workspace with no skills", func(t *testing.T) {
			ws := &mockWorkspace{
				skills: &mockWorkspaceSkills{
					skills:   []SkillMeta{},
					skillMap: map[string]*Skill{},
				},
			}
			sp := NewSkillsProcessor(SkillsProcessorOptions{Workspace: ws})
			skills, err := sp.ListSkills()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(skills) != 0 {
				t.Fatalf("expected 0 skills, got %d", len(skills))
			}
		})
	})

	t.Run("skill activation tracking", func(t *testing.T) {
		t.Run("should track activated skills", func(t *testing.T) {
			ws := &mockWorkspace{
				skills: &mockWorkspaceSkills{
					skills: []SkillMeta{
						{Name: "test-skill", Description: "A test skill"},
					},
					skillMap: map[string]*Skill{
						"test-skill": {
							Name:         "test-skill",
							Description:  "A test skill",
							Instructions: "Do the thing",
						},
					},
				},
			}
			sp := NewSkillsProcessor(SkillsProcessorOptions{Workspace: ws})

			// Initially not activated
			if sp.IsSkillActivated("test-skill") {
				t.Fatal("expected skill to not be activated initially")
			}

			// Activate
			sp.ActivateSkill("test-skill")

			// Should now be activated
			if !sp.IsSkillActivated("test-skill") {
				t.Fatal("expected skill to be activated after ActivateSkill")
			}
		})
	})
}
