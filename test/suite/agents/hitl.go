package agents

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// HITL — Human-in-the-Loop tool approval via generateWithApproval.
//
// These tests exercise the bus-based approval flow that brainkit exposes
// to deployments through `generateWithApproval(agent, prompt, opts)`.
// The .ts side calls a tool with `requireApproval: true`; the agent
// suspends; brainkit publishes an approval request to opts.approvalTopic
// and awaits a `{approved: bool}` reply.
//
// Contract codified by these tests:
//
//   - Approve must run the tool's execute() exactly once
//   - Decline must NOT run the tool's execute()
//   - A raw `new Agent` (no Mastra parent) must EITHER work end-to-end
//     OR fail loudly — silent no-op is not acceptable
//   - Multi-call retries (model declines, retries, declines, ...) must
//     loop through approval cycles and ultimately reach "stop"
//
// Observability: tool execute() emits a bus event to a per-test topic.
// The Go test subscribes to that topic so the side-effect count is
// observable across the QuickJS/Compartment boundary (a globalThis
// counter would not survive Compartment isolation).

const hitlSharedToolBlock = `
	const publishTool = createTool({
		id: "hitl-publish",
		description: "Publishes an article. Has a real side effect.",
		inputSchema: z.object({ slug: z.string() }),
		outputSchema: z.object({ ok: z.boolean(), slug: z.string() }),
		requireApproval: true,
		execute: async (input) => {
			var args = (input && input.context !== undefined) ? input.context : input;
			var slug = (args && args.slug) || "untitled";
			bus.emit(__hitl_fired_topic, { slug: slug });
			return { ok: true, slug: slug };
		},
	});
`

// fireCounter tracks how often the tool emits its "fired" event and the
// most recent slug seen.
type fireCounter struct {
	mu   sync.Mutex
	n    int
	slug string
}

func (f *fireCounter) snapshot() (int, string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.n, f.slug
}

// installFireWatcher subscribes to the per-test execute-fired topic and
// returns an unsubscribe function plus the counter to inspect.
func installFireWatcher(t *testing.T, env *suite.TestEnv, ctx context.Context, firedTopic string) (func(), *fireCounter) {
	t.Helper()
	fc := &fireCounter{}
	unsub, err := env.Kit.SubscribeRaw(ctx, firedTopic, func(msg sdk.Message) {
		var data struct {
			Slug string `json:"slug"`
		}
		_ = json.Unmarshal(msg.Payload, &data)
		fc.mu.Lock()
		fc.n++
		if data.Slug != "" {
			fc.slug = data.Slug
		}
		fc.mu.Unlock()
	})
	require.NoError(t, err)
	return unsub, fc
}

// installApprover subscribes to the approval topic and replies to every
// request with the supplied response. Returns an unsubscribe function and
// a counter pointer that tracks how many requests were observed.
//
// approvalTopic is the LOCAL topic — the kit's namespace prefix is added
// transparently by SubscribeRaw / PublishRaw, so this matches what the
// deployment publishes via __go_brainkit_await_approval.
func installApprover(t *testing.T, env *suite.TestEnv, ctx context.Context, approvalTopic string, response map[string]any) (func(), *int) {
	t.Helper()
	count := 0
	respBytes, err := json.Marshal(response)
	require.NoError(t, err)

	unsub, err := env.Kit.SubscribeRaw(ctx, approvalTopic, func(msg sdk.Message) {
		count++
		replyTo := msg.Metadata["replyTo"]
		if replyTo == "" {
			t.Logf("approver: missing replyTo on %s", approvalTopic)
			return
		}
		// PublishRaw on the kit auto-namespaces. The bridge already
		// resolved replyTo with the publisher's namespace, so we must
		// publish without re-prefixing — strip the namespace prefix
		// before handing it back to PublishRaw.
		stripped := replyTo
		if strings.HasPrefix(replyTo, "test.") {
			stripped = strings.TrimPrefix(replyTo, "test.")
		}
		if _, err := env.Kit.PublishRaw(ctx, stripped, respBytes); err != nil {
			t.Logf("approver: publish reply: %v", err)
		}
	})
	require.NoError(t, err)
	return unsub, &count
}

// triggerHitl invokes the deployment's "run" handler and waits for the
// reply payload.
func triggerHitl(t *testing.T, env *suite.TestEnv, ctx context.Context, service string) map[string]any {
	t.Helper()
	pr, err := sdk.SendToService(env.Kit, ctx, service, "run", json.RawMessage(`{}`))
	require.NoError(t, err)

	replyCh := make(chan sdk.Message, 1)
	unsub, err := env.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(msg sdk.Message) {
		select {
		case replyCh <- msg:
		default:
		}
	})
	require.NoError(t, err)
	defer unsub()

	select {
	case msg := <-replyCh:
		var parsed map[string]any
		require.NoError(t, json.Unmarshal(suite.ResponseDataFromMsg(msg), &parsed))
		return parsed
	case <-ctx.Done():
		t.Fatalf("triggerHitl: timeout waiting for reply on %s", pr.ReplyTo)
		return nil
	}
}

// hitlAgentBlock builds the deployment code that wires the agent and
// the bus.on("run") handler. wrapInMastra controls whether the agent
// goes through `new Mastra({ agents, storage }).getAgent(name)` or is
// registered raw. instructions controls how aggressive the agent is
// about calling the tool (e.g., "retry if rejected" stresses the
// approval-cycle loop).
func hitlAgentBlock(agentName, slug, approvalTopic, firedTopic, instructions string, wrapInMastra bool, maxSteps int) string {
	if maxSteps == 0 {
		maxSteps = 3
	}
	header := `globalThis.__hitl_fired_topic = "` + firedTopic + `";` + "\n" + hitlSharedToolBlock
	agent := `
		const rawAgent = new Agent({
			name: "` + agentName + `",
			model: model("openai", "gpt-4o-mini"),
			instructions: "` + instructions + `",
			tools: { publish: publishTool },
			maxSteps: ` + strconv.Itoa(maxSteps) + `,
		});
	`
	register := `
		kit.register("agent", "` + agentName + `", rawAgent);
		const writer = rawAgent;
	`
	if wrapInMastra {
		register = `
		const mastra = new Mastra({
			agents: { "` + agentName + `": rawAgent },
			storage: new InMemoryStore(),
		});
		const writer = mastra.getAgent("` + agentName + `");
		kit.register("agent", "` + agentName + `", writer);
		`
	}
	handler := `
		bus.on("run", async (msg) => {
			try {
				const result = await generateWithApproval(writer, "publish slug '` + slug + `' please.", {
					approvalTopic: "` + approvalTopic + `",
					timeout: 30000,
				});
				msg.reply({
					ok: true,
					finishReason: result.finishReason,
					toolResults: result.toolResults || [],
					text: result.text,
				});
			} catch (e) {
				msg.reply({ ok: false, error: (e && e.message) || String(e) });
			}
		});
	`
	return header + agent + register + handler
}

// testGenerateWithApprovalNoMastraWrap pins the contract for an agent
// registered via `kit.register("agent", ...)` WITHOUT first being
// wrapped in `new Mastra({ agents, storage })`. Mastra's resumeGenerate
// requires `#mastra.getStorage()` to load the workflow snapshot, so
// without that wrap the approve path silently degrades.
//
// Contract: either fail-loud (clear error mentioning Mastra parent) OR
// finish cleanly (finishReason="stop", non-empty text, execute fired
// once). Silent half-completion is the bug this test guards against.
func testGenerateWithApprovalNoMastraWrap(t *testing.T, env *suite.TestEnv) {
	env.RequireAI(t)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	const (
		source        = "hitl-no-mastra-adv.ts"
		approvalTopic = "hitl.no-mastra.approvals"
		firedTopic    = "hitl.no-mastra.fired"
	)

	require.NoError(t, testutil.DeployErr(env.Kit, source,
		hitlAgentBlock("hitl-no-mastra-writer", "no-mastra", approvalTopic, firedTopic,
			"Call hitl-publish with slug 'no-mastra'. Then stop. Do not call any other tool. Do not retry.",
			false, 3)))
	t.Cleanup(func() { testutil.Teardown(t, env.Kit, source) })

	unsubFire, fc := installFireWatcher(t, env, ctx, firedTopic)
	defer unsubFire()
	unsubApprove, approveCount := installApprover(t, env, ctx, approvalTopic, map[string]any{"approved": true})
	defer unsubApprove()

	reply := triggerHitl(t, env, ctx, source)
	// Give the bus a beat to deliver the fired event after reply lands.
	time.Sleep(100 * time.Millisecond)
	fired, slug := fc.snapshot()

	t.Logf("no-mastra reply: %v", reply)
	t.Logf("approver requests=%d, fire count=%d, slug=%q", *approveCount, fired, slug)

	// Contract: either fail loudly OR succeed end-to-end. The user's bug
	// is the silent-degradation middle ground — execute may or may not
	// fire, finishReason stays "suspended", text stays empty, while
	// ok=true gives the caller false confidence.
	ok, _ := reply["ok"].(bool)
	if !ok {
		errMsg, _ := reply["error"].(string)
		require.NotEmpty(t, errMsg, "if not ok, error must be non-empty")
		require.Contains(t, errMsg, "Mastra parent",
			"failure must come from the fail-loud Mastra-parent guard, not some unrelated error")
		t.Logf("no-mastra path failed loudly (acceptable): %s", errMsg)
		return
	}

	require.Greater(t, *approveCount, 0, "approver should have been asked at least once")
	require.Equal(t, 1, fired, "tool execute() should have run exactly once on approve")
	require.Equal(t, "no-mastra", slug, "tool received the right slug")
	finishReason, _ := reply["finishReason"].(string)
	require.Equal(t, "stop", finishReason,
		"approve path must finish, not stay suspended (silent half-completion guard)")
	text, _ := reply["text"].(string)
	require.NotEmpty(t, text,
		"approve path must produce a final text response (no confused empty reply)")
}

// testGenerateWithApprovalWithMastraWrap_Approve verifies the
// supported wiring: wrapping the agent in `new Mastra({ agents,
// storage })` before registering yields a working HITL approve flow
// end-to-end (execute fires once, finishReason="stop").
func testGenerateWithApprovalWithMastraWrap_Approve(t *testing.T, env *suite.TestEnv) {
	env.RequireAI(t)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	const (
		source        = "hitl-mastra-approve-adv.ts"
		approvalTopic = "hitl.mastra.approve.approvals"
		firedTopic    = "hitl.mastra.approve.fired"
	)

	require.NoError(t, testutil.DeployErr(env.Kit, source,
		hitlAgentBlock("hitl-wrapped-writer-a", "wrapped-yes", approvalTopic, firedTopic,
			"Call hitl-publish with slug 'wrapped-yes'. Then stop. Do not call any other tool. Do not retry.",
			true, 3)))
	t.Cleanup(func() { testutil.Teardown(t, env.Kit, source) })

	unsubFire, fc := installFireWatcher(t, env, ctx, firedTopic)
	defer unsubFire()
	unsubApprove, approveCount := installApprover(t, env, ctx, approvalTopic, map[string]any{"approved": true})
	defer unsubApprove()

	reply := triggerHitl(t, env, ctx, source)
	time.Sleep(100 * time.Millisecond)
	fired, slug := fc.snapshot()

	t.Logf("wrapped/approve reply: %v", reply)
	t.Logf("approver requests=%d, fire count=%d, slug=%q", *approveCount, fired, slug)

	require.True(t, reply["ok"].(bool), "wrapped agent + approve must succeed: %v", reply["error"])
	assert.Greater(t, *approveCount, 0, "approver should have been asked at least once")
	assert.Equal(t, 1, fired, "tool execute() must run exactly once on approve")
	assert.Equal(t, "wrapped-yes", slug, "tool received the right slug")
}

// testGenerateWithApprovalWithMastraWrap_Decline pins the decline
// contract: with a wrapped agent, replying `{approved: false}` to the
// approval topic MUST NOT run the tool's execute(). Zero fire-events;
// agent reaches "stop" with explanatory text from the model.
func testGenerateWithApprovalWithMastraWrap_Decline(t *testing.T, env *suite.TestEnv) {
	env.RequireAI(t)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	const (
		source        = "hitl-mastra-decline-adv.ts"
		approvalTopic = "hitl.mastra.decline.approvals"
		firedTopic    = "hitl.mastra.decline.fired"
	)

	require.NoError(t, testutil.DeployErr(env.Kit, source,
		hitlAgentBlock("hitl-wrapped-writer-d", "should-not-run", approvalTopic, firedTopic,
			"Call hitl-publish with slug 'should-not-run'. Then stop. Do not call any other tool. Do not retry.",
			true, 3)))
	t.Cleanup(func() { testutil.Teardown(t, env.Kit, source) })

	unsubFire, fc := installFireWatcher(t, env, ctx, firedTopic)
	defer unsubFire()
	unsubDecline, declineCount := installApprover(t, env, ctx, approvalTopic, map[string]any{
		"approved": false,
		"reason":   "test rejection",
	})
	defer unsubDecline()

	reply := triggerHitl(t, env, ctx, source)
	time.Sleep(100 * time.Millisecond)
	fired, slug := fc.snapshot()

	t.Logf("wrapped/decline reply: %v", reply)
	t.Logf("approver requests=%d, fire count=%d, slug=%q", *declineCount, fired, slug)

	require.Greater(t, *declineCount, 0, "approver should have been asked at least once")
	assert.Equal(t, 0, fired, "tool execute() MUST NOT run on decline")
	assert.Empty(t, slug, "tool should not have set a slug on decline")
}

// testGenerateWithApprovalDeclineWithRetries stresses the
// approval-cycle loop: maxSteps=5 + aggressive retry instructions +
// always-decline approver. Each retry triggers a fresh approval cycle,
// the loop in __kit_generateWithApproval handles all of them, no
// execute() ever fires, and the agent eventually reaches "stop" with
// non-empty text instead of half-completing in a suspended state.
func testGenerateWithApprovalDeclineWithRetries(t *testing.T, env *suite.TestEnv) {
	env.RequireAI(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	const (
		source        = "hitl-mastra-retry-decline-adv.ts"
		approvalTopic = "hitl.mastra.retry.approvals"
		firedTopic    = "hitl.mastra.retry.fired"
	)

	instructions := "You MUST publish an article. Call hitl-publish with slug 'rejected-1'. " +
		"If the tool result says it was not approved, RETRY by calling hitl-publish again with " +
		"a different slug like 'rejected-2', then 'rejected-3', then 'rejected-4'. Do not give up. " +
		"Do not stop until publish returns ok:true."

	require.NoError(t, testutil.DeployErr(env.Kit, source,
		hitlAgentBlock("hitl-retry-decline-writer", "rejected-1", approvalTopic, firedTopic,
			instructions, true, 5)))
	t.Cleanup(func() { testutil.Teardown(t, env.Kit, source) })

	unsubFire, fc := installFireWatcher(t, env, ctx, firedTopic)
	defer unsubFire()
	unsubDecline, declineCount := installApprover(t, env, ctx, approvalTopic, map[string]any{
		"approved": false,
		"reason":   "always reject for this test",
	})
	defer unsubDecline()

	reply := triggerHitl(t, env, ctx, source)
	time.Sleep(200 * time.Millisecond)
	fired, slug := fc.snapshot()

	t.Logf("retry/decline reply: %v", reply)
	t.Logf("approver requests=%d, fire count=%d, slug=%q", *declineCount, fired, slug)

	require.Greater(t, *declineCount, 0, "approver should have been asked at least once")
	assert.Equal(t, 0, fired,
		"tool execute() MUST NOT run on decline even across retries")

	// Loop-end-state contract: after declines, the agent must reach a
	// terminal "stop" state with explanatory text. Empty text +
	// finishReason='suspended' is the silent half-completion the loop
	// fix in __kit_generateWithApproval prevents.
	finishReason, _ := reply["finishReason"].(string)
	assert.Equal(t, "stop", finishReason,
		"decline path must reach a terminal state, not stay suspended")
	text, _ := reply["text"].(string)
	assert.NotEmpty(t, text,
		"decline path must produce a final text response (no silent suspend)")
}
