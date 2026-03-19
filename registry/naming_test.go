package registry

import "testing"

func TestParseToolName_FullyQualified(t *testing.T) {
	owner, pkg, version, tool := ParseToolName("brainlet/cron@1.0.0/create")
	expect := [4]string{"brainlet", "cron", "1.0.0", "create"}
	got := [4]string{owner, pkg, version, tool}
	if got != expect {
		t.Errorf("got %v, want %v", got, expect)
	}
}

func TestParseToolName_NoOwner(t *testing.T) {
	owner, pkg, version, tool := ParseToolName("cron@1.0.0/create")
	expect := [4]string{"", "cron", "1.0.0", "create"}
	got := [4]string{owner, pkg, version, tool}
	if got != expect {
		t.Errorf("got %v, want %v", got, expect)
	}
}

func TestParseToolName_NoVersion(t *testing.T) {
	owner, pkg, version, tool := ParseToolName("brainlet/cron/create")
	expect := [4]string{"brainlet", "cron", "", "create"}
	got := [4]string{owner, pkg, version, tool}
	if got != expect {
		t.Errorf("got %v, want %v", got, expect)
	}
}

func TestParseToolName_Bare(t *testing.T) {
	owner, pkg, version, tool := ParseToolName("cron/create")
	expect := [4]string{"", "cron", "", "create"}
	got := [4]string{owner, pkg, version, tool}
	if got != expect {
		t.Errorf("got %v, want %v", got, expect)
	}
}

func TestParseToolName_ShortNameOnly(t *testing.T) {
	owner, pkg, version, tool := ParseToolName("create")
	expect := [4]string{"", "", "", "create"}
	got := [4]string{owner, pkg, version, tool}
	if got != expect {
		t.Errorf("got %v, want %v", got, expect)
	}
}

func TestParseToolName_ThirdParty(t *testing.T) {
	owner, pkg, version, tool := ParseToolName("acme-corp/postgres@2.1.0/query")
	expect := [4]string{"acme-corp", "postgres", "2.1.0", "query"}
	got := [4]string{owner, pkg, version, tool}
	if got != expect {
		t.Errorf("got %v, want %v", got, expect)
	}
}

func TestSplitVersion(t *testing.T) {
	cases := []struct {
		input   string
		pkg     string
		version string
	}{
		{"cron@1.0.0", "cron", "1.0.0"},
		{"cron", "cron", ""},
		{"postgres@2.1.0-beta.1", "postgres", "2.1.0-beta.1"},
		{"knowledge-graph@1.0.0", "knowledge-graph", "1.0.0"},
	}
	for _, tc := range cases {
		pkg, version := splitVersion(tc.input)
		if pkg != tc.pkg || version != tc.version {
			t.Errorf("splitVersion(%q) = (%q,%q), want (%q,%q)",
				tc.input, pkg, version, tc.pkg, tc.version)
		}
	}
}

func TestComposeName(t *testing.T) {
	name := ComposeName("brainlet", "cron", "1.0.0", "create")
	if name != "brainlet/cron@1.0.0/create" {
		t.Errorf("ComposeName = %q", name)
	}
}

func TestCompareSemver(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "2.0.0", -1},
		{"2.0.0", "1.0.0", 1},
		{"1.1.0", "1.0.0", 1},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "1.0.1", -1},
		{"2.1.0", "2.0.3", 1},
		{"2.0.0-beta.1", "2.0.0", -1},
		{"2.0.0", "2.0.0-beta.1", 1},
		{"2.0.0-alpha", "2.0.0-beta", -1},
		{"1.0.0-rc.1", "1.0.0-rc.1", 0},
	}
	for _, tc := range cases {
		got := CompareSemver(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("CompareSemver(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestIsPrerelease(t *testing.T) {
	if !IsPrerelease("2.0.0-beta.1") {
		t.Error("expected pre-release")
	}
	if IsPrerelease("2.0.0") {
		t.Error("expected not pre-release")
	}
}

func TestIsNewFormat(t *testing.T) {
	if !IsNewFormat("brainlet/cron@1.0.0/create") {
		t.Error("expected new format")
	}
	if IsNewFormat("echo") {
		t.Error("bare name should not be new format")
	}
}
