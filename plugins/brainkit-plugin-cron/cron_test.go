package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseSchedule_ValidDurations(t *testing.T) {
	cases := []struct {
		input string
		want  time.Duration
	}{
		{"5s", 5 * time.Second},
		{"1m", 1 * time.Minute},
		{"30m", 30 * time.Minute},
		{"1h", 1 * time.Hour},
		{"24h", 24 * time.Hour},
	}

	for _, tc := range cases {
		d, err := parseSchedule(tc.input)
		if err != nil {
			t.Errorf("parseSchedule(%q): unexpected error: %v", tc.input, err)
			continue
		}
		if d != tc.want {
			t.Errorf("parseSchedule(%q) = %v, want %v", tc.input, d, tc.want)
		}
	}
}

func TestParseSchedule_Invalid(t *testing.T) {
	cases := []string{
		"",
		"invalid",
		"* * * * *",
		"100ns",
		"500ms",
	}

	for _, input := range cases {
		_, err := parseSchedule(input)
		if err == nil {
			t.Errorf("parseSchedule(%q): expected error, got nil", input)
		}
	}
}

func TestCronJobInfoSerialization(t *testing.T) {
	info := CronJobInfo{
		Name:     "test-job",
		Schedule: "5m",
		Action:   CronAction{Type: "event", Topic: "data.sync"},
		Paused:   false,
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatal(err)
	}

	var decoded CronJobInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.Name != info.Name || decoded.Schedule != info.Schedule {
		t.Errorf("round-trip failed: got %+v", decoded)
	}
}

func TestCronFiredEvent_BusTopic(t *testing.T) {
	e := CronFiredEvent{JobName: "test"}
	if e.BusTopic() != "cron.fired" {
		t.Errorf("BusTopic() = %q, want cron.fired", e.BusTopic())
	}
}
