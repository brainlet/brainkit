# Workspace in Brainkit

Workspaces give agents filesystem access, command execution, skill discovery, and content search. When an agent has a workspace, it automatically gets tools like `read_file`, `write_file`, `grep`, `execute_command`, and `search` — no manual tool registration needed.

---

## Quick Start

```ts
import { agent, Workspace, LocalFilesystem, LocalSandbox } from "kit";

const ws = new Workspace({
  id: "my-project",
  filesystem: new LocalFilesystem({ basePath: "./project" }),
  sandbox: new LocalSandbox({ workingDirectory: "./project" }),
  bm25: true,
});

const coder = agent({
  model: "openai/gpt-4o-mini",
  instructions: "You are a coding assistant. Use workspace tools to read and edit files.",
  workspace: ws,
});

const result = await coder.generate("Read main.go and add error handling");
```

The agent now has 15 auto-injected tools and can autonomously read, write, search, and execute commands in the workspace directory.

---

## What Gets Auto-Injected

When you attach a workspace to an agent, Mastra injects these tools automatically:

### Filesystem Tools
| Tool | Description |
|------|-------------|
| `read_file` | Read file contents |
| `write_file` | Write/create a file |
| `edit_file` | Edit a file (find & replace) |
| `list_files` | List directory contents |
| `file_stat` | Get file metadata (size, modified time) |
| `mkdir` | Create directories |
| `delete` | Delete files/directories |
| `copy_file` | Copy a file |
| `move_file` | Move/rename a file |
| `grep` | Search file contents by pattern |

### Sandbox Tools
| Tool | Description |
|------|-------------|
| `execute_command` | Run a shell command |

### Search Tools
| Tool | Description |
|------|-------------|
| `search` | Search indexed workspace content (BM25, vector, or hybrid) |

### Skill Tools
| Tool | Description |
|------|-------------|
| `skill` | Execute a skill from SKILL.md definitions |
| `skill_read` | Read a skill's definition |
| `skill_search` | Search available skills |

---

## Search Modes

Workspace search supports three modes depending on what's configured.

### BM25 (Keyword Search)

In-memory keyword search. No external dependencies.

```ts
const ws = new Workspace({
  filesystem: new LocalFilesystem({ basePath: "./project" }),
  bm25: true,
});
```

The agent can then search with:
```
search("error handling patterns", { mode: "bm25", topK: 5 })
```

### Vector (Semantic Search)

Requires a vector store and an embedding function. Finds results by meaning, not just keywords.

```ts
const ws = new Workspace({
  filesystem: new LocalFilesystem({ basePath: "./project" }),
  vectorStore: new LibSQLVector({ id: "ws-search", url: "http://libsql-server:8080" }),
  embedder: async (text) => {
    const r = await ai.embed({ model: "openai/text-embedding-3-small", value: text });
    return r.embedding;
  },
});
```

**Important:** Vector search requires a vector-capable backend. The embedded SQLite bridge (`StorageConfig.Path`) does NOT support vector operations (`vector32()` function). Use a real `libsql-server`, PgVector, or other vector store.

### Hybrid (BM25 + Vector)

Combines keyword and semantic search. Auto-detected when both BM25 and vector are configured.

```ts
const ws = new Workspace({
  filesystem: new LocalFilesystem({ basePath: "./project" }),
  bm25: true,
  vectorStore: vectors,
  embedder: embedder,
});

// Auto-detects hybrid mode
await ws.search("concurrent error handling");

// Or explicit
await ws.search("concurrent error handling", { mode: "hybrid", vectorWeight: 0.7 });
```

The `vectorWeight` parameter (0-1, default 0.5) controls the balance: higher values favor semantic matches, lower values favor keyword matches.

### Mode Auto-Detection

| Config | Default Mode |
|--------|-------------|
| `bm25: true` only | `bm25` |
| `vectorStore + embedder` only | `vector` |
| Both | `hybrid` |
| Neither | Search disabled |

---

## Tool Name Remapping

Rename workspace tools to customize the LLM's interface. Useful for making tool names more intuitive or matching a specific coding assistant style.

```ts
const ws = new Workspace({
  filesystem: new LocalFilesystem({ basePath: "./project" }),
  tools: {
    mastra_workspace_read_file: { name: "view" },
    mastra_workspace_edit_file: { name: "edit" },
    mastra_workspace_write_file: { name: "write" },
    mastra_workspace_list_files: { name: "find_files" },
    mastra_workspace_grep: { name: "search_content" },
    mastra_workspace_execute_command: { name: "run" },
  },
});
```

The agent sees `view`, `edit`, `write`, `find_files`, `search_content`, `run` instead of the default names.

---

## Mode Switching (Enable/Disable Tools)

Disable tools dynamically based on the agent's mode. For example, a "plan mode" that can read but not write:

```ts
const TOOLS_BUILD = {
  mastra_workspace_read_file: { name: "view" },
  mastra_workspace_write_file: { name: "write" },
  mastra_workspace_edit_file: { name: "edit" },
};

const TOOLS_PLAN = {
  mastra_workspace_read_file: { name: "view" },
  mastra_workspace_write_file: { enabled: false },
  mastra_workspace_edit_file: { enabled: false },
};

// Start in build mode
const ws = new Workspace({
  filesystem: new LocalFilesystem({ basePath: "./project" }),
  tools: TOOLS_BUILD,
});

// Switch to plan mode at runtime
ws.setToolsConfig(TOOLS_PLAN);

// Switch back
ws.setToolsConfig(TOOLS_BUILD);
```

Each tool config supports:

| Field | Type | Description |
|-------|------|-------------|
| `name` | `string` | Custom name exposed to the LLM |
| `enabled` | `boolean` | Whether the tool is available (default: true) |
| `requireApproval` | `boolean` | Require human approval before execution |
| `requireReadBeforeWrite` | `boolean` | Force read_file before write_file |
| `maxOutputTokens` | `number` | Limit tool output size |

---

## Dynamic Workspace Factory

Create workspaces per-request based on runtime context. The workspace config can be a function that receives `requestContext` and returns a Workspace instance:

```ts
const coder = agent({
  model: "openai/gpt-4o-mini",
  instructions: "You are a coding assistant.",
  workspace: ({ requestContext }) => {
    const projectPath = requestContext.get("projectPath");
    const mode = requestContext.get("mode") || "build";

    return new Workspace({
      id: `ws-${projectPath}`,
      filesystem: new LocalFilesystem({
        basePath: projectPath,
        allowedPaths: [projectPath, "/tmp"],
      }),
      sandbox: new LocalSandbox({ workingDirectory: projectPath }),
      tools: mode === "plan" ? TOOLS_PLAN : TOOLS_BUILD,
      bm25: true,
    });
  },
});

// Each call can have a different workspace
await coder.generate("Fix the bug in main.go", {
  requestContext: new RequestContext({ projectPath: "/home/user/project-a", mode: "build" }),
});

await coder.generate("Review the architecture", {
  requestContext: new RequestContext({ projectPath: "/home/user/project-b", mode: "plan" }),
});
```

This is the pattern mastracode uses: workspace properties (base path, allowed paths, tool configuration) change per-request based on the user's project and current mode.

---

## Skills

Skills are reusable instructions defined in SKILL.md files. The workspace discovers them from the filesystem and exposes `skill`, `skill_read`, and `skill_search` tools.

### Skill Directory Structure

```
project/
  skills/
    code-review/
      SKILL.md
    testing/
      SKILL.md
    deployment/
      SKILL.md
```

Each SKILL.md uses frontmatter:

```markdown
---
name: code-review
description: Review code for quality, security, and best practices
---

## Instructions

When reviewing code:
1. Check for security vulnerabilities
2. Verify error handling
3. Look for performance issues
```

The directory name must match the skill name in frontmatter.

### Agent Usage

```ts
const ws = new Workspace({
  filesystem: new LocalFilesystem({ basePath: "./project" }),
  skills: ["./project/skills"],
});

const a = agent({
  model: "openai/gpt-4o-mini",
  workspace: ws,
});

// Agent can now use: skill("code-review"), skill_read("testing"), skill_search("deploy")
```

---

## LSP (Language Server Protocol)

Workspace can integrate with language servers for post-edit diagnostics. After the agent edits a file, the LSP returns type errors, missing imports, syntax issues — the agent sees these and can fix them.

```ts
const ws = new Workspace({
  filesystem: new LocalFilesystem({ basePath: "./project" }),
  sandbox: new LocalSandbox({ workingDirectory: "./project" }),
  lsp: true,
});
```

With `lsp: true`, Mastra auto-detects language servers based on project files:

| Language | Server | Detected By |
|----------|--------|-------------|
| TypeScript/JavaScript | `typescript-language-server` | `tsconfig.json`, `package.json` |
| Go | `gopls` | `go.mod` |
| Rust | `rust-analyzer` | `Cargo.toml` |
| Python | `pyright` | `pyproject.toml`, `setup.py` |

Language servers must be installed on the system (or in `node_modules/.bin/`).

### LSP Config

```ts
const ws = new Workspace({
  filesystem: new LocalFilesystem({ basePath: "./project" }),
  lsp: {
    diagnosticTimeout: 3000,
    disableServers: ["eslint"],
    binaryOverrides: { typescript: "/usr/local/bin/typescript-language-server --stdio" },
    packageRunner: "npx --yes",
  },
});
```

---

## Filesystem Containment

`LocalFilesystem` with `contained: true` (default) prevents path traversal:

- Relative paths: `data.txt` → `basePath/data.txt`
- Absolute paths inside basePath: used as-is
- Absolute paths outside basePath: blocked
- `..` traversal: blocked

```ts
// Contained (default) — agent can only access files under basePath
new LocalFilesystem({ basePath: "./project" })

// With extra allowed paths
new LocalFilesystem({
  basePath: "./project",
  allowedPaths: ["/tmp/scratch", "/home/user/.config"],
})
```

---

## What's Not Supported

| Feature | Why |
|---------|-----|
| S3 Filesystem | Cloud provider — needs `@mastra/s3` bundled |
| GCS Filesystem | Cloud provider — needs GCS SDK |
| AgentFS | Virtual filesystem — needs Mastra server |
| E2B Sandbox | Cloud sandbox — needs E2B SDK |
| Daytona Sandbox | Cloud sandbox — needs Daytona SDK |
| Blaxel Sandbox | Cloud sandbox — needs Blaxel SDK |

These are all cloud/managed providers. For self-hosted deployments, `LocalFilesystem` + `LocalSandbox` cover all use cases.

---

## Testing

All workspace features are tested with real fixtures:

| Feature | Test |
|---------|------|
| Tool name remapping | `TestWorkspaceToolRemapping` — rename, enable/disable, mode switching |
| BM25 search | `TestWorkspaceBM25Search` — index, search, auto-detect mode |
| Vector + hybrid search | `TestWorkspaceVectorSearch` — needs libsql-server testcontainer |
| Dynamic workspace factory | `TestWorkspaceDynamicFactory` — factory called per-request |
| Skills config | `TestWorkspaceSkillsConfig` — SKILL.md discovery from configured paths |
| Containment + setAllowedPaths | `TestWorkspaceAllowedPaths` — blocked → allowed after runtime update |
| Filesystem operations | `workspace-read-write.js` — all 14 Go bridge operations |
| Agent auto-tools | `workspace-agent-tools.js` — agent reads/writes via injected tools |
| LSP diagnostics | `workspace-lsp.js` — TypeScript type errors detected |

### Vector Search Limitation

The embedded SQLite bridge (`StorageConfig.Path`) does NOT support workspace vector search — it lacks the `vector32()` function that LibSQL's native server provides. BM25 keyword search works fine with the embedded bridge.

For vector search in workspaces, use:
- A real `libsql-server` (testcontainer or remote Turso)
- PgVector
- Any other vector-capable backend
