# Brainkit Documentation

Brainkit is the execution engine for [Brainlet](https://github.com/brainlet/brainlet). A Kit is a self-contained environment — one QuickJS runtime with Mastra + AI SDK + polyfills loaded, plus Go services (bus, tool registry, WASM, storage). All agents, AI calls, workflows, and `.ts` code run inside a Kit.

## Guides

Conceptual documentation — what things are, when to use them, how to choose.

- [Storage](guides/storage.md) — Storage providers, embedded SQLite, memory backends, vector stores. What's supported, what isn't, and why.
- [Workspace](guides/workspace.md) — Filesystem, sandbox, search, skills, LSP, tool remapping, dynamic factories.

## API Reference

Technical reference — Go config structs, TypeScript constructors, method signatures, error cases.

- [Storage API](api/storage/README.md) — `StorageConfig`, `LibSQLStore`, `LibSQLVector`, `AddStorage`/`RemoveStorage`
- [Workspace API](api/workspace/README.md) — `Workspace`, `LocalFilesystem`, `LocalSandbox`, search, tools config, LSP
