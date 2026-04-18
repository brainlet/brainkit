// Command agent-spawner shows the peak brainkit use case: an
// agent that designs and deploys other agents at runtime.
//
// Flow:
//
//  1. Deploy `architect.ts` — an Agent with a `deploy_agent`
//     tool. The tool templates a new `.ts` package whose contents
//     register an Agent (name + instructions from the tool call)
//     and expose `bus.on("ask")` for external callers. The tool
//     invokes `bus.call("package.deploy", …)` to deploy that
//     package on the running Kit.
//
//  2. From Go, call `ts.architect.create` with a free-form
//     request: "I need an agent that writes haikus. Call it
//     haiku-bot." The architect's LLM decides name + instructions
//     and calls `deploy_agent`.
//
//  3. From Go, call the newly deployed agent's public topic
//     (`ts.<name>.ask`) directly. No architect in the loop —
//     once spawned, the new agent is a first-class bus citizen.
//
// Requires OPENAI_API_KEY (two round trips per run: architect
// design pass + spawned-agent answer pass). The example uses
// `gpt-4o-mini` by default; pass `-model` to override.
//
// Run from the repo root:
//
//	OPENAI_API_KEY=sk-... go run ./examples/agent-spawner
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
)

func main() {
	modelID := flag.String("model", "gpt-4o-mini", "OpenAI model id used by both agents")
	request := flag.String("request", "I need an agent that writes a single short haiku about whatever topic the caller provides. Keep it to three lines. Name it haiku-bot.",
		"free-form instruction for the architect")
	askPrompt := flag.String("ask", "autumn leaves drifting past a mountain stream",
		"prompt sent to the newly spawned agent's ts.<name>.ask topic")
	flag.Parse()

	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		log.Fatalf("OPENAI_API_KEY is required — this example runs two live OpenAI calls (architect + spawned agent)")
	}

	if err := run(key, *modelID, *request, *askPrompt); err != nil {
		log.Fatalf("agent-spawner: %v", err)
	}
}

func run(apiKey, modelID, request, askPrompt string) error {
	tmp, err := os.MkdirTemp("", "brainkit-agent-spawner-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "agent-spawner-demo",
		Transport: brainkit.Memory(),
		FSRoot:    tmp,
		Providers: []brainkit.ProviderConfig{brainkit.OpenAI(apiKey)},
	})
	if err != nil {
		return fmt.Errorf("new kit: %w", err)
	}
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// ── Step 1: deploy the architect ──────────────────────────
	architectCode := fmt.Sprintf(architectTemplate, modelID, modelID)
	if _, err := kit.Deploy(ctx, brainkit.PackageInline("architect", "architect.ts", architectCode)); err != nil {
		return fmt.Errorf("deploy architect: %w", err)
	}
	fmt.Println("[1/3] architect deployed")

	// ── Step 2: ask the architect to design + deploy a new agent ──
	fmt.Printf("[2/3] architect request: %q\n", request)
	req := json.RawMessage(fmt.Sprintf(`{"prompt":%q}`, request))
	reply, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](kit, ctx, sdk.CustomMsg{
		Topic:   "ts.architect.create",
		Payload: req,
	}, brainkit.WithCallTimeout(75*time.Second))
	if err != nil {
		return fmt.Errorf("architect create: %w", err)
	}
	var designed struct {
		Text     string `json:"text"`
		Deployed *struct {
			Deployed bool   `json:"deployed"`
			Name     string `json:"name"`
			Topic    string `json:"topic"`
		} `json:"deployed"`
		ToolCall *struct {
			Name         string `json:"name"`
			Instructions string `json:"instructions"`
		} `json:"toolCall"`
	}
	if err := json.Unmarshal(reply, &designed); err != nil {
		return fmt.Errorf("parse architect reply: %w\nraw: %s", err, string(reply))
	}
	if designed.Deployed == nil || !designed.Deployed.Deployed {
		return fmt.Errorf("architect did not deploy an agent.\n  text=%q\n  raw=%s",
			designed.Text, string(reply))
	}
	if designed.ToolCall != nil {
		fmt.Printf("        architect picked name=%q\n", designed.ToolCall.Name)
		fmt.Printf("        architect wrote instructions=%q\n", truncate(designed.ToolCall.Instructions, 140))
	}
	fmt.Printf("        architect deployed %s on %s\n",
		designed.Deployed.Name, designed.Deployed.Topic)

	// ── Step 3: call the spawned agent directly via the bus ──
	spawnedTopic := designed.Deployed.Topic
	fmt.Printf("[3/3] calling %s with prompt=%q\n", spawnedTopic, askPrompt)
	askPayload := json.RawMessage(fmt.Sprintf(`{"prompt":%q}`, askPrompt))
	spawnedReply, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](kit, ctx, sdk.CustomMsg{
		Topic:   spawnedTopic,
		Payload: askPayload,
	}, brainkit.WithCallTimeout(60*time.Second))
	if err != nil {
		return fmt.Errorf("call spawned agent: %w", err)
	}
	var ans struct {
		Text  string         `json:"text"`
		Usage map[string]any `json:"usage"`
	}
	if err := json.Unmarshal(spawnedReply, &ans); err != nil {
		fmt.Printf("        raw reply: %s\n", string(spawnedReply))
	} else {
		fmt.Println()
		fmt.Println("--- spawned agent reply ---")
		fmt.Println(ans.Text)
		fmt.Println("---")
		if ans.Usage != nil {
			fmt.Printf("usage: prompt=%v completion=%v total=%v\n",
				ans.Usage["promptTokens"], ans.Usage["completionTokens"], ans.Usage["totalTokens"])
		}
	}
	return nil
}

// architectTemplate is the source for the architect deployment.
// Two %s slots: the model ID for the architect itself, and the
// model ID baked into every spawned agent. Kept simple on
// purpose — a real deployment factory would let the architect
// pick models, tools, and memories per request.
const architectTemplate = `
const deployAgent = createTool({
    id: "deploy_agent",
    description: "Spawn a brand new agent on this Kit. Pass a concise lowercase dash-separated name and well-crafted instructions. Returns the deployment name + the bus topic external callers should use to reach the agent.",
    inputSchema: z.object({
        name: z.string().describe("agent id, lowercase-kebab"),
        instructions: z.string().describe("system prompt for the new agent"),
    }),
    execute: async (args) => {
        // Mastra wraps the call as { context, runtimeContext } in some
        // versions, passes raw args in others. Normalize.
        const input = (args && args.context) || args || {};
        const name = input.name;
        const instructions = input.instructions;
        const modelID = %q;
        const spawnSource =
            "const a = new Agent({ name: " + JSON.stringify(name) +
            ", model: model(\"openai\", " + JSON.stringify(modelID) + ")" +
            ", instructions: " + JSON.stringify(instructions) + " });\n" +
            "kit.register(\"agent\", " + JSON.stringify(name) + ", a);\n" +
            "bus.on(\"ask\", async (msg) => {\n" +
            "  const r = await a.generate(msg.payload.prompt);\n" +
            "  const u = r.usage || {};\n" +
            "  msg.reply({\n" +
            "    text: r.text,\n" +
            "    usage: {\n" +
            "      promptTokens: u.inputTokens || u.promptTokens || 0,\n" +
            "      completionTokens: u.outputTokens || u.completionTokens || 0,\n" +
            "      totalTokens: u.totalTokens || 0,\n" +
            "    },\n" +
            "  });\n" +
            "});\n";
        const manifest = { name: name, entry: name + ".ts" };
        const files = {};
        files[name + ".ts"] = spawnSource;
        const resp = await bus.call("package.deploy", { manifest: manifest, files: files }, { timeoutMs: 30000 });
        return {
            deployed: !!resp.deployed,
            name: resp.name || name,
            topic: "ts." + name + ".ask",
        };
    },
});

const architect = new Agent({
    name: "architect",
    model: model("openai", %q),
    instructions: "You are an agent architect. When asked to create an agent, call the deploy_agent tool exactly once with a concise lowercase-kebab name and crisp instructions tailored to the request. Do not ask clarifying questions. After the tool returns, reply briefly noting the deployed name.",
    tools: { deploy_agent: deployAgent },
});
kit.register("agent", "architect", architect);

// Mastra / AI SDK 5 stores the actual tool invocations in
// step.content as entries with type "tool-call" / "tool-result".
// Extract the ones matching our target tool.
function _findToolCall(result, toolName) {
    for (const step of (result && result.steps) || []) {
        for (const c of step.content || []) {
            if (c && c.type === "tool-call" && c.toolName === toolName) {
                return c.input || c.args || null;
            }
        }
    }
    return null;
}

function _findToolResult(result, toolName) {
    for (const step of (result && result.steps) || []) {
        for (const c of step.content || []) {
            if (c && c.type === "tool-result" && c.toolName === toolName) {
                if (c.output && c.output.value !== undefined) return c.output.value;
                if (c.output !== undefined) return c.output;
                if (c.result !== undefined) return c.result;
            }
        }
    }
    return null;
}

bus.on("create", async (msg) => {
    const prompt = (msg.payload && msg.payload.prompt) || "";
    let result;
    let err = null;
    try {
        result = await architect.generate(prompt, { maxSteps: 5 });
    } catch (e) {
        err = String(e && e.message || e);
        result = null;
    }

    msg.reply({
        text: (result && result.text) || "",
        error: err,
        toolCall: result ? _findToolCall(result, "deploy_agent") : null,
        deployed: result ? _findToolResult(result, "deploy_agent") : null,
    });
});
`

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
