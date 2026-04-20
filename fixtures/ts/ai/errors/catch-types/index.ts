// AI SDK error classes — each subclasses AISDKError and ships an
// `isInstance()` guard so user code can discriminate between failure
// modes when `generateObject`, `generateText`, etc. throw. This
// fixture doesn't reach the wire: it constructs each class, throws,
// and catches via `instanceof` and `isInstance` to prove the surface.
import {
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

function tryCatch(ctor: new (...args: any[]) => Error, args: any[]) {
  try {
    throw new ctor(...args);
  } catch (e) {
    return {
      isAiError: e instanceof AISDKError,
      isInstance:
        typeof (ctor as any).isInstance === "function"
          ? (ctor as any).isInstance(e)
          : false,
      name: (e as Error).name,
    };
  }
}

const apiCall = tryCatch(APICallError, [
  {
    message: "rate limited",
    url: "https://api.example.com/v1/chat",
    requestBodyValues: {},
  },
]);
const noObject = tryCatch(NoObjectGeneratedError, [
  { message: "could not produce object", cause: new Error("token limit") },
]);
const noModel = tryCatch(NoSuchModelError, [
  { modelId: "openai/gpt-999", modelType: "languageModel" },
]);
const noTool = tryCatch(NoSuchToolError, [
  { toolName: "delete-record" },
]);
const invalidPrompt = tryCatch(InvalidPromptError, [
  { prompt: {}, message: "prompt missing text" },
]);
const invalidToolInput = tryCatch(InvalidToolInputError, [
  { toolName: "add", toolInput: '{"a":1}', cause: new Error("missing b") },
]);
const retry = tryCatch(RetryError, [
  { message: "gave up", reason: "maxRetriesExceeded", errors: [] },
]);
const typeValidation = tryCatch(TypeValidationError, [
  { value: {}, cause: new Error("schema mismatch") },
]);
const loadKey = tryCatch(LoadAPIKeyError, [{ message: "missing key" }]);

output({
  apiCallIsAi: apiCall.isAiError,
  apiCallIsInstance: apiCall.isInstance,
  noObjectIsInstance: noObject.isInstance,
  noModelIsInstance: noModel.isInstance,
  noToolIsInstance: noTool.isInstance,
  invalidPromptIsInstance: invalidPrompt.isInstance,
  invalidToolInputIsInstance: invalidToolInput.isInstance,
  retryIsInstance: retry.isInstance,
  typeValidationIsInstance: typeValidation.isInstance,
  loadKeyIsInstance: loadKey.isInstance,
});
