// Package brainkit is an embeddable runtime for AI agent teams. It
// provides a light control-plane [Kit] with a typed pub/sub bus, and
// lets you compose opt-in subsystems through github.com/brainlet/brainkit/module.
//
// # Two entry points
//
// Library mode — embed a Kit inside your Go service:
//
//	kit, err := brainkit.New(brainkit.Config{
//	    Namespace: "myapp",
//	    Transport: transports.EmbeddedNATS(),
//	    Providers: []brainkit.ProviderConfig{
//	        brainkit.OpenAI(os.Getenv("OPENAI_API_KEY")),
//	    },
//	})
//	defer kit.Close()
//
// Service mode — run brainkit as a long-lived server, composed from
// the quickstart standard module set (gateway, probes, tracing, audit):
//
//	srv, _ := quickstart.New("my-app", "/var/brainkit")
//	defer srv.Close()
//	_ = srv.Start(ctx)
//
// Full server composition lives in the sibling server package.
//
// # Interaction model
//
// Features are exposed as typed bus commands. Deploy packages, schedule
// messages, manage secrets, call AI providers, talk to plugins —
// request/reply paths go through [Call], [CallStream], or the SDK's
// generated synchronous wrappers (one per Msg/Resp pair):
//
//	resp, err := packagemsg.CallPackageDeploy(kit, ctx,
//	    packagemsg.PackageDeployMsg{Path: "./agents/support"},
//	)
//
// The generic [Call] / [CallStream] helpers stay exported for Kit-specific
// behavior, such as topology-aware [WithCallTo]. The SDK wrappers saturate
// type parameters without adding module command names to the root API.
//
// # Runtime administration modules
//
// Provider, storage, vector, and secret management are module-owned bus APIs.
// Mount modules/registry or modules/secrets when a process needs to mutate or
// inspect those tables after Kit startup:
//
//	resp, err := registrymsg.CallRegistryList(kit, ctx,
//	    registrymsg.RegistryListMsg{Category: "provider"})
//	_, err = secretmsg.CallSecretsSet(kit, ctx,
//	    secretmsg.SecretsSetMsg{Name: "API_KEY", Value: "..."})
//
// # Modules
//
// Opt-in subsystems implement module.Module. The standard set lives under
// modules/*: gateway, mcp, plugins, schedules, audit, tracing,
// probes, discovery, topology, workflow, harness. Pass instances in
// [Config.Modules] or rely on server mode's curated defaults.
package brainkit
