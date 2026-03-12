import { generateText, streamText } from 'ai';
import { createOpenAI } from '@ai-sdk/openai';

globalThis.__ai_sdk = {
  generateText,
  streamText,
  createOpenAI,
};
