// Ported from: packages/core/src/tools/is-vercel-tool.ts
package tools

// This file corresponds to is-vercel-tool.ts which is a re-export file:
//
//   export { isVercelTool } from './toolchecks';
//
// In TypeScript, this file exists to provide a separate import path for
// isVercelTool. In Go, since all files in the tools package share the same
// namespace, IsVercelTool is already exported directly from toolchecks.go.
//
// No additional code is needed. The function IsVercelTool is defined in
// toolchecks.go and is accessible as tools.IsVercelTool from any importing
// package.
