package common

// BuiltinNames contains compiler-generated and standard library names.
// These string constants are used by the compiler and builtins packages.
// Ported from: assemblyscript/src/builtins.ts BuiltinNames namespace.

const (
	// Compiler-generated names.
	BuiltinNameStart              = "~start"
	BuiltinNameStarted            = "~started"
	BuiltinNameArgumentsLength    = "~argumentsLength"
	BuiltinNameSetArgumentsLength = "~setArgumentsLength"

	// Runtime globals.
	BuiltinNameDataEnd      = "~lib/memory/__data_end"
	BuiltinNameStackPointer = "~lib/memory/__stack_pointer"
	BuiltinNameHeapBase     = "~lib/memory/__heap_base"
	BuiltinNameRttiBase     = "~lib/rt/__rtti_base"
	BuiltinNameVisitGlobals = "~lib/rt/__visit_globals"
	BuiltinNameVisitMembers = "~lib/rt/__visit_members"
	BuiltinNameTostack      = "~lib/rt/__tostack"

	// Standard library builtins.
	BuiltinNameAbort     = "~lib/builtins/abort"
	BuiltinNameTrace     = "~lib/builtins/trace"
	BuiltinNameSeed      = "~lib/builtins/seed"

	// Standard library class names.
	BuiltinNameObject = "~lib/object/Object"
	BuiltinNameStringEq  = "~lib/string/String.__eq"
	BuiltinNameStringNe  = "~lib/string/String.__ne"
	BuiltinNameStringNot = "~lib/string/String.__not"
)
