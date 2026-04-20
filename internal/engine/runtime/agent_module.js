// "agent" module — direct re-exports of Mastra framework from __agent_embed.
// Does NOT re-export AI SDK functions (those come from "ai").

// Core
export const Agent = globalThis.__agent_embed.Agent;
export const Mastra = globalThis.__agent_embed.Mastra;
export const TripWire = globalThis.__agent_embed.TripWire;
export const MessageList = globalThis.__agent_embed.MessageList;
export const convertMessages = globalThis.__agent_embed.convertMessages;
export const TypeDetector = globalThis.__agent_embed.TypeDetector;
export const createTool = globalThis.__agent_embed.createTool;
export const createWorkflow = globalThis.__agent_embed.createWorkflow;
export const createStep = globalThis.__agent_embed.createStep;
export const Workflow = globalThis.__agent_embed.Workflow;
export const cloneWorkflow = globalThis.__agent_embed.cloneWorkflow;
export const cloneStep = globalThis.__agent_embed.cloneStep;
export const mapVariable = globalThis.__agent_embed.mapVariable;
export const Memory = globalThis.__agent_embed.Memory;
export const RequestContext = globalThis.__agent_embed.RequestContext;
export const z = globalThis.__agent_embed.z;

// Logger — required by Mastra config; previously no way to construct one.
export const ConsoleLogger = globalThis.__agent_embed.ConsoleLogger;
export const MultiLogger = globalThis.__agent_embed.MultiLogger;
export const DualLogger = globalThis.__agent_embed.DualLogger;

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
export const CompositeFilesystem = globalThis.__agent_embed.CompositeFilesystem;
export const createWorkspaceTools = globalThis.__agent_embed.createWorkspaceTools;
export const readFileTool = globalThis.__agent_embed.readFileTool;
export const writeFileTool = globalThis.__agent_embed.writeFileTool;
export const editFileTool = globalThis.__agent_embed.editFileTool;
export const listFilesTool = globalThis.__agent_embed.listFilesTool;
export const deleteFileTool = globalThis.__agent_embed.deleteFileTool;
export const fileStatTool = globalThis.__agent_embed.fileStatTool;
export const mkdirTool = globalThis.__agent_embed.mkdirTool;
export const searchTool = globalThis.__agent_embed.searchTool;
export const indexContentTool = globalThis.__agent_embed.indexContentTool;
export const executeCommandTool = globalThis.__agent_embed.executeCommandTool;

// RAG
export const MDocument = globalThis.__agent_embed.MDocument;
export const GraphRAG = globalThis.__agent_embed.GraphRAG;
export const createVectorQueryTool = globalThis.__agent_embed.createVectorQueryTool;
export const createDocumentChunkerTool = globalThis.__agent_embed.createDocumentChunkerTool;
export const createGraphRAGTool = globalThis.__agent_embed.createGraphRAGTool;
export const rerank = globalThis.__agent_embed.rerank;
export const rerankWithScorer = globalThis.__agent_embed.rerankWithScorer;
export const CohereRelevanceScorer = globalThis.__agent_embed.CohereRelevanceScorer;
export const MastraAgentRelevanceScorer = globalThis.__agent_embed.MastraAgentRelevanceScorer;
export const ZeroEntropyRelevanceScorer = globalThis.__agent_embed.ZeroEntropyRelevanceScorer;

// Observability
export const Observability = globalThis.__agent_embed.Observability;
export const DefaultExporter = globalThis.__agent_embed.DefaultExporter;
export const SensitiveDataFilter = globalThis.__agent_embed.SensitiveDataFilter;
export const BaseExporter = globalThis.__agent_embed.BaseExporter;
export const CloudExporter = globalThis.__agent_embed.CloudExporter;
export const ConsoleExporter = globalThis.__agent_embed.ConsoleExporter;
export const TestExporter = globalThis.__agent_embed.TestExporter;
export const TrackingExporter = globalThis.__agent_embed.TrackingExporter;
export const chainFormatters = globalThis.__agent_embed.chainFormatters;
// OpenTelemetry span processors + exporters
export const BatchSpanProcessor = globalThis.__agent_embed.BatchSpanProcessor;
export const SimpleSpanProcessor = globalThis.__agent_embed.SimpleSpanProcessor;
export const NoopSpanProcessor = globalThis.__agent_embed.NoopSpanProcessor;
export const ConsoleSpanExporter = globalThis.__agent_embed.ConsoleSpanExporter;
export const InMemorySpanExporter = globalThis.__agent_embed.InMemorySpanExporter;
export const BasicTracerProvider = globalThis.__agent_embed.BasicTracerProvider;
export const AlwaysOnSampler = globalThis.__agent_embed.AlwaysOnSampler;
export const AlwaysOffSampler = globalThis.__agent_embed.AlwaysOffSampler;
export const ParentBasedSampler = globalThis.__agent_embed.ParentBasedSampler;
export const TraceIdRatioBasedSampler = globalThis.__agent_embed.TraceIdRatioBasedSampler;

// Voice
export const MastraVoice = globalThis.__agent_embed.MastraVoice;
export const CompositeVoice = globalThis.__agent_embed.CompositeVoice;
export const DefaultVoice = globalThis.__agent_embed.DefaultVoice;
export const AISDKSpeech = globalThis.__agent_embed.AISDKSpeech;
export const AISDKTranscription = globalThis.__agent_embed.AISDKTranscription;
export const OpenAIVoice = globalThis.__agent_embed.OpenAIVoice;
export const OpenAIRealtimeVoice = globalThis.__agent_embed.OpenAIRealtimeVoice;
export const AzureVoice = globalThis.__agent_embed.AzureVoice;
export const ElevenLabsVoice = globalThis.__agent_embed.ElevenLabsVoice;
export const CloudflareVoice = globalThis.__agent_embed.CloudflareVoice;
export const DeepgramVoice = globalThis.__agent_embed.DeepgramVoice;
export const PlayAIVoice = globalThis.__agent_embed.PlayAIVoice;
export const PLAYAI_VOICES = globalThis.__agent_embed.PLAYAI_VOICES;
export const SpeechifyVoice = globalThis.__agent_embed.SpeechifyVoice;
export const SarvamVoice = globalThis.__agent_embed.SarvamVoice;
export const MurfVoice = globalThis.__agent_embed.MurfVoice;

// Evals
export const createScorer = globalThis.__agent_embed.createScorer;
export const runEvals = globalThis.__agent_embed.runEvals;
export const MastraScorer = globalThis.__agent_embed.MastraScorer;
export const registerHook = globalThis.__agent_embed.registerHook;
export const executeHook = globalThis.__agent_embed.executeHook;
export const AvailableHooks = globalThis.__agent_embed.AvailableHooks;

// Prebuilt scorer factories (@mastra/evals/scorers/prebuilt).
export const createCompletenessScorer = globalThis.__agent_embed.createCompletenessScorer;
export const createTextualDifferenceScorer = globalThis.__agent_embed.createTextualDifferenceScorer;
export const createKeywordCoverageScorer = globalThis.__agent_embed.createKeywordCoverageScorer;
export const createContentSimilarityScorer = globalThis.__agent_embed.createContentSimilarityScorer;
export const createToneScorer = globalThis.__agent_embed.createToneScorer;
export const createAnswerRelevancyScorer = globalThis.__agent_embed.createAnswerRelevancyScorer;
export const createAnswerSimilarityScorer = globalThis.__agent_embed.createAnswerSimilarityScorer;
export const createFaithfulnessScorer = globalThis.__agent_embed.createFaithfulnessScorer;
export const createHallucinationScorer = globalThis.__agent_embed.createHallucinationScorer;
export const createBiasScorer = globalThis.__agent_embed.createBiasScorer;
export const createToxicityScorer = globalThis.__agent_embed.createToxicityScorer;
export const createContextPrecisionScorer = globalThis.__agent_embed.createContextPrecisionScorer;
export const createContextRelevanceScorerLLM = globalThis.__agent_embed.createContextRelevanceScorerLLM;
export const createNoiseSensitivityScorerLLM = globalThis.__agent_embed.createNoiseSensitivityScorerLLM;
export const createPromptAlignmentScorerLLM = globalThis.__agent_embed.createPromptAlignmentScorerLLM;
export const createToolCallAccuracyScorerLLM = globalThis.__agent_embed.createToolCallAccuracyScorerLLM;
export const createTrajectoryAccuracyScorerLLM = globalThis.__agent_embed.createTrajectoryAccuracyScorerLLM;
export const createToolCallAccuracyScorerCode = globalThis.__agent_embed.createToolCallAccuracyScorerCode;
export const createTrajectoryAccuracyScorerCode = globalThis.__agent_embed.createTrajectoryAccuracyScorerCode;
export const createTrajectoryScorerCode = globalThis.__agent_embed.createTrajectoryScorerCode;

// Processors (@mastra/core/processors).
export const ModerationProcessor = globalThis.__agent_embed.ModerationProcessor;
export const PromptInjectionDetector = globalThis.__agent_embed.PromptInjectionDetector;
export const PIIDetector = globalThis.__agent_embed.PIIDetector;
export const SystemPromptScrubber = globalThis.__agent_embed.SystemPromptScrubber;
export const UnicodeNormalizer = globalThis.__agent_embed.UnicodeNormalizer;
export const LanguageDetector = globalThis.__agent_embed.LanguageDetector;
export const TokenLimiterProcessor = globalThis.__agent_embed.TokenLimiterProcessor;
export const BatchPartsProcessor = globalThis.__agent_embed.BatchPartsProcessor;
export const StructuredOutputProcessor = globalThis.__agent_embed.StructuredOutputProcessor;
export const ToolCallFilter = globalThis.__agent_embed.ToolCallFilter;
export const ToolSearchProcessor = globalThis.__agent_embed.ToolSearchProcessor;
export const AgentsMDInjector = globalThis.__agent_embed.AgentsMDInjector;
export const SkillsProcessor = globalThis.__agent_embed.SkillsProcessor;
export const SkillSearchProcessor = globalThis.__agent_embed.SkillSearchProcessor;
export const WorkspaceInstructionsProcessor = globalThis.__agent_embed.WorkspaceInstructionsProcessor;

// Observability (@mastra/observability) — already exported above, so keep this section for the new
// primitives that landed in the recent gap fill; the three original exports are above.
