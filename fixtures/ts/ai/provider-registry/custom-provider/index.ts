// `customProvider` maps short names to pre-wired models; useful for
// project-internal aliasing ("fast" → gpt-4o-mini, "safe" →
// claude-sonnet). `createProviderRegistry` composes multiple
// providers behind one lookup and resolves
// `provider.languageModel("fast")` to the right underlying model.
// This fixture proves construction and routing without reaching the
// wire — calling the model is deferred to the paid fixtures under
// agent/generate/.
import { customProvider, createProviderRegistry } from "ai";
import { model } from "kit";
import { output } from "kit";

const fast = (model as any)("openai", "gpt-4o-mini");

const myProvider = customProvider({
  languageModels: {
    fast,
  },
});

const registry = createProviderRegistry({
  my: myProvider,
});

const resolvedFromProvider = myProvider.languageModel("fast");
const resolvedFromRegistry = (registry as any).languageModel("my:fast");

output({
  providerHasLanguageModel: typeof myProvider.languageModel === "function",
  registryHasLanguageModel: typeof (registry as any).languageModel === "function",
  providerResolves: !!resolvedFromProvider,
  registryResolves: !!resolvedFromRegistry,
  resolvedIsObject: typeof resolvedFromProvider === "object",
});
