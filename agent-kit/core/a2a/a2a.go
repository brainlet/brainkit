// Ported from: packages/core/src/a2a/index.ts
//
// In TypeScript the index.ts barrel file re-exports:
//   export * from './error';
//   export * from '@a2a-js/sdk';
//   export type { JSONRPCResponse, JSONRPCError, TaskContext } from './types';
//
// In Go all exported symbols within the same package are automatically
// accessible to importers, so no explicit re-export mechanism is needed.
// This file serves as the package documentation entry point.
//
// Importers use:
//   import "github.com/brainlet/brainkit/agent-kit/core/a2a"
//
// Which gives access to all exported types, constants, and functions defined
// across types.go, error.go, and this file:
//
// Types (from types.go — includes both locally defined and @a2a-js/sdk stubs):
//   - JSONRPCError, JSONRPCResponse, JSONRPCMessage
//   - TaskContext
//   - Task, TaskState, TaskStatus
//   - Message, MessageRole
//   - Part, PartKind, TextPart, FilePart, DataPart
//   - FileContent, FileWithBytes, FileWithURI
//   - Artifact
//   - KnownErrorCode and all ErrorCode* constants
//
// Error utilities (from error.go):
//   - A2AError (struct implementing error interface)
//   - NewA2AError
//   - Factory functions: ParseError, InvalidRequest, MethodNotFound,
//     InvalidParams, InternalError, TaskNotFound, TaskNotCancelable,
//     PushNotificationNotSupported, UnsupportedOperation
package a2a
