// Ported from: packages/ai/src/generate-object/repair-text.ts
package generateobject

// RepairTextFunc is a function that attempts to repair the raw output of the model
// to enable JSON parsing.
// Should return the repaired text or an error if the text cannot be repaired.
type RepairTextFunc func(text string, parseError error) (string, error)
