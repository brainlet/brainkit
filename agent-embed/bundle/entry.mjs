// agent-embed entry point
// Mastra framework + AI SDK providers for QuickJS embedding

import { Agent } from '@mastra/core/agent';
import { createTool } from '@mastra/core/tools';
import { Mastra } from '@mastra/core/mastra';
import { z } from 'zod';

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
  Mastra,
  z,

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
