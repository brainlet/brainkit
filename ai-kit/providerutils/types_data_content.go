// Ported from: packages/provider-utils/src/types/data-content.ts
package providerutils

// DataContent represents data content. It can be a base64-encoded string or raw bytes.
// In TypeScript this is: string | Uint8Array | ArrayBuffer | Buffer
// In Go we use interface{} to represent this union; callers should use string or []byte.
type DataContent = interface{}
