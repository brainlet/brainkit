// Command package-workflow demonstrates the on-disk package
// lifecycle: scaffold → edit → add a sibling file → deploy →
// call → teardown → redeploy after edit.
//
// This is the workflow the brainkit CLI (`brainkit new package`
// + `brainkit deploy <dir>`) uses internally, unpacked into a Go
// process so you can see every step.
//
// After the example exits, the scaffolded directory survives on
// disk (default: ./package-workflow-demo/) so you can open it in
// an IDE — manifest.json, tsconfig.json, types/*.d.ts, and the
// source files are all right there, with the paths mapped so
// `import { bus } from "kit"` resolves to the bundled
// declarations.
//
// Run from the repo root:
//
//	go run ./examples/package-workflow
//
// Flags:
//
//	-out     where to scaffold the package (default ./package-workflow-demo)
//	-keep    keep the scaffolded dir on exit (default true)
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
)

const initialSource = `// Initial scaffold: a single greeting handler.
// Edit this file, call kit.Redeploy (or re-run brainkit deploy),
// and the new version takes over.

bus.on("greet", (msg) => {
    const name = (msg.payload && msg.payload.name) || "world";
    msg.reply({ message: "Hello, " + name + "!" });
});
`

const editedSource = `// Edited by the example: the greeting got a weather line.

bus.on("greet", (msg) => {
    const name = (msg.payload && msg.payload.name) || "world";
    msg.reply({ message: "Hello, " + name + "! The weather is fine." });
});

bus.on("facts", (msg) => {
    msg.reply({
        facts: [
            "brainkit packages live on disk as a plain directory.",
            "tsconfig.json + types/ gives the IDE first-class autocomplete.",
            "kit.Deploy(PackageFromDir(path)) ships whatever's on disk.",
        ],
    });
});
`

// extraFileSource shows the pattern for adding sibling files to
// the same package. The bundler (esbuild, via the deploy path)
// pulls in every file referenced by the entry point; here we
// keep it simple by having the entry re-export from a helper.
const helperFileSource = `export function greeting(name) {
    return "Howdy, " + name + ". This greeting comes from a sibling file under the same package.";
}
`

const entryWithHelperSource = `import { greeting } from "./greetings";

bus.on("greet", (msg) => {
    const name = (msg.payload && msg.payload.name) || "stranger";
    msg.reply({ message: greeting(name) });
});
`

func main() {
	outRaw := flag.String("out", "./package-workflow-demo",
		"scaffold destination — survives the process so you can open it in an IDE")
	keep := flag.Bool("keep", true,
		"keep the scaffold on disk after the example exits (default true)")
	flag.Parse()

	out, err := filepath.Abs(*outRaw)
	if err != nil {
		log.Fatalf("package-workflow: resolve out: %v", err)
	}
	if err := run(out, *keep); err != nil {
		log.Fatalf("package-workflow: %v", err)
	}
}

func run(out string, keep bool) error {
	if _, err := os.Stat(out); err == nil {
		if err := os.RemoveAll(out); err != nil {
			return fmt.Errorf("clear stale scaffold at %s: %w", out, err)
		}
	}
	if !keep {
		defer os.RemoveAll(out)
	}

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "package-workflow-demo",
		Transport: brainkit.Memory(),
		FSRoot:    filepath.Dir(out),
	})
	if err != nil {
		return fmt.Errorf("new kit: %w", err)
	}
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// ── Step 1: scaffold a fresh package on disk ─────────────────
	fmt.Printf("[1/5] scaffolding package at %s\n", out)
	if err := brainkit.ScaffoldPackage(out, "greeter", "index.ts", initialSource); err != nil {
		return fmt.Errorf("scaffold: %w", err)
	}
	listDir(out)

	// ── Step 2: deploy via PackageFromDir ───────────────────────
	fmt.Println("[2/5] deploying PackageFromDir(...)")
	pkg, err := brainkit.PackageFromDir(out)
	if err != nil {
		return fmt.Errorf("PackageFromDir: %w", err)
	}
	if _, err := kit.Deploy(ctx, pkg); err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	greet, err := callGreet(kit, ctx, "alice")
	if err != nil {
		return fmt.Errorf("call greet: %w", err)
	}
	fmt.Printf("        ts.greeter.greet reply: %s\n", greet)

	// ── Step 3: edit the entry file on disk, redeploy ───────────
	fmt.Println("[3/5] editing index.ts on disk (adding a 'facts' handler)")
	if err := os.WriteFile(filepath.Join(out, "index.ts"), []byte(editedSource), 0o644); err != nil {
		return fmt.Errorf("edit: %w", err)
	}
	// kit.Deploy hot-replaces by name — same dir, new content.
	if _, err := kit.Deploy(ctx, pkg); err != nil {
		return fmt.Errorf("redeploy: %w", err)
	}

	greet2, err := callGreet(kit, ctx, "bob")
	if err != nil {
		return fmt.Errorf("call greet after edit: %w", err)
	}
	fmt.Printf("        ts.greeter.greet reply (edited): %s\n", greet2)

	factsReply, err := callFacts(kit, ctx)
	if err != nil {
		return fmt.Errorf("call facts: %w", err)
	}
	fmt.Println("        ts.greeter.facts reply:")
	for _, f := range factsReply.Facts {
		fmt.Printf("          • %s\n", f)
	}

	// ── Step 4: add a new sibling file, wire it from the entry ──
	fmt.Println("[4/5] adding a sibling file (greetings.ts) + updating index.ts to import it")
	if err := os.WriteFile(filepath.Join(out, "greetings.ts"), []byte(helperFileSource), 0o644); err != nil {
		return fmt.Errorf("write sibling: %w", err)
	}
	if err := os.WriteFile(filepath.Join(out, "index.ts"), []byte(entryWithHelperSource), 0o644); err != nil {
		return fmt.Errorf("rewrite entry: %w", err)
	}
	if _, err := kit.Deploy(ctx, pkg); err != nil {
		return fmt.Errorf("redeploy with sibling: %w", err)
	}

	greet3, err := callGreet(kit, ctx, "carol")
	if err != nil {
		return fmt.Errorf("call greet after sibling: %w", err)
	}
	fmt.Printf("        ts.greeter.greet reply (with sibling): %s\n", greet3)

	// ── Step 5: teardown ────────────────────────────────────────
	fmt.Println("[5/5] tearing down the deployment")
	if err := kit.Teardown(ctx, "greeter"); err != nil {
		return fmt.Errorf("teardown: %w", err)
	}
	if _, err := callGreet(kit, ctx, "dana"); err == nil {
		fmt.Println("        warning: post-teardown call returned a reply — expected failure")
	} else {
		fmt.Printf("        post-teardown call errored as expected: %v\n", cleanErr(err))
	}

	fmt.Println()
	if keep {
		fmt.Printf("Package kept on disk at: %s\n", out)
		fmt.Println("  Open it in an IDE — tsconfig.json + types/ give full autocomplete.")
		fmt.Println("  Edit index.ts or greetings.ts, then re-run this example or `brainkit deploy` to push changes.")
	}
	return nil
}

func listDir(dir string) {
	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		fmt.Printf("          %s  (%d bytes)\n", rel, info.Size())
		return nil
	})
}

type greetReply struct {
	Message string `json:"message"`
}

type factsReply struct {
	Facts []string `json:"facts"`
}

func callGreet(kit *brainkit.Kit, ctx context.Context, name string) (string, error) {
	reply, err := brainkit.Call[sdk.CustomMsg, greetReply](kit, ctx, sdk.CustomMsg{
		Topic:   "ts.greeter.greet",
		Payload: json.RawMessage(fmt.Sprintf(`{"name":%q}`, name)),
	}, brainkit.WithCallTimeout(5*time.Second))
	if err != nil {
		return "", err
	}
	return reply.Message, nil
}

func callFacts(kit *brainkit.Kit, ctx context.Context) (factsReply, error) {
	reply, err := brainkit.Call[sdk.CustomMsg, factsReply](kit, ctx, sdk.CustomMsg{
		Topic:   "ts.greeter.facts",
		Payload: json.RawMessage(`{}`),
	}, brainkit.WithCallTimeout(5*time.Second))
	return reply, err
}

// cleanErr trims the noisy prefix brainkit.Call adds around bus
// timeouts so the demo output stays readable.
func cleanErr(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	const prefix = "call timeout on "
	if idx := indexOf(s, prefix); idx >= 0 {
		return s[idx:]
	}
	return s
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
