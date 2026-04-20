package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// newInspectCmd creates the `brainkit inspect <subject>` verb.
// Each subject maps onto one bus topic on the server via
// POST /api/bus. Renders human-readable tables by default; pass
// --json for raw payload output.
func newInspectCmd() *cobra.Command {
	var endpoint string
	c := &cobra.Command{
		Use:   "inspect <subject>",
		Short: "Inspect state of a running brainkit server",
		Long: `Inspect queries a running server for a specific subject and
prints the result. Subjects:

  health     — overall health status (kit.health)
  packages   — deployed packages (package.list)
  plugins    — running plugins (plugin.list)
  schedules  — active schedules (schedules.list)
  agents     — registered agents (agents.list)
  tools      — registered tools (tools.list)
  workflows  — registered workflows (workflow.list)
  resources  — every registered tool + agent + workflow, grouped
  audit      — recent audit events (audit.query)
  traces     — recent traces (trace.list)
  routes     — HTTP gateway routes (gateway.http.route.list)

Use --json to emit the raw payload instead of the table
rendering.`,
		Args: cobra.ExactArgs(1),
		ValidArgs: []string{
			"health", "packages", "plugins", "schedules",
			"agents", "tools", "workflows", "resources",
			"audit", "traces", "routes",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			subject := args[0]

			ctx, cancel := withTimeout(cmd.Context())
			defer cancel()

			client := newBusClient(endpoint)

			if subject == "resources" {
				return renderResources(ctx, cmd, client)
			}

			spec, ok := inspectSubjects[subject]
			if !ok {
				return fmt.Errorf("unknown subject %q — see `brainkit inspect -h`", subject)
			}
			payload, err := client.call(ctx, spec.topic, json.RawMessage(spec.payload))
			if err != nil {
				return err
			}
			if jsonOutput {
				return writeJSONPretty(cmd.OutOrStdout(), payload)
			}
			return spec.render(cmd.OutOrStdout(), payload)
		},
	}
	c.Flags().StringVarP(&endpoint, "endpoint", "e", "", "server endpoint (default http://127.0.0.1:8080)")
	return c
}

// inspectSubject binds a CLI subject to its bus topic + payload +
// renderer.
type inspectSubject struct {
	topic   string
	payload string
	render  func(io.Writer, json.RawMessage) error
}

var inspectSubjects = map[string]inspectSubject{
	"health":    {topic: "kit.health", payload: "{}", render: renderHealth},
	"packages":  {topic: "package.list", payload: "{}", render: renderPackages},
	"plugins":   {topic: "plugin.list", payload: "{}", render: renderPlugins},
	"schedules": {topic: "schedules.list", payload: "{}", render: renderSchedules},
	"agents":    {topic: "agents.list", payload: "{}", render: renderAgents},
	"tools":     {topic: "tools.list", payload: "{}", render: renderTools},
	"workflows": {topic: "workflow.list", payload: "{}", render: renderWorkflows},
	"audit":     {topic: "audit.query", payload: `{"limit":20}`, render: renderAudit},
	"traces":    {topic: "trace.list", payload: `{"limit":20}`, render: renderTraces},
	"routes":    {topic: "gateway.http.route.list", payload: "{}", render: renderRoutes},
}

func renderHealth(w io.Writer, payload json.RawMessage) error {
	var env struct {
		Health json.RawMessage `json:"health"`
	}
	_ = json.Unmarshal(payload, &env)
	body := env.Health
	if len(body) == 0 {
		body = payload
	}
	var shape struct {
		Status string `json:"status"`
		Uptime any    `json:"uptime,omitempty"`
	}
	if err := json.Unmarshal(body, &shape); err != nil {
		return writeJSONPretty(w, payload)
	}
	tw := newTW(w)
	fmt.Fprintf(tw, "STATUS\t%s\n", nonEmpty(shape.Status, "(unknown)"))
	if shape.Uptime != nil {
		fmt.Fprintf(tw, "UPTIME\t%v\n", shape.Uptime)
	}
	return tw.Flush()
}

func renderPackages(w io.Writer, payload json.RawMessage) error {
	var resp struct {
		Packages []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
			Source  string `json:"source"`
			Status  string `json:"status"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(payload, &resp); err != nil {
		return writeJSONPretty(w, payload)
	}
	tw := newTW(w)
	fmt.Fprintln(tw, "NAME\tVERSION\tSOURCE\tSTATUS")
	for _, p := range resp.Packages {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			nonEmpty(p.Name, "-"),
			nonEmpty(p.Version, "-"),
			nonEmpty(p.Source, "-"),
			nonEmpty(p.Status, "-"))
	}
	return tw.Flush()
}

func renderPlugins(w io.Writer, payload json.RawMessage) error {
	var resp struct {
		Plugins []struct {
			Name     string `json:"name"`
			Version  string `json:"version"`
			PID      int    `json:"pid"`
			Identity string `json:"identity"`
		} `json:"plugins"`
	}
	if err := json.Unmarshal(payload, &resp); err != nil {
		return writeJSONPretty(w, payload)
	}
	tw := newTW(w)
	fmt.Fprintln(tw, "NAME\tVERSION\tPID\tIDENTITY")
	for _, p := range resp.Plugins {
		fmt.Fprintf(tw, "%s\t%s\t%d\t%s\n",
			nonEmpty(p.Name, "-"),
			nonEmpty(p.Version, "-"),
			p.PID,
			nonEmpty(p.Identity, "-"))
	}
	return tw.Flush()
}

func renderSchedules(w io.Writer, payload json.RawMessage) error {
	var resp struct {
		Schedules []struct {
			ID         string `json:"id"`
			Expression string `json:"expression"`
			Topic      string `json:"topic"`
			Source     string `json:"source"`
		} `json:"schedules"`
	}
	if err := json.Unmarshal(payload, &resp); err != nil {
		return writeJSONPretty(w, payload)
	}
	tw := newTW(w)
	fmt.Fprintln(tw, "ID\tEXPRESSION\tTOPIC\tSOURCE")
	for _, s := range resp.Schedules {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			nonEmpty(s.ID, "-"),
			nonEmpty(s.Expression, "-"),
			nonEmpty(s.Topic, "-"),
			nonEmpty(s.Source, "-"))
	}
	return tw.Flush()
}

func renderAgents(w io.Writer, payload json.RawMessage) error {
	var resp struct {
		Agents []struct {
			Name   string `json:"name"`
			Source string `json:"source"`
			Status string `json:"status"`
		} `json:"agents"`
	}
	if err := json.Unmarshal(payload, &resp); err != nil {
		return writeJSONPretty(w, payload)
	}
	tw := newTW(w)
	fmt.Fprintln(tw, "NAME\tSOURCE\tSTATUS")
	for _, a := range resp.Agents {
		fmt.Fprintf(tw, "%s\t%s\t%s\n",
			nonEmpty(a.Name, "-"),
			nonEmpty(a.Source, "-"),
			nonEmpty(a.Status, "-"))
	}
	return tw.Flush()
}

func renderAudit(w io.Writer, payload json.RawMessage) error {
	var resp struct {
		Events []struct {
			Timestamp string `json:"timestamp"`
			Type      string `json:"type"`
			Category  string `json:"category"`
			Source    string `json:"source"`
		} `json:"events"`
	}
	if err := json.Unmarshal(payload, &resp); err != nil {
		return writeJSONPretty(w, payload)
	}
	tw := newTW(w)
	fmt.Fprintln(tw, "TIMESTAMP\tCATEGORY\tTYPE\tSOURCE")
	for _, e := range resp.Events {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			nonEmpty(e.Timestamp, "-"),
			nonEmpty(e.Category, "-"),
			nonEmpty(e.Type, "-"),
			nonEmpty(e.Source, "-"))
	}
	return tw.Flush()
}

func renderTraces(w io.Writer, payload json.RawMessage) error {
	var resp struct {
		Traces []struct {
			TraceID string `json:"traceId"`
			Name    string `json:"name"`
			Source  string `json:"source"`
			Status  string `json:"status"`
			Span    int    `json:"spanCount"`
		} `json:"traces"`
	}
	if err := json.Unmarshal(payload, &resp); err != nil {
		return writeJSONPretty(w, payload)
	}
	sort.Slice(resp.Traces, func(i, j int) bool {
		return resp.Traces[i].TraceID < resp.Traces[j].TraceID
	})
	tw := newTW(w)
	fmt.Fprintln(tw, "TRACE ID\tNAME\tSOURCE\tSTATUS\tSPANS")
	for _, t := range resp.Traces {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d\n",
			nonEmpty(t.TraceID, "-"),
			nonEmpty(t.Name, "-"),
			nonEmpty(t.Source, "-"),
			nonEmpty(t.Status, "-"),
			t.Span)
	}
	return tw.Flush()
}

func renderRoutes(w io.Writer, payload json.RawMessage) error {
	var resp struct {
		Routes []struct {
			Method string `json:"method"`
			Path   string `json:"path"`
			Topic  string `json:"topic"`
			Type   string `json:"type"`
			Owner  string `json:"owner"`
		} `json:"routes"`
	}
	if err := json.Unmarshal(payload, &resp); err != nil {
		return writeJSONPretty(w, payload)
	}
	tw := newTW(w)
	fmt.Fprintln(tw, "METHOD\tPATH\tTOPIC\tTYPE\tOWNER")
	for _, r := range resp.Routes {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			nonEmpty(r.Method, "-"),
			nonEmpty(r.Path, "-"),
			nonEmpty(r.Topic, "-"),
			nonEmpty(r.Type, "-"),
			nonEmpty(r.Owner, "-"))
	}
	return tw.Flush()
}

func newTW(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
}

func nonEmpty(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

func renderTools(w io.Writer, payload json.RawMessage) error {
	var resp struct {
		Tools []struct {
			Name        string `json:"name"`
			ShortName   string `json:"shortName"`
			Description string `json:"description"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(payload, &resp); err != nil {
		return writeJSONPretty(w, payload)
	}
	sort.Slice(resp.Tools, func(i, j int) bool {
		return resp.Tools[i].Name < resp.Tools[j].Name
	})
	tw := newTW(w)
	fmt.Fprintln(tw, "NAME\tSHORT\tDESCRIPTION")
	for _, t := range resp.Tools {
		fmt.Fprintf(tw, "%s\t%s\t%s\n",
			nonEmpty(t.Name, "-"),
			nonEmpty(t.ShortName, "-"),
			truncate(nonEmpty(t.Description, "-"), 60))
	}
	return tw.Flush()
}

func renderWorkflows(w io.Writer, payload json.RawMessage) error {
	var resp struct {
		Workflows []struct {
			Name        string `json:"name"`
			Source      string `json:"source"`
			Description string `json:"description"`
		} `json:"workflows"`
	}
	if err := json.Unmarshal(payload, &resp); err != nil {
		return writeJSONPretty(w, payload)
	}
	sort.Slice(resp.Workflows, func(i, j int) bool {
		return resp.Workflows[i].Name < resp.Workflows[j].Name
	})
	tw := newTW(w)
	fmt.Fprintln(tw, "NAME\tSOURCE\tDESCRIPTION")
	for _, wf := range resp.Workflows {
		fmt.Fprintf(tw, "%s\t%s\t%s\n",
			nonEmpty(wf.Name, "-"),
			nonEmpty(wf.Source, "-"),
			truncate(nonEmpty(wf.Description, "-"), 60))
	}
	return tw.Flush()
}

// renderResources fans out to tools.list + agents.list + workflow.list
// and prints each section under its own heading. The old CLI's
// `brainkit resources` verb.
func renderResources(ctx context.Context, cmd *cobra.Command, client *busClient) error {
	out := cmd.OutOrStdout()

	sections := []struct {
		heading string
		topic   string
		render  func(io.Writer, json.RawMessage) error
	}{
		{"Tools", "tools.list", renderTools},
		{"Agents", "agents.list", renderAgents},
		{"Workflows", "workflow.list", renderWorkflows},
	}

	for i, s := range sections {
		payload, err := client.call(ctx, s.topic, json.RawMessage("{}"))
		if err != nil {
			fmt.Fprintf(out, "%s: error — %v\n", s.heading, err)
			continue
		}
		if jsonOutput {
			fmt.Fprintf(out, "%s:\n", s.heading)
			if err := writeJSONPretty(out, payload); err != nil {
				return err
			}
			continue
		}
		fmt.Fprintf(out, "%s\n", s.heading)
		if err := s.render(out, payload); err != nil {
			return err
		}
		if i < len(sections)-1 {
			fmt.Fprintln(out)
		}
	}
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	return s[:max-1] + "…"
}
