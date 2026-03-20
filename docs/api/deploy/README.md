# Deploy API Reference

Deploy, teardown, and redeploy TypeScript code into isolated SES Compartments. Resources created inside a deployment are tracked by source name and cleaned up on teardown.

---

## Go API

### Kit.Deploy

Evaluates code in a new SES Compartment with isolated globals. Resources created inside the compartment are tracked by `source`.

```go
func (k *Kit) Deploy(ctx context.Context, source, code string) ([]ResourceInfo, error)
```

| Param | Type | Description |
|-------|------|-------------|
| `ctx` | `context.Context` | Cancellation context |
| `source` | `string` | Deployment identifier (e.g. `"agents.ts"`). Must be unique -- returns error if already deployed. |
| `code` | `string` | TypeScript code to evaluate inside the compartment |

**Returns**: Slice of resources created by the code, or error. On error, any partial resources are cleaned up automatically.

```go
resources, err := kit.Deploy(ctx, "agents.ts", `
    // No import needed — agent() is endowed as a global inside the Compartment
    agent({ name: "greeter", model: "openai/gpt-4o-mini", instructions: "Say hello." });
`)
// resources = [{Type:"agent", ID:"greeter", Name:"greeter", Source:"agents.ts", CreatedAt:1710792000000}]
```

### Kit.Teardown

Removes all resources from a deployed source and drops its compartment. Idempotent -- returns 0 if the source was not deployed.

```go
func (k *Kit) Teardown(ctx context.Context, source string) (int, error)
```

| Param | Type | Description |
|-------|------|-------------|
| `ctx` | `context.Context` | Cancellation context |
| `source` | `string` | Source identifier to tear down |

**Returns**: Number of resources removed, or error.

```go
removed, err := kit.Teardown(ctx, "agents.ts")
// removed = 1
```

### Kit.Redeploy

Tears down the old deployment and deploys new code in one call. If teardown fails, it is logged but deploy proceeds.

```go
func (k *Kit) Redeploy(ctx context.Context, source, code string) ([]ResourceInfo, error)
```

| Param | Type | Description |
|-------|------|-------------|
| `ctx` | `context.Context` | Cancellation context |
| `source` | `string` | Source identifier to redeploy |
| `code` | `string` | New TypeScript code |

**Returns**: Slice of resources from the new deployment, or error.

```go
resources, err := kit.Redeploy(ctx, "agents.ts", `
    agent({ name: "greeter-v2", model: "openai/gpt-4o-mini", instructions: "Updated." });
`)
```

### Kit.ListDeployments

Returns all currently deployed sources with their resources. Refreshes resource lists from the registry at call time.

```go
func (k *Kit) ListDeployments() []deploymentInfo
```

**Returns**: Slice of `deploymentInfo` (may be empty).

---

## Types

### deploymentInfo

```go
type deploymentInfo struct {
    Source    string         `json:"source"`
    CreatedAt time.Time     `json:"createdAt"`
    Resources []ResourceInfo `json:"resources,omitempty"`
}
```

| Field | Type | Description |
|-------|------|-------------|
| `Source` | `string` | Deployment identifier |
| `CreatedAt` | `time.Time` | When the deployment was created |
| `Resources` | `[]ResourceInfo` | Resources registered by this deployment |

### ResourceInfo

```go
type ResourceInfo struct {
    Type      string `json:"type"`
    ID        string `json:"id"`
    Name      string `json:"name"`
    Source    string `json:"source"`
    CreatedAt int64  `json:"createdAt"`
}
```

| Field | Type | Description |
|-------|------|-------------|
| `Type` | `string` | Resource kind: `"agent"`, `"tool"`, `"workflow"`, `"wasm"`, `"memory"`, `"harness"` |
| `ID` | `string` | Unique within type |
| `Name` | `string` | Display name |
| `Source` | `string` | `.ts` filename that created it |
| `CreatedAt` | `int64` | Unix timestamp (milliseconds) |

---

## Bus Topics

All deploy operations are available over the bus via Ask (request/response).

### kit.deploy

Deploy code from a source.

| Direction | Shape |
|-----------|-------|
| **Request** | `{"source": string, "code": string}` |
| **Response** | `{"deployed": true, "resources": ResourceInfo[]}` |

### kit.teardown

Tear down all resources from a source.

| Direction | Shape |
|-----------|-------|
| **Request** | `{"source": string}` |
| **Response** | `{"removed": int}` |

### kit.list

List all active deployments.

| Direction | Shape |
|-----------|-------|
| **Request** | `{}` |
| **Response** | `deploymentInfo[]` |

### kit.redeploy

Teardown + deploy in one call.

| Direction | Shape |
|-----------|-------|
| **Request** | `{"source": string, "code": string}` |
| **Response** | `{"deployed": true, "resources": ResourceInfo[]}` |

---

## SDK Messages

Typed messages for bus interactions. All implement `BusMessage`.

```go
import "github.com/brainlet/brainkit/sdk/messages"
```

| Message | Fields | BusTopic() |
|---------|--------|------------|
| `KitDeployMsg` | `Source string`, `Code string` | `"kit.deploy"` |
| `KitTeardownMsg` | `Source string` | `"kit.teardown"` |
| `KitListMsg` | *(none)* | `"kit.list"` |
| `KitRedeployMsg` | `Source string`, `Code string` | `"kit.redeploy"` |

---

## SDK Client

Available on the plugin SDK client for remote deploy operations.

### client.Deploy

```go
func (c *grpcClient) Deploy(ctx context.Context, source, code string) error
```

Sends a `KitDeployMsg` via Ask and blocks until the Kit responds. Returns error if the deploy fails or context is cancelled.

### client.Teardown

```go
func (c *grpcClient) Teardown(ctx context.Context, source string) error
```

Sends a `KitTeardownMsg` via Ask and blocks until the Kit responds. Returns error if teardown fails or context is cancelled.
