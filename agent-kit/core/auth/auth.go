// Ported from: packages/core/src/auth/index.ts
//
// Package auth provides authentication capabilities for Mastra.
//
// This package re-exports interfaces and default implementations:
//
//   - Interfaces: See the authinterfaces sub-package for Session, ISessionProvider,
//     User, IUserProvider, ICredentialsProvider, and ISSOProvider.
//
//   - Default session providers:
//     - CookieSessionProvider: Signed cookie sessions (auth/defaults/session)
//     - MemorySessionProvider: In-memory sessions for development (auth/defaults/session)
//
//   - Enterprise features (RBAC, ACL, license validation): See the auth/ee sub-package.
//
// In Go, re-exports are not idiomatic. Import the sub-packages directly:
//
//	import "github.com/brainlet/brainkit/agent-kit/core/auth/authinterfaces"
//	import "github.com/brainlet/brainkit/agent-kit/core/auth/defaults/session"
//	import "github.com/brainlet/brainkit/agent-kit/core/auth/ee"
//	import "github.com/brainlet/brainkit/agent-kit/core/auth/ee/eeinterfaces"
//	import "github.com/brainlet/brainkit/agent-kit/core/auth/ee/defaults"
//	import "github.com/brainlet/brainkit/agent-kit/core/auth/ee/defaults/rbac"
package auth
