import {
  Agent,
  Harness,
  InMemoryStore,
  Memory,
  assignTaskIds,
  defaultDisplayState,
  defaultOMProgressState,
  parseSubagentMeta,
} from "agent";
import { model, output } from "kit";

function resolveHarnessModel(modelId: string) {
  const slash = modelId.indexOf("/");
  if (slash < 0) return model("openai", modelId);
  return model(modelId.slice(0, slash), modelId.slice(slash + 1));
}

async function waitMicrotask() {
  await Promise.resolve();
}

async function* omStream() {
  yield {
    type: "data-om-buffering-start",
    data: {
      cycleId: "buffer-observation-1",
      operationType: "observation",
      tokensToBuffer: 120,
    },
  };
  yield {
    type: "data-om-buffering-end",
    data: {
      cycleId: "buffer-observation-1",
      operationType: "observation",
      tokensBuffered: 120,
      bufferedTokens: 120,
      observations: "- buffered observation",
    },
  };
  yield {
    type: "data-om-observation-start",
    data: {
      cycleId: "observe-1",
      operationType: "observation",
      tokensToObserve: 222,
    },
  };
  yield {
    type: "data-om-observation-end",
    data: {
      cycleId: "observe-1",
      operationType: "observation",
      durationMs: 17,
      tokensObserved: 222,
      observationTokens: 333,
      observations: "- User likes cobalt UI accents",
      currentTask: "observe preference",
      suggestedResponse: "Noted.",
    },
  };
  yield {
    type: "data-om-observation-start",
    data: {
      cycleId: "reflect-1",
      operationType: "reflection",
      tokensToObserve: 444,
    },
  };
  yield {
    type: "data-om-observation-end",
    data: {
      cycleId: "reflect-1",
      operationType: "reflection",
      durationMs: 29,
      observationTokens: 111,
      observations: "- User prefers cobalt accents",
    },
  };
  yield {
    type: "data-om-activation",
    data: {
      cycleId: "activation-1",
      operationType: "observation",
      chunksActivated: 1,
      tokensActivated: 50,
      observationTokens: 333,
      messagesActivated: 2,
      generationCount: 4,
      triggeredBy: "threshold",
    },
  };
  yield {
    type: "data-om-thread-update",
    data: {
      cycleId: "title-1",
      threadId: "thread-om",
      oldTitle: "Old title",
      newTitle: "Cobalt UI Preferences",
    },
  };
  yield { type: "finish", payload: { stepResult: { reason: "stop" } } };
}

async function* omFailureStream() {
  yield {
    type: "data-om-buffering-failed",
    data: {
      cycleId: "buffer-failed-1",
      operationType: "observation",
      error: "fixture buffering failure",
    },
  };
}

const events: any[] = [];
const storage = new InMemoryStore();
const memory = new Memory({
  storage,
  options: {
    lastMessages: 10,
    observationalMemory: {
      model: () => resolveHarnessModel("openai/gpt-4o-mini"),
      observation: {
        messageTokens: 128,
        bufferTokens: false,
      },
      reflection: {
        observationTokens: 256,
      },
    },
  },
});

const agent = new Agent({
  name: "harness-om-agent",
  model: model("openai", "gpt-4o-mini"),
  instructions: "You are a deterministic Harness observational memory fixture agent.",
  memory,
});

const harness = new Harness<{
  observerModelId?: string;
  reflectorModelId?: string;
  observationThreshold?: number;
  reflectionThreshold?: number;
}>({
  id: "harness-observational-memory-basic",
  storage,
  memory,
  initialState: {
    observationThreshold: 30_000,
    reflectionThreshold: 40_000,
  },
  omConfig: {
    defaultObserverModelId: "openai/gpt-4o-mini",
    defaultReflectorModelId: "openai/gpt-4o-mini",
    defaultObservationThreshold: 30_000,
    defaultReflectionThreshold: 40_000,
  },
  resolveModel: resolveHarnessModel,
  modes: [
    {
      id: "default",
      name: "Default",
      default: true,
      defaultModelId: "openai/gpt-4o-mini",
      agent,
    },
  ],
});

const unsubscribe = harness.subscribe((event) => {
  events.push(event);
});

let destroyed = false;

try {
  const defaultOM = defaultOMProgressState();
  const displayA = defaultDisplayState();
  const displayB = defaultDisplayState();
  displayA.tasks.push({ id: "task-a", content: "mutate only A", status: "pending", activeForm: "Mutating" });
  const assignedTasks = assignTaskIds([{ content: "capture OM", status: "pending", activeForm: "Capturing" }]);
  const parsedMeta = parseSubagentMeta(
    'done\n<subagent-meta modelId="openai/gpt-4o-mini" durationMs="123" tools="read:ok,write:err" />',
  );

  await harness.init();
  const defaultObserverBeforeSwitch = harness.getObserverModelId();
  const defaultReflectorBeforeSwitch = harness.getReflectorModelId();
  const resolvedObserver = harness.getResolvedObserverModel();
  const resolvedReflector = harness.getResolvedReflectorModel();

  const threadA = await harness.createThread({ title: "OM thread A" });
  await harness.setState({ observationThreshold: 12_000, reflectionThreshold: 21_000 });
  await harness.setThreadSetting({ key: "observationThreshold", value: 12_000 });
  await harness.setThreadSetting({ key: "reflectionThreshold", value: 21_000 });

  await harness.switchObserverModel({ modelId: "openai/gpt-4o-mini" });
  await harness.switchReflectorModel({ modelId: "openai/gpt-4o-mini" });
  await waitMicrotask();

  await harness.createThread({ title: "OM thread B" });
  await harness.setState({ observationThreshold: 33_000, reflectionThreshold: 44_000 });
  await harness.switchThread({ threadId: threadA.id });

  const memoryStorage = (await storage.getStore("memory")) as any;
  const record = await memoryStorage.initializeObservationalMemory({
    threadId: threadA.id,
    resourceId: harness.getResourceId(),
    scope: "thread",
    config: {
      observationThreshold: { min: 10_000, max: 12_000 },
      reflectionThreshold: { min: 20_000, max: 21_000 },
    },
  });
  await memoryStorage.updateActiveObservations({
    id: record.id,
    observations: "- User prefers cobalt UI accents\n- User asks for rigorous live fixtures",
    tokenCount: 99,
    lastObservedAt: new Date(),
  });
  await memoryStorage.saveMessages({
    messages: [
      {
        id: "om-status-message",
        role: "assistant",
        threadId: threadA.id,
        resourceId: harness.getResourceId(),
        createdAt: new Date(),
        content: {
          format: 2,
          parts: [
            {
              type: "data-om-status",
              data: {
                windows: {
                  active: {
                    messages: { tokens: 6_000, threshold: 12_000 },
                    observations: { tokens: 4_200, threshold: 21_000 },
                  },
                  buffered: {
                    observations: {
                      status: "running",
                      chunks: 2,
                      messageTokens: 1_200,
                      projectedMessageRemoval: 800,
                      observationTokens: 600,
                    },
                    reflection: {
                      status: "idle",
                      inputObservationTokens: 0,
                      observationTokens: 0,
                    },
                  },
                },
                recordId: record.id,
                threadId: threadA.id,
                stepNumber: 7,
                generationCount: 3,
              },
            },
          ],
          content: "",
        },
      },
    ],
  });

  await harness.loadOMProgress();
  const loadedRecord = await harness.getObservationalMemoryRecord();
  const afterLoadDisplay = harness.getDisplayState() as any;
  const loadProgressPendingTokens = afterLoadDisplay.omProgress.pendingTokens === 6_000;
  const loadProgressObservationTokens = afterLoadDisplay.omProgress.observationTokens === 4_200;
  const loadProgressBufferedMessages = afterLoadDisplay.bufferingMessages === true;
  const savedThread = await memoryStorage.getThreadById({ threadId: threadA.id });

  await (harness as any).processStream({ fullStream: omStream() });
  const afterStreamDisplay = harness.getDisplayState() as any;
  await (harness as any).processStream({ fullStream: omFailureStream() });
  const errorEvent = events.find(
    (event) =>
      event.type === "error" &&
      String(event.error?.message || "").includes("Observational memory observation buffering failed"),
  );

  await harness.destroy();
  destroyed = true;

  output({
    defaultOMProgressExport:
      defaultOM.status === "idle" &&
      defaultOM.threshold === 30_000 &&
      defaultOM.reflectionThreshold === 40_000,
    defaultDisplayIndependent: displayB.tasks.length === 0,
    assignTaskIdsExport: typeof assignedTasks[0]?.id === "string" && assignedTasks[0].content === "capture OM",
    parseSubagentMetaExport:
      parsedMeta.text.trim() === "done" &&
      parsedMeta.modelId === "openai/gpt-4o-mini" &&
      parsedMeta.durationMs === 123 &&
      parsedMeta.toolCalls?.some((tool) => tool.name === "write" && tool.isError === true) === true,
    defaultObserverModel: defaultObserverBeforeSwitch === "openai/gpt-4o-mini",
    defaultReflectorModel: defaultReflectorBeforeSwitch === "openai/gpt-4o-mini",
    resolvedObserverModel: typeof resolvedObserver === "object",
    resolvedReflectorModel: typeof resolvedReflector === "object",
    observerSwitchEvent: events.some(
      (event) => event.type === "om_model_changed" && event.role === "observer" && event.modelId === "openai/gpt-4o-mini",
    ),
    reflectorSwitchEvent: events.some(
      (event) => event.type === "om_model_changed" && event.role === "reflector" && event.modelId === "openai/gpt-4o-mini",
    ),
    threadThresholdRestored:
      harness.getObservationThreshold() === 12_000 && harness.getReflectionThreshold() === 21_000,
    threadMetadataPersisted:
      savedThread?.metadata?.observationThreshold === 12_000 &&
      savedThread?.metadata?.reflectionThreshold === 21_000,
    loadedRecordFound: loadedRecord?.id === record.id,
    loadedRecordObservations: String(loadedRecord?.activeObservations || "").includes("cobalt UI accents"),
    loadProgressEvent: events.some((event) => event.type === "om_status" && event.recordId === record.id),
    loadProgressPendingTokens,
    loadProgressObservationTokens,
    loadProgressBufferedMessages,
    streamObservationStart: events.some((event) => event.type === "om_observation_start" && event.cycleId === "observe-1"),
    streamObservationEnd: events.some(
      (event) =>
        event.type === "om_observation_end" &&
        event.cycleId === "observe-1" &&
        event.observationTokens === 333,
    ),
    streamReflectionStart: events.some((event) => event.type === "om_reflection_start" && event.cycleId === "reflect-1"),
    streamReflectionEnd: events.some(
      (event) => event.type === "om_reflection_end" && event.cycleId === "reflect-1" && event.compressedTokens === 111,
    ),
    streamActivation: events.some(
      (event) =>
        event.type === "om_activation" &&
        event.cycleId === "activation-1" &&
        event.triggeredBy === "threshold",
    ),
    streamThreadTitleUpdate: events.some(
      (event) =>
        event.type === "om_thread_title_updated" &&
        event.threadId === "thread-om" &&
        event.newTitle === "Cobalt UI Preferences",
    ),
    streamDisplayIdle:
      afterStreamDisplay.omProgress.status === "idle" &&
      afterStreamDisplay.bufferingMessages === false &&
      afterStreamDisplay.bufferingObservations === false,
    failureAbortError: Boolean(errorEvent),
    failureAgentAborted: events.some((event) => event.type === "agent_end" && event.reason === "aborted"),
  });
} finally {
  unsubscribe();
  if (!destroyed) {
    await harness.destroy();
  }
}
