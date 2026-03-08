// Ported from: packages/ai/src/util/deep-partial.ts
//
// License for this File only:
//
// MIT License
//
// Copyright (c) Sindre Sorhus <sindresorhus@gmail.com> (https://sindresorhus.com)
// Copyright (c) Vercel, Inc. (https://vercel.com)
//
// The original TypeScript file defines a DeepPartial<T> type utility that makes
// all keys and nested keys of an object optional. Since Go does not have structural
// type-level partial constructs, this file serves as a documentation marker.
//
// In Go, optional fields are idiomatically represented using pointers (*T) or the
// omitempty JSON struct tag. There is no direct equivalent of TypeScript's Partial<T>.
//
// Users of this port should use pointer fields in structs where partial/optional
// semantics are needed.
package util
