import {
  Agent,
  AgentChannels,
  ChatChannelProcessor,
  InMemoryStore,
  Mastra,
  RequestContext,
  type ChannelAdapter,
} from "agent";
import { model, output } from "kit";

const adapter: ChannelAdapter = {
  name: "local",
  userName: "BrainkitBot",
  async postMessage() {
    return { id: "sent-1", text: "ok" };
  },
  async editMessage() {},
  async deleteMessage() {},
  async addReaction() {},
  async removeReaction() {},
  async handleWebhook() {
    return new Response("ok", { status: 200 });
  },
  async initialize() {},
  async fetchMessages() {
    return [];
  },
  encodeThreadId: (...parts: string[]) => parts.join(":"),
  decodeThreadId: (id: string) => id.split(":"),
  channelIdFromThreadId: (id: string) => id.split(":").slice(0, 2).join(":"),
  renderFormatted: (message: string | Record<string, unknown>) => message,
  async fetchThread() {
    return null;
  },
  async startTyping() {},
  parseMessage: (raw: unknown) => raw,
};

const agent = new Agent({
  id: "channel-agent",
  name: "Channel Agent",
  instructions: "You are a channel-aware agent.",
  model: model("openai", "gpt-4o-mini"),
  channels: {
    adapters: { local: adapter },
    userName: "BrainkitBot",
    tools: true,
    handlers: {
      onMention: false,
    },
  },
});

const channels = agent.getChannels();
if (!(channels instanceof AgentChannels)) {
  throw new Error("agent did not create AgentChannels");
}

const mastra = new Mastra({
  agents: { "channel-agent": agent },
  storage: new InMemoryStore(),
  server: {
    apiRoutes: [
      {
        path: "/api/custom",
        method: "GET",
        createHandler: async () => async () => new Response("ok"),
      },
    ],
  },
});

const aggregated = mastra.getChannels();
const routes = channels.getWebhookRoutes();
const serverRoutes = (mastra.getServer()?.apiRoutes ?? []).map((route: any) => route.path);
const channelTools = channels.getTools();

const requestContext = new RequestContext();
requestContext.set("channel", {
  platform: "local",
  isDM: false,
  userId: "u-1",
  userName: "Dana",
  botUserName: "BrainkitBot",
  botMention: "@BrainkitBot",
});

const processor = new ChatChannelProcessor();
const processed = processor.processInputStep({
  requestContext,
  systemMessages: [],
});
const systemText = String(processed?.systemMessages?.[0]?.content ?? "");

output({
  hasAgentChannels: channels instanceof AgentChannels,
  adapterKeys: Object.keys(channels.adapters).join(","),
  hasLocalAdapter: channels.hasAdapter("local"),
  hasMissingAdapter: channels.hasAdapter("missing"),
  routeCount: routes.length,
  routePath: routes[0]?.path,
  routeMethod: routes[0]?.method,
  routeRequiresAuth: routes[0]?.requiresAuth,
  serverHasCustomRoute: serverRoutes.includes("/api/custom"),
  serverHasChannelRoute: serverRoutes.includes("/api/agents/channel-agent/channels/local/webhook"),
  mastraAggregated: aggregated["channel-agent"] instanceof AgentChannels,
  addReactionTool: typeof channelTools.add_reaction?.execute === "function",
  removeReactionTool: typeof channelTools.remove_reaction?.execute === "function",
  processorId: processor.id,
  processorMentionsPlatform: systemText.includes("communicating via local"),
  processorMentionsPublicThread: systemText.includes("public channel or thread"),
  processorMentionsBotIdentity: systemText.includes("BrainkitBot") && systemText.includes("@BrainkitBot"),
});
