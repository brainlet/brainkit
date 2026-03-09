// Package interfaces provides shared interface definitions that break
// circular dependencies between agent-kit packages.
//
// This package exists because several agent-kit packages need to reference
// each other's types (e.g., mastra needs Agent, agent needs Mastra).
// Instead of creating circular imports, shared interfaces are extracted here.
//
// Key interfaces:
//   - Agent: core agent behavior, implemented by agent.Agent
//   - MastraScorer: evaluation scorer, implemented by evals.MastraScorer
//
// Key shared types:
//   - AgentMethodType: shared enum used by both agent and llm/model packages
//
// Design constraints:
//   - This package has ZERO dependencies on other agent-kit packages.
//   - Only stdlib and external library imports are permitted.
//   - Interfaces are MINIMAL — only the methods actually needed by consumers.
//   - No concrete types live here, only interfaces and simple value types.
//
// TS equivalent: These interfaces correspond to the TypeScript abstract classes
// and interfaces that are passed around in @mastra/core. In TS, circular
// references work due to runtime resolution. In Go, we extract them here.
package interfaces
