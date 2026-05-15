package jsbridge

import "testing"

func TestIntlDateTimeFormatFormatToPartsTimeZone(t *testing.T) {
	b := newTestBridge(t, Intl())
	got := evalString(t, b, `
		const d = new Date("2026-01-02T14:05:06Z");
		const parts = new Intl.DateTimeFormat("en-US", {
			timeZone: "America/New_York",
			year: "numeric",
			month: "numeric",
			day: "numeric",
			hour: "numeric",
			minute: "numeric",
			second: "numeric",
			hour12: false,
		}).formatToParts(d);
		JSON.stringify(Object.fromEntries(parts.filter((p) => p.type !== "literal").map((p) => [p.type, p.value])));
	`)
	if got != `{"month":"1","day":"2","year":"2026","hour":"09","minute":"05","second":"06"}` {
		t.Fatalf("formatToParts = %s", got)
	}
}

func TestIntlDateTimeFormatFormatUsesTimeZone(t *testing.T) {
	b := newTestBridge(t, Intl())
	got := evalString(t, b, `
		new Intl.DateTimeFormat("en-US", { timeZone: "UTC" }).format(new Date("2026-01-02T14:05:06Z"));
	`)
	if got != "2026-01-02 14:05" {
		t.Fatalf("format = %q", got)
	}
}

func TestIntlDateTimeFormatInvalidTimeZone(t *testing.T) {
	b := newTestBridge(t, Intl())
	got := evalString(t, b, `
		let message = "";
		try {
			new Intl.DateTimeFormat("en-US", { timeZone: "Not/AZone" }).formatToParts(new Date("2026-01-02T14:05:06Z"));
		} catch (err) {
			message = String(err && err.message || err);
		}
		message;
	`)
	if got == "" {
		t.Fatal("expected invalid timezone to throw")
	}
}
