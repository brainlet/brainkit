// Package brainkit is an embeddable runtime for AI agent teams.
//
// Create a runtime with [New]. Interact through typed async messages
// using [sdk.Publish] and [sdk.SubscribeTo]:
//
//	kit, _ := brainkit.New(brainkit.Config{
//	    Namespace: "myapp",
//	    Storages: map[string]brainkit.StorageConfig{
//	        "default": brainkit.SQLiteStorage("./data/app.db"),
//	    },
//	    Providers: []brainkit.ProviderConfig{
//	        brainkit.OpenAI(os.Getenv("OPENAI_API_KEY")),
//	    },
//	})
//	defer kit.Close()
//
//	// Deploy a package (async)
//	pr, _ := sdk.PublishPackageDeploy(kit, ctx, messages.PackageDeployMsg{
//	    Path: "./agents/support.ts",
//	})
//	sdk.SubscribePackageDeployResp(kit, ctx, pr.ReplyTo,
//	    func(resp messages.PackageDeployResp, msg messages.Message) {
//	        fmt.Println("Deployed:", resp.Name)
//	    },
//	)
//
// Every feature is a typed bus command — deploy packages, manage providers,
// schedule messages, manage secrets, control plugins. The SDK generates
// type-safe Publish/Subscribe wrappers for every command.
package brainkit
