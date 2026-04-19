// agent-embed entry point
// Mastra framework + AI SDK providers for QuickJS embedding

import { Agent } from '@mastra/core/agent';
import { createTool } from '@mastra/core/tools';
import { createWorkflow, createStep } from '@mastra/core/workflows';
import { Mastra } from '@mastra/core/mastra';
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
import { embed, embedMany, generateText, streamText, generateObject, streamObject } from 'ai';
import { ModelRouterEmbeddingModel } from '@mastra/core/llm';
import { RequestContext } from '@mastra/core/request-context';

// Evals — scorer infrastructure from core + pre-built rule-based scorers
import { createScorer, runEvals } from '@mastra/core/evals';
import {
  // Rule-based (no LLM)
  createCompletenessScorer,
  createTextualDifferenceScorer,
  createKeywordCoverageScorer,
  createContentSimilarityScorer,
  createToneScorer,
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
} from '@mastra/core/processors';

// RAG — document chunking, vector query tools, graph RAG, reranking
import { MDocument, GraphRAG } from '@mastra/rag';
import { createVectorQueryTool, createDocumentChunkerTool, createGraphRAGTool } from '@mastra/rag';
import { rerank, rerankWithScorer } from '@mastra/rag';
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
import { Observability, DefaultExporter, SensitiveDataFilter } from '@mastra/observability';
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
import { Workspace, LocalFilesystem, LocalSandbox } from '@mastra/core/workspace';

// Voice — STT/TTS. CompositeVoice lives in @mastra/core, the
// OpenAI provider lives in its own package.
// MastraVoice is the abstract base every provider extends.
// Exposing it lets .ts code type-check custom voice provider
// subclasses without leaving the "agent" module.
import { CompositeVoice, MastraVoice } from '@mastra/core/voice';
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
import { askUserTool, submitPlanTool, taskWriteTool, taskCheckTool } from '@mastra/core/harness';

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

// Expose on globalThis for QuickJS access
globalThis.__agent_embed = {
  // Mastra core
  Agent,
  createTool,
  createWorkflow,
  createStep,
  Mastra,
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
  createCompletenessScorer,
  createTextualDifferenceScorer,
  createKeywordCoverageScorer,
  createContentSimilarityScorer,
  createToneScorer,
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

  // RAG
  MDocument,
  GraphRAG,
  createVectorQueryTool,
  createDocumentChunkerTool,
  createGraphRAGTool,
  rerank,
  rerankWithScorer,

  // Observability
  Observability,
  DefaultExporter,
  SensitiveDataFilter,
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
  MastraVoice,
  CompositeVoice,
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
  askUserTool,
  submitPlanTool,
  taskWriteTool,
  taskCheckTool,

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
};
