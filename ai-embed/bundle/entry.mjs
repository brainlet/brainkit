import { generateText } from 'ai';
import { createOpenAI } from '@ai-sdk/openai';

globalThis.__ai_sdk = {
  generateText,
  createOpenAI,
};
