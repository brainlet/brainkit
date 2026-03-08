package graymatter

import "testing"

func TestSections(t *testing.T) {
	input := `---yaml
title: I'm front matter
---

This is an excerpt.
---

---aaa
title: First section
---

Section one.

---bbb
title: Second section
---

Part 1.

---

Part 2.

---

Part 3.

---ccc
title: Third section
---

Section three.
`

	file, err := Parse(input, Options{Excerpt: true, Sections: true})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if file.Excerpt != "\nThis is an excerpt.\n" {
		t.Fatalf("unexpected excerpt: %q", file.Excerpt)
	}
	if file.Content != "\nThis is an excerpt.\n---\n" {
		t.Fatalf("unexpected content: %q", file.Content)
	}
	if len(file.Sections) != 3 {
		t.Fatalf("expected 3 sections, got %d", len(file.Sections))
	}

	if file.Sections[0].Key != "aaa" || file.Sections[0].Data != "title: First section" {
		t.Fatalf("unexpected first section: %#v", file.Sections[0])
	}
	if file.Sections[1].Key != "bbb" || file.Sections[1].Content != "\nPart 1.\n\n---\n\nPart 2.\n\n---\n\nPart 3.\n" {
		t.Fatalf("unexpected second section: %#v", file.Sections[1])
	}
	if file.Sections[2].Key != "ccc" || file.Sections[2].Data != "title: Third section" {
		t.Fatalf("unexpected third section: %#v", file.Sections[2])
	}
}
