// Command workspace-agent demonstrates a Mastra Agent wired to a
// Workspace — a bundle of LocalFilesystem (file read/write) +
// LocalSandbox (shell-command execution). When an Agent has a
// workspace, Mastra auto-injects a set of tools:
// `mastra_workspace_read_file`, `..._write_file`,
// `..._list_files`, `..._search_files`, `..._execute_command`.
// The agent then reasons about tasks and calls these tools to
// actually touch files + run commands.
//
// The example seeds a tiny workspace with two files, then asks
// the agent to:
//
//  1. list the files
//  2. write a TODO.md derived from sample.ts
//  3. count the lines of sample.ts via `wc -l` and store in
//     COUNT.txt
//
// After the run, Go reads the workspace back from disk and prints
// what the agent actually produced.
//
// Requires OPENAI_API_KEY.
//
// Path note: brainkit's fs polyfill rebases every user path under
// Kit FSRoot (see `internal/jsbridge/fs.go:resolve`). The example
// sets FSRoot to a tempdir and passes `basePath: "ws"` (relative)
// — the agent's tools end up writing under `<FSRoot>/ws/`.
//
// Run from the repo root:
//
//	OPENAI_API_KEY=sk-... go run ./examples/workspace-agent
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("workspace-agent: %v", err)
	}
}

func run() error {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return fmt.Errorf("OPENAI_API_KEY is required")
	}

	tmp, err := os.MkdirTemp("", "brainkit-workspace-agent-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	// Seed workspace on disk so the agent has something to read.
	// Paths are inside <tmp>/ws — LocalFilesystem({basePath: "ws"})
	// reaches the same place after fs-polyfill rebasing.
	wsDir := filepath.Join(tmp, "ws")
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(wsDir, "README.md"),
		[]byte("# demo workspace\n\nA sandboxed place for the agent to work.\n"), 0o644); err != nil {
		return err
	}
	sampleTS := `// sample.ts — demo source the agent will inspect.

export function add(a: number, b: number): number {
    return a + b;
}

export function greet(name: string): string {
    return "Hello, " + name + "!";
}

// TODO: add a subtract function
// TODO: write unit tests
// TODO: document the API
`
	if err := os.WriteFile(filepath.Join(wsDir, "sample.ts"), []byte(sampleTS), 0o644); err != nil {
		return err
	}

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "workspace-agent-demo",
		Transport: brainkit.Memory(),
		FSRoot:    tmp,
		Providers: []brainkit.ProviderConfig{brainkit.OpenAI(key)},
	})
	if err != nil {
		return fmt.Errorf("new kit: %w", err)
	}
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	if _, err := kit.Deploy(ctx, brainkit.PackageInline("workspace-agent", "workspace.ts", workspaceSource)); err != nil {
		return fmt.Errorf("deploy: %w", err)
	}
	fmt.Println("[1/5] workspace-agent deployed")
	fmt.Printf("        workspace dir: %s\n", wsDir)
	printTree(wsDir)

	// Introspect the auto-injected workspace tools.
	type toolsList struct {
		Names []string `json:"names"`
	}
	tl, err := brainkit.Call[sdk.CustomMsg, toolsList](kit, ctx, sdk.CustomMsg{
		Topic:   "ts.workspace-agent.tools-list",
		Payload: json.RawMessage(`{}`),
	}, brainkit.WithCallTimeout(5*time.Second))
	if err == nil {
		fmt.Printf("        auto-injected tools (%d): %v\n", len(tl.Names), tl.Names)
	}

	type traceEvent struct {
		Kind    string `json:"kind"`
		Tool    string `json:"tool,omitempty"`
		Input   any    `json:"input,omitempty"`
		Output  any    `json:"output,omitempty"`
		IsError bool   `json:"isError,omitempty"`
		Text    string `json:"text,omitempty"`
	}
	type reply struct {
		Text         string       `json:"text"`
		FinishReason string       `json:"finishReason"`
		Trace        []traceEvent `json:"trace"`
	}
	ask := func(task string) (reply, error) {
		payload := json.RawMessage(fmt.Sprintf(`{"task":%q}`, task))
		return brainkit.Call[sdk.CustomMsg, reply](kit, ctx, sdk.CustomMsg{
			Topic:   "ts.workspace-agent.task",
			Payload: payload,
		}, brainkit.WithCallTimeout(120*time.Second))
	}

	tasks := []struct {
		label string
		task  string
	}{
		{"[2/5] list files", "List every file in this workspace with a one-line summary of its contents."},
		{"[3/5] write TODO.md", "Read sample.ts, find the three TODO comments, and write TODO.md with each one on its own line (prefix each with \"- \")."},
		{"[4/5] run wc -l + write COUNT.txt", "Run `wc -l sample.ts` and write its numeric output (just the number) into COUNT.txt."},
	}
	for _, t := range tasks {
		fmt.Printf("\n%s\n        task: %q\n", t.label, t.task)
		r, err := ask(t.task)
		if err != nil {
			return fmt.Errorf("%s: %w", t.label, err)
		}
		fmt.Printf("        finishReason: %s\n", r.FinishReason)
		for _, e := range r.Trace {
			switch e.Kind {
			case "call":
				b, _ := json.Marshal(e.Input)
				fmt.Printf("        → call %s(%s)\n", e.Tool, truncate(string(b), 180))
			case "result":
				mark := "✓"
				if e.IsError {
					mark = "✗"
				}
				fmt.Printf("        %s %s ← %v\n", mark, e.Tool, truncate(fmt.Sprint(e.Output), 200))
			case "text":
				fmt.Printf("        … %s\n", truncate(e.Text, 200))
			}
		}
		fmt.Printf("        agent: %s\n", truncate(r.Text, 200))
	}

	// ── Final state ──
	fmt.Println("\n[5/5] workspace after the run:")
	printTree(wsDir)
	for _, name := range []string{"TODO.md", "COUNT.txt"} {
		p := filepath.Join(wsDir, name)
		if b, err := os.ReadFile(p); err == nil {
			fmt.Printf("--- %s ---\n%s", name, string(b))
			if len(b) > 0 && b[len(b)-1] != '\n' {
				fmt.Println()
			}
			fmt.Println("---")
		}
	}
	return nil
}

func printTree(dir string) {
	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		fmt.Printf("          %s  (%d bytes)\n", rel, info.Size())
		return nil
	})
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

const workspaceSource = `
// A coding agent with four explicit tools: read, write, list,
// and execute_command. We use createTool directly (over Mastra's
// Workspace class) because the tools then route through
// brainkit's fs + exec polyfills without any extra adapter
// layer. LocalSandbox's execute_command works out of the box
// via the refreshed exec polyfill (which rebases cwd under
// FSRoot); file I/O goes through fs.promises on the endowed
// fs object.

const readFileTool = createTool({
    id: "read_file",
    description: "Read the contents of a file inside the workspace.",
    inputSchema: z.object({ path: z.string().describe("relative file path under ws/") }),
    outputSchema: z.object({ content: z.string() }),
    execute: async (args) => {
        const input = (args && args.context) || args || {};
        const content = await fs.readFile("ws/" + input.path, "utf8");
        return { content: String(content) };
    },
});

const writeFileTool = createTool({
    id: "write_file",
    description: "Write or overwrite a file inside the workspace.",
    inputSchema: z.object({
        path: z.string().describe("relative file path under ws/"),
        content: z.string().describe("full file contents"),
    }),
    outputSchema: z.object({ bytes: z.number() }),
    execute: async (args) => {
        const input = (args && args.context) || args || {};
        await fs.writeFile("ws/" + input.path, input.content);
        return { bytes: input.content.length };
    },
});

const listFilesTool = createTool({
    id: "list_files",
    description: "List files inside the workspace.",
    inputSchema: z.object({ subdir: z.string().optional() }),
    outputSchema: z.object({ files: z.array(z.string()) }),
    execute: async (args) => {
        const input = (args && args.context) || args || {};
        const sub = input.subdir ? "ws/" + input.subdir : "ws";
        const entries = await fs.readdir(sub);
        return { files: entries };
    },
});

// execute_command runs through brainkit's exec polyfill via
// child_process.exec. The polyfill prepends Kit FSRoot to
// relative paths so commands can't reach outside the sandbox.
// "ws/" prefix points the shell at the workspace root.
const execTool = createTool({
    id: "execute_command",
    description: "Run a shell command inside the workspace.",
    inputSchema: z.object({ command: z.string() }),
    outputSchema: z.object({
        stdout: z.string(),
        stderr: z.string(),
        exitCode: z.number(),
    }),
    execute: async (args) => {
        const input = (args && args.context) || args || {};
        // Run with cd into ws so relative paths resolve to the
        // workspace. child_process.exec forwards the raw string
        // to sh -c, so "cd ws && ..." is safe.
        const out = await child_process.exec("cd ws && " + input.command);
        return {
            stdout: String(out.stdout || ""),
            stderr: String(out.stderr || ""),
            exitCode: Number(out.exitCode || 0),
        };
    },
});

const agent = new Agent({
    name: "workspace-agent",
    model: model("openai", "gpt-4o-mini"),
    instructions:
        "You have four tools: read_file, write_file, list_files, execute_command. " +
        "Always use tools — never fabricate file contents. " +
        "write_file takes a \"content\" string and writes it in full. " +
        "execute_command runs in the workspace root; use plain relative paths (e.g. \"wc -l sample.ts\").",
    tools: { read_file: readFileTool, write_file: writeFileTool, list_files: listFilesTool, execute_command: execTool },
});

kit.register("agent", "workspace-agent", agent);

function collectTrace(result) {
    const events = [];
    for (const step of (result && result.steps) || []) {
        for (const c of (step.content || [])) {
            if (!c) continue;
            if (c.type === "tool-call") {
                events.push({
                    kind: "call",
                    tool: c.toolName,
                    input: c.input || c.args,
                });
            } else if (c.type === "tool-result") {
                let out = c.output;
                if (out && out.value !== undefined) out = out.value;
                events.push({
                    kind: "result",
                    tool: c.toolName,
                    isError: !!c.isError,
                    output: typeof out === "string" ? out : JSON.stringify(out).slice(0, 200),
                });
            } else if (c.type === "text" && c.text) {
                events.push({ kind: "text", text: String(c.text).slice(0, 200) });
            }
        }
    }
    return events;
}

bus.on("task", async (msg) => {
    const task = (msg.payload && msg.payload.task) || "";
    const result = await agent.generate(task, { maxSteps: 12 });
    msg.reply({
        text: result.text || "",
        finishReason: result.finishReason || "",
        trace: collectTrace(result),
    });
});

bus.on("tools-list", async (msg) => {
    // Runtime introspection — what tools did Mastra auto-inject
    // onto the agent from its workspace?
    const tools = agent.tools || {};
    const names = Object.keys(typeof tools === "function" ? {} : tools);
    msg.reply({ names });
});
`
