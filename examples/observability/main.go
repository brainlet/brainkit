// Command observability demonstrates the audit + tracing query
// surfaces. Wires modules/audit + modules/tracing on a Kit,
// generates a handful of events by deploying + calling, then
// queries both stores and pretty-prints the results.
//
// Run from the repo root:
//
//	go run ./examples/observability
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/modules/audit"
	auditstores "github.com/brainlet/brainkit/modules/audit/stores"
	"github.com/brainlet/brainkit/modules/tracing"
	"github.com/brainlet/brainkit/sdk"

	_ "modernc.org/sqlite"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("observability: %v", err)
	}
}

func run() error {
	tmp, err := os.MkdirTemp("", "brainkit-observability-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	auditStore, err := auditstores.NewSQLite(filepath.Join(tmp, "audit.db"))
	if err != nil {
		return fmt.Errorf("open audit store: %w", err)
	}

	traceDB, err := sql.Open("sqlite", filepath.Join(tmp, "traces.db"))
	if err != nil {
		return fmt.Errorf("open trace db: %w", err)
	}
	traceStore, err := tracing.NewSQLiteTraceStore(traceDB)
	if err != nil {
		return fmt.Errorf("open trace store: %w", err)
	}

	kit, err := brainkit.New(brainkit.Config{
		Namespace:       "observability-demo",
		Transport:       brainkit.Memory(),
		FSRoot:          tmp,
		TraceSampleRate: 1.0,
		Modules: []brainkit.Module{
			audit.NewModule(audit.Config{Store: auditStore}),
			tracing.New(tracing.Config{Store: traceStore}),
		},
	})
	if err != nil {
		return fmt.Errorf("new kit: %w", err)
	}
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Generate events: deploy a handler, call it twice.
	tsCode := `
		bus.on("ping", (msg) => { msg.reply({pong: true}); });
	`
	if _, err := kit.Deploy(ctx, brainkit.PackageInline("observability-demo", "obs.ts", tsCode)); err != nil {
		return fmt.Errorf("deploy: %w", err)
	}
	for i := 0; i < 2; i++ {
		_, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](kit, ctx, sdk.CustomMsg{
			Topic:   "ts.observability-demo.ping",
			Payload: json.RawMessage(`{}`),
		}, brainkit.WithCallTimeout(3*time.Second))
		if err != nil {
			return fmt.Errorf("call: %w", err)
		}
	}

	// Short pause so events + spans commit before we query.
	time.Sleep(250 * time.Millisecond)

	// ── Audit ──
	fmt.Println("audit.query (last 20 events):")
	queryResp, err := brainkit.CallAuditQuery(kit, ctx, sdk.AuditQueryMsg{Limit: 20},
		brainkit.WithCallTimeout(3*time.Second))
	if err != nil {
		return fmt.Errorf("audit.query: %w", err)
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "  TIMESTAMP\tCATEGORY\tTYPE\tSOURCE")
	for _, e := range queryResp.Events {
		fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n",
			e.Timestamp.Format(time.RFC3339),
			defaultIfEmpty(e.Category, "-"),
			defaultIfEmpty(e.Type, "-"),
			defaultIfEmpty(e.Source, "-"))
	}
	tw.Flush()

	fmt.Println()
	fmt.Println("audit.stats:")
	statsResp, err := brainkit.CallAuditStats(kit, ctx, sdk.AuditStatsMsg{},
		brainkit.WithCallTimeout(3*time.Second))
	if err != nil {
		return fmt.Errorf("audit.stats: %w", err)
	}
	fmt.Printf("  total=%d\n", statsResp.TotalEvents)
	for cat, n := range statsResp.EventsByCategory {
		fmt.Printf("  %s: %d\n", cat, n)
	}

	// ── Traces ──
	fmt.Println()
	fmt.Println("trace.list (last 20):")
	traceResp, err := brainkit.CallTraceList(kit, ctx, sdk.TraceListMsg{Limit: 20},
		brainkit.WithCallTimeout(3*time.Second))
	if err != nil {
		return fmt.Errorf("trace.list: %w", err)
	}
	// traceResp.Traces is a json.RawMessage — decode to TraceSummary.
	var traces []struct {
		TraceID   string `json:"traceId"`
		Name      string `json:"name"`
		Status    string `json:"status"`
		SpanCount int    `json:"spanCount"`
	}
	if err := json.Unmarshal(traceResp.Traces, &traces); err != nil {
		return fmt.Errorf("decode trace list: %w", err)
	}
	if len(traces) == 0 {
		fmt.Println("  (no traces recorded yet — trace sampling may be off or the handler paths haven't been instrumented)")
	} else {
		for _, tr := range traces {
			id := tr.TraceID
			if len(id) > 8 {
				id = id[:8]
			}
			fmt.Printf("  %s  %s  %s  %d span(s)\n", id, tr.Name, tr.Status, tr.SpanCount)
		}
	}
	return nil
}

func defaultIfEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
