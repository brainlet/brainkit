// Package tracing adds persistent distributed-trace capture to a
// Kit. Init attaches the module's store to the Kit's Tracer; Close
// detaches it. Cross-cutting spans emitted by other subsystems
// (bus handlers, plugins, workflow) drop into the store.
//
// Status: beta.
package tracing
