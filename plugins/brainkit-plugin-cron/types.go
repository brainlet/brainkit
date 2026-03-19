package main

import "encoding/json"

// ── Tool input/output types ──

type CreateInput struct {
	Name     string     `json:"name"`
	Schedule string     `json:"schedule"` // interval string: "5s", "1m", "1h"
	Action   CronAction `json:"action"`
}

type CreateOutput struct {
	Created string `json:"created"`
}

type RemoveInput struct {
	Name string `json:"name"`
}

type RemoveOutput struct {
	Removed string `json:"removed"`
}

type PauseInput struct {
	Name string `json:"name"`
}

type PauseOutput struct {
	Paused string `json:"paused"`
}

type ResumeInput struct {
	Name string `json:"name"`
}

type ResumeOutput struct {
	Resumed string `json:"resumed"`
}

type ListInput struct{}

type ListOutput struct {
	Jobs []CronJobInfo `json:"jobs"`
}

type CronJobInfo struct {
	Name     string     `json:"name"`
	Schedule string     `json:"schedule"`
	Action   CronAction `json:"action"`
	Paused   bool       `json:"paused"`
}

type CronAction struct {
	Type  string          `json:"type"`  // "event" | "tool"
	Topic string          `json:"topic"` // bus topic for "event", tool name for "tool"
	Data  json.RawMessage `json:"data,omitempty"`
}

// ── Event type ──

type CronFiredEvent struct {
	JobName  string `json:"jobName"`
	Schedule string `json:"schedule"`
	Action   string `json:"action"`
}

func (CronFiredEvent) BusTopic() string { return "cron.fired" }
