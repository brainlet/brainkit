// Ported from: packages/ai/src/util/job.ts
package util

// Job is a function that performs an asynchronous unit of work.
type Job func() error
