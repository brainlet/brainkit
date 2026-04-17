# Agents Test Map

**Purpose:** Verifies agent registration, discovery, status management, and AI-backed agent execution via the bus
**Tests:** 10 functions across 3 files
**Entry point:** `agents_test.go` → `Run(t, env)`
**Campaigns:** transport (all 5), fullstack (all 3)

## Files

### lifecycle.go — Agent CRUD operations without AI

| Function | Purpose |
|----------|---------|
| testListEmpty | Publishes AgentListMsg on a fresh kernel and asserts the response contains an empty agents slice |
| testDiscoverNoMatch | Publishes AgentDiscoverMsg with capability "teleportation" and asserts no agents are returned |
| testGetStatusNotFound | Publishes AgentGetStatusMsg for a nonexistent agent name and asserts a non-empty error string is returned |
| testSetStatusNotFound | Publishes AgentSetStatusMsg for a nonexistent agent and asserts a non-empty error string is returned |
| testSetStatusInvalid | Publishes AgentSetStatusMsg with an invalid status value ("flying") and asserts a non-empty error string |

### ai.go — AI agent deploy + list + status lifecycle (requires OPENAI_API_KEY)

| Function | Purpose |
|----------|---------|
| testDeployAgentThenList | Deploys a .ts file that creates a Mastra Agent, calls generate, verifies non-empty text+usage in output, confirms agent appears in AgentListMsg, then verifies get/set status transitions from "idle" to "busy" |

### surface.go — AI SDK surface tests through deployed .ts (requires OPENAI_API_KEY)

| Function | Purpose |
|----------|---------|
| testSurfaceGenerateTextReal | Deploys .ts that calls generateText with gpt-4o-mini, verifies the response text contains "4" and has token usage |
| testSurfaceAgentGenerate | Deploys .ts that creates an Agent, calls generate, verifies the response contains "SURFACE_AGENT_OK" and the agent appears in AgentListMsg |
| testSurfaceAgentWithTool | Deploys .ts that creates an Agent with a custom addTool, calls generate ("What is 17+25?"), verifies non-empty text and that steps array is populated |
| testSurfaceBusServiceAIProxy | Deploys .ts as a bus service that calls generateText on incoming messages, Go sends a message via SendToService, verifies the AI response text and usage are returned through the bus reply |

## Cross-references

- **Campaigns:** `transport/{sqlite,nats,postgres,redis,amqp}_test.go`, `fullstack/{redis_mongodb,amqp_postgres_vector}_test.go`
- **Related domains:** deploy (agent registration), bus (agent discovery via kit.register)
- **Fixtures:** AI fixtures (OPENAI_API_KEY dependent)
