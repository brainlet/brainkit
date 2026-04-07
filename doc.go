// Package brainkit is an embeddable runtime for AI agent teams.
//
// Create a Kernel for standalone use or a Node for transport-connected deployment.
// Deploy .ts code, register Go tools, configure AI providers, storage backends,
// RBAC roles, secrets, and tracing — all through KernelConfig.
//
//	k, _ := brainkit.NewKernel(brainkit.KernelConfig{
//	    Namespace: "myapp",
//	    Storages: map[string]brainkit.StorageConfig{
//	        "default": brainkit.SQLiteStorage("./data/app.db"),
//	    },
//	})
//	defer k.Close()
//
//	k.Deploy(ctx, "agent.ts", code)
package brainkit
