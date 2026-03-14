import {
  generateText, streamText,
  generateObject, streamObject,
  embed, embedMany,
  tool,
  jsonSchema,
  wrapLanguageModel,
  defaultSettingsMiddleware,
  extractReasoningMiddleware,
} from 'ai';
import { createOpenAI } from '@ai-sdk/openai';
import { createAnthropic } from '@ai-sdk/anthropic';
import { createGoogleGenerativeAI } from '@ai-sdk/google';

globalThis.__ai_sdk = {
  generateText, streamText,
  generateObject, streamObject,
  embed, embedMany,
  tool,
  jsonSchema,
  createOpenAI,
  createAnthropic,
  createGoogleGenerativeAI,
  wrapLanguageModel,
  defaultSettingsMiddleware,
  extractReasoningMiddleware,
};
