// Gap 12 — surface audit. Every symbol below was added after the
// brainkit-maps/brainkit/plans-03 audit flagged it as missing from the
// deployed .ts runtime. This fixture proves each is reachable; it does
// not attempt to instantiate or call them (some need real providers /
// Mastra instances / etc).
import {
  // Mastra core additions
  Mastra,
  TripWire,
  MessageList,
  convertMessages,
  TypeDetector,
  Workflow,
  cloneWorkflow,
  cloneStep,
  mapVariable,
  ConsoleLogger,
  MultiLogger,
  DualLogger,
  // Evals additions
  MastraScorer,
  registerHook,
  executeHook,
  AvailableHooks,
  createTrajectoryAccuracyScorerLLM,
  createToolCallAccuracyScorerCode,
  createTrajectoryAccuracyScorerCode,
  createTrajectoryScorerCode,
  // RAG additions
  CohereRelevanceScorer,
  MastraAgentRelevanceScorer,
  ZeroEntropyRelevanceScorer,
  // Workspace additions
  CompositeFilesystem,
  createWorkspaceTools,
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
  // Voice additions
  DefaultVoice,
  AISDKSpeech,
  AISDKTranscription,
  // Observability additions
  BaseExporter,
  CloudExporter,
  ConsoleExporter,
  TestExporter,
  TrackingExporter,
  chainFormatters,
} from "agent";

import {
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
  simulateStreamingMiddleware,
  smoothStream,
  // AI SDK — provider registry
  createProviderRegistry,
  customProvider,
  // AI SDK — message utilities
  convertToModelMessages,
  pruneMessages,
  validateUIMessages,
  consumeStream,
  // AI SDK — media
  generateImage,
  experimental_transcribe,
  experimental_generateSpeech,
  // AI SDK — misc
  cosineSimilarity,
  simulateReadableStream,
  parsePartialJson,
  // AI SDK — gateway
  gateway,
  createGateway,
  // AI SDK — error classes
  AISDKError,
  APICallError,
  NoObjectGeneratedError,
  NoSuchModelError,
  NoSuchToolError,
  InvalidPromptError,
  InvalidToolInputError,
  RetryError,
  TypeValidationError,
  LoadAPIKeyError,
} from "ai";

import { output } from "kit";

function isFn(v: any) { return typeof v === "function"; }
function isDef(v: any) { return typeof v !== "undefined"; }

output({
  // Classes (typeof "function")
  mastra: isFn(Mastra),
  tripWire: isFn(TripWire),
  messageList: isFn(MessageList),
  workflow: isFn(Workflow),
  consoleLogger: isFn(ConsoleLogger),
  multiLogger: isFn(MultiLogger),
  dualLogger: isFn(DualLogger),
  mastraScorer: isFn(MastraScorer),
  cohereRerank: isFn(CohereRelevanceScorer),
  mastraAgentRerank: isFn(MastraAgentRelevanceScorer),
  zeroentropyRerank: isFn(ZeroEntropyRelevanceScorer),
  compositeFilesystem: isFn(CompositeFilesystem),
  defaultVoice: isFn(DefaultVoice),
  aisdkSpeech: isFn(AISDKSpeech),
  aisdkTranscription: isFn(AISDKTranscription),
  baseExporter: isFn(BaseExporter),
  cloudExporter: isFn(CloudExporter),
  consoleExporter: isFn(ConsoleExporter),
  testExporter: isFn(TestExporter),
  trackingExporter: isFn(TrackingExporter),

  // Functions
  convertMessages: isFn(convertMessages),
  typeDetector: isFn(TypeDetector),
  cloneWorkflow: isFn(cloneWorkflow),
  cloneStep: isFn(cloneStep),
  mapVariable: isFn(mapVariable),
  registerHook: isFn(registerHook),
  executeHook: isFn(executeHook),
  trajectoryLLM: isFn(createTrajectoryAccuracyScorerLLM),
  toolCallAccuracyCode: isFn(createToolCallAccuracyScorerCode),
  trajectoryAccuracyCode: isFn(createTrajectoryAccuracyScorerCode),
  trajectoryCode: isFn(createTrajectoryScorerCode),
  createWorkspaceTools: isFn(createWorkspaceTools),
  // Individual workspace tools are pre-built Tool objects, not factories.
  readFileTool: isDef(readFileTool),
  writeFileTool: isDef(writeFileTool),
  editFileTool: isDef(editFileTool),
  listFilesTool: isDef(listFilesTool),
  deleteFileTool: isDef(deleteFileTool),
  fileStatTool: isDef(fileStatTool),
  mkdirTool: isDef(mkdirTool),
  searchTool: isDef(searchTool),
  indexContentTool: isDef(indexContentTool),
  executeCommandTool: isDef(executeCommandTool),
  chainFormatters: isFn(chainFormatters),

  // Enum
  availableHooks: isDef(AvailableHooks),

  // AI SDK — tool authoring
  tool: isFn(tool),
  dynamicTool: isFn(dynamicTool),
  jsonSchema: isFn(jsonSchema),
  zodSchema: isFn(zodSchema),
  asSchema: isFn(asSchema),
  generateId: isFn(generateId),
  createIdGenerator: isFn(createIdGenerator),
  hasToolCall: isFn(hasToolCall),
  stepCountIs: isFn(stepCountIs),
  isLoopFinished: isFn(isLoopFinished),

  // AI SDK — middleware
  wrapLanguageModel: isFn(wrapLanguageModel),
  wrapEmbeddingModel: isFn(wrapEmbeddingModel),
  wrapImageModel: isFn(wrapImageModel),
  wrapProvider: isFn(wrapProvider),
  extractReasoningMiddleware: isFn(extractReasoningMiddleware),
  extractJsonMiddleware: isFn(extractJsonMiddleware),
  defaultSettingsMiddleware: isFn(defaultSettingsMiddleware),
  simulateStreamingMiddleware: isFn(simulateStreamingMiddleware),
  smoothStream: isFn(smoothStream),

  // AI SDK — provider registry
  createProviderRegistry: isFn(createProviderRegistry),
  customProvider: isFn(customProvider),

  // AI SDK — messages
  convertToModelMessages: isFn(convertToModelMessages),
  pruneMessages: isFn(pruneMessages),
  validateUIMessages: isFn(validateUIMessages),
  consumeStream: isFn(consumeStream),

  // AI SDK — media
  generateImage: isFn(generateImage),
  experimental_transcribe: isFn(experimental_transcribe),
  experimental_generateSpeech: isFn(experimental_generateSpeech),

  // AI SDK — misc
  cosineSimilarity: isFn(cosineSimilarity),
  simulateReadableStream: isFn(simulateReadableStream),
  parsePartialJson: isFn(parsePartialJson),

  // AI SDK — gateway
  gateway: isDef(gateway),
  createGateway: isFn(createGateway),

  // AI SDK — error classes
  aisdkError: isFn(AISDKError),
  apiCallError: isFn(APICallError),
  noObjectGeneratedError: isFn(NoObjectGeneratedError),
  noSuchModelError: isFn(NoSuchModelError),
  noSuchToolError: isFn(NoSuchToolError),
  invalidPromptError: isFn(InvalidPromptError),
  invalidToolInputError: isFn(InvalidToolInputError),
  retryError: isFn(RetryError),
  typeValidationError: isFn(TypeValidationError),
  loadAPIKeyError: isFn(LoadAPIKeyError),
});
