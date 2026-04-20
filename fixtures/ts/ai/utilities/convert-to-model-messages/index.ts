// `convertToModelMessages(uiMessages)` normalizes the chat-UI
// message shape into the ModelMessage shape that `generateText` /
// `streamText` expect. Used on server handlers that receive UI state
// from `useChat()` and need to forward it to an LLM.
import { convertToModelMessages } from "ai";
import { output } from "kit";

let isFn = false;
let convertErr: string | null = null;
let count = 0;
let firstRole = "";
try {
  isFn = typeof convertToModelMessages === "function";
  if (!isFn) throw new Error("convertToModelMessages is not a function");

  const uiMessages: any[] = [
    {
      id: "m1",
      role: "user",
      parts: [{ type: "text", text: "Hello" }],
    },
    {
      id: "m2",
      role: "assistant",
      parts: [{ type: "text", text: "Hi there" }],
    },
  ];
  const modelMessages: any[] = await convertToModelMessages(uiMessages);
  count = modelMessages.length;
  firstRole = modelMessages[0]?.role ?? "";
} catch (e: any) {
  convertErr = String(e?.message || e).substring(0, 200);
}

output({ isFn, convertErr, count, firstRole });
