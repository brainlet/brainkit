package engine

import (
	"testing"
)

func TestCatalogPerInstance(t *testing.T) {
	// Two catalogs built independently must not share state.
	cat1 := buildCommandCatalog()
	cat2 := buildCommandCatalog()

	// They should have the same commands (both built from the same source)
	if len(cat1.ordered) != len(cat2.ordered) {
		t.Fatalf("catalogs have different lengths: %d vs %d", len(cat1.ordered), len(cat2.ordered))
	}

	// But they must be distinct instances — mutating one doesn't affect the other.
	originalLen := len(cat2.ordered)
	cat1.byTopic["__test_isolation"] = commandSpec{topic: "__test_isolation"}
	cat1.ordered = append(cat1.ordered, commandSpec{topic: "__test_isolation"})

	if len(cat2.ordered) != originalLen {
		t.Fatal("cat2 was mutated when cat1 was modified — catalogs share state")
	}
	if _, found := cat2.byTopic["__test_isolation"]; found {
		t.Fatal("cat2.byTopic was mutated when cat1 was modified — catalogs share state")
	}
}

func TestEventCatalogPerInstance(t *testing.T) {
	cat := buildCommandCatalog()
	ev1 := buildEventCatalog(cat)
	ev2 := buildEventCatalog(cat)

	if len(ev1.byTopic) != len(ev2.byTopic) {
		t.Fatalf("event catalogs have different lengths: %d vs %d", len(ev1.byTopic), len(ev2.byTopic))
	}

	// Verify the commands back-reference works
	if ev1.commands == nil {
		t.Fatal("ev1.commands is nil — back-reference not set")
	}
	if ev1.commands != cat {
		t.Fatal("ev1.commands does not point to the command catalog")
	}
}

func TestModuleRegisterCommand(t *testing.T) {
	cat := buildCommandCatalog()
	k := &Kernel{catalog: cat}

	initialCount := len(cat.ordered)

	k.RegisterCommand(commandSpec{topic: "__test.module.ping"})

	if len(cat.ordered) != initialCount+1 {
		t.Fatalf("expected %d commands, got %d", initialCount+1, len(cat.ordered))
	}
	if !cat.HasCommand("__test.module.ping") {
		t.Fatal("registered command not found in catalog")
	}
}

func TestModuleRegisterCommandDuplicatePanics(t *testing.T) {
	cat := buildCommandCatalog()
	k := &Kernel{catalog: cat}

	k.RegisterCommand(commandSpec{topic: "__test.dup"})

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on duplicate registration")
		}
	}()
	k.RegisterCommand(commandSpec{topic: "__test.dup"})
}

// Module close ordering is owned by brainkit.Kit (reverses its own
// modules slice). The kernel-scoped Module interface was retired
// when every shipped module migrated to brainkit.Module.
