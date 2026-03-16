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

// RAG — document chunking, vector query tools, graph RAG, reranking
import { MDocument, GraphRAG } from '@mastra/rag';
import { createVectorQueryTool, createDocumentChunkerTool, createGraphRAGTool } from '@mastra/rag';
import { rerank, rerankWithScorer } from '@mastra/rag';

// Observability — tracing, spans, exporters
import { Observability, DefaultExporter, SensitiveDataFilter } from '@mastra/observability';

// Workspace — filesystem, sandbox, skills, search
import { Workspace, LocalFilesystem, LocalSandbox } from '@mastra/core/workspace';

// tiktoken: the tiktoken-unify esbuild plugin redirects 'js-tiktoken/lite'
// and 'js-tiktoken/ranks/*' to full 'js-tiktoken'. getTiktoken() in
// @mastra/core/utils/tiktoken.ts now resolves to the already-bundled module.

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
