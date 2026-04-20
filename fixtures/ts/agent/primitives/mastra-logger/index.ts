// Mastra loggers — ConsoleLogger writes to stdout/stderr,
// MultiLogger fans out across multiple IMastraLogger instances.
// Exposed so custom Mastra configs can build a proper logger
// instead of falling back to `any`.
import { ConsoleLogger, MultiLogger, Mastra, Agent, InMemoryStore } from "agent";
import { model, output } from "kit";

const primary = new (ConsoleLogger as any)({ level: "INFO", name: "primary" });
const secondary = new (ConsoleLogger as any)({ level: "DEBUG", name: "audit" });
const multi = new (MultiLogger as any)([primary, secondary]);

// Call each log level — each logger must accept without throwing.
let logError: string | null = null;
try {
  primary.info("primary hello");
  primary.warn("primary warn");
  multi.info("fanout hello", { requestId: "abc-123" });
  multi.debug("debug diagnostic");
  multi.error("error diagnostic");
} catch (e: any) {
  logError = String(e?.message || e);
}

// Pass the MultiLogger through a Mastra config; `new Mastra` must
// accept it without throwing.
let mastraError: string | null = null;
let loggerAccepted = false;
try {
  const mastra = new (Mastra as any)({
    agents: {
      smoke: new Agent({
        name: "smoke",
        model: model("openai", "gpt-4o-mini"),
        instructions: "test",
      }),
    },
    storage: new InMemoryStore(),
    logger: multi,
  });
  loggerAccepted = !!mastra.getAgent("smoke");
} catch (e: any) {
  mastraError = String(e?.message || e);
}

output({
  consoleLoggerHasInfo: typeof primary.info === "function",
  consoleLoggerHasWarn: typeof primary.warn === "function",
  consoleLoggerHasError: typeof primary.error === "function",
  consoleLoggerHasDebug: typeof primary.debug === "function",
  multiLoggerFanout: typeof multi.info === "function",
  logError,
  mastraError,
  loggerAccepted,
});
