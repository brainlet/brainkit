package graymatter

import (
	"testing"
)

func TestWindows(t *testing.T) {
	t.Run("should extract YAML front matter", func(t *testing.T) {
		actual, _ := Parse("---\r\nabc: xyz\r\n---")
		data := dataMap(t, actual.Data)
		if data["abc"] != "xyz" {
			t.Errorf("expected abc to be xyz, got %v", data["abc"])
		}
	})

	t.Run("should cache orig string on the orig property", func(t *testing.T) {
		fixture := "---\r\nabc: xyz\r\n---"
		actual, _ := Parse(fixture)
		orig, ok := actual.Orig.([]byte)
		if !ok {
			t.Fatalf("expected orig to be []byte, got %T", actual.Orig)
		}
		if string(orig) != fixture {
			t.Errorf("expected orig to be %q, got %q", fixture, string(orig))
		}
	})

	t.Run("should throw parsing errors", func(t *testing.T) {
		_, err := Parse("---whatever\r\nabc: xyz\r\n---")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("should throw an error when a string is not passed", func(t *testing.T) {
		_, err := Parse("")
		if err != nil {
			t.Error("expected no error for empty string")
		}
	})

	t.Run("should return an object when the string is 0 length", func(t *testing.T) {
		result, _ := Parse("")
		if result.Data == nil {
			t.Error("expected result to not be nil")
		}
	})

	t.Run("should extract YAML front matter and content", func(t *testing.T) {
		fixture := "---\r\nabc: xyz\r\nversion: 2\r\n---\r\n\r\n<span class=\"alert alert-info\">This is an alert</span>\r\n"
		actual, _ := Parse(fixture)
		data := dataMap(t, actual.Data)
		if data["abc"] != "xyz" || data["version"] != 2 {
			t.Errorf("expected data to be {abc: xyz, version: 2}, got %v", actual.Data)
		}
	})

	t.Run("should use a custom delimiter as a string", func(t *testing.T) {
		fixture := "~~~\r\nabc: xyz\r\nversion: 2\r\n~~~\r\n\r\n<span class=\"alert alert-info\">This is an alert</span>\r\n"
		actual, _ := Parse(fixture, Options{Delimiters: "~~~"})
		data := dataMap(t, actual.Data)
		if data["abc"] != "xyz" || data["version"] != 2 {
			t.Errorf("expected data to be {abc: xyz, version: 2}, got %v", actual.Data)
		}
	})

	t.Run("should use custom delimiters as an array", func(t *testing.T) {
		fixture := "~~~\r\nabc: xyz\r\nversion: 2\r\n~~~\r\n\r\n<span class=\"alert alert-info\">This is an alert</span>\r\n"
		actual, _ := Parse(fixture, Options{Delimiters: []string{"~~~"}})
		data := dataMap(t, actual.Data)
		if data["abc"] != "xyz" || data["version"] != 2 {
			t.Errorf("expected data to be {abc: xyz, version: 2}, got %v", actual.Data)
		}
	})

	t.Run("should correctly identify delimiters and ignore strings that look like delimiters", func(t *testing.T) {
		fixture := "---\r\nname: \"troublesome --- value\"\r\n---\r\nhere is some content\r\n"
		actual, _ := Parse(fixture)
		data := dataMap(t, actual.Data)
		if data["name"] != "troublesome --- value" {
			t.Errorf("expected name to be 'troublesome --- value', got %v", data["name"])
		}
	})

	t.Run("should correctly parse a string that only has an opening delimiter", func(t *testing.T) {
		fixture := "---\r\nname: \"troublesome --- value\"\r\n"
		actual, _ := Parse(fixture)
		data := dataMap(t, actual.Data)
		if data["name"] != "troublesome --- value" {
			t.Errorf("expected name to be 'troublesome --- value', got %v", data["name"])
		}
		if actual.Content != "" {
			t.Errorf("expected content to be empty, got %q", actual.Content)
		}
	})

	t.Run("should not try to parse a string has content that looks like front-matter", func(t *testing.T) {
		fixture := "-----------name--------------value\r\nfoo"
		actual, _ := Parse(fixture)
		if len(dataMap(t, actual.Data)) != 0 {
			t.Errorf("expected empty data, got %v", actual.Data)
		}
	})
}
