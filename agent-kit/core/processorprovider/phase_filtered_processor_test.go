// Ported from: packages/core/src/processor-provider/phase-filtered-processor.test.ts
package processorprovider

import (
	"github.com/brainlet/brainkit/agent-kit/core/processors"
	"testing"
)

// ---------------------------------------------------------------------------
// Mock processor for testing
// ---------------------------------------------------------------------------

// mockFullProcessor implements Processor, InputProcessorMethods, OutputProcessorMethods,
// and MastraRegistrable to allow testing all phase-filtering scenarios.
type mockFullProcessor struct {
	id          string
	name        string
	description string
	index       int

	processInputCalled        bool
	processInputStepCalled    bool
	processOutputStreamCalled bool
	processOutputResultCalled bool
	processOutputStepCalled   bool
	registerMastraCalled      bool
}

func newMockFullProcessor(id, name string) *mockFullProcessor {
	return &mockFullProcessor{id: id, name: name}
}

func (m *mockFullProcessor) ID() string             { return m.id }
func (m *mockFullProcessor) Name() string           { return m.name }
func (m *mockFullProcessor) Description() string    { return m.description }
func (m *mockFullProcessor) ProcessorIndex() int    { return m.index }
func (m *mockFullProcessor) SetProcessorIndex(i int) { m.index = i }

func (m *mockFullProcessor) ProcessInput(args processors.ProcessInputArgs) ([]processors.MastraDBMessage, *processors.MessageList, *processors.ProcessInputResultWithSystemMessages, error) {
	m.processInputCalled = true
	return nil, nil, nil, nil
}

func (m *mockFullProcessor) ProcessInputStep(args processors.ProcessInputStepArgs) (*processors.ProcessInputStepResult, []processors.MastraDBMessage, error) {
	m.processInputStepCalled = true
	return nil, nil, nil
}

func (m *mockFullProcessor) ProcessOutputStream(args processors.ProcessOutputStreamArgs) (*processors.ChunkType, error) {
	m.processOutputStreamCalled = true
	return nil, nil
}

func (m *mockFullProcessor) ProcessOutputResult(args processors.ProcessOutputResultArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	m.processOutputResultCalled = true
	return nil, nil, nil
}

func (m *mockFullProcessor) ProcessOutputStep(args processors.ProcessOutputStepArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	m.processOutputStepCalled = true
	return nil, nil, nil
}

func (m *mockFullProcessor) RegisterMastra(mastra processors.Mastra) {
	m.registerMastraCalled = true
}

// mockInputOnlyProcessor only implements InputProcessorMethods (not Output).
type mockInputOnlyProcessor struct {
	id    string
	name  string
	index int
}

func (m *mockInputOnlyProcessor) ID() string             { return m.id }
func (m *mockInputOnlyProcessor) Name() string           { return m.name }
func (m *mockInputOnlyProcessor) Description() string    { return "" }
func (m *mockInputOnlyProcessor) ProcessorIndex() int    { return m.index }
func (m *mockInputOnlyProcessor) SetProcessorIndex(i int) { m.index = i }

func (m *mockInputOnlyProcessor) ProcessInput(args processors.ProcessInputArgs) ([]processors.MastraDBMessage, *processors.MessageList, *processors.ProcessInputResultWithSystemMessages, error) {
	return nil, nil, nil, nil
}

func (m *mockInputOnlyProcessor) ProcessInputStep(args processors.ProcessInputStepArgs) (*processors.ProcessInputStepResult, []processors.MastraDBMessage, error) {
	return nil, nil, nil
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestPhaseFilteredProcessor_AllPhasesEnabled(t *testing.T) {
	t.Run("should expose all phases when all are enabled", func(t *testing.T) {
		inner := newMockFullProcessor("test", "Test Processor")
		pfp := NewPhaseFilteredProcessor(inner, AllProcessorPhases)

		for _, phase := range AllProcessorPhases {
			if !pfp.HasPhase(phase) {
				t.Errorf("expected phase %s to be enabled", phase)
			}
		}
	})
}

func TestPhaseFilteredProcessor_DisabledPhases(t *testing.T) {
	t.Run("should hide disabled phases", func(t *testing.T) {
		inner := newMockFullProcessor("test", "Test")
		pfp := NewPhaseFilteredProcessor(inner, []ProcessorPhase{
			PhaseProcessInput,
			PhaseProcessOutputResult,
		})

		if !pfp.HasPhase(PhaseProcessInput) {
			t.Error("expected processInput to be enabled")
		}
		if !pfp.HasPhase(PhaseProcessOutputResult) {
			t.Error("expected processOutputResult to be enabled")
		}

		if pfp.HasPhase(PhaseProcessInputStep) {
			t.Error("expected processInputStep to be disabled")
		}
		if pfp.HasPhase(PhaseProcessOutputStream) {
			t.Error("expected processOutputStream to be disabled")
		}
		if pfp.HasPhase(PhaseProcessOutputStep) {
			t.Error("expected processOutputStep to be disabled")
		}
	})
}

func TestPhaseFilteredProcessor_UnimplementedInnerPhases(t *testing.T) {
	t.Run("should not expose unimplemented inner phases", func(t *testing.T) {
		// mockInputOnlyProcessor does NOT implement OutputProcessorMethods
		inner := &mockInputOnlyProcessor{id: "input-only", name: "Input Only"}
		pfp := NewPhaseFilteredProcessor(inner, AllProcessorPhases)

		// Input phases should be enabled (inner implements them)
		if !pfp.HasPhase(PhaseProcessInput) {
			t.Error("expected processInput to be enabled")
		}
		if !pfp.HasPhase(PhaseProcessInputStep) {
			t.Error("expected processInputStep to be enabled")
		}

		// Output phases should NOT be enabled (inner doesn't implement them)
		if pfp.HasPhase(PhaseProcessOutputStream) {
			t.Error("expected processOutputStream to be disabled (not implemented by inner)")
		}
		if pfp.HasPhase(PhaseProcessOutputResult) {
			t.Error("expected processOutputResult to be disabled (not implemented by inner)")
		}
		if pfp.HasPhase(PhaseProcessOutputStep) {
			t.Error("expected processOutputStep to be disabled (not implemented by inner)")
		}
	})
}

func TestPhaseFilteredProcessor_DelegateProcessInput(t *testing.T) {
	t.Run("should delegate processInput to inner when enabled", func(t *testing.T) {
		inner := newMockFullProcessor("test", "Test")
		pfp := NewPhaseFilteredProcessor(inner, []ProcessorPhase{PhaseProcessInput})

		_, _, _, err := pfp.ProcessInput(processors.ProcessInputArgs{})
		if err != nil {
			t.Fatalf("ProcessInput returned error: %v", err)
		}
		if !inner.processInputCalled {
			t.Error("expected inner ProcessInput to be called")
		}
	})

	t.Run("should not delegate processInput when disabled", func(t *testing.T) {
		inner := newMockFullProcessor("test", "Test")
		pfp := NewPhaseFilteredProcessor(inner, []ProcessorPhase{PhaseProcessOutputResult})

		_, _, _, err := pfp.ProcessInput(processors.ProcessInputArgs{})
		if err != nil {
			t.Fatalf("ProcessInput returned error: %v", err)
		}
		if inner.processInputCalled {
			t.Error("expected inner ProcessInput NOT to be called")
		}
	})
}

func TestPhaseFilteredProcessor_EmptyEnabledPhases(t *testing.T) {
	t.Run("should have no active phases with empty enabled list", func(t *testing.T) {
		inner := newMockFullProcessor("test", "Test")
		pfp := NewPhaseFilteredProcessor(inner, []ProcessorPhase{})

		for _, phase := range AllProcessorPhases {
			if pfp.HasPhase(phase) {
				t.Errorf("expected phase %s to be disabled with empty enabledPhases", phase)
			}
		}
	})
}

func TestPhaseFilteredProcessor_Identity(t *testing.T) {
	t.Run("should preserve inner processor identity", func(t *testing.T) {
		inner := newMockFullProcessor("my-id", "My Processor")
		inner.description = "A test processor"
		pfp := NewPhaseFilteredProcessor(inner, AllProcessorPhases)

		if pfp.ID() != "my-id" {
			t.Errorf("expected ID()=my-id, got %s", pfp.ID())
		}
		if pfp.Name() != "My Processor" {
			t.Errorf("expected Name()=My Processor, got %s", pfp.Name())
		}
		if pfp.Description() != "A test processor" {
			t.Errorf("expected Description()='A test processor', got %s", pfp.Description())
		}
	})
}

func TestPhaseFilteredProcessor_RegisterMastra(t *testing.T) {
	t.Run("should forward RegisterMastra to inner", func(t *testing.T) {
		inner := newMockFullProcessor("test", "Test")
		pfp := NewPhaseFilteredProcessor(inner, AllProcessorPhases)

		pfp.RegisterMastra(nil)
		if !inner.registerMastraCalled {
			t.Error("expected inner RegisterMastra to be called")
		}
	})
}
