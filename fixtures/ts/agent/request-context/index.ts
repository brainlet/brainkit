import { Agent, RequestContext } from "agent";
import { model, output } from "kit";
const agent = new Agent({ name: "ctx-agent", model: model("openai", "gpt-4o-mini"), instructions: (ctx: any) => { const persona = ctx?.requestContext?.get?.("persona") || "assistant"; return "You are a " + persona + ". Respond in character in one sentence."; } });
const ctx = new RequestContext([["persona", "pirate"]]);
const result = await agent.generate("Say hello!", { requestContext: ctx });
output({ text: result.text, hasText: result.text.length > 0 });
