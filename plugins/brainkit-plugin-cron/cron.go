package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
)

// Package-global state — one plugin process = one instance.
var (
	client sdk.Client
	crons  = make(map[string]*cronEntry)
	mu     sync.Mutex
)

type cronEntry struct {
	Info   CronJobInfo
	ticker *time.Ticker
	done   chan struct{}
}

// ── Lifecycle ──

func onStart(c sdk.Client) error {
	client = c

	state, err := c.GetState(context.Background(), "crons")
	if err != nil {
		log.Printf("[cron] failed to load state: %v", err)
		return nil
	}
	if state == "" {
		return nil
	}

	var saved []CronJobInfo
	if err := json.Unmarshal([]byte(state), &saved); err != nil {
		log.Printf("[cron] failed to parse saved state: %v", err)
		return nil
	}

	for _, info := range saved {
		if !info.Paused {
			startCron(info)
		} else {
			mu.Lock()
			crons[info.Name] = &cronEntry{Info: info}
			mu.Unlock()
		}
	}

	log.Printf("[cron] loaded %d jobs from state", len(saved))
	return nil
}

func onStop() error {
	mu.Lock()
	defer mu.Unlock()

	for _, entry := range crons {
		if entry.done != nil {
			close(entry.done)
		}
	}

	persistState()
	return nil
}

// ── Tool handlers ──

func handleCreate(_ context.Context, _ sdk.Client, in CreateInput) (CreateOutput, error) {
	if in.Name == "" {
		return CreateOutput{}, fmt.Errorf("name is required")
	}
	if in.Schedule == "" {
		return CreateOutput{}, fmt.Errorf("schedule is required")
	}

	mu.Lock()
	defer mu.Unlock()

	if _, exists := crons[in.Name]; exists {
		return CreateOutput{}, fmt.Errorf("cron job %q already exists", in.Name)
	}

	info := CronJobInfo{
		Name:     in.Name,
		Schedule: in.Schedule,
		Action:   in.Action,
		Paused:   false,
	}

	startCronLocked(info)
	persistState()

	return CreateOutput{Created: in.Name}, nil
}

func handleList(_ context.Context, _ sdk.Client, _ ListInput) (ListOutput, error) {
	mu.Lock()
	defer mu.Unlock()

	var jobs []CronJobInfo
	for _, entry := range crons {
		jobs = append(jobs, entry.Info)
	}
	return ListOutput{Jobs: jobs}, nil
}

func handleRemove(_ context.Context, _ sdk.Client, in RemoveInput) (RemoveOutput, error) {
	if in.Name == "" {
		return RemoveOutput{}, fmt.Errorf("name is required")
	}

	mu.Lock()
	defer mu.Unlock()

	entry, exists := crons[in.Name]
	if !exists {
		return RemoveOutput{}, fmt.Errorf("cron job %q not found", in.Name)
	}

	if entry.done != nil {
		close(entry.done)
	}
	delete(crons, in.Name)
	persistState()

	return RemoveOutput{Removed: in.Name}, nil
}

func handlePause(_ context.Context, _ sdk.Client, in PauseInput) (PauseOutput, error) {
	if in.Name == "" {
		return PauseOutput{}, fmt.Errorf("name is required")
	}

	mu.Lock()
	defer mu.Unlock()

	entry, exists := crons[in.Name]
	if !exists {
		return PauseOutput{}, fmt.Errorf("cron job %q not found", in.Name)
	}
	if entry.Info.Paused {
		return PauseOutput{}, fmt.Errorf("cron job %q is already paused", in.Name)
	}

	if entry.done != nil {
		close(entry.done)
		entry.done = nil
	}
	if entry.ticker != nil {
		entry.ticker.Stop()
		entry.ticker = nil
	}
	entry.Info.Paused = true
	persistState()

	return PauseOutput{Paused: in.Name}, nil
}

func handleResume(_ context.Context, _ sdk.Client, in ResumeInput) (ResumeOutput, error) {
	if in.Name == "" {
		return ResumeOutput{}, fmt.Errorf("name is required")
	}

	mu.Lock()
	defer mu.Unlock()

	entry, exists := crons[in.Name]
	if !exists {
		return ResumeOutput{}, fmt.Errorf("cron job %q not found", in.Name)
	}
	if !entry.Info.Paused {
		return ResumeOutput{}, fmt.Errorf("cron job %q is not paused", in.Name)
	}

	entry.Info.Paused = false
	startCronLocked(entry.Info)
	persistState()

	return ResumeOutput{Resumed: in.Name}, nil
}

// ── Internal ──

func startCron(info CronJobInfo) {
	mu.Lock()
	defer mu.Unlock()
	startCronLocked(info)
}

func startCronLocked(info CronJobInfo) {
	dur, err := parseSchedule(info.Schedule)
	if err != nil {
		log.Printf("[cron] invalid schedule %q for %q: %v", info.Schedule, info.Name, err)
		return
	}

	entry := &cronEntry{
		Info:   info,
		ticker: time.NewTicker(dur),
		done:   make(chan struct{}),
	}
	crons[info.Name] = entry

	go func() {
		for {
			select {
			case <-entry.ticker.C:
				fireCron(entry.Info)
			case <-entry.done:
				if entry.ticker != nil {
					entry.ticker.Stop()
				}
				return
			}
		}
	}()
}

func fireCron(info CronJobInfo) {
	ctx := context.Background()

	// Emit typed event
	client.Send(ctx, CronFiredEvent{
		JobName:  info.Name,
		Schedule: info.Schedule,
		Action:   info.Action.Type,
	})

	// Execute the action
	switch info.Action.Type {
	case "event":
		if info.Action.Topic != "" {
			payload := info.Action.Data
			if payload == nil {
				payload = json.RawMessage(`{}`)
			}
			client.Send(ctx, &rawBusMessage{
				topic:   info.Action.Topic,
				payload: payload,
			})
		}
	case "tool":
		if info.Action.Topic != "" {
			input := info.Action.Data
			if input == nil {
				input = json.RawMessage(`{}`)
			}
			client.Ask(ctx, &rawBusMessage{
				topic:   "tools.call",
				payload: mustMarshal(map[string]any{"name": info.Action.Topic, "input": json.RawMessage(input)}),
			}, func(msg messages.Message) {
				log.Printf("[cron:%s] tool %q result: %s", info.Name, info.Action.Topic, string(msg.Payload))
			})
		}
	}
}

// rawBusMessage implements BusMessage for dynamic topics.
type rawBusMessage struct {
	topic   string
	payload json.RawMessage
}

func (m *rawBusMessage) BusTopic() string                { return m.topic }
func (m *rawBusMessage) MarshalJSON() ([]byte, error)    { return m.payload, nil }

func mustMarshal(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func persistState() {
	var infos []CronJobInfo
	for _, entry := range crons {
		infos = append(infos, entry.Info)
	}
	data, _ := json.Marshal(infos)
	if client != nil {
		client.SetState(context.Background(), "crons", string(data))
	}
}

func parseSchedule(s string) (time.Duration, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid schedule %q: must be a Go duration (e.g. 5s, 1m, 1h): %w", s, err)
	}
	if d < 1*time.Second {
		return 0, fmt.Errorf("schedule %q too short: minimum 1s", s)
	}
	return d, nil
}
