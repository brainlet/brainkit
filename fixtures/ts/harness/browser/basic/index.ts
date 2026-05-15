import {
  Agent,
  BrowserContextProcessor,
  Harness,
  InMemoryStore,
  MastraBrowser,
  Memory,
  RequestContext,
  createTool,
  z,
} from "agent";
import { model, output } from "kit";

async function withTimeout<T>(label: string, timeoutMs: number, fn: () => Promise<T>): Promise<T> {
  let timer: ReturnType<typeof setTimeout> | undefined;
  try {
    return await Promise.race([
      fn(),
      new Promise<T>((_, reject) => {
        timer = setTimeout(() => reject(new Error(`timeout:${label}`)), timeoutMs);
      }),
    ]);
  } finally {
    if (timer) clearTimeout(timer);
  }
}

class FixtureBrowser extends MastraBrowser {
  readonly id = "fixture-browser";
  readonly name = "FixtureBrowser";
  readonly provider = "brainkit-fixture-browser";
  readonly pageUrl = "https://brainkit.test/harness-browser";
  launchCount = 0;
  closeCount = 0;
  toolCalls = 0;
  lastToolContext: any;
  launchHookBrowserId = "";
  closeHookBrowserId = "";

  constructor() {
    super({
      headless: true,
      scope: "shared",
      onLaunch: ({ browser }) => {
        this.launchHookBrowserId = browser.id;
      },
      onClose: ({ browser }) => {
        this.closeHookBrowserId = browser.id;
      },
    });
  }

  protected async doLaunch(): Promise<void> {
    this.launchCount++;
  }

  protected async doClose(): Promise<void> {
    this.closeCount++;
  }

  protected async getActivePage(): Promise<{ url(): string } | null> {
    return { url: () => this.pageUrl };
  }

  protected getBrowserStateForThread(): { currentUrl: string; tabs: Array<{ url: string; title: string }>; activeTabIndex: number } {
    return {
      currentUrl: this.pageUrl,
      tabs: [{ url: this.pageUrl, title: "Brainkit Harness Browser Fixture" }],
      activeTabIndex: 0,
    };
  }

  async getCurrentUrl(): Promise<string | null> {
    return this.pageUrl;
  }

  getTools() {
    return {
      browser_read_current_page: createTool({
        id: "browser_read_current_page",
        description:
          "Read the current browser page. Always use this tool when the user asks for the browser fixture secret.",
        inputSchema: z.object({
          expected: z.string(),
        }),
        outputSchema: z.object({
          ok: z.boolean(),
          provider: z.string(),
          url: z.string(),
          sessionId: z.string(),
          contextProvider: z.string(),
          contextUrl: z.string(),
        }),
        execute: async ({ expected }: any, context: any) => {
          this.toolCalls++;
          this.lastToolContext = context?.requestContext?.get("browser");
          await this.ensureReady();
          return {
            ok: expected === "BROWSER_FIXTURE_SECRET",
            provider: this.provider,
            url: this.pageUrl,
            sessionId: this.getSessionId(context?.threadId),
            contextProvider: String(this.lastToolContext?.provider || ""),
            contextUrl: String(this.lastToolContext?.currentUrl || ""),
          };
        },
      }),
    };
  }
}

const events: any[] = [];
const storage = new InMemoryStore();
const memory = new Memory({
  storage,
  options: { lastMessages: 10 },
});
const browser = new FixtureBrowser();

const agent = new Agent({
  name: "harness-browser-agent",
  model: model("openai", "gpt-4o-mini"),
  instructions: [
    "You are a deterministic Harness browser fixture agent.",
    'You must call browser_read_current_page exactly once with expected "BROWSER_FIXTURE_SECRET".',
    "After the browser tool result, reply with exactly BROWSER_FIXTURE_OK and no punctuation.",
  ].join("\n"),
  memory,
  maxSteps: 3,
});

const harness = new Harness({
  id: "harness-browser-basic",
  storage,
  memory,
  initialState: { yolo: true },
  browser,
  modes: [
    {
      id: "default",
      name: "Default",
      default: true,
      agent,
    },
  ],
});

const requestContext = new RequestContext();
const unsubscribe = harness.subscribe((event) => {
  events.push(event);
});

let harnessDestroyed = false;
let browserClosed = false;

try {
  const processor = new BrowserContextProcessor();
  await harness.init();

  const propagatedBrowser = agent.browser === browser;
  const hasOwnBrowser = agent.hasOwnBrowser();
  const listedTools = await agent.listTools({ requestContext });
  const browserInputProcessors = browser.getInputProcessors([]);

  await withTimeout("sendMessage", 120_000, () =>
    harness.sendMessage({
      content:
        'Use the browser_read_current_page tool with expected "BROWSER_FIXTURE_SECRET", then answer BROWSER_FIXTURE_OK.',
      requestContext,
    }),
  );

  const browserContext = requestContext.get("browser") as any;
  const displayState = harness.getDisplayState() as any;
  const assistantEnd = [...events].reverse().find(
    (event) => event.type === "message_end" && event.message?.role === "assistant",
  );
  const text = String(
    assistantEnd?.message?.content
      ?.filter((part: any) => part?.type === "text")
      ?.map((part: any) => part.text)
      ?.join("") || "",
  );

  await harness.destroy();
  harnessDestroyed = true;
  await browser.close();
  browserClosed = true;

  output({
    mastraBrowserExport: typeof MastraBrowser === "function",
    browserContextProcessorExport: typeof BrowserContextProcessor === "function",
    processorId: processor.id === "browser-context",
    browserLaunchHook: browser.launchHookBrowserId === browser.id,
    propagatedBrowser,
    hasOwnBrowser,
    listToolsDoesNotExposeBrowser: !("browser_read_current_page" in listedTools),
    browserInputProcessorAvailable: browserInputProcessors.some((candidate: any) => candidate?.id === "browser-context"),
    browserContextProvider: browserContext?.provider === browser.provider,
    browserContextProviderType: browserContext?.providerType === "sdk",
    browserContextSessionId: browserContext?.sessionId === browser.id,
    browserContextUrlAbsentBeforeToolLaunch: typeof browserContext?.currentUrl === "undefined",
    toolCalledOnce: browser.toolCalls === 1,
    toolContextProvider: browser.lastToolContext?.provider === browser.provider,
    toolContextUrlAbsentBeforeToolLaunch: typeof browser.lastToolContext?.currentUrl === "undefined",
    toolStarted: events.some((event) => event.type === "tool_start" && event.toolName === "browser_read_current_page"),
    toolEnded: events.some((event) => event.type === "tool_end"),
    toolApprovalNotRequired: !events.some((event) => event.type === "tool_approval_required"),
    agentEndedComplete: events.some((event) => event.type === "agent_end" && event.reason === "complete"),
    messageEnded: Boolean(assistantEnd),
    containsToken: text.includes("BROWSER_FIXTURE_OK"),
    displayIdle: displayState.isRunning === false,
    browserClosed,
    browserCloseHook: browser.closeHookBrowserId === browser.id,
  });
} finally {
  unsubscribe();
  if (!harnessDestroyed) await harness.destroy();
  if (!browserClosed) await browser.close().catch(() => undefined);
}
