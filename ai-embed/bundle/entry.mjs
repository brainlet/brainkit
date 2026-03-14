import {
  generateText, streamText,
  generateObject, streamObject,
  embed, embedMany,
  tool,
} from 'ai';
import { createOpenAI } from '@ai-sdk/openai';

globalThis.__ai_sdk = {
  generateText, streamText,
  generateObject, streamObject,
  embed, embedMany,
  tool,
  createOpenAI,
};
