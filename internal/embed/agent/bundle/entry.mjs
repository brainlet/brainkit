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
} from '@mastra/core/processors';

// RAG — document chunking, vector query tools, graph RAG, reranking
import { MDocument, GraphRAG } from '@mastra/rag';
import { createVectorQueryTool, createDocumentChunkerTool, createGraphRAGTool } from '@mastra/rag';
import { rerank, rerankWithScorer } from '@mastra/rag';

// Observability — tracing, spans, exporters
import { Observability, DefaultExporter, SensitiveDataFilter } from '@mastra/observability';

// Workspace — filesystem, sandbox, skills, search
import { Workspace, LocalFilesystem, LocalSandbox } from '@mastra/core/workspace';

// Voice — STT/TTS. CompositeVoice lives in @mastra/core, the
// OpenAI provider lives in its own package.
import { CompositeVoice } from '@mastra/core/voice';
import { OpenAIVoice } from '@mastra/voice-openai';
import { OpenAIRealtimeVoice } from '@mastra/voice-openai-realtime';

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

  // Workspace
  Workspace,
  LocalFilesystem,
  LocalSandbox,
  CompositeVoice,
  OpenAIVoice,
  OpenAIRealtimeVoice,

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
