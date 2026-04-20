// "ai" module — direct re-exports of AI SDK from __agent_embed.

// ── Core generation ─────────────────────────────────────────────
export const generateText = globalThis.__agent_embed.generateText;
export const streamText = globalThis.__agent_embed.streamText;
export const generateObject = globalThis.__agent_embed.generateObject;
export const streamObject = globalThis.__agent_embed.streamObject;
export const embed = globalThis.__agent_embed.embed;
export const embedMany = globalThis.__agent_embed.embedMany;
export const z = globalThis.__agent_embed.z;

// ── Tool authoring ─────────────────────────────────────────────
export const tool = globalThis.__agent_embed.tool;
export const dynamicTool = globalThis.__agent_embed.dynamicTool;
export const jsonSchema = globalThis.__agent_embed.jsonSchema;
export const zodSchema = globalThis.__agent_embed.zodSchema;
export const asSchema = globalThis.__agent_embed.asSchema;
export const generateId = globalThis.__agent_embed.generateId;
export const createIdGenerator = globalThis.__agent_embed.createIdGenerator;
export const hasToolCall = globalThis.__agent_embed.hasToolCall;
export const stepCountIs = globalThis.__agent_embed.stepCountIs;
export const isLoopFinished = globalThis.__agent_embed.isLoopFinished;

// ── Middleware ─────────────────────────────────────────────────
export const wrapLanguageModel = globalThis.__agent_embed.wrapLanguageModel;
export const wrapEmbeddingModel = globalThis.__agent_embed.wrapEmbeddingModel;
export const wrapImageModel = globalThis.__agent_embed.wrapImageModel;
export const wrapProvider = globalThis.__agent_embed.wrapProvider;
export const extractReasoningMiddleware = globalThis.__agent_embed.extractReasoningMiddleware;
export const extractJsonMiddleware = globalThis.__agent_embed.extractJsonMiddleware;
export const defaultSettingsMiddleware = globalThis.__agent_embed.defaultSettingsMiddleware;
export const defaultEmbeddingSettingsMiddleware = globalThis.__agent_embed.defaultEmbeddingSettingsMiddleware;
export const simulateStreamingMiddleware = globalThis.__agent_embed.simulateStreamingMiddleware;
export const smoothStream = globalThis.__agent_embed.smoothStream;
export const addToolInputExamplesMiddleware = globalThis.__agent_embed.addToolInputExamplesMiddleware;

// ── Provider registry ──────────────────────────────────────────
export const createProviderRegistry = globalThis.__agent_embed.createProviderRegistry;
export const customProvider = globalThis.__agent_embed.customProvider;
export const experimental_createProviderRegistry = globalThis.__agent_embed.experimental_createProviderRegistry;
export const experimental_customProvider = globalThis.__agent_embed.experimental_customProvider;

// ── Message utilities ──────────────────────────────────────────
export const convertToModelMessages = globalThis.__agent_embed.convertToModelMessages;
export const pruneMessages = globalThis.__agent_embed.pruneMessages;
export const validateUIMessages = globalThis.__agent_embed.validateUIMessages;
export const safeValidateUIMessages = globalThis.__agent_embed.safeValidateUIMessages;
export const readUIMessageStream = globalThis.__agent_embed.readUIMessageStream;
export const consumeStream = globalThis.__agent_embed.consumeStream;
export const convertFileListToFileUIParts = globalThis.__agent_embed.convertFileListToFileUIParts;

// ── Media ──────────────────────────────────────────────────────
export const generateImage = globalThis.__agent_embed.generateImage;
export const experimental_generateImage = globalThis.__agent_embed.experimental_generateImage;
export const experimental_generateVideo = globalThis.__agent_embed.experimental_generateVideo;
export const experimental_transcribe = globalThis.__agent_embed.experimental_transcribe;
export const experimental_generateSpeech = globalThis.__agent_embed.experimental_generateSpeech;

// ── Misc ───────────────────────────────────────────────────────
export const cosineSimilarity = globalThis.__agent_embed.cosineSimilarity;
export const simulateReadableStream = globalThis.__agent_embed.simulateReadableStream;
export const parsePartialJson = globalThis.__agent_embed.parsePartialJson;
export const parseJsonEventStream = globalThis.__agent_embed.parseJsonEventStream;

// ── Gateway ────────────────────────────────────────────────────
export const gateway = globalThis.__agent_embed.gateway;
export const createGateway = globalThis.__agent_embed.createGateway;

// ── Error classes (for `instanceof` in catch blocks) ───────────
export const AISDKError = globalThis.__agent_embed.AISDKError;
export const APICallError = globalThis.__agent_embed.APICallError;
export const NoObjectGeneratedError = globalThis.__agent_embed.NoObjectGeneratedError;
export const NoSuchModelError = globalThis.__agent_embed.NoSuchModelError;
export const NoSuchToolError = globalThis.__agent_embed.NoSuchToolError;
export const InvalidArgumentError = globalThis.__agent_embed.InvalidArgumentError;
export const InvalidDataContentError = globalThis.__agent_embed.InvalidDataContentError;
export const InvalidPromptError = globalThis.__agent_embed.InvalidPromptError;
export const InvalidToolInputError = globalThis.__agent_embed.InvalidToolInputError;
export const NoContentGeneratedError = globalThis.__agent_embed.NoContentGeneratedError;
export const NoSpeechGeneratedError = globalThis.__agent_embed.NoSpeechGeneratedError;
export const NoTranscriptGeneratedError = globalThis.__agent_embed.NoTranscriptGeneratedError;
export const NoVideoGeneratedError = globalThis.__agent_embed.NoVideoGeneratedError;
export const RetryError = globalThis.__agent_embed.RetryError;
export const ToolCallRepairError = globalThis.__agent_embed.ToolCallRepairError;
export const TypeValidationError = globalThis.__agent_embed.TypeValidationError;
export const MessageConversionError = globalThis.__agent_embed.MessageConversionError;
export const MissingToolResultsError = globalThis.__agent_embed.MissingToolResultsError;
export const LoadAPIKeyError = globalThis.__agent_embed.LoadAPIKeyError;
export const InvalidToolApprovalError = globalThis.__agent_embed.InvalidToolApprovalError;
export const ToolCallNotFoundForApprovalError = globalThis.__agent_embed.ToolCallNotFoundForApprovalError;
