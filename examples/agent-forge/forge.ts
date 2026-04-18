// agent-forge: a multi-agent pipeline that designs, writes,
// reviews, and deploys a brand-new brainkit .ts agent based on
// a freeform user request. Wraps every major brainkit primitive
// in one realistic use case.
//
// Wired as a Mastra workflow inside a SES Compartment:
//   architect  → spec
//   coder      → first-pass code (gpt-5.3)
//   dountil:
//     reviewer (supervisor + subagents) → approved? | issues
//     patch-coder (gpt-5.3)             → fixed code
//   deploy     → bus.call("package.deploy", …)
//
// Handlers exposed:
//   ts.agent-forge.create  — input: { request }, reply: ForgeResult
//
// Ownership: Go side (main.go) deploys this TS, calls create,
// then calls the freshly-forged agent's ts.<name>.ask topic.

// Model assignment: coder slot benefits from the highest-quality
// available model; architect + reviewers can run on a cheaper
// tier. Swap these constants if a stronger model is available in
// your provider account.
const CODER_MODEL = "gpt-4o";
const REVIEWER_MODEL = "gpt-4o";   // reviewers hallucinated on gpt-4o-mini — need the bigger model to reliably ground on the actual source
const ARCHITECT_MODEL = "gpt-4o-mini";

// ── Full brainkit self-description for the coder ────────────
// Every .d.ts + every .md the Kit ships. Coder + patch-coder
// both get the whole thing — ~250kb, well inside gpt-4o's 128k
// context, and it eliminates the class of bugs where the coder
// guesses a symbol shape that doesn't exist. Reviewers receive
// a leaner "tool-author" pack because they only need to pattern
// match on the surface, not generate against every API.
const FULL_REFERENCE = await reference.get("everything");
const REVIEWER_REFERENCE = await reference.get("tool-author");

// The agent.generate({output: ZodSchema}) path in this runtime
// doesn't reliably populate result.object — Mastra's fork forwards
// the zod-described shape to the LLM as an instruction, but the
// model returns markdown-fenced JSON in .text and the parsed
// object never materializes. We sidestep by instructing the LLM
// to emit JSON only and parsing .text ourselves. Works every time.

// Shape the architect emits.
const SpecShape = {
    name: "lowercase-kebab agent id (e.g. tweet-bot, haiku-writer)",
    purpose: "one-sentence description of what the agent does",
    instructions: "the system prompt that will be baked into the forged agent",
    askShape: "a JSON schema string describing msg.payload shape for the ask topic",
    needsMemory: "boolean — true iff the agent should carry a Mastra Memory across turns",
};

// Shape each reviewer emits.
const SingleReviewShape = {
    ok: "boolean — true iff nothing in your review scope is wrong",
    issues: "array of short strings describing problems in your scope (empty when ok=true)",
};

// ── ARCHITECT ────────────────────────────────────────────────
const architect = new Agent({
    id: "architect",
    name: "Architect",
    model: model("openai", ARCHITECT_MODEL),
    instructions:
        "You turn a user request into a structured spec for a new brainkit agent. " +
        "Pick a concise lowercase-kebab name from the request (if the user suggests one, use it). " +
        "Design focused, safe instructions for the new agent. " +
        "Do not invent tools the agent will need; they get none by default. " +
        "askShape must always be exactly the string '{\"prompt\":\"string\"}' — the calling convention is fixed. " +
        "needsMemory must be false unless the request explicitly asks for the agent to remember past turns or carry context across calls.\n\n" +
        "Respond with JSON and nothing else (no markdown fences, no commentary, no explanation). " +
        "The JSON object must have exactly these keys:\n" +
        JSON.stringify(SpecShape, null, 2) +
        "\n\nExample valid output for 'Build me a haiku-writer agent':\n" +
        '{"name":"haiku-writer","purpose":"Writes a short haiku about a topic","instructions":"You are a haiku poet. Write a single three-line haiku about the topic provided in the prompt. Reply with ONLY the haiku text.","askShape":"{\\"prompt\\":\\"string\\"}","needsMemory":false}',
});

const SpecZodSchema = z.object({
    name: z.string(),
    purpose: z.string(),
    instructions: z.string(),
    askShape: z.string(),
    needsMemory: z.boolean(),
});

const architectStep = createStep({
    id: "architect",
    inputSchema: z.object({ request: z.string() }),
    outputSchema: z.object({ request: z.string(), spec: SpecZodSchema }),
    execute: async ({ inputData }) => {
        const result = await architect.generate(inputData.request);
        const parsed = parseJSON(result.text);
        if (!parsed || typeof parsed.name !== "string" || !parsed.name) {
            throw new BrainkitError(
                "architect did not produce a usable spec (name missing). Raw: " + (result.text || "").slice(0, 300),
                "VALIDATION_ERROR",
                { field: "spec" });
        }
        const spec = {
            name: String(parsed.name).toLowerCase().replace(/[^a-z0-9-]/g, "-"),
            purpose: String(parsed.purpose || inputData.request),
            instructions: String(parsed.instructions || "Be helpful."),
            askShape: String(parsed.askShape || '{"prompt":"string"}'),
            needsMemory: !!parsed.needsMemory,
        };
        return { request: inputData.request, spec };
    },
});

// ── CODER ────────────────────────────────────────────────────
const coder = new Agent({
    id: "coder",
    name: "Coder",
    model: model("openai", CODER_MODEL),
    instructions:
        "You write brainkit .ts deployment source for a SES Compartment. " +
        "You receive a spec: {name, purpose, instructions, askShape, needsMemory}. " +
        "Produce ONLY valid TypeScript source — no markdown, no ``` fences, no prose, no comments other than a one-line header.\n\n" +
        "Follow this exact template (substitute <name>, <instructions>, and optionally Memory wiring):\n\n" +
        "// Forged agent: <name>\n" +
        "const agent = new Agent({\n" +
        "    name: <name>,\n" +
        "    model: model(\"openai\", \"gpt-4o-mini\"),\n" +
        "    instructions: <instructions>,\n" +
        "});\n" +
        "kit.register(\"agent\", <name>, agent);\n\n" +
        "bus.on(\"ask\", async (msg) => {\n" +
        "    const prompt = (msg.payload && msg.payload.prompt) || \"\";\n" +
        "    const result = await agent.generate(prompt);\n" +
        "    const u = result.usage || {};\n" +
        "    msg.reply({\n" +
        "        text: result.text || \"\",\n" +
        "        usage: {\n" +
        "            promptTokens: u.inputTokens || u.promptTokens || 0,\n" +
        "            completionTokens: u.outputTokens || u.completionTokens || 0,\n" +
        "            totalTokens: u.totalTokens || 0,\n" +
        "        },\n" +
        "    });\n" +
        "});\n\n" +
        "Strict rules:\n" +
        "  - Use JSON.stringify-quoted string literals for <name> and <instructions>. The name and instructions MUST come verbatim from the spec.\n" +
        "  - The bus.on topic is always exactly the string \"ask\". Do not change it.\n" +
        "  - The incoming payload shape is always {\"prompt\":\"string\"}. Read msg.payload.prompt. Do NOT destructure other field names.\n" +
        "  - If spec.needsMemory is true, declare a memory before the agent (`const memory = new Memory({ storage: new InMemoryStore() });`) and pass it into new Agent({ ..., memory }). Otherwise do NOT add a Memory.\n" +
        "  - Do NOT import anything (imports are stripped). Use only globals: Agent, model, Memory, InMemoryStore, z, bus, kit, msg, console.\n" +
        "  - Do NOT use fetch, fs, setTimeout, or any network/filesystem APIs.\n" +
        "  - No error handling. Keep it flat. Trust the runtime.\n\n" +
        "Reference (use for symbol shape):\n\n" +
        FULL_REFERENCE,
});

const coderStep = createStep({
    id: "coder",
    inputSchema: z.object({ request: z.string(), spec: SpecZodSchema }),
    outputSchema: z.object({
        spec: SpecZodSchema,
        code: z.string(),
        iterationCount: z.number(),
        approved: z.boolean(),
        issues: z.array(z.object({
            category: z.string(),
            message: z.string(),
        })),
    }),
    execute: async ({ inputData }) => {
        const spec = inputData.spec;
        const prompt =
            "Write the .ts source for this brainkit agent:\n\n" +
            JSON.stringify(spec, null, 2) +
            "\n\nReturn TypeScript ONLY.";
        const result = await coder.generate(prompt);
        return {
            spec,
            code: stripFences(result.text || ""),
            iterationCount: 0,
            approved: false,
            issues: [],
        };
    },
});

// ── REVIEWER (supervisor + subagents) ────────────────────────
// Every reviewer receives the same contract: "return a JSON
// object with { ok, issues[] } and NOTHING else". We parse text,
// not .object.
const REVIEWER_VERDICT_INSTRUCTIONS =
    "\n\n=== VERDICT FORMAT ===\n" +
    "Respond with JSON and nothing else (no markdown fences, no prose, no commentary). " +
    "Exactly two keys: `ok` (boolean) and `issues` (array of short strings).\n\n" +
    "=== GROUNDING RULES (NON-NEGOTIABLE) ===\n" +
    "1. You only flag a problem you can QUOTE from the source. Every issue string MUST include a code excerpt that demonstrates the problem.\n" +
    "2. If the source already does the right thing, ok=true with issues=[].\n" +
    "3. DO NOT guess. DO NOT output issues that your instructions mention as examples. Hallucinating issues causes an infinite loop.\n" +
    "4. DO NOT list positive observations. Only problems that actually exist in the source.\n" +
    "5. Your default output is {\"ok\":true,\"issues\":[]}. Only deviate when you can point at the specific offending code.\n\n" +
    "=== EXAMPLES ===\n" +
    'Source contains `kit.register("agent", "bot", agent);` AND `bus.on("ask", …)` AND `msg.reply({text, usage})` → {"ok":true,"issues":[]}\n' +
    'Source contains `import { Agent } from "@mastra/core"` → {"ok":false,"issues":["uses forbidden ES import: import { Agent } from \\"@mastra/core\\""]}\n' +
    'Source is missing any call to kit.register → {"ok":false,"issues":["no kit.register call found — the agent is never registered"]}\n\n' +
    "=== REFERENCE CORPUS ===\n" +
    "Use this as the source of truth for what brainkit symbols exist. If the source uses a symbol not declared here, that's an issue. If the source uses a symbol correctly per the reference, do NOT flag it.\n\n" +
    REVIEWER_REFERENCE;

const safetyReviewer = new Agent({
    id: "safety-reviewer",
    name: "Safety Reviewer",
    description: "Flags filesystem escapes, credential exfiltration, unsafe fetch, and other risky patterns.",
    model: model("openai", REVIEWER_MODEL),
    instructions:
        "You review brainkit .ts deployment source for SAFETY issues ONLY. " +
        "Flag: credential exposure (hardcoded keys, leaking secrets through replies), " +
        "filesystem path traversal, unrestricted fetch to user-controlled URLs, " +
        "bus.call used for privilege escalation, prototype pollution. " +
        "Do NOT comment on style or correctness." +
        REVIEWER_VERDICT_INSTRUCTIONS,
});

const styleReviewer = new Agent({
    id: "style-reviewer",
    name: "Style Reviewer",
    description: "Checks brainkit convention adherence: kit.register, bus.on shape, msg.reply, usage mapping.",
    model: model("openai", REVIEWER_MODEL),
    instructions:
        "You review brainkit .ts deployment source for STYLE issues ONLY. " +
        "Flag: missing kit.register for the created Agent, wrong bus.on topic name (must be \"ask\"), " +
        "missing or malformed msg.reply, token-usage mapping that doesn't defensively read " +
        "inputTokens/outputTokens alongside promptTokens/completionTokens, ES `import` statements " +
        "(forbidden — compartment strips them), or use of undeclared globals. " +
        "Do NOT comment on safety or correctness." +
        REVIEWER_VERDICT_INSTRUCTIONS,
});

const correctnessReviewer = new Agent({
    id: "correctness-reviewer",
    name: "Correctness Reviewer",
    description: "Checks that the spec is faithfully implemented.",
    model: model("openai", REVIEWER_MODEL),
    instructions:
        "You review brainkit .ts deployment source for CORRECTNESS against the spec ONLY. " +
        "Flag mismatches: agent `name` field not matching spec.name, agent `instructions` not " +
        "derived from spec.instructions, needsMemory=true but no Memory wired into the Agent, " +
        "askShape claims fields that msg.payload destructure misses. " +
        "Do NOT comment on style or safety." +
        REVIEWER_VERDICT_INSTRUCTIONS,
});

// Aggregate the three specialist reviewers in parallel and fold
// the verdicts deterministically. Three sub-agents running
// concurrently → three structured verdicts → one aggregated
// boolean + issue list.
async function runReviewPanel(spec, code) {
    const prompt =
        "Spec:\n" +
        JSON.stringify(spec, null, 2) +
        "\n\nSource:\n" +
        code;
    const [safetyV, styleV, correctnessV] = await Promise.all([
        safetyReviewer.generate(prompt),
        styleReviewer.generate(prompt),
        correctnessReviewer.generate(prompt),
    ]);
    const panel = [
        { category: "safety", verdict: coerceReview(safetyV.text) },
        { category: "style", verdict: coerceReview(styleV.text) },
        { category: "correctness", verdict: coerceReview(correctnessV.text) },
    ];
    // One line per reviewer in kit logs so the decision trail
    // shows up in audit events.
    console.log("[forge] review:",
        panel.map(p => p.category + "=" + (p.verdict.ok ? "ok" : p.verdict.issues.length + " issue(s)")).join(" | "));
    const aggIssues = [];
    let allOk = true;
    for (const p of panel) {
        if (!p.verdict.ok) allOk = false;
        for (const msg of p.verdict.issues || []) {
            if (typeof msg === "string" && msg.trim().length > 0) {
                aggIssues.push({ category: p.category, message: msg });
            }
        }
    }
    return { approved: allOk, issues: aggIssues };
}

// coerceReview takes whatever text a reviewer produced and
// normalizes it to { ok, issues[] }. It handles: pure JSON,
// markdown-fenced JSON, JSON with extra keys, and (as a last
// resort) unstructured prose — treated as ok=false with the raw
// text as the issue.
function coerceReview(raw) {
    const parsed = parseJSON(raw);
    if (parsed && typeof parsed === "object") {
        const ok = parsed.ok === true;
        let issues = [];
        if (Array.isArray(parsed.issues)) {
            issues = parsed.issues.map(i => typeof i === "string" ? i : (i && i.message) || JSON.stringify(i));
        } else if (parsed.issues && typeof parsed.issues === "object") {
            // Some LLMs nest issues as an object of categories.
            for (const key of Object.keys(parsed.issues)) {
                const v = parsed.issues[key];
                if (Array.isArray(v)) issues = issues.concat(v.map(String));
                else if (typeof v === "string") issues.push(v);
                else if (v) issues.push(key + ": " + JSON.stringify(v));
            }
        }
        return { ok: ok && issues.length === 0, issues };
    }
    const trimmed = (raw || "").trim();
    if (/(^|\s)(no issues|looks good|nothing wrong|approved)/i.test(trimmed)) {
        return { ok: true, issues: [] };
    }
    return { ok: false, issues: trimmed ? [trimmed.slice(0, 240)] : ["reviewer returned nothing"] };
}

// Patch coder — same model, separate prompt that focuses on
// diff-style fixes rather than fresh generation.
const patchCoder = new Agent({
    id: "patch-coder",
    name: "Patch Coder",
    model: model("openai", CODER_MODEL),
    instructions:
        "You are given existing brainkit .ts source and a list of issues from reviewers. " +
        "Produce a corrected version of the source, addressing every issue. " +
        "Output ONLY the full patched TypeScript source — no fences, no prose, no diff markers. " +
        "Preserve the original spec (same agent name + instructions) unless an issue explicitly asks otherwise.\n\n" +
        "Reference corpus:\n\n" +
        FULL_REFERENCE,
});

const reviewAndPatchStep = createStep({
    id: "review-and-patch",
    inputSchema: z.object({
        spec: SpecZodSchema,
        code: z.string(),
        iterationCount: z.number(),
        approved: z.boolean(),
        issues: z.array(z.object({
            category: z.string(),
            message: z.string(),
        })),
    }),
    outputSchema: z.object({
        spec: SpecZodSchema,
        code: z.string(),
        iterationCount: z.number(),
        approved: z.boolean(),
        issues: z.array(z.object({
            category: z.string(),
            message: z.string(),
        })),
    }),
    execute: async ({ inputData }) => {
        const nextIter = inputData.iterationCount + 1;

        const verdict = await runReviewPanel(inputData.spec, inputData.code);

        if (verdict.approved) {
            return {
                spec: inputData.spec,
                code: inputData.code,
                iterationCount: nextIter,
                approved: true,
                issues: [],
            };
        }

        // Not approved → ask the patch coder to fix.
        const patchPrompt =
            "Spec:\n" +
            JSON.stringify(inputData.spec, null, 2) +
            "\n\nCurrent source:\n" +
            inputData.code +
            "\n\nReviewer issues to fix:\n" +
            JSON.stringify(verdict.issues, null, 2);
        const patched = await patchCoder.generate(patchPrompt);

        return {
            spec: inputData.spec,
            code: stripFences(patched.text || inputData.code),
            iterationCount: nextIter,
            approved: false,
            issues: verdict.issues,
        };
    },
});

// ── DEPLOY ──────────────────────────────────────────────────
const deployStep = createStep({
    id: "deploy",
    inputSchema: z.object({
        spec: SpecZodSchema,
        code: z.string(),
        iterationCount: z.number(),
        approved: z.boolean(),
        issues: z.array(z.object({
            category: z.string(),
            message: z.string(),
        })),
    }),
    outputSchema: z.object({
        deployed: z.boolean(),
        name: z.string(),
        topic: z.string(),
        iterations: z.number(),
        approved: z.boolean(),
        issues: z.array(z.object({
            category: z.string(),
            message: z.string(),
        })),
        code: z.string(),
    }),
    execute: async ({ inputData }) => {
        const name = inputData.spec.name || "unnamed-agent";
        if (!inputData.approved) {
            // Review loop exhausted without approval — surface the
            // best-effort code + issue log. Deploy is skipped so a
            // broken agent never becomes callable.
            return {
                deployed: false,
                name,
                topic: "",
                iterations: inputData.iterationCount,
                approved: false,
                issues: inputData.issues,
                code: inputData.code,
            };
        }
        const manifest = { name, entry: name + ".ts" };
        const files = {};
        files[name + ".ts"] = inputData.code;
        const resp = await bus.call("package.deploy", { manifest, files }, { timeoutMs: 30000 });
        return {
            deployed: !!resp.deployed,
            name,
            topic: "ts." + name + ".ask",
            iterations: inputData.iterationCount,
            approved: true,
            issues: [],
            code: inputData.code,
        };
    },
});

// ── WORKFLOW ────────────────────────────────────────────────
const forgeWorkflow = createWorkflow({
    id: "forge",
    inputSchema: z.object({ request: z.string() }),
    outputSchema: z.object({
        deployed: z.boolean(),
        name: z.string(),
        topic: z.string(),
        iterations: z.number(),
        approved: z.boolean(),
        issues: z.array(z.object({
            category: z.string(),
            message: z.string(),
        })),
        code: z.string(),
    }),
})
    .then(architectStep)
    .then(coderStep)
    .dountil(reviewAndPatchStep, async ({ inputData }) => {
        // Stop either when the supervisor approves or after 3
        // passes (original + 2 patches). Iteration count is
        // tracked on inputData because brainkit's Mastra fork
        // doesn't inject an iterationCount helper.
        if (inputData.iterationCount >= 3) return true;
        return inputData.approved;
    })
    .then(deployStep)
    .commit();

kit.register("workflow", "forge", forgeWorkflow);

// ── Public handler ───────────────────────────────────────────
bus.on("create", async (msg) => {
    const request = (msg.payload && msg.payload.request) || "";
    if (!request) {
        msg.reply({ error: "request is required" });
        return;
    }
    try {
        const run = await forgeWorkflow.createRun();
        const result = await run.start({ inputData: { request } });
        msg.reply(result.result || result);
    } catch (e) {
        msg.reply({ error: String((e && e.message) || e) });
    }
});

// ── Helpers ─────────────────────────────────────────────────
// Post-process LLM output into something the compartment can
// actually evaluate: strip markdown fences and any ES `import`
// lines the coder sneaks in despite explicit instructions. The
// compartment strips imports at compile time too, but removing
// them here keeps the source readable to reviewers on the next
// iteration (reviewers shouldn't waste a turn re-flagging them).
function stripFences(text) {
    let s = (text || "").trim();
    const fence = /^```(?:typescript|ts|javascript|js)?\s*\n([\s\S]*?)\n```\s*$/;
    const m = s.match(fence);
    if (m) s = m[1].trim();
    s = s.replace(/^\s*import\b[^;\n]*;?\s*$/gm, "");
    return s.trim();
}

// Parse JSON from LLM output, tolerating markdown fences and
// leading/trailing prose. Returns null when no JSON object can
// be extracted.
function parseJSON(text) {
    if (!text || typeof text !== "string") return null;
    let s = text.trim();
    // Strip fenced block if present.
    const fence = /```(?:json)?\s*\n([\s\S]*?)\n```/;
    const m = s.match(fence);
    if (m) s = m[1].trim();
    // Try direct parse first.
    try { return JSON.parse(s); } catch (_) {}
    // Fall back: find the first { and match to the outermost }.
    const start = s.indexOf("{");
    if (start < 0) return null;
    let depth = 0;
    for (let i = start; i < s.length; i++) {
        const c = s[i];
        if (c === "{") depth++;
        else if (c === "}") {
            depth--;
            if (depth === 0) {
                const candidate = s.slice(start, i + 1);
                try { return JSON.parse(candidate); } catch (_) { return null; }
            }
        }
    }
    return null;
}
