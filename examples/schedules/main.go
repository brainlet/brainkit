// Command schedules demonstrates cron-style scheduled bus
// messages. Wires modules/schedules onto a persistent SQLite
// store, deploys a .ts handler that ticks, schedules a cron
// expression that fires the handler every 2 seconds, collects
// 3 ticks, cancels the schedule, exits.
//
// Run from the repo root:
//
//	go run ./examples/schedules
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/brainlet/brainkit"
	schedulesmod "github.com/brainlet/brainkit/modules/schedules"
	"github.com/brainlet/brainkit/sdk"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("schedules: %v", err)
	}
}

func run() error {
	tmp, err := os.MkdirTemp("", "brainkit-schedules-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	store, err := brainkit.NewSQLiteStore(filepath.Join(tmp, "kit.db"))
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "schedules-demo",
		Transport: brainkit.Memory(),
		FSRoot:    tmp,
		Store:     store,
		Modules: []brainkit.Module{
			schedulesmod.NewModule(schedulesmod.Config{Store: store}),
		},
	})
	if err != nil {
		return fmt.Errorf("new kit: %w", err)
	}
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Deploy a .ts that listens for the scheduled topic and
	// replies with a counter increment.
	tsCode := `
		let ticks = 0;
		bus.on("heartbeat", (msg) => {
			ticks++;
			console.log("tick", ticks, "at", new Date().toISOString());
		});
	`
	if _, err := kit.Deploy(ctx, brainkit.PackageInline("heartbeat-demo", "hb.ts", tsCode)); err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	// Subscribe to the scheduled topic on the Go side so we can
	// count ticks independently.
	var ticks atomic.Int32
	ch := make(chan struct{}, 8)
	unsub, err := kit.SubscribeRaw(ctx, "ts.heartbeat-demo.heartbeat", func(msg sdk.Message) {
		n := ticks.Add(1)
		fmt.Printf("received tick %d\n", n)
		if n <= 3 {
			select {
			case ch <- struct{}{}:
			default:
			}
		}
	})
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}
	defer unsub()

	// Schedule: fire ts.heartbeat-demo.heartbeat every 2 seconds.
	created, err := brainkit.CallScheduleCreate(kit, ctx, sdk.ScheduleCreateMsg{
		Expression: "every 2s",
		Topic:      "ts.heartbeat-demo.heartbeat",
		Payload:    json.RawMessage(`{}`),
	}, brainkit.WithCallTimeout(5*time.Second))
	if err != nil {
		return fmt.Errorf("schedule create: %w", err)
	}
	fmt.Printf("scheduled heartbeat every 2s (id=%s)\n", created.ID)

	// Wait for 3 ticks.
	for i := 0; i < 3; i++ {
		select {
		case <-ch:
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for tick %d", i+1)
		}
	}

	// Cancel the schedule.
	_, err = brainkit.CallScheduleCancel(kit, ctx, sdk.ScheduleCancelMsg{ID: created.ID},
		brainkit.WithCallTimeout(3*time.Second))
	if err != nil {
		return fmt.Errorf("schedule cancel: %w", err)
	}
	fmt.Printf("cancelled schedule %s\n", created.ID)

	return nil
}
