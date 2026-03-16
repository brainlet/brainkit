# Workspace API

## TypeScript API

### Workspace

```ts
import { Workspace, LocalFilesystem, LocalSandbox } from "brainlet";
```

#### Constructor

```ts
const ws = new Workspace(config: WorkspaceConfig);
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | `string` | no | Workspace identifier |
| `name` | `string` | no | Display name |
| `filesystem` | `FilesystemInstance` | yes | Filesystem provider |
| `sandbox` | `SandboxInstance` | no | Sandbox for command execution |
| `bm25` | `boolean \| { k1?: number; b?: number }` | no | Enable BM25 keyword search |
| `vectorStore` | `VectorStore` | no | Vector store for semantic search. Requires `embedder`. |
| `embedder` | `(text: string) => Promise<number[]>` | no | Embedding function. Required if `vectorStore` is set. |
| `searchIndexName` | `string` | no | Custom vector index name. Default: auto-generated. |
| `tools` | `WorkspaceToolsConfig` | no | Per-tool rename, enable/disable, approval |
| `skills` | `string[]` | no | Paths to skills directories |
| `lsp` | `boolean \| LSPConfig` | no | Enable LSP diagnostics |

#### Methods

```ts
await ws.init();                              // Initialize (create indexes, start LSP)
await ws.destroy();                           // Clean up (stop LSP, release resources)
await ws.search(query, options?);             // Search indexed content
await ws.index(filePath, content);            // Index a file for search
ws.getInfo();                                 // Get workspace metadata
ws.getInstructions();                         // Get agent context instructions
ws.getToolsConfig();                          // Get current tools config
ws.setToolsConfig(config);                    // Update tools config at runtime
```

#### search()

```ts
const results = await ws.search(query: string, options?: WorkspaceSearchOptions);
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `topK` | `number` | `10` | Number of results |
| `minScore` | `number` | — | Minimum score threshold |
| `mode` | `"bm25" \| "vector" \| "hybrid"` | auto | Search mode |
| `vectorWeight` | `number` | `0.5` | Vector weight in hybrid mode (0-1) |
| `filter` | `Record<string, any>` | — | Metadata filter (vector only) |

Returns `WorkspaceSearchResult[]`:

| Field | Type | Description |
|-------|------|-------------|
| `id` | `string` | Document/file ID |
| `content` | `string` | Matched content |
| `score` | `number` | Relevance score |
| `scoreDetails` | `{ vector?: number; bm25?: number }` | Per-mode scores |
| `metadata` | `Record<string, any>` | Document metadata |
| `lineRange` | `{ start: number; end: number }` | Line range of match |

---

### WorkspaceToolsConfig

Per-tool configuration passed to `Workspace({ tools })` or `ws.setToolsConfig()`.

```ts
type WorkspaceToolsConfig = {
  enabled?: boolean;          // Default for all tools
  requireApproval?: boolean;  // Default for all tools
} & Record<string, WorkspaceToolConfig>;
```

#### WorkspaceToolConfig

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `name` | `string` | — | Custom name exposed to the LLM |
| `enabled` | `boolean` | `true` | Whether tool is available |
| `requireApproval` | `boolean` | `false` | Require human approval |
| `requireReadBeforeWrite` | `boolean` | `false` | Force read before write |
| `maxOutputTokens` | `number` | — | Limit tool output size |

#### Tool Constants

Use these as keys in the tools config:

| Constant | Default Name |
|----------|-------------|
| `mastra_workspace_read_file` | `read_file` |
| `mastra_workspace_write_file` | `write_file` |
| `mastra_workspace_edit_file` | `edit_file` |
| `mastra_workspace_list_files` | `list_files` |
| `mastra_workspace_file_stat` | `file_stat` |
| `mastra_workspace_mkdir` | `mkdir` |
| `mastra_workspace_delete` | `delete` |
| `mastra_workspace_copy_file` | `copy_file` |
| `mastra_workspace_move_file` | `move_file` |
| `mastra_workspace_grep` | `grep` |
| `mastra_workspace_execute_command` | `execute_command` |

---

### LSPConfig

```ts
const ws = new Workspace({
  lsp: {
    diagnosticTimeout: 5000,
    initTimeout: 15000,
    disableServers: ["eslint"],
    binaryOverrides: { typescript: "/path/to/tls --stdio" },
    packageRunner: "npx --yes",
  },
});
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `diagnosticTimeout` | `number` | `5000` | Wait time for diagnostics (ms) |
| `initTimeout` | `number` | `15000` | Server init timeout (ms) |
| `disableServers` | `string[]` | — | Skip specific LSP servers |
| `binaryOverrides` | `Record<string, string>` | — | Custom binary paths |
| `packageRunner` | `string` | — | Fallback runner (e.g. `"npx --yes"`) |

---

### LocalFilesystem

```ts
import { LocalFilesystem } from "brainlet";

const fs = new LocalFilesystem({
  basePath: "./project",
  allowedPaths: ["/tmp/scratch"],
  contained: true,  // default: true — prevents path traversal
});
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `basePath` | `string` | required | Root directory for filesystem operations |
| `allowedPaths` | `string[]` | — | Additional paths the agent can access |
| `contained` | `boolean` | `true` | Prevent path traversal outside basePath |

#### Methods

```ts
await fs.readFile(path);
await fs.writeFile(path, content);
await fs.stat(path);
await fs.readdir(path);
await fs.mkdir(path, { recursive: true });
await fs.rm(path, { recursive: true });
fs.setAllowedPaths(paths);  // Update at runtime
```

---

### LocalSandbox

```ts
import { LocalSandbox } from "brainlet";

const sandbox = new LocalSandbox({
  workingDirectory: "./project",
  env: { NODE_ENV: "development" },
});
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `workingDirectory` | `string` | — | CWD for commands |
| `env` | `Record<string, string>` | — | Environment variables |
| `defaultShell` | `string` | — | Shell to use (default: system shell) |

---

### Agent Integration

#### Static workspace

```ts
const a = agent({
  model: "openai/gpt-4o-mini",
  workspace: ws,
});
```

#### Dynamic workspace (per-request)

```ts
const a = agent({
  model: "openai/gpt-4o-mini",
  workspace: ({ requestContext }) => {
    const path = requestContext.get("projectPath");
    return new Workspace({
      filesystem: new LocalFilesystem({ basePath: path }),
    });
  },
});
```

---

## Go API

Workspace is configured entirely from TypeScript. The Go side provides:

- **Filesystem bridges** (`jsbridge/fs.go`): 14 async Go functions backing `fs/promises` operations
- **Exec bridges** (`jsbridge/exec.go`): Process spawning with stdin/stdout for `execute_command` and LSP
- **Containment**: Path validation happens in `LocalFilesystem` using `path.resolve` + comparison

No Go-side workspace configuration is needed. The workspace lives in the JS runtime.
