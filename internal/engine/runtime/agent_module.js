// "agent" module — direct re-exports of Mastra framework from __agent_embed.
// Does NOT re-export AI SDK functions (those come from "ai").

// Core
export const Agent = globalThis.__agent_embed.Agent;
export const createTool = globalThis.__agent_embed.createTool;
export const createWorkflow = globalThis.__agent_embed.createWorkflow;
export const createStep = globalThis.__agent_embed.createStep;
export const Memory = globalThis.__agent_embed.Memory;
export const RequestContext = globalThis.__agent_embed.RequestContext;
export const z = globalThis.__agent_embed.z;

// Storage backends
export const InMemoryStore = globalThis.__agent_embed.InMemoryStore;
export const LibSQLStore = globalThis.__agent_embed.LibSQLStore;
export const UpstashStore = globalThis.__agent_embed.UpstashStore;
export const PostgresStore = globalThis.__agent_embed.PostgresStore;
export const MongoDBStore = globalThis.__agent_embed.MongoDBStore;

// Vector backends
export const LibSQLVector = globalThis.__agent_embed.LibSQLVector;
export const PgVector = globalThis.__agent_embed.PgVector;
export const MongoDBVector = globalThis.__agent_embed.MongoDBVector;
export const ModelRouterEmbeddingModel = globalThis.__agent_embed.ModelRouterEmbeddingModel;

// Workspace
export const Workspace = globalThis.__agent_embed.Workspace;
export const LocalFilesystem = globalThis.__agent_embed.LocalFilesystem;
export const LocalSandbox = globalThis.__agent_embed.LocalSandbox;

// RAG
export const MDocument = globalThis.__agent_embed.MDocument;
export const GraphRAG = globalThis.__agent_embed.GraphRAG;
export const createVectorQueryTool = globalThis.__agent_embed.createVectorQueryTool;
export const createDocumentChunkerTool = globalThis.__agent_embed.createDocumentChunkerTool;
export const createGraphRAGTool = globalThis.__agent_embed.createGraphRAGTool;
export const rerank = globalThis.__agent_embed.rerank;
export const rerankWithScorer = globalThis.__agent_embed.rerankWithScorer;

// Observability
export const Observability = globalThis.__agent_embed.Observability;
export const DefaultExporter = globalThis.__agent_embed.DefaultExporter;
export const SensitiveDataFilter = globalThis.__agent_embed.SensitiveDataFilter;

// Evals
export const createScorer = globalThis.__agent_embed.createScorer;
export const runEvals = globalThis.__agent_embed.runEvals;
