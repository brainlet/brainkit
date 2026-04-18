// Package workflow adds declarative multi-step agent workflows to
// a Kit. Each workflow is a directed graph of bus-driven steps; the
// module exposes workflow.* bus commands to start / cancel / inspect
// live instances and persists run state through the Kit's store.
//
// Status: beta.
package workflow
