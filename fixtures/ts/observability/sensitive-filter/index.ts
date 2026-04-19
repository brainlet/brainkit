// Test: SensitiveDataFilter — process a span whose attributes include
// an api-key. Asserts the key is redacted while non-sensitive fields
// stay intact.
import { SensitiveDataFilter } from "agent";
import { output } from "kit";

const filter = new SensitiveDataFilter({
  sensitiveFields: ["apikey", "password"],
  redactionToken: "[HIDDEN]",
  redactionStyle: "full",
} as any);

const span: any = {
  id: "span-1",
  name: "agent.generate",
  attributes: {
    userId: "u123",
    apiKey: "sk-live-abcdef123456",
    body: { password: "super-secret", nickname: "alice" },
  },
};

const filtered: any = (filter as any).process(span);

output({
  userIdIntact: filtered?.attributes?.userId === "u123",
  apiKeyRedacted: filtered?.attributes?.apiKey === "[HIDDEN]",
  passwordRedacted: filtered?.attributes?.body?.password === "[HIDDEN]",
  nicknameIntact: filtered?.attributes?.body?.nickname === "alice",
});
