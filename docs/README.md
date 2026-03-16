# Brainkit Documentation

Brainkit is the execution engine for [Brainlet](https://github.com/brainlet/brainlet). A Kit is a self-contained environment — one QuickJS runtime with Mastra + AI SDK + polyfills loaded, plus Go services (bus, tool registry, WASM, storage). All agents, AI calls, workflows, and `.ts` code run inside a Kit.

## Guides

Conceptual documentation — what things are, when to use them, how to choose.

- [Storage](guides/storage.md) — Storage providers, embedded SQLite, memory backends, vector stores. What's supported, what isn't, and why.

## API Reference

Technical reference — Go config structs, TypeScript constructors, method signatures, error cases.

- [Storage API](api/storage/README.md) — `StorageConfig`, `LibSQLStore`, `LibSQLVector`, `AddStorage`/`RemoveStorage`
