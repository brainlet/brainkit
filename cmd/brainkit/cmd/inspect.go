package cmd

import (
	"bytes"
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
  modules    — mounted module manifests (kit.modules)
  module ID  — one module manifest with preflight (kit.module.describe)
  lifecycle  — runtime lifecycle/debug counters (kit.lifecycle)
  tools      — registered tools (tools.list)
  workflows  — registered workflows (workflow.list)
  resources  — every registered tool + agent + workflow, grouped
  audit      — recent audit events (audit.query)
  traces     — recent traces (trace.list)
  routes     — HTTP gateway routes (gateway.http.route.list)

Use --json to emit the raw payload instead of the table
rendering.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("requires a subject")
			}
			if args[0] == "module" {
				if len(args) != 2 {
					return fmt.Errorf("inspect module requires a module ID")
				}
				return nil
			}
			if len(args) != 1 {
				return fmt.Errorf("inspect %s does not accept extra arguments", args[0])
			}
			return nil
		},
		ValidArgs: []string{
			"health", "packages", "plugins", "schedules",
			"agents", "modules", "module", "tools", "workflows", "resources",
			"audit", "traces", "routes", "lifecycle",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			subject := args[0]

			ctx, cancel := withTimeout(cmd.Context())
			defer cancel()

			client := newBusClient(endpoint)

			if subject == "resources" {
				return renderResources(ctx, cmd, client)
			}
			if subject == "module" {
				payload, err := json.Marshal(map[string]string{"id": args[1]})
				if err != nil {
					return err
				}
				resp, err := client.call(ctx, "kit.module.describe", json.RawMessage(payload))
				if err != nil {
					return err
				}
				if jsonOutput {
					return writeJSONPretty(cmd.OutOrStdout(), resp)
				}
				return renderModule(cmd.OutOrStdout(), resp)
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
	"modules":   {topic: "kit.modules", payload: "{}", render: renderModules},
	"lifecycle": {topic: "kit.lifecycle", payload: "{}", render: renderLifecycle},
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

func renderLifecycle(w io.Writer, payload json.RawMessage) error {
	var resp struct {
		Lifecycle struct {
			Runtime struct {
				RuntimeID      string `json:"runtimeId"`
				Namespace      string `json:"namespace"`
				CallerID       string `json:"callerId"`
				MountedModules int    `json:"mountedModules"`
				ActiveHandlers int64  `json:"activeHandlers"`
				Draining       bool   `json:"draining"`
				Provider       struct {
					AIProviders      int   `json:"aiProviders"`
					VectorStores     int   `json:"vectorStores"`
					Storages         int   `json:"storages"`
					Closing          bool  `json:"closing"`
					Closed           bool  `json:"closed"`
					ActiveProbes     int64 `json:"activeProbes"`
					ActiveOperations int64 `json:"activeOperations"`
				} `json:"provider"`
				Storage struct {
					Closing      bool     `json:"closing"`
					BridgeCount  int      `json:"bridgeCount"`
					BridgeNames  []string `json:"bridgeNames"`
					ActiveCloses int      `json:"activeCloses"`
				} `json:"storage"`
				Transport struct {
					Kind                   string `json:"kind"`
					OwnsTransport          bool   `json:"ownsTransport"`
					ActiveSubscriptions    int64  `json:"activeSubscriptions"`
					ActiveStreamHeartbeats int    `json:"activeStreamHeartbeats"`
					ClosingRouter          bool   `json:"closingRouter"`
					ClosingCaller          bool   `json:"closingCaller"`
					ClosingTransport       bool   `json:"closingTransport"`
					ClosedRouter           bool   `json:"closedRouter"`
					ClosedCaller           bool   `json:"closedCaller"`
					ClosedTransport        bool   `json:"closedTransport"`
					Router                 struct {
						Handlers        int            `json:"handlers"`
						StartedHandlers int            `json:"startedHandlers"`
						StoppedHandlers int            `json:"stoppedHandlers"`
						Topics          map[string]int `json:"topics"`
					} `json:"router"`
				} `json:"transport"`
			} `json:"runtime"`
			Components []struct {
				Name  string          `json:"name"`
				Data  json.RawMessage `json:"data"`
				Error string          `json:"error"`
			} `json:"components"`
		} `json:"lifecycle"`
	}
	if err := json.Unmarshal(payload, &resp); err != nil {
		return writeJSONPretty(w, payload)
	}
	tw := newTW(w)
	fmt.Fprintln(tw, "AREA\tMETRIC\tVALUE")
	r := resp.Lifecycle.Runtime
	fmt.Fprintf(tw, "runtime\truntimeId\t%s\n", nonEmpty(r.RuntimeID, "-"))
	fmt.Fprintf(tw, "runtime\tnamespace\t%s\n", nonEmpty(r.Namespace, "-"))
	fmt.Fprintf(tw, "runtime\tcallerId\t%s\n", nonEmpty(r.CallerID, "-"))
	fmt.Fprintf(tw, "runtime\tmountedModules\t%d\n", r.MountedModules)
	fmt.Fprintf(tw, "runtime\tactiveHandlers\t%d\n", r.ActiveHandlers)
	fmt.Fprintf(tw, "runtime\tdraining\t%t\n", r.Draining)
	fmt.Fprintf(tw, "provider\taiProviders\t%d\n", r.Provider.AIProviders)
	fmt.Fprintf(tw, "provider\tvectorStores\t%d\n", r.Provider.VectorStores)
	fmt.Fprintf(tw, "provider\tstorages\t%d\n", r.Provider.Storages)
	fmt.Fprintf(tw, "provider\tclosing\t%t\n", r.Provider.Closing)
	fmt.Fprintf(tw, "provider\tclosed\t%t\n", r.Provider.Closed)
	fmt.Fprintf(tw, "provider\tactiveProbes\t%d\n", r.Provider.ActiveProbes)
	fmt.Fprintf(tw, "provider\tactiveOperations\t%d\n", r.Provider.ActiveOperations)
	fmt.Fprintf(tw, "storage\tclosing\t%t\n", r.Storage.Closing)
	fmt.Fprintf(tw, "storage\tbridgeCount\t%d\n", r.Storage.BridgeCount)
	fmt.Fprintf(tw, "storage\tbridgeNames\t%s\n", nonEmpty(strings.Join(r.Storage.BridgeNames, ","), "-"))
	fmt.Fprintf(tw, "storage\tactiveCloses\t%d\n", r.Storage.ActiveCloses)
	fmt.Fprintf(tw, "transport\tkind\t%s\n", nonEmpty(r.Transport.Kind, "-"))
	fmt.Fprintf(tw, "transport\townsTransport\t%t\n", r.Transport.OwnsTransport)
	fmt.Fprintf(tw, "transport\tactiveSubscriptions\t%d\n", r.Transport.ActiveSubscriptions)
	fmt.Fprintf(tw, "transport\tactiveStreamHeartbeats\t%d\n", r.Transport.ActiveStreamHeartbeats)
	fmt.Fprintf(tw, "transport\tclosingRouter\t%t\n", r.Transport.ClosingRouter)
	fmt.Fprintf(tw, "transport\tclosingCaller\t%t\n", r.Transport.ClosingCaller)
	fmt.Fprintf(tw, "transport\tclosingTransport\t%t\n", r.Transport.ClosingTransport)
	fmt.Fprintf(tw, "transport\tclosedRouter\t%t\n", r.Transport.ClosedRouter)
	fmt.Fprintf(tw, "transport\tclosedCaller\t%t\n", r.Transport.ClosedCaller)
	fmt.Fprintf(tw, "transport\tclosedTransport\t%t\n", r.Transport.ClosedTransport)
	fmt.Fprintf(tw, "transport.router\thandlers\t%d\n", r.Transport.Router.Handlers)
	fmt.Fprintf(tw, "transport.router\tstartedHandlers\t%d\n", r.Transport.Router.StartedHandlers)
	fmt.Fprintf(tw, "transport.router\tstoppedHandlers\t%d\n", r.Transport.Router.StoppedHandlers)
	for _, component := range resp.Lifecycle.Components {
		if component.Error != "" {
			fmt.Fprintf(tw, "%s\terror\t%s\n", nonEmpty(component.Name, "component"), component.Error)
			continue
		}
		fields := map[string]any{}
		if err := json.Unmarshal(component.Data, &fields); err != nil {
			fmt.Fprintf(tw, "%s\tdata\t%s\n", nonEmpty(component.Name, "component"), compactJSON(component.Data))
			continue
		}
		keys := make([]string, 0, len(fields))
		for key := range fields {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Fprintf(tw, "%s\t%s\t%s\n", nonEmpty(component.Name, "component"), key, inspectValueString(fields[key]))
		}
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

func inspectValueString(v any) string {
	switch value := v.(type) {
	case nil:
		return "null"
	case string:
		return nonEmpty(value, "-")
	case float64:
		if value == float64(int64(value)) {
			return fmt.Sprintf("%d", int64(value))
		}
		return fmt.Sprintf("%g", value)
	case bool:
		return fmt.Sprintf("%t", value)
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Sprintf("%v", value)
		}
		return string(data)
	}
}

func compactJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "-"
	}
	var out bytes.Buffer
	if err := json.Compact(&out, raw); err != nil {
		return string(raw)
	}
	return out.String()
}

type inspectModuleCapability struct {
	Name      string `json:"name"`
	Direction string `json:"direction"`
	Type      string `json:"type"`
	Summary   string `json:"summary"`
}

type inspectModuleCapabilityGroups struct {
	Required []inspectModuleCapability `json:"required"`
	Optional []inspectModuleCapability `json:"optional"`
	Provided []inspectModuleCapability `json:"provided"`
}

type inspectModuleCapabilityAvailability struct {
	Module    string `json:"module"`
	Name      string `json:"name"`
	Direction string `json:"direction"`
	Type      string `json:"type"`
	Summary   string `json:"summary"`
	Available bool   `json:"available"`
	Source    string `json:"source"`
	Provider  string `json:"provider"`
}

type inspectModuleDependencyStatus struct {
	Name        string `json:"name"`
	RequestedBy string `json:"requestedBy"`
	Mounted     bool   `json:"mounted"`
	Registered  bool   `json:"registered"`
	Available   bool   `json:"available"`
}

type inspectModulePreflight struct {
	Ready                       bool                                  `json:"ready"`
	RequiredModules             []inspectModuleDependencyStatus       `json:"requiredModules"`
	MissingModules              []string                              `json:"missingModules"`
	RequiredCapabilities        []inspectModuleCapabilityAvailability `json:"requiredCapabilities"`
	OptionalCapabilities        []inspectModuleCapabilityAvailability `json:"optionalCapabilities"`
	MissingRequiredCapabilities []inspectModuleCapabilityAvailability `json:"missingRequiredCapabilities"`
	Errors                      []string                              `json:"errors"`
}

func renderModules(w io.Writer, payload json.RawMessage) error {
	var resp struct {
		Modules []struct {
			Name             string                         `json:"name"`
			Status           string                         `json:"status"`
			Summary          string                         `json:"summary"`
			Provides         []string                       `json:"provides"`
			Requires         []string                       `json:"requires"`
			Commands         []any                          `json:"commands"`
			Events           []any                          `json:"events"`
			Capabilities     []inspectModuleCapability      `json:"capabilities"`
			CapabilityGroups *inspectModuleCapabilityGroups `json:"capabilityGroups"`
			Resources        []any                          `json:"resources"`
		} `json:"modules"`
		Preflights map[string]inspectModulePreflight `json:"preflights"`
	}
	if err := json.Unmarshal(payload, &resp); err != nil {
		return writeJSONPretty(w, payload)
	}
	tw := newTW(w)
	fmt.Fprintln(tw, "NAME\tSTATUS\tPROVIDES\tREQUIRES\tCOMMANDS\tEVENTS\tREQCAPS\tOPTCAPS\tPROVCAPS\tREADY\tMISSING\tRES\tSUMMARY")
	for _, m := range resp.Modules {
		required, optional, provided := moduleCapabilityCounts(m.CapabilityGroups, m.Capabilities)
		ready, missing := modulePreflightSummary(resp.Preflights, m.Name)
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%s\t%s\t%d\t%s\n",
			nonEmpty(m.Name, "-"),
			nonEmpty(m.Status, "-"),
			nonEmpty(strings.Join(m.Provides, ","), "-"),
			nonEmpty(strings.Join(m.Requires, ","), "-"),
			len(m.Commands),
			len(m.Events),
			required,
			optional,
			provided,
			ready,
			missing,
			len(m.Resources),
			nonEmpty(m.Summary, "-"))
	}
	return tw.Flush()
}

func modulePreflightSummary(preflights map[string]inspectModulePreflight, name string) (ready, missing string) {
	if preflights == nil {
		return "-", "-"
	}
	preflight, ok := preflights[name]
	if !ok {
		return "-", "-"
	}
	var missingParts []string
	for _, module := range preflight.MissingModules {
		missingParts = append(missingParts, "module:"+module)
	}
	for _, cap := range preflight.MissingRequiredCapabilities {
		label := cap.Name
		if cap.Module != "" && cap.Module != name {
			label = cap.Module + ":" + label
		}
		missingParts = append(missingParts, "cap:"+label)
	}
	missingParts = append(missingParts, preflight.Errors...)
	sort.Strings(missingParts)
	return fmt.Sprintf("%t", preflight.Ready), nonEmpty(strings.Join(missingParts, ","), "-")
}

func renderModule(w io.Writer, payload json.RawMessage) error {
	var resp struct {
		Module struct {
			Name             string                         `json:"name"`
			Status           string                         `json:"status"`
			Summary          string                         `json:"summary"`
			Provides         []string                       `json:"provides"`
			Requires         []string                       `json:"requires"`
			Commands         []any                          `json:"commands"`
			Events           []any                          `json:"events"`
			Subscriptions    []any                          `json:"subscriptions"`
			Capabilities     []inspectModuleCapability      `json:"capabilities"`
			CapabilityGroups *inspectModuleCapabilityGroups `json:"capabilityGroups"`
			Resources        []any                          `json:"resources"`
		} `json:"module"`
		Mounted   bool                   `json:"mounted"`
		Preflight inspectModulePreflight `json:"preflight"`
	}
	if err := json.Unmarshal(payload, &resp); err != nil {
		return writeJSONPretty(w, payload)
	}

	tw := newTW(w)
	fmt.Fprintln(tw, "FIELD\tVALUE")
	fmt.Fprintf(tw, "name\t%s\n", nonEmpty(resp.Module.Name, "-"))
	fmt.Fprintf(tw, "status\t%s\n", nonEmpty(resp.Module.Status, "-"))
	fmt.Fprintf(tw, "mounted\t%t\n", resp.Mounted)
	fmt.Fprintf(tw, "ready\t%t\n", resp.Preflight.Ready)
	fmt.Fprintf(tw, "missingModules\t%s\n", nonEmpty(strings.Join(resp.Preflight.MissingModules, ","), "-"))
	fmt.Fprintf(tw, "missingCapabilities\t%s\n", nonEmpty(strings.Join(moduleCapabilityNames(resp.Preflight.MissingRequiredCapabilities), ","), "-"))
	fmt.Fprintf(tw, "errors\t%s\n", nonEmpty(strings.Join(resp.Preflight.Errors, ","), "-"))
	fmt.Fprintf(tw, "provides\t%s\n", nonEmpty(strings.Join(resp.Module.Provides, ","), "-"))
	fmt.Fprintf(tw, "requires\t%s\n", nonEmpty(strings.Join(resp.Module.Requires, ","), "-"))
	fmt.Fprintf(tw, "commands\t%d\n", len(resp.Module.Commands))
	fmt.Fprintf(tw, "events\t%d\n", len(resp.Module.Events))
	fmt.Fprintf(tw, "subscriptions\t%d\n", len(resp.Module.Subscriptions))
	fmt.Fprintf(tw, "resources\t%d\n", len(resp.Module.Resources))
	fmt.Fprintf(tw, "summary\t%s\n", nonEmpty(resp.Module.Summary, "-"))
	if err := tw.Flush(); err != nil {
		return err
	}

	if len(resp.Preflight.RequiredModules) > 0 || len(resp.Preflight.MissingModules) > 0 {
		fmt.Fprintln(w)
		if err := renderModuleDependencies(w, resp.Preflight.RequiredModules); err != nil {
			return err
		}
	}

	fmt.Fprintln(w)
	return renderModuleCapabilities(w, resp.Module.Name, resp.Module.CapabilityGroups, resp.Module.Capabilities, resp.Preflight)
}

func renderModuleDependencies(w io.Writer, deps []inspectModuleDependencyStatus) error {
	tw := newTW(w)
	fmt.Fprintln(tw, "DEPENDENCY\tREQUESTED_BY\tMOUNTED\tREGISTERED\tAVAILABLE")
	for _, dep := range deps {
		fmt.Fprintf(tw, "%s\t%s\t%t\t%t\t%t\n",
			nonEmpty(dep.Name, "-"),
			nonEmpty(dep.RequestedBy, "-"),
			dep.Mounted,
			dep.Registered,
			dep.Available)
	}
	return tw.Flush()
}

func moduleCapabilityNames(caps []inspectModuleCapabilityAvailability) []string {
	out := make([]string, 0, len(caps))
	for _, cap := range caps {
		name := cap.Name
		if cap.Module != "" {
			name = cap.Module + ":" + name
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func renderModuleCapabilities(w io.Writer, moduleName string, groups *inspectModuleCapabilityGroups, caps []inspectModuleCapability, preflight inspectModulePreflight) error {
	rows := moduleCapabilityRows(moduleName, groups, caps, preflight)
	tw := newTW(w)
	fmt.Fprintln(tw, "DIRECTION\tNAME\tTYPE\tAVAILABLE\tSOURCE\tPROVIDER\tMODULE\tSUMMARY")
	for _, row := range rows {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%t\t%s\t%s\t%s\t%s\n",
			nonEmpty(row.Direction, "-"),
			nonEmpty(row.Name, "-"),
			nonEmpty(row.Type, "-"),
			row.Available,
			nonEmpty(row.Source, "-"),
			nonEmpty(row.Provider, "-"),
			nonEmpty(row.Module, "-"),
			nonEmpty(row.Summary, "-"))
	}
	return tw.Flush()
}

func moduleCapabilityRows(moduleName string, groups *inspectModuleCapabilityGroups, caps []inspectModuleCapability, preflight inspectModulePreflight) []inspectModuleCapabilityAvailability {
	var rows []inspectModuleCapabilityAvailability
	rows = append(rows, preflight.RequiredCapabilities...)
	rows = append(rows, preflight.OptionalCapabilities...)
	for _, cap := range providedCapabilities(groups, caps) {
		rows = append(rows, inspectModuleCapabilityAvailability{
			Name:      cap.Name,
			Direction: "provided",
			Type:      cap.Type,
			Summary:   cap.Summary,
			Available: true,
			Source:    "self",
			Provider:  moduleName,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Direction != rows[j].Direction {
			return capabilityDirectionRank(rows[i].Direction) < capabilityDirectionRank(rows[j].Direction)
		}
		if rows[i].Module != rows[j].Module {
			return rows[i].Module < rows[j].Module
		}
		return rows[i].Name < rows[j].Name
	})
	return rows
}

func providedCapabilities(groups *inspectModuleCapabilityGroups, caps []inspectModuleCapability) []inspectModuleCapability {
	if groups != nil {
		return groups.Provided
	}
	var out []inspectModuleCapability
	for _, cap := range caps {
		if cap.Direction == "provided" {
			out = append(out, cap)
		}
	}
	return out
}

func capabilityDirectionRank(direction string) int {
	switch direction {
	case "required":
		return 0
	case "optional":
		return 1
	case "provided":
		return 2
	default:
		return 3
	}
}

func moduleCapabilityCounts(groups *inspectModuleCapabilityGroups, caps []inspectModuleCapability) (required, optional, provided int) {
	if groups != nil {
		return len(groups.Required), len(groups.Optional), len(groups.Provided)
	}
	for _, cap := range caps {
		switch cap.Direction {
		case "required":
			required++
		case "optional":
			optional++
		case "provided":
			provided++
		}
	}
	return required, optional, provided
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
