// agent-embed entry point
// Mastra framework + AI SDK providers for QuickJS embedding

import { Agent, TripWire, MessageList, convertMessages, TypeDetector } from '@mastra/core/agent';
import { createTool } from '@mastra/core/tools';
import { createWorkflow, createStep, Workflow, cloneWorkflow, cloneStep, mapVariable } from '@mastra/core/workflows';
import { Mastra } from '@mastra/core/mastra';
import { BackgroundTaskManager, createBackgroundTask, generateBackgroundTaskSystemPrompt } from '@mastra/core/background-tasks';
import { ConsoleLogger, MultiLogger, DualLogger } from '@mastra/core/logger';
import { Memory } from '@mastra/memory';
import { MockMemory } from '@mastra/core/memory';
import { InMemoryStore } from '@mastra/core/storage';
import { LibSQLStore, LibSQLVector } from '@mastra/libsql';
import { UpstashStore } from '@mastra/upstash';
// TCP-based stores — require jsbridge/net.go polyfill at runtime
import { PostgresStore, PgVector } from '@mastra/pg';
import { MongoDBStore, MongoDBVector } from '@mastra/mongodb';
// Managed vector backends. Each extends Error via older TS __extends —
// safe now because the bundle eval precedes SES lockdown.
import { PineconeVector } from '@mastra/pinecone';
import { ChromaVector } from '@mastra/chroma';
import { QdrantVector } from '@mastra/qdrant';
// With the zod-unify esbuild plugin, 'zod' resolves to 'zod/v4'.
// ONE Zod version everywhere — z.toJSONSchema exists because v4 includes it.
import { z, toJSONSchema } from 'zod';
// Register for the dynamic require("zod/v4") resolution via createRequire stub.
globalThis.__zod_v4_module = { z, toJSONSchema };
import {
  // core generation
  embed,
  embedMany,
  generateText,
  streamText,
  generateObject,
  streamObject,
  // tool authoring
  tool,
  dynamicTool,
  jsonSchema,
  zodSchema,
  asSchema,
  generateId,
  createIdGenerator,
  hasToolCall,
  stepCountIs,
  isLoopFinished,
  // middleware
  wrapLanguageModel,
  wrapEmbeddingModel,
  wrapImageModel,
  wrapProvider,
  extractReasoningMiddleware,
  extractJsonMiddleware,
  defaultSettingsMiddleware,
  defaultEmbeddingSettingsMiddleware,
  simulateStreamingMiddleware,
  smoothStream,
  addToolInputExamplesMiddleware,
  // provider registry
  createProviderRegistry,
  customProvider,
  experimental_createProviderRegistry,
  experimental_customProvider,
  // message utils
  convertToModelMessages,
  pruneMessages,
  validateUIMessages,
  safeValidateUIMessages,
  readUIMessageStream,
  consumeStream,
  convertFileListToFileUIParts,
  // media
  generateImage,
  experimental_generateImage,
  experimental_generateVideo,
  experimental_transcribe,
  experimental_generateSpeech,
  // misc
  cosineSimilarity,
  simulateReadableStream,
  parsePartialJson,
  parseJsonEventStream,
  // gateway
  gateway,
  createGateway,
  // errors
  AISDKError,
  APICallError,
  NoObjectGeneratedError,
  NoSuchModelError,
  NoSuchToolError,
  InvalidArgumentError,
  InvalidDataContentError,
  InvalidPromptError,
  InvalidToolInputError,
  NoContentGeneratedError,
  NoSpeechGeneratedError,
  NoTranscriptGeneratedError,
  NoVideoGeneratedError,
  RetryError,
  ToolCallRepairError,
  TypeValidationError,
  MessageConversionError,
  MissingToolResultsError,
  LoadAPIKeyError,
  InvalidToolApprovalError,
  ToolCallNotFoundForApprovalError,
} from 'ai';
import { ModelRouterEmbeddingModel } from '@mastra/core/llm';
import { RequestContext } from '@mastra/core/request-context';

// Evals — scorer infrastructure from core + pre-built rule-based scorers
import { createScorer, runEvals, MastraScorer } from '@mastra/core/evals';
import { registerHook, executeHook, AvailableHooks } from '@mastra/core/hooks';
import {
  // Rule-based (no LLM)
  createCompletenessScorer,
  createTextualDifferenceScorer,
  createKeywordCoverageScorer,
  createContentSimilarityScorer,
  createToneScorer,
  createToolCallAccuracyScorerCode,
  createTrajectoryAccuracyScorerCode,
  createTrajectoryScorerCode,
  // LLM-based (require judge model)
  createHallucinationScorer,
  createFaithfulnessScorer,
  createAnswerRelevancyScorer,
  createAnswerSimilarityScorer,
  createBiasScorer,
  createToxicityScorer,
  createContextPrecisionScorer,
  createContextRelevanceScorerLLM,
  createNoiseSensitivityScorerLLM,
  createPromptAlignmentScorerLLM,
  createToolCallAccuracyScorerLLM,
  createTrajectoryAccuracyScorerLLM,
} from '@mastra/evals/scorers/prebuilt';

// Processors — built-in input/output middleware
import {
  ModerationProcessor,
  PromptInjectionDetector,
  PIIDetector,
  SystemPromptScrubber,
  UnicodeNormalizer,
  LanguageDetector,
  TokenLimiterProcessor,
  BatchPartsProcessor,
  StructuredOutputProcessor,
  ToolCallFilter,
  ToolSearchProcessor,
  AgentsMDInjector,
  SkillsProcessor,
  SkillSearchProcessor,
  WorkspaceInstructionsProcessor,
  ResponseCache,
  DEFAULT_RESPONSE_CACHE_TTL_SECONDS,
  RESPONSE_CACHE_CONTEXT_KEY,
  buildResponseCacheKey,
} from '@mastra/core/processors';
import { InMemoryServerCache, MastraServerCache } from '@mastra/core/cache';

// RAG — document chunking, vector query tools, graph RAG, reranking
import { MDocument, GraphRAG } from '@mastra/rag';
import { createVectorQueryTool, createDocumentChunkerTool, createGraphRAGTool } from '@mastra/rag';
import { rerank, rerankWithScorer } from '@mastra/rag';
import { CohereRelevanceScorer } from '@mastra/rag';
import { MastraAgentRelevanceScorer } from '@mastra/rag';
import { ZeroEntropyRelevanceScorer } from '@mastra/rag';
// MDocument.fromCSV extension — pure-JS, no SES surprises.
import Papa from 'papaparse';
MDocument.fromCSV = function fromCSV(csv, metadata) {
  const parsed = Papa.parse(String(csv || ''), {
    skipEmptyLines: true,
    header: false,
  });
  const rows = Array.isArray(parsed.data) ? parsed.data : [];
  const text = rows
    .map((row) => (Array.isArray(row) ? row.join(' | ') : String(row)))
    .join('\n');
  return MDocument.fromText(text, metadata || {});
};
// MDocument.fromDocx extension — mammoth's browser build needs writable
// intrinsics during init, which the reorder now allows.
import mammoth from 'mammoth/mammoth.browser.js';
MDocument.fromDocx = async function fromDocx(source, metadata) {
  let arrayBuffer;
  if (source instanceof ArrayBuffer) {
    arrayBuffer = source;
  } else if (source && typeof source === 'object' && source.buffer instanceof ArrayBuffer) {
    arrayBuffer = source.buffer.slice(
      source.byteOffset || 0,
      (source.byteOffset || 0) + source.byteLength,
    );
  } else if (typeof source === 'string') {
    const bin = atob(source);
    const u8 = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) u8[i] = bin.charCodeAt(i);
    arrayBuffer = u8.buffer;
  } else {
    throw new TypeError('MDocument.fromDocx: source must be ArrayBuffer, Uint8Array, or base64 string');
  }
  const { value } = await mammoth.extractRawText({ arrayBuffer });
  return MDocument.fromText(value || '', metadata || {});
};
// MDocument.fromPDF via pdfjs-dist legacy build. Worker disabled —
// brainkit's QuickJS runtime is single-threaded so pdfjs runs inline.
// dom-stubs must import BEFORE pdfjs so DOMMatrix/Path2D/ImageData are
// in place when pdfjs evaluates its top-level references.
import './dom-stubs.mjs';
import * as pdfjs from 'pdfjs-dist/legacy/build/pdf.mjs';
import * as pdfjsWorker from 'pdfjs-dist/legacy/build/pdf.worker.mjs';
import { ecosystemCanaries } from './ecosystem-canaries.mjs';
// pdfjs picks the main-thread fake-worker path when
// globalThis.pdfjsWorker.WorkerMessageHandler is available — skipping
// the dynamic import of a worker script that our single-threaded
// QuickJS runtime can't execute.
globalThis.pdfjsWorker = { WorkerMessageHandler: pdfjsWorker.WorkerMessageHandler };
MDocument.fromPDF = async function fromPDF(source, metadata) {
  let data;
  if (source instanceof Uint8Array) {
    data = source;
  } else if (source instanceof ArrayBuffer) {
    data = new Uint8Array(source);
  } else if (source && typeof source === 'object' && source.buffer instanceof ArrayBuffer) {
    data = new Uint8Array(source.buffer, source.byteOffset || 0, source.byteLength);
  } else if (typeof source === 'string') {
    const bin = atob(source);
    data = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) data[i] = bin.charCodeAt(i);
  } else {
    throw new TypeError('MDocument.fromPDF: source must be ArrayBuffer, Uint8Array, or base64 string');
  }
  // pdfjs transfers ArrayBuffer ownership on its inbound fetch path —
  // pass a detached copy so callers' buffer (and our checks) stay
  // intact, and so pdfjs's size check sees the true byteLength.
  const dataCopy = new Uint8Array(data.byteLength);
  dataCopy.set(data);
  const doc = await pdfjs.getDocument({
    data: dataCopy,
    disableWorker: true,
    disableFontFace: true,
    useSystemFonts: false,
    isEvalSupported: false,
    useWorkerFetch: false,
    verbosity: 0,
  }).promise;
  const pages = [];
  for (let p = 1; p <= doc.numPages; p++) {
    const page = await doc.getPage(p);
    const tc = await page.getTextContent();
    pages.push(tc.items.map((it) => it.str || '').join(' '));
  }
  await doc.destroy();
  return MDocument.fromText(pages.join('\n\n'), metadata || {});
};

// Observability — tracing, spans, exporters
import {
  Observability,
  DefaultExporter,
  SensitiveDataFilter,
  BaseExporter,
  CloudExporter,
  ConsoleExporter,
  TestExporter,
  TrackingExporter,
  chainFormatters,
} from '@mastra/observability';
import {
  AlwaysOnSampler,
  AlwaysOffSampler,
  ParentBasedSampler,
  TraceIdRatioBasedSampler,
  NoopSpanProcessor,
  ConsoleSpanExporter,
  InMemorySpanExporter,
  BasicTracerProvider,
  SimpleSpanProcessor,
  BatchSpanProcessor,
} from '@opentelemetry/sdk-trace-base';

// Workspace — filesystem, sandbox, skills, search
import {
  Workspace,
  LocalFilesystem,
  LocalSandbox,
  CompositeFilesystem,
  WORKSPACE_TOOLS_PREFIX,
  WORKSPACE_TOOLS,
  createWorkspaceTools,
  resolveToolConfig,
  readFileTool,
  writeFileTool,
  editFileTool,
  listFilesTool,
  deleteFileTool,
  fileStatTool,
  mkdirTool,
  searchTool,
  indexContentTool,
  executeCommandTool,
  requireWorkspace,
  requireFilesystem,
  requireSandbox,
} from '@mastra/core/workspace';

// Browser — core provider base + context processor. Concrete browser
// providers such as @mastra/agent-browser and Stagehand stay outside the
// curated embed until the explicit npm resolver/provider story can own their
// dependencies, browser processes, and gateway lifecycle.
import { MastraBrowser, BrowserContextProcessor } from '@mastra/core/browser';

// Channels — core orchestration primitives. Concrete Slack/Discord/Telegram
// adapters stay outside the curated embed until provider credentials, gateway
// routes, and adapter lifecycle are owned explicitly.
import { AgentChannels, ChatChannelProcessor, MastraStateAdapter } from '@mastra/core/channels';

// Voice — STT/TTS. CompositeVoice lives in @mastra/core, the
// OpenAI provider lives in its own package.
// MastraVoice is the abstract base every provider extends.
// Exposing it lets .ts code type-check custom voice provider
// subclasses without leaving the "agent" module.
import { CompositeVoice, MastraVoice, DefaultVoice, AISDKSpeech, AISDKTranscription } from '@mastra/core/voice';
import { OpenAIVoice } from '@mastra/voice-openai';
import { OpenAIRealtimeVoice } from '@mastra/voice-openai-realtime';
import { AzureVoice } from '@mastra/voice-azure';
import { ElevenLabsVoice } from '@mastra/voice-elevenlabs';
// GoogleVoice (classic) pulls @google-cloud/speech which
// requires a full gRPC-over-HTTP2 polyfill brainkit doesn't
// ship; GeminiLiveVoice below uses the HTTP @google/genai SDK
// + ws which the runtime already polyfills. Skip classic
// GoogleVoice for now; users needing Google TTS/STT route
// through Gemini Live.
import { CloudflareVoice } from '@mastra/voice-cloudflare';
import { DeepgramVoice } from '@mastra/voice-deepgram';
import { PlayAIVoice, PLAYAI_VOICES } from '@mastra/voice-playai';
import { SpeechifyVoice } from '@mastra/voice-speechify';
import { SarvamVoice } from '@mastra/voice-sarvam';
import { MurfVoice } from '@mastra/voice-murf';
// GeminiLiveVoice temporarily removed — transitive deps
// caused SES lockdown to reject the bundle ("not a prototype"
// on load). Needs a focused investigation; track separately.
// import { GeminiLiveVoice } from '@mastra/voice-google-gemini-live';

// Harness — orchestration layer for agent execution, threads, modes, tool approval
import { Harness } from '@mastra/core/harness';
import {
  assignTaskIds,
  askUserTool,
  defaultDisplayState,
  defaultOMProgressState,
  parseSubagentMeta,
  submitPlanTool,
  taskWriteTool,
  taskUpdateTool,
  taskCompleteTool,
  taskCheckTool,
} from '@mastra/core/harness';

function cloneJSON(value) {
  return JSON.parse(JSON.stringify(value));
}

function standardizeHarnessToolSchemaForSES(schema) {
  if (!schema) return;
  if (typeof schema === 'function') {
    schema = schema();
  }
  if (!schema || typeof schema !== 'object') return;
  // Zod v4 lazily materializes object-shape metadata with
  // Object.defineProperty while traversing schemas. The bundle is evaluated
  // before SES lockdown, but Harness tool schemas are normally traversed
  // later during Agent.prepareTools(). Warm them now while intrinsics and
  // schema definition objects are still writable.
  if (!schema._zod) return schema;

  const inputJSONSchema = toJSONSchema(schema, { target: 'draft-7', io: 'input' });
  const outputJSONSchema = toJSONSchema(schema, { target: 'draft-7', io: 'output' });
  return {
    '~standard': {
      version: 1,
      vendor: 'brainkit-precomputed-zod',
      validate(value) {
        const result = schema.safeParse(value);
        if (result.success) {
          return { value: result.data };
        }
        return {
          issues: (result.error?.issues || []).map((issue) => ({
            message: issue.message || 'Validation error',
            path: issue.path || [],
          })),
        };
      },
      jsonSchema: {
        input() {
          return cloneJSON(inputJSONSchema);
        },
        output() {
          return cloneJSON(outputJSONSchema);
        },
      },
    },
  };
}

function standardizeHarnessToolSchemasForSES() {
  standardizeToolSchemasForSES([
    askUserTool,
    submitPlanTool,
    taskWriteTool,
    taskUpdateTool,
    taskCompleteTool,
    taskCheckTool,
  ]);
}

function standardizeWorkspaceToolSchemasForSES() {
  standardizeToolSchemasForSES([
    readFileTool,
    writeFileTool,
    editFileTool,
    listFilesTool,
    deleteFileTool,
    fileStatTool,
    mkdirTool,
    searchTool,
    indexContentTool,
    executeCommandTool,
  ]);
}

function standardizeToolSchemasForSES(tools) {
  for (const tool of tools) {
    if (!tool) continue;
    const inputSchema = standardizeHarnessToolSchemaForSES(tool.inputSchema);
    if (inputSchema) tool.inputSchema = inputSchema;
    const outputSchema = standardizeHarnessToolSchemaForSES(tool.outputSchema);
    if (outputSchema) tool.outputSchema = outputSchema;
  }
}

standardizeHarnessToolSchemasForSES();
standardizeWorkspaceToolSchemasForSES();

function installBrainkitHarnessSendMessageWait() {
  const proto = Harness?.prototype;
  if (!proto || proto.__brainkitSendMessageWaitInstalled) return;
  Object.defineProperty(proto, '__brainkitSendMessageWaitInstalled', { value: true });

  const originalSendMessage = proto.sendMessage;
  proto.sendMessage = async function brainkitSendMessage(...args) {
    const wasActive =
      typeof this.isCurrentThreadStreamActive === 'function' &&
      this.isCurrentThreadStreamActive();
    let sawRunStart = false;
    let sawRunEnd = false;

    const unsubscribe =
      typeof this.subscribe === 'function'
        ? this.subscribe((event) => {
            if (event?.type === 'agent_start') {
              sawRunStart = true;
              sawRunEnd = false;
              return;
            }
            if (sawRunStart && event?.type === 'agent_end') {
              sawRunEnd = true;
            }
          })
        : undefined;

    try {
      const result = await originalSendMessage.apply(this, args);
      if (wasActive || typeof this.isRunning !== 'function') return result;

      const timeoutMs = Number(globalThis.__brainkit_harness_send_message_timeout_ms ?? 300000);
      const startedAt = Date.now();
      while (this.isRunning() && !sawRunEnd) {
        if (timeoutMs > 0 && Date.now() - startedAt > timeoutMs) {
          throw new Error('Harness.sendMessage timed out waiting for the Brainkit stream run to become idle');
        }
        await new Promise((resolve) => setTimeout(resolve, 5));
      }
      return result;
    } finally {
      unsubscribe?.();
    }
  };
}

installBrainkitHarnessSendMessageWait();

function installBrainkitHarnessToolApprovalResume() {
  const proto = Harness?.prototype;
  if (!proto || proto.__brainkitToolApprovalResumeInstalled) return;
  Object.defineProperty(proto, '__brainkitToolApprovalResumeInstalled', { value: true });

  async function processApprovalResumeOutput(harness, output, requestContext) {
    if (!output || typeof output !== 'object' || !output.fullStream) return;
    if (typeof harness.processStream !== 'function') return;
    await harness.processStream(output, requestContext);
    // Brainkit consumes the resume stream directly. The original subscribed
    // stream has no more useful work here, and leaving it active keeps
    // Harness.sendMessage() waiting on the old run forever.
    if (typeof harness.cleanupAgentThreadSubscription === 'function') {
      harness.cleanupAgentThreadSubscription();
    }
  }

  proto.handleToolApprove = async function brainkitHandleToolApprove({ toolCallId, requestContext }) {
    if (!this.currentRunId) {
      throw new Error('No active run to approve tool call for');
    }

    const agent = this.getCurrentAgent();
    if (!this.abortController) {
      this.abortController = new AbortController();
    }

    const resolvedRequestContext = await this.buildRequestContext(requestContext);
    const isYolo = this.state?.yolo === true;
    const output = await agent.approveToolCall({
      runId: this.currentRunId,
      toolCallId,
      requireToolApproval: !isYolo,
      memory: this.currentThreadId ? { thread: this.currentThreadId, resource: this.resourceId } : undefined,
      abortSignal: this.abortController.signal,
      requestContext: resolvedRequestContext,
      toolsets: await this.buildToolsets(resolvedRequestContext),
    });
    await processApprovalResumeOutput(this, output, resolvedRequestContext);
  };

  proto.handleToolDecline = async function brainkitHandleToolDecline({ toolCallId, requestContext }) {
    if (!this.currentRunId) {
      throw new Error('No active run to decline tool call for');
    }

    const agent = this.getCurrentAgent();
    if (!this.abortController) {
      this.abortController = new AbortController();
    }

    const resolvedRequestContext = await this.buildRequestContext(requestContext);
    const isYolo = this.state?.yolo === true;
    const output = await agent.declineToolCall({
      runId: this.currentRunId,
      toolCallId,
      requireToolApproval: !isYolo,
      memory: this.currentThreadId ? { thread: this.currentThreadId, resource: this.resourceId } : undefined,
      abortSignal: this.abortController.signal,
      requestContext: resolvedRequestContext,
      toolsets: await this.buildToolsets(resolvedRequestContext),
    });
    await processApprovalResumeOutput(this, output, resolvedRequestContext);
  };
}

installBrainkitHarnessToolApprovalResume();

// tiktoken: the tiktoken-unify esbuild plugin redirects 'js-tiktoken/lite'
// and 'js-tiktoken/ranks/*' to full 'js-tiktoken'. getTiktoken() in
// @mastra/core/utils/tiktoken.ts now resolves to the already-bundled module.

// LSP dependencies — pre-loaded so Mastra's createRequire('vscode-jsonrpc/node') finds them.
// The LSPClient uses dynamic require() to load these optional deps.
import * as _vscodeJsonrpc from 'vscode-jsonrpc/node';
import * as _vscodeProtocol from 'vscode-languageserver-protocol';
globalThis.__vscode_jsonrpc_node = _vscodeJsonrpc;
globalThis.__vscode_lsp_protocol = _vscodeProtocol;

// execa polyfill — Mastra's LocalProcessManager uses execa to spawn LSP servers.
// Minimal implementation backed by our Go spawn bridge (child_process.spawn).
globalThis.__execa_polyfill = function execa(command, args, options) {
  var shell = options?.shell !== false;
  var cwd = options?.cwd || '';
  var fullCommand = shell
    ? (args?.length ? command + ' ' + args.join(' ') : command)
    : command;

  var proc = globalThis.child_process.spawn(
    shell ? 'sh' : command,
    shell ? ['-c', fullCommand] : (args || []),
    cwd
  );

  var stdoutListeners = [];
  var stderrListeners = [];
  var closeListeners = [];
  var errorListeners = [];

  var result = {
    pid: proc.pid,
    stdout: {
      on: function(ev, fn) { if (ev === 'data') stdoutListeners.push(fn); },
      off: function(ev, fn) { stdoutListeners = stdoutListeners.filter(function(f) { return f !== fn; }); },
    },
    stderr: {
      on: function(ev, fn) { if (ev === 'data') stderrListeners.push(fn); },
      off: function(ev, fn) { stderrListeners = stderrListeners.filter(function(f) { return f !== fn; }); },
    },
    stdin: {
      write: function(data, cb) {
        proc.write(data).then(
          function() { if (cb) cb(null); },
          function(err) { if (cb) cb(err); }
        );
      },
    },
    on: function(ev, fn) {
      if (ev === 'close') closeListeners.push(fn);
      else if (ev === 'error') errorListeners.push(fn);
    },
    off: function(ev, fn) {
      if (ev === 'close') closeListeners = closeListeners.filter(function(f) { return f !== fn; });
      else if (ev === 'error') errorListeners = errorListeners.filter(function(f) { return f !== fn; });
    },
    kill: function() { proc.kill(); },
  };

  // Background: read stdout chunks and dispatch to listeners
  (async function() {
    try {
      while (true) {
        var chunk = await proc.readChunk();
        if (chunk === null) break;
        var buf = typeof Buffer !== 'undefined' ? Buffer.from(chunk) : chunk;
        for (var i = 0; i < stdoutListeners.length; i++) stdoutListeners[i](buf);
      }
    } catch(e) {
      for (var i = 0; i < errorListeners.length; i++) errorListeners[i](e);
    }
    var exitCode = await proc.wait();
    for (var i = 0; i < closeListeners.length; i++) closeListeners[i](exitCode, null);
  })();

  return result;
};

// AI SDK providers
import { createOpenAI } from '@ai-sdk/openai';
import { createAnthropic } from '@ai-sdk/anthropic';
import { createGoogleGenerativeAI } from '@ai-sdk/google';
import { createMistral } from '@ai-sdk/mistral';
import { createXai } from '@ai-sdk/xai';
import { createGroq } from '@ai-sdk/groq';
import { createDeepSeek } from '@ai-sdk/deepseek';
import { createCerebras } from '@ai-sdk/cerebras';
import { createPerplexity } from '@ai-sdk/perplexity';
import { createTogetherAI } from '@ai-sdk/togetherai';
import { createFireworks } from '@ai-sdk/fireworks';
import { createCohere } from '@ai-sdk/cohere';

function installBrainkitBackgroundTaskInlineExecution() {
  const proto = BackgroundTaskManager?.prototype;
  if (!proto || proto.__brainkitInlineExecutionInstalled) return;
  Object.defineProperty(proto, '__brainkitInlineExecutionInstalled', { value: true });

  const originalEnqueue = proto.enqueue;
  const originalResume = proto.resume;
  const originalDrainPending = proto.drainPending;

  const makeTaskId = () => {
    if (globalThis.crypto && typeof globalThis.crypto.randomUUID === 'function') {
      return globalThis.crypto.randomUUID();
    }
    return `task_${Date.now()}_${Math.random().toString(16).slice(2)}`;
  };

  const waitForInlineTaskStarted = async (manager, taskId) => {
    const storage = await manager.getStorage();
    for (let i = 0; i < 100; i++) {
      const task = await storage.getTask(taskId);
      if (!task || task.status !== 'pending') return task;
      await new Promise((resolve) => setTimeout(resolve, 1));
    }
    return storage.getTask(taskId);
  };

  const runInlineTask = async (manager, taskId, resumeData) => {
    const storage = await manager.getStorage();
    let task = await storage.getTask(taskId);
    if (!task || task.status === 'cancelled') {
      manager.deregisterTaskContext(taskId);
      return;
    }

    const startedAt = new Date();
    await storage.updateTask(taskId, {
      status: 'running',
      startedAt,
      suspendPayload: undefined,
      suspendedAt: undefined,
    });
    task = await storage.getTask(taskId);
    if (!task) return;

    await manager.publishLifecycleEvent(resumeData === undefined ? 'task.running' : 'task.resumed', task);
    await manager.runLocalExecutionHook(task);

    while (true) {
      task = await storage.getTask(taskId);
      if (!task || task.status === 'cancelled') {
        manager.deregisterTaskContext(taskId);
        return;
      }

      const ctx = manager.taskContexts.get(taskId);
      const executor = ctx?.executor ?? manager.getStaticExecutor(task.toolName);
      if (!executor) {
        const errorInfo = {
          message:
            `No executor registered for tool "${task.toolName}". ` +
            `Register the tool on Mastra or run the task in the same Brainkit process as the producer.`,
        };
        await storage.updateTask(taskId, { status: 'failed', error: errorInfo, completedAt: new Date() });
        const failedTask = await storage.getTask(taskId);
        if (failedTask) {
          await manager.runLocalCompletionHooks(failedTask, 'failed', { error: errorInfo });
          await manager.publishLifecycleEvent('task.failed', failedTask);
        }
        return;
      }

      const abortController = new AbortController();
      manager.activeAbortControllers.set(taskId, abortController);
      const timeoutHandle = setTimeout(() => {
        abortController.abort(new Error(`Task timed out after ${task.timeoutMs}ms`));
      }, task.timeoutMs);

      let didSuspend = false;
      const onProgress = async (chunk) => {
        const current = await storage.getTask(taskId);
        if (current) await manager.publishLifecycleEvent('task.output', { ...current, chunk });
      };
      const suspend = async (data) => {
        didSuspend = true;
        await storage.updateTask(taskId, {
          status: 'suspended',
          suspendPayload: data,
          suspendedAt: new Date(),
        });
        const suspendedTask = await storage.getTask(taskId);
        if (suspendedTask) {
          await manager.runLocalSuspendHooks(suspendedTask);
          await manager.publishLifecycleEvent('task.suspended', suspendedTask);
        }
      };

      try {
        const result = await executor.execute(task.args, {
          abortSignal: abortController.signal,
          onProgress,
          suspend,
          resumeData,
        });

        if (didSuspend) return;

        const latest = await storage.getTask(taskId);
        if (!latest || latest.status === 'cancelled') {
          manager.deregisterTaskContext(taskId);
          return;
        }

        await storage.updateTask(taskId, { status: 'completed', result, completedAt: new Date() });
        const completedTask = await storage.getTask(taskId);
        if (completedTask) {
          await manager.runLocalCompletionHooks(completedTask, 'completed', { result });
          await manager.publishLifecycleEvent('task.completed', completedTask);
        }
        if (typeof manager.__brainkitDrainInlinePending === 'function') await manager.__brainkitDrainInlinePending();
        return;
      } catch (error) {
        const latest = await storage.getTask(taskId);
        if (!latest || latest.status === 'cancelled') {
          manager.deregisterTaskContext(taskId);
          return;
        }

        const timedOut =
          abortController.signal.aborted ||
          error?.name === 'AbortError' ||
          error?.message === 'Task cancelled' ||
          error?.message?.startsWith('Task timed out after ');

        if (!timedOut && latest.retryCount < latest.maxRetries) {
          await storage.updateTask(taskId, {
            retryCount: latest.retryCount + 1,
            error: undefined,
            startedAt: new Date(),
          });
          resumeData = undefined;
          continue;
        }

        const errorInfo = {
          message: timedOut ? `Task timed out after ${latest.timeoutMs}ms` : (error?.message ?? 'Unknown error'),
          stack: error?.stack,
        };
        await storage.updateTask(taskId, {
          status: timedOut ? 'timed_out' : 'failed',
          error: errorInfo,
          completedAt: new Date(),
        });
        const failedTask = await storage.getTask(taskId);
        if (failedTask) {
          await manager.runLocalCompletionHooks(failedTask, 'failed', { error: errorInfo });
          await manager.publishLifecycleEvent('task.failed', failedTask);
        }
        if (typeof manager.__brainkitDrainInlinePending === 'function') await manager.__brainkitDrainInlinePending();
        return;
      } finally {
        clearTimeout(timeoutHandle);
        manager.activeAbortControllers.delete(taskId);
      }
    }
  };

  proto.__brainkitDrainInlinePending = async function drainInlinePending() {
    if (this.__brainkitDrainingInlinePending) return;
    this.__brainkitDrainingInlinePending = true;
    try {
      const storage = await this.getStorage();
      const { tasks: pending } = await storage.listTasks({
        status: 'pending',
        orderBy: 'createdAt',
        orderDirection: 'asc',
      });

      for (const task of pending) {
        const canRun = typeof this.checkConcurrency === 'function' ? await this.checkConcurrency(task.agentId) : true;
        if (!canRun) return;

        const ctx = this.taskContexts.get(task.id);
        if (ctx?.executor && !globalThis.__brainkit_disable_inline_background_tasks) {
          void runInlineTask(this, task.id).catch((error) => {
            console.error?.('[brainkit] inline background queued task failed', error);
          });
          await waitForInlineTaskStarted(this, task.id);
          continue;
        }

        if (typeof originalDrainPending === 'function') {
          await originalDrainPending.call(this);
        }
        return;
      }
    } finally {
      this.__brainkitDrainingInlinePending = false;
    }
  };

  proto.enqueue = async function enqueueInline(payload, context) {
    if (!context?.executor || globalThis.__brainkit_disable_inline_background_tasks) {
      return originalEnqueue.call(this, payload, context);
    }
    if (this.shuttingDown) {
      throw new Error('BackgroundTaskManager is shutting down, cannot enqueue new tasks');
    }
    if (this.initPromise) await this.initPromise;

    const task = {
      id: makeTaskId(),
      status: 'pending',
      toolName: payload.toolName,
      toolCallId: payload.toolCallId,
      args: payload.args,
      agentId: payload.agentId,
      threadId: payload.threadId,
      resourceId: payload.resourceId,
      runId: payload.runId,
      retryCount: 0,
      maxRetries: payload.maxRetries ?? this.config.defaultRetries?.maxRetries ?? 0,
      timeoutMs: payload.timeoutMs ?? this.config.defaultTimeoutMs,
      createdAt: new Date(),
    };

    this.registerTaskContext(task.id, context);
    const storage = await this.getStorage();
    await storage.createTask(task);

    const canRun = typeof this.checkConcurrency === 'function' ? await this.checkConcurrency(task.agentId) : true;
    if (!canRun) {
      switch (this.config.backpressure) {
        case 'reject':
          this.deregisterTaskContext(task.id);
          await storage.deleteTask(task.id);
          throw new Error(`Concurrency limit reached, cannot enqueue task for tool "${task.toolName}"`);
        case 'fallback-sync':
          this.deregisterTaskContext(task.id);
          await storage.deleteTask(task.id);
          return { task, fallbackToSync: true };
        case 'queue':
        default:
          return { task };
      }
    }

    void runInlineTask(this, task.id).catch((error) => {
      console.error?.('[brainkit] inline background task failed', error);
    });
    await waitForInlineTaskStarted(this, task.id);
    return { task };
  };

  proto.resume = async function resumeInline(taskId, resumeData) {
    if (globalThis.__brainkit_disable_inline_background_tasks) {
      return originalResume.call(this, taskId, resumeData);
    }
    if (this.initPromise) await this.initPromise;
    const storage = await this.getStorage();
    const task = await storage.getTask(taskId);
    if (!task) throw new Error(`Task not found: ${taskId}`);
    if (task.status !== 'suspended') {
      throw new Error(`Cannot resume task in status '${task.status}' (expected 'suspended')`);
    }
    void runInlineTask(this, taskId, resumeData).catch((error) => {
      console.error?.('[brainkit] inline background task resume failed', error);
    });
    return task;
  };
}

installBrainkitBackgroundTaskInlineExecution();

function brainkitBackgroundTaskLifecycleState() {
  if (!globalThis.__brainkit_background_task_lifecycle_state) {
    Object.defineProperty(globalThis, '__brainkit_background_task_lifecycle_state', {
      value: {
        managers: new Set(),
        mastraInstances: new Set(),
        nextMastraId: 1,
        openedStreams: 0,
        closedStreams: 0,
        shutdownCalls: 0,
        workerStartCalls: 0,
        workerStopCalls: 0,
        workflowEventSubscriptions: new Set(),
        workflowEventsReceived: 0,
        workflowEventsOk: 0,
        workflowEventsRetry: 0,
        workflowEventsFailed: 0,
      },
      configurable: true,
    });
  }
  return globalThis.__brainkit_background_task_lifecycle_state;
}

function rememberBrainkitBackgroundTaskManager(manager) {
  if (!manager) return;
  const state = brainkitBackgroundTaskLifecycleState();
  state.managers.add(manager);
  if (!manager.__brainkitTrackedStreamAbortControllers) {
    Object.defineProperty(manager, '__brainkitTrackedStreamAbortControllers', {
      value: new Set(),
      configurable: true,
    });
  }
}

function rememberBrainkitMastraWorkerInstance(mastra) {
  if (!mastra) return;
  const state = brainkitBackgroundTaskLifecycleState();
  state.mastraInstances.add(mastra);
  if (!mastra.__brainkitWorkerInstanceId) {
    Object.defineProperty(mastra, '__brainkitWorkerInstanceId', {
      value: `mastra-${state.nextMastraId++}`,
      configurable: true,
    });
  }
}

function installBrainkitBackgroundTaskLifecycleTracking() {
  const proto = BackgroundTaskManager?.prototype;
  if (!proto || proto.__brainkitLifecycleTrackingInstalled) return;
  Object.defineProperty(proto, '__brainkitLifecycleTrackingInstalled', { value: true });

  const originalStream = proto.stream;
  const originalShutdown = proto.shutdown;

  proto.stream = function streamBrainkitTracked(options = {}) {
    rememberBrainkitBackgroundTaskManager(this);
    const state = brainkitBackgroundTaskLifecycleState();
    const tracked = this.__brainkitTrackedStreamAbortControllers;
    const upstreamSignal = options?.abortSignal;
    const internalAbort = new AbortController();
    const streamOptions = { ...options, abortSignal: internalAbort.signal };

    let closed = false;
    const markClosed = () => {
      if (closed) return;
      closed = true;
      tracked.delete(internalAbort);
      if (state.closedStreams < state.openedStreams) state.closedStreams++;
    };

    tracked.add(internalAbort);
    state.openedStreams++;
    internalAbort.signal.addEventListener('abort', markClosed, { once: true });

    if (upstreamSignal) {
      if (upstreamSignal.aborted) {
        internalAbort.abort(upstreamSignal.reason);
      } else {
        upstreamSignal.addEventListener(
          'abort',
          () => internalAbort.abort(upstreamSignal.reason),
          { once: true },
        );
      }
    }

    const inner = originalStream.call(this, streamOptions);
    let reader;

    return new ReadableStream({
      async start(controller) {
        reader = inner.getReader();
        try {
          while (true) {
            const next = await reader.read();
            if (next.done) {
              markClosed();
              try {
                controller.close();
              } catch {
                // Already closed by cancellation.
              }
              return;
            }
            controller.enqueue(next.value);
          }
        } catch (error) {
          markClosed();
          try {
            controller.error(error);
          } catch {
            // Already closed by cancellation.
          }
        }
      },
      async cancel(reason) {
        internalAbort.abort(reason);
        markClosed();
        if (reader && typeof reader.cancel === 'function') {
          await reader.cancel(reason);
        } else if (typeof inner.cancel === 'function') {
          await inner.cancel(reason);
        }
      },
    });
  };

  proto.shutdown = async function shutdownBrainkitTracked() {
    rememberBrainkitBackgroundTaskManager(this);
    const state = brainkitBackgroundTaskLifecycleState();
    state.shutdownCalls++;
    const tracked = Array.from(this.__brainkitTrackedStreamAbortControllers ?? []);
    for (const controller of tracked) {
      if (!controller.signal.aborted) {
        controller.abort(new Error('BackgroundTaskManager shutdown'));
      }
    }
    return originalShutdown.call(this);
  };
}

installBrainkitBackgroundTaskLifecycleTracking();

function installBrainkitMastraWorkerBridge() {
  const proto = Mastra?.prototype;
  if (!proto || proto.__brainkitWorkerBridgeInstalled) return;
  Object.defineProperty(proto, '__brainkitWorkerBridgeInstalled', { value: true });

  const originalStartWorkers = proto.startWorkers;
  const originalStopWorkers = proto.stopWorkers;

  const getWorkers = (mastra) => {
    try {
      return Array.from(mastra.workers ?? []);
    } catch {
      return [];
    }
  };

  const patchOrchestrationWorker = (worker) => {
    if (!worker || worker.name !== 'orchestration' || worker.__brainkitPushWorkflowPatched) return;
    Object.defineProperty(worker, '__brainkitPushWorkflowPatched', { value: true });
    Object.defineProperty(worker, '__brainkitOriginalStart', { value: worker.start });

    // QuickJS runs Brainkit deployments in a single cooperative JS isolate.
    // Mastra's pull OrchestrationWorker is a server/worker-process concern; in
    // this runtime workflow events are safer as a push listener owned by the
    // Mastra instance below.
    worker.start = async function startBrainkitOrchestrationNoop() {};
  };

  const ensureWorkflowEventPushSubscription = async (mastra) => {
    rememberBrainkitMastraWorkerInstance(mastra);
    if (!mastra || mastra.__brainkitWorkflowEventSubscription) return;
    if (typeof mastra.addTopicListener !== 'function' || typeof mastra.handleWorkflowEvent !== 'function') {
      throw new Error('Brainkit workflow event bridge requires Mastra.addTopicListener and Mastra.handleWorkflowEvent');
    }
    const state = brainkitBackgroundTaskLifecycleState();
    const cb = (event, ack) => {
      state.workflowEventsReceived++;
      void mastra.handleWorkflowEvent(event)
        .then((result) => {
          if (result?.ok) {
            state.workflowEventsOk++;
            if (ack) {
              return ack().catch((err) =>
                console.error?.('[brainkit] error acking workflow event', err),
              );
            }
            return undefined;
          }
          state.workflowEventsRetry++;
          return undefined;
        })
        .catch((err) => {
          state.workflowEventsFailed++;
          console.error?.('[brainkit] unhandled workflow event error', err);
        });
    };
    Object.defineProperty(mastra, '__brainkitWorkflowEventSubscription', {
      value: { topic: 'workflows', cb },
      configurable: true,
      writable: true,
    });
    await mastra.addTopicListener('workflows', cb);
    state.workflowEventSubscriptions.add(mastra);
  };

  const removeWorkflowEventPushSubscription = async (mastra) => {
    if (!mastra?.__brainkitWorkflowEventSubscription) return;
    const state = brainkitBackgroundTaskLifecycleState();
    const sub = mastra.__brainkitWorkflowEventSubscription;
    mastra.__brainkitWorkflowEventSubscription = undefined;
    state.workflowEventSubscriptions.delete(mastra);
    if (typeof mastra.removeTopicListener === 'function') {
      await mastra.removeTopicListener(sub.topic, sub.cb);
    }
  };

  proto.startWorkers = async function startWorkersBrainkit(name) {
    rememberBrainkitMastraWorkerInstance(this);
    const state = brainkitBackgroundTaskLifecycleState();
    state.workerStartCalls++;
    const workers = getWorkers(this);

    if (!name || name === 'orchestration') {
      for (const worker of workers) patchOrchestrationWorker(worker);
      await ensureWorkflowEventPushSubscription(this);
    }

    if (!name) {
      for (const worker of workers) {
        if (!worker || worker.name === 'orchestration' || worker.name === 'scheduler') continue;
        await originalStartWorkers.call(this, worker.name);
      }
      return;
    }

    return originalStartWorkers.call(this, name);
  };

  proto.stopWorkers = async function stopWorkersBrainkit() {
    rememberBrainkitMastraWorkerInstance(this);
    const state = brainkitBackgroundTaskLifecycleState();
    state.workerStopCalls++;
    if (this.backgroundTaskManager) {
      rememberBrainkitBackgroundTaskManager(this.backgroundTaskManager);
    }
    let err;
    try {
      await removeWorkflowEventPushSubscription(this);
    } catch (removeErr) {
      err = removeErr;
    }
    try {
      await originalStopWorkers.call(this);
    } catch (stopErr) {
      err = err ? new AggregateError([err, stopErr], 'Brainkit stopWorkers failed') : stopErr;
    }
    if (err) throw err;
  };
}

installBrainkitMastraWorkerBridge();

globalThis.__brainkit_mastra_background_task_debug = function brainkitMastraBackgroundTaskDebug() {
  const state = brainkitBackgroundTaskLifecycleState();
  let activeTrackedStreams = 0;
  let activeTaskContexts = 0;
  let activeAbortControllers = 0;
  let shuttingDownManagers = 0;
  for (const manager of state.managers) {
    activeTrackedStreams += manager.__brainkitTrackedStreamAbortControllers?.size ?? 0;
    activeTaskContexts += manager.taskContexts?.size ?? 0;
    activeAbortControllers += manager.activeAbortControllers?.size ?? 0;
    if (manager.shuttingDown) shuttingDownManagers++;
  }

  const activeWorkerNames = [];
  let activeSchedulerCount = 0;
  for (const mastra of state.mastraInstances) {
    const instanceID = mastra.__brainkitWorkerInstanceId ?? 'mastra';
    if (mastra.scheduler?.isRunning) activeSchedulerCount++;
    for (const worker of Array.from(mastra.workers ?? [])) {
      if (worker?.isRunning) activeWorkerNames.push(`${instanceID}:${worker.name}`);
    }
  }
  activeWorkerNames.sort();

  return {
    openedStreams: state.openedStreams,
    closedStreams: state.closedStreams,
    activeStreams: Math.max(0, state.openedStreams - state.closedStreams),
    activeTrackedStreams,
    activeTaskContexts,
    activeAbortControllers,
    shuttingDownManagers,
    shutdownCalls: state.shutdownCalls,
    workerStartCalls: state.workerStartCalls,
    workerStopCalls: state.workerStopCalls,
    activeWorkerCount: activeWorkerNames.length,
    activeWorkerNames,
    activeWorkflowEventSubscriptions: state.workflowEventSubscriptions.size,
    workflowEventsReceived: state.workflowEventsReceived,
    workflowEventsOk: state.workflowEventsOk,
    workflowEventsRetry: state.workflowEventsRetry,
    workflowEventsFailed: state.workflowEventsFailed,
    activeSchedulerCount,
  };
};

// Expose on globalThis for QuickJS access
globalThis.__agent_embed = {
  // Mastra core
  Agent,
  TripWire,
  MessageList,
  convertMessages,
  TypeDetector,
  createTool,
  createWorkflow,
  createStep,
  Workflow,
  cloneWorkflow,
  cloneStep,
  mapVariable,
  Mastra,
  BackgroundTaskManager,
  createBackgroundTask,
  generateBackgroundTaskSystemPrompt,
  __brainkitMastraBackgroundTaskDebug: globalThis.__brainkit_mastra_background_task_debug,
  ConsoleLogger,
  MultiLogger,
  DualLogger,
  Memory,
  MockMemory,
  InMemoryStore,
  LibSQLStore,
  LibSQLVector,
  UpstashStore,
  PostgresStore,
  PgVector,
  MongoDBStore,
  MongoDBVector,
  PineconeVector,
  ChromaVector,
  QdrantVector,
  z,
  embed,
  embedMany,
  generateText,
  streamText,
  generateObject,
  streamObject,
  ModelRouterEmbeddingModel,
  RequestContext,

  // Evals
  createScorer,
  runEvals,
  MastraScorer,
  registerHook,
  executeHook,
  AvailableHooks,
  createCompletenessScorer,
  createTextualDifferenceScorer,
  createKeywordCoverageScorer,
  createContentSimilarityScorer,
  createToneScorer,
  createToolCallAccuracyScorerCode,
  createTrajectoryAccuracyScorerCode,
  createTrajectoryScorerCode,
  createHallucinationScorer,
  createFaithfulnessScorer,
  createAnswerRelevancyScorer,
  createAnswerSimilarityScorer,
  createBiasScorer,
  createToxicityScorer,
  createContextPrecisionScorer,
  createContextRelevanceScorerLLM,
  createNoiseSensitivityScorerLLM,
  createPromptAlignmentScorerLLM,
  createToolCallAccuracyScorerLLM,
  createTrajectoryAccuracyScorerLLM,

  // Processors
  ModerationProcessor,
  PromptInjectionDetector,
  PIIDetector,
  SystemPromptScrubber,
  UnicodeNormalizer,
  LanguageDetector,
  TokenLimiterProcessor,
  BatchPartsProcessor,
  StructuredOutputProcessor,
  ToolCallFilter,
  ToolSearchProcessor,
  AgentsMDInjector,
  SkillsProcessor,
  SkillSearchProcessor,
  WorkspaceInstructionsProcessor,
  ResponseCache,
  DEFAULT_RESPONSE_CACHE_TTL_SECONDS,
  RESPONSE_CACHE_CONTEXT_KEY,
  buildResponseCacheKey,
  InMemoryServerCache,
  MastraServerCache,

  // RAG
  MDocument,
  GraphRAG,
  createVectorQueryTool,
  createDocumentChunkerTool,
  createGraphRAGTool,
  rerank,
  rerankWithScorer,
  CohereRelevanceScorer,
  MastraAgentRelevanceScorer,
  ZeroEntropyRelevanceScorer,

  // Observability
  Observability,
  DefaultExporter,
  SensitiveDataFilter,
  BaseExporter,
  CloudExporter,
  ConsoleExporter,
  TestExporter,
  TrackingExporter,
  chainFormatters,
  // OpenTelemetry span processors + exporters
  AlwaysOnSampler,
  AlwaysOffSampler,
  ParentBasedSampler,
  TraceIdRatioBasedSampler,
  NoopSpanProcessor,
  ConsoleSpanExporter,
  InMemorySpanExporter,
  BasicTracerProvider,
  SimpleSpanProcessor,
  BatchSpanProcessor,

  // Workspace
  Workspace,
  LocalFilesystem,
  LocalSandbox,
  CompositeFilesystem,
  WORKSPACE_TOOLS_PREFIX,
  WORKSPACE_TOOLS,
  createWorkspaceTools,
  resolveToolConfig,
  readFileTool,
  writeFileTool,
  editFileTool,
  listFilesTool,
  deleteFileTool,
  fileStatTool,
  mkdirTool,
  searchTool,
  indexContentTool,
  executeCommandTool,
  requireWorkspace,
  requireFilesystem,
  requireSandbox,

  // Browser
  MastraBrowser,
  BrowserContextProcessor,

  // Channels
  AgentChannels,
  ChatChannelProcessor,
  MastraStateAdapter,

  // Voice
  MastraVoice,
  CompositeVoice,
  DefaultVoice,
  AISDKSpeech,
  AISDKTranscription,
  OpenAIVoice,
  OpenAIRealtimeVoice,
  AzureVoice,
  ElevenLabsVoice,
  CloudflareVoice,
  DeepgramVoice,
  PlayAIVoice,
  PLAYAI_VOICES,
  SpeechifyVoice,
  SarvamVoice,
  MurfVoice,

  // Harness
  Harness,
  assignTaskIds,
  askUserTool,
  defaultDisplayState,
  defaultOMProgressState,
  parseSubagentMeta,
  submitPlanTool,
  taskWriteTool,
  taskUpdateTool,
  taskCompleteTool,
  taskCheckTool,

  // Ecosystem compatibility canaries
  ecosystemCanaries,

  // AI SDK providers
  createOpenAI,
  createAnthropic,
  createGoogleGenerativeAI,
  createMistral,
  createXai,
  createGroq,
  createDeepSeek,
  createCerebras,
  createPerplexity,
  createTogetherAI,
  createFireworks,
  createCohere,

  // AI SDK — tool authoring
  tool,
  dynamicTool,
  jsonSchema,
  zodSchema,
  asSchema,
  generateId,
  createIdGenerator,
  hasToolCall,
  stepCountIs,
  isLoopFinished,

  // AI SDK — middleware
  wrapLanguageModel,
  wrapEmbeddingModel,
  wrapImageModel,
  wrapProvider,
  extractReasoningMiddleware,
  extractJsonMiddleware,
  defaultSettingsMiddleware,
  defaultEmbeddingSettingsMiddleware,
  simulateStreamingMiddleware,
  smoothStream,
  addToolInputExamplesMiddleware,

  // AI SDK — provider registry
  createProviderRegistry,
  customProvider,
  experimental_createProviderRegistry,
  experimental_customProvider,

  // AI SDK — message utilities
  convertToModelMessages,
  pruneMessages,
  validateUIMessages,
  safeValidateUIMessages,
  readUIMessageStream,
  consumeStream,
  convertFileListToFileUIParts,

  // AI SDK — media
  generateImage,
  experimental_generateImage,
  experimental_generateVideo,
  experimental_transcribe,
  experimental_generateSpeech,

  // AI SDK — misc
  cosineSimilarity,
  simulateReadableStream,
  parsePartialJson,
  parseJsonEventStream,

  // AI SDK — gateway
  gateway,
  createGateway,

  // AI SDK — error classes (needed for instanceof in catch blocks)
  AISDKError,
  APICallError,
  NoObjectGeneratedError,
  NoSuchModelError,
  NoSuchToolError,
  InvalidArgumentError,
  InvalidDataContentError,
  InvalidPromptError,
  InvalidToolInputError,
  NoContentGeneratedError,
  NoSpeechGeneratedError,
  NoTranscriptGeneratedError,
  NoVideoGeneratedError,
  RetryError,
  ToolCallRepairError,
  TypeValidationError,
  MessageConversionError,
  MissingToolResultsError,
  LoadAPIKeyError,
  InvalidToolApprovalError,
  ToolCallNotFoundForApprovalError,
};
