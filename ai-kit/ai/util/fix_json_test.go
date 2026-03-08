// Ported from: packages/ai/src/util/fix-json.test.ts
package util

import (
	"strings"
	"testing"
)

func assertFixJSON(t *testing.T, input, expected string) {
	t.Helper()
	result := FixJSON(input)
	if result != expected {
		t.Fatalf("FixJSON(%q) = %q, want %q", input, result, expected)
	}
}

func TestFixJSON_EmptyInput(t *testing.T) {
	assertFixJSON(t, "", "")
}

func TestFixJSON_IncompleteNull(t *testing.T) {
	assertFixJSON(t, "nul", "null")
}

func TestFixJSON_IncompleteTrue(t *testing.T) {
	assertFixJSON(t, "t", "true")
}

func TestFixJSON_IncompleteFalse(t *testing.T) {
	assertFixJSON(t, "fals", "false")
}

func TestFixJSON_IncompleteNumbers(t *testing.T) {
	assertFixJSON(t, "12.", "12")
}

func TestFixJSON_NumbersWithDot(t *testing.T) {
	assertFixJSON(t, "12.2", "12.2")
}

func TestFixJSON_NegativeNumbers(t *testing.T) {
	assertFixJSON(t, "-12", "-12")
}

func TestFixJSON_IncompleteNegativeNumbers(t *testing.T) {
	assertFixJSON(t, "-", "")
}

func TestFixJSON_ENotation(t *testing.T) {
	assertFixJSON(t, "2.5e", "2.5")
	assertFixJSON(t, "2.5e-", "2.5")
	assertFixJSON(t, "2.5e3", "2.5e3")
	assertFixJSON(t, "-2.5e3", "-2.5e3")
}

func TestFixJSON_UppercaseENotation(t *testing.T) {
	assertFixJSON(t, "2.5E", "2.5")
	assertFixJSON(t, "2.5E-", "2.5")
	assertFixJSON(t, "2.5E3", "2.5E3")
	assertFixJSON(t, "-2.5E3", "-2.5E3")
}

func TestFixJSON_IncompleteNumbersWithE(t *testing.T) {
	assertFixJSON(t, "12.e", "12")
	assertFixJSON(t, "12.34e", "12.34")
	assertFixJSON(t, "5e", "5")
}

func TestFixJSON_IncompleteStrings(t *testing.T) {
	assertFixJSON(t, `"abc`, `"abc"`)
}

func TestFixJSON_EscapeSequences(t *testing.T) {
	assertFixJSON(t,
		`"value with \"quoted\" text and \\ escape`,
		`"value with \"quoted\" text and \\ escape"`,
	)
}

func TestFixJSON_IncompleteEscapeSequences(t *testing.T) {
	assertFixJSON(t, `"value with \`, `"value with "`)
}

func TestFixJSON_UnicodeCharacters(t *testing.T) {
	assertFixJSON(t, `"value with unicode <"`, `"value with unicode <"`)
}

func TestFixJSON_IncompleteArray(t *testing.T) {
	assertFixJSON(t, "[", "[]")
}

func TestFixJSON_ClosingBracketAfterNumberInArray(t *testing.T) {
	assertFixJSON(t, "[[1], [2", "[[1], [2]]")
}

func TestFixJSON_ClosingBracketAfterStringInArray(t *testing.T) {
	assertFixJSON(t, `[["1"], ["2`, `[["1"], ["2"]]`)
}

func TestFixJSON_ClosingBracketAfterLiteralInArray(t *testing.T) {
	assertFixJSON(t, "[[false], [nu", "[[false], [null]]")
}

func TestFixJSON_ClosingBracketAfterArrayInArray(t *testing.T) {
	assertFixJSON(t, "[[[]], [[]", "[[[]], [[]]]")
}

func TestFixJSON_ClosingBracketAfterObjectInArray(t *testing.T) {
	assertFixJSON(t, "[[{}], [{", "[[{}], [{}]]")
}

func TestFixJSON_TrailingCommaInArray(t *testing.T) {
	assertFixJSON(t, "[1, ", "[1]")
}

func TestFixJSON_ClosingArray(t *testing.T) {
	assertFixJSON(t, "[[], 123", "[[], 123]")
}

func TestFixJSON_KeysWithoutValues(t *testing.T) {
	assertFixJSON(t, `{"key":`, "{}")
}

func TestFixJSON_ClosingBraceAfterNumberInObject(t *testing.T) {
	assertFixJSON(t, `{"a": {"b": 1}, "c": {"d": 2`, `{"a": {"b": 1}, "c": {"d": 2}}`)
}

func TestFixJSON_ClosingBraceAfterStringInObject(t *testing.T) {
	assertFixJSON(t, `{"a": {"b": "1"}, "c": {"d": 2`, `{"a": {"b": "1"}, "c": {"d": 2}}`)
}

func TestFixJSON_ClosingBraceAfterLiteralInObject(t *testing.T) {
	assertFixJSON(t, `{"a": {"b": false}, "c": {"d": 2`, `{"a": {"b": false}, "c": {"d": 2}}`)
}

func TestFixJSON_ClosingBraceAfterArrayInObject(t *testing.T) {
	assertFixJSON(t, `{"a": {"b": []}, "c": {"d": 2`, `{"a": {"b": []}, "c": {"d": 2}}`)
}

func TestFixJSON_ClosingBraceAfterObjectInObject(t *testing.T) {
	assertFixJSON(t, `{"a": {"b": {}}, "c": {"d": 2`, `{"a": {"b": {}}, "c": {"d": 2}}`)
}

func TestFixJSON_PartialKeysFirst(t *testing.T) {
	assertFixJSON(t, `{"ke`, "{}")
}

func TestFixJSON_PartialKeysSecond(t *testing.T) {
	assertFixJSON(t, `{"k1": 1, "k2`, `{"k1": 1}`)
}

func TestFixJSON_PartialKeysWithColonSecond(t *testing.T) {
	assertFixJSON(t, `{"k1": 1, "k2":`, `{"k1": 1}`)
}

func TestFixJSON_TrailingWhitespace(t *testing.T) {
	assertFixJSON(t, `{"key": "value"  `, `{"key": "value"}`)
}

func TestFixJSON_ClosingAfterEmptyObject(t *testing.T) {
	assertFixJSON(t, `{"a": {"b": {}`, `{"a": {"b": {}}}`)
}

func TestFixJSON_NestedArraysWithNumbers(t *testing.T) {
	assertFixJSON(t, "[1, [2, 3, [", "[1, [2, 3, []]]")
}

func TestFixJSON_NestedArraysWithLiterals(t *testing.T) {
	assertFixJSON(t, "[false, [true, [", "[false, [true, []]]")
}

func TestFixJSON_NestedObjects(t *testing.T) {
	assertFixJSON(t, `{"key": {"subKey":`, `{"key": {}}`)
}

func TestFixJSON_NestedObjectsWithNumbers(t *testing.T) {
	assertFixJSON(t, `{"key": 123, "key2": {"subKey":`, `{"key": 123, "key2": {}}`)
}

func TestFixJSON_NestedObjectsWithLiterals(t *testing.T) {
	assertFixJSON(t, `{"key": null, "key2": {"subKey":`, `{"key": null, "key2": {}}`)
}

func TestFixJSON_ArraysWithinObjects(t *testing.T) {
	assertFixJSON(t, `{"key": [1, 2, {`, `{"key": [1, 2, {}]}`)
}

func TestFixJSON_ObjectsWithinArrays(t *testing.T) {
	assertFixJSON(t, `[1, 2, {"key": "value",`, `[1, 2, {"key": "value"}]`)
}

func TestFixJSON_NestedArraysAndObjects(t *testing.T) {
	assertFixJSON(t, `{"a": {"b": ["c", {"d": "e",`, `{"a": {"b": ["c", {"d": "e"}]}}`)
}

func TestFixJSON_DeeplyNestedObjects(t *testing.T) {
	assertFixJSON(t, `{"a": {"b": {"c": {"d":`, `{"a": {"b": {"c": {}}}}`)
}

func TestFixJSON_PotentialNestedArraysOrObjects(t *testing.T) {
	assertFixJSON(t, `{"a": 1, "b": [`, `{"a": 1, "b": []}`)
	assertFixJSON(t, `{"a": 1, "b": {`, `{"a": 1, "b": {}}`)
	assertFixJSON(t, `{"a": 1, "b": "`, `{"a": 1, "b": ""}`)
}

func TestFixJSON_ComplexNesting1(t *testing.T) {
	input := strings.Join([]string{
		"{",
		`  "a": [`,
		"    {",
		`      "a1": "v1",`,
		`      "a2": "v2",`,
		`      "a3": "v3"`,
		"    }",
		"  ],",
		`  "b": [`,
		"    {",
		`      "b1": "n`,
	}, "\n")

	expected := strings.Join([]string{
		"{",
		`  "a": [`,
		"    {",
		`      "a1": "v1",`,
		`      "a2": "v2",`,
		`      "a3": "v3"`,
		"    }",
		"  ],",
		`  "b": [`,
		"    {",
		`      "b1": "n"}]}`,
	}, "\n")

	assertFixJSON(t, input, expected)
}

func TestFixJSON_EmptyObjectsInsideNestedObjectsAndArrays(t *testing.T) {
	assertFixJSON(t,
		`{"type":"div","children":[{"type":"Card","props":{}`,
		`{"type":"div","children":[{"type":"Card","props":{}}]}`,
	)
}
