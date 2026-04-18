// Package brainkit is an embeddable runtime for AI agent teams. It
// combines an in-process JS/TS compartment (QuickJS + SES) with a
// typed pub/sub bus (Watermill), exposes the result as [Kit], and
// lets you compose opt-in subsystems through [Module].
//
// # Two entry points
//
// Library mode — embed a Kit inside your Go service:
//
//	kit, err := brainkit.New(brainkit.Config{
//	    Namespace: "myapp",
//	    Transport: brainkit.EmbeddedNATS(),
//	    Providers: []brainkit.ProviderConfig{
//	        brainkit.OpenAI(os.Getenv("OPENAI_API_KEY")),
//	    },
//	})
//	defer kit.Close()
//
// Service mode — run brainkit as a long-lived server, composed from
// the standard module set (gateway, probes, tracing, audit):
//
//	srv, _ := server.QuickStart("my-app", "/var/brainkit")
//	defer srv.Close()
//	_ = srv.Start(ctx)
//
// Full server composition lives in the sibling server package.
//
// # Interaction model
//
// Every feature is a typed bus command. Deploy packages, schedule
// messages, manage secrets, call AI providers, talk to plugins —
// each goes through [sdk.Publish] / [sdk.SubscribeTo] or the
// generated synchronous wrappers (one per Msg/Resp pair):
//
//	resp, err := brainkit.CallPackageDeploy(kit, ctx,
//	    sdk.PackageDeployMsg{Path: "./agents/support"},
//	)
//
// The generic [Call] / [CallStream] helpers stay exported for
// advanced uses; the wrappers just saturate the type parameters
// so the call site doesn't guess.
//
// # Accessors
//
// Provider, storage, vector, and secret management consolidate
// behind narrow accessors that cache a single instance per Kit:
//
//	kit.Providers().Register("openai", "openai", cfg)
//	kit.Secrets().Set(ctx, "API_KEY", "…")
//
// # Modules
//
// Opt-in subsystems implement [Module]. The standard set lives under
// modules/*: gateway, mcp, plugins, schedules, audit, tracing,
// probes, discovery, topology, workflow, harness. Pass instances in
// [Config.Modules] or rely on server mode's curated defaults.
package brainkit
