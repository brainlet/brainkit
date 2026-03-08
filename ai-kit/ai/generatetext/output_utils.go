// Ported from: packages/ai/src/generate-text/output-utils.ts
package generatetext

// Note: In TypeScript, InferCompleteOutput, InferPartialOutput, and InferElementOutput
// are conditional types that extract type parameters from Output<COMPLETE, PARTIAL, ELEMENT>.
// In Go, without generics on the Output interface, these are simply interface{}.
// The actual types are determined at runtime through the Output methods.
