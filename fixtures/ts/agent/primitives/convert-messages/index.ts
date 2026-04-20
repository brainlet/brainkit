// `convertMessages(input).to(format)` translates between:
//   "Mastra.V2", "AIV4.UI", "AIV4.Core", "AIV5.UI", "AIV5.Model"
// Needed when code reaches into Memory storage (Mastra.V2 shape) and
// hands the result back to the AI SDK (AIV5.Model), or when adapting
// v4 UI messages from older code.
import { convertMessages } from "agent";
import { output } from "kit";

// v5 UI-message format → Mastra.V2
const uiMessages = [
  {
    id: "u1",
    role: "user" as const,
    parts: [{ type: "text" as const, text: "Hi there" }],
  },
  {
    id: "a1",
    role: "assistant" as const,
    parts: [{ type: "text" as const, text: "Hello!" }],
  },
];

let v2: any = null;
let modelMsgs: any = null;
let convertError: string | null = null;
try {
  v2 = (convertMessages as any)(uiMessages).to("Mastra.V2");
  modelMsgs = (convertMessages as any)(uiMessages).to("AIV5.Model");
} catch (e: any) {
  convertError = String(e?.message || e);
}

output({
  convertError,
  v2IsArray: Array.isArray(v2),
  v2Length: Array.isArray(v2) ? v2.length : 0,
  modelIsArray: Array.isArray(modelMsgs),
  modelLength: Array.isArray(modelMsgs) ? modelMsgs.length : 0,
  modelFirstRole: Array.isArray(modelMsgs) && modelMsgs[0] ? modelMsgs[0].role : "",
});
