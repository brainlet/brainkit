package asembed

import (
	"unsafe"

	"github.com/fastschema/qjs"
)

func registerFunctionImpls(ctx *qjs.Context, lm *LinearMemory) {
	ctx.SetFunc("_BinaryenAddFunction", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		params := argU(a, 2)
		results := argU(a, 3)
		varTypesPtr := argI(a, 4)
		numVarTypes := argI(a, 5)
		body := argU(a, 6)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		varTypes := readPtrArray(lm, varTypesPtr, numVarTypes)
		return retF(this.Context(), cgoAddFunction(module, name, params, results, varTypes, body))
	})
	ctx.SetFunc("_BinaryenGetFunction", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(this.Context(), cgoGetFunction(module, name))
	})
	ctx.SetFunc("_BinaryenRemoveFunction", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		cgoRemoveFunction(module, name)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenGetNumFunctions", func(this *qjs.This) (*qjs.Value, error) {
		return retI(this.Context(), cgoGetNumFunctions(argU(this.Args(), 0)))
	})
	ctx.SetFunc("_BinaryenGetFunctionByIndex", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoGetFunctionByIndex(argU(a, 0), argI(a, 1)))
	})
	ctx.SetFunc("_BinaryenFunctionGetName", func(this *qjs.This) (*qjs.Value, error) {
		cName := cgoFunctionGetName(argU(this.Args(), 0))
		if cName == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cName)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})
	ctx.SetFunc("_BinaryenFunctionGetParams", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoFunctionGetParams(argU(this.Args(), 0)))
	})
	ctx.SetFunc("_BinaryenFunctionGetResults", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoFunctionGetResults(argU(this.Args(), 0)))
	})
	ctx.SetFunc("_BinaryenFunctionGetNumVars", func(this *qjs.This) (*qjs.Value, error) {
		return retI(this.Context(), cgoFunctionGetNumVars(argU(this.Args(), 0)))
	})
	ctx.SetFunc("_BinaryenFunctionGetVar", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retF(this.Context(), cgoFunctionGetVar(argU(a, 0), argI(a, 1)))
	})
	ctx.SetFunc("_BinaryenFunctionGetBody", func(this *qjs.This) (*qjs.Value, error) {
		return retF(this.Context(), cgoFunctionGetBody(argU(this.Args(), 0)))
	})
	ctx.SetFunc("_BinaryenFunctionSetBody", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cgoFunctionSetBody(argU(a, 0), argU(a, 1))
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenFunctionGetNumLocals", func(this *qjs.This) (*qjs.Value, error) {
		return retI(this.Context(), cgoFunctionGetNumLocals(argU(this.Args(), 0)))
	})
	ctx.SetFunc("_BinaryenFunctionHasLocalName", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		return retBool(this.Context(), cgoFunctionHasLocalName(argU(a, 0), argI(a, 1)))
	})
	ctx.SetFunc("_BinaryenFunctionGetLocalName", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		cName := cgoFunctionGetLocalName(argU(a, 0), argI(a, 1))
		if cName == nil {
			return retI(this.Context(), 0)
		}
		s := cgoGoString(cName)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(this.Context(), ptr)
	})
	ctx.SetFunc("_BinaryenFunctionSetLocalName", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		fn := argU(a, 0)
		index := argI(a, 1)
		namePtr := argI(a, 2)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		cgoFunctionSetLocalName(fn, index, name)
		return retVoid(this.Context())
	})
	// FunctionAddVar, FunctionGetType, FunctionSetType, FunctionOptimize,
	// FunctionRunPasses, FunctionSetDebugLocation — keep as stubs for now
}

func registerGlobalImpls(ctx *qjs.Context, lm *LinearMemory) {
	ctx.SetFunc("_BinaryenAddGlobal", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		typ := argU(a, 2)
		mutable := argBool(a, 3)
		init := argU(a, 4)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		result := cgoAddGlobal(module, name, typ, mutable, init)
		return retF(this.Context(), result)
	})
	ctx.SetFunc("_BinaryenGetGlobal", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(this.Context(), cgoGetGlobal(module, name))
	})
	ctx.SetFunc("_BinaryenRemoveGlobal", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		cgoRemoveGlobal(module, name)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenGetNumGlobals", func(this *qjs.This) (*qjs.Value, error) {
		return retI(this.Context(), cgoGetNumGlobals(argU(this.Args(), 0)))
	})
	// GlobalGetName, GlobalGetType, GlobalIsMutable, GlobalGetInitExpr,
	// GetGlobalByIndex — keep as stubs
}

func registerExportImpls(ctx *qjs.Context, lm *LinearMemory) {
	ctx.SetFunc("_BinaryenAddFunctionExport", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		intNamePtr := argI(a, 1)
		extNamePtr := argI(a, 2)
		intName := cgoCString(lm.ReadString(intNamePtr))
		defer cgoFree(unsafe.Pointer(intName))
		extName := cgoCString(lm.ReadString(extNamePtr))
		defer cgoFree(unsafe.Pointer(extName))
		return retF(this.Context(), cgoAddFunctionExport(module, intName, extName))
	})
	ctx.SetFunc("_BinaryenAddTableExport", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		intNamePtr := argI(a, 1)
		extNamePtr := argI(a, 2)
		intName := cgoCString(lm.ReadString(intNamePtr))
		defer cgoFree(unsafe.Pointer(intName))
		extName := cgoCString(lm.ReadString(extNamePtr))
		defer cgoFree(unsafe.Pointer(extName))
		return retF(this.Context(), cgoAddTableExport(module, intName, extName))
	})
	ctx.SetFunc("_BinaryenAddMemoryExport", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		intNamePtr := argI(a, 1)
		extNamePtr := argI(a, 2)
		intName := cgoCString(lm.ReadString(intNamePtr))
		defer cgoFree(unsafe.Pointer(intName))
		extName := cgoCString(lm.ReadString(extNamePtr))
		defer cgoFree(unsafe.Pointer(extName))
		return retF(this.Context(), cgoAddMemoryExport(module, intName, extName))
	})
	ctx.SetFunc("_BinaryenAddGlobalExport", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		intNamePtr := argI(a, 1)
		extNamePtr := argI(a, 2)
		intName := cgoCString(lm.ReadString(intNamePtr))
		defer cgoFree(unsafe.Pointer(intName))
		extName := cgoCString(lm.ReadString(extNamePtr))
		defer cgoFree(unsafe.Pointer(extName))
		return retF(this.Context(), cgoAddGlobalExport(module, intName, extName))
	})
	ctx.SetFunc("_BinaryenAddTagExport", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		intNamePtr := argI(a, 1)
		extNamePtr := argI(a, 2)
		intName := cgoCString(lm.ReadString(intNamePtr))
		defer cgoFree(unsafe.Pointer(intName))
		extName := cgoCString(lm.ReadString(extNamePtr))
		defer cgoFree(unsafe.Pointer(extName))
		return retF(this.Context(), cgoAddTagExport(module, intName, extName))
	})
	ctx.SetFunc("_BinaryenGetExport", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(this.Context(), cgoGetExport(module, name))
	})
	ctx.SetFunc("_BinaryenRemoveExport", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		cgoRemoveExport(module, name)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenGetNumExports", func(this *qjs.This) (*qjs.Value, error) {
		return retI(this.Context(), cgoGetNumExports(argU(this.Args(), 0)))
	})
	// ExportGetKind, ExportGetName, ExportGetValue, GetExportByIndex — keep as stubs
}

func registerImportImpls(ctx *qjs.Context, lm *LinearMemory) {
	ctx.SetFunc("_BinaryenAddFunctionImport", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		intNamePtr := argI(a, 1)
		extModPtr := argI(a, 2)
		extBasePtr := argI(a, 3)
		params := argU(a, 4)
		results := argU(a, 5)
		intName := cgoCString(lm.ReadString(intNamePtr))
		defer cgoFree(unsafe.Pointer(intName))
		extMod := cgoCString(lm.ReadString(extModPtr))
		defer cgoFree(unsafe.Pointer(extMod))
		extBase := cgoCString(lm.ReadString(extBasePtr))
		defer cgoFree(unsafe.Pointer(extBase))
		cgoAddFunctionImport(module, intName, extMod, extBase, params, results)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenAddGlobalImport", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		intNamePtr := argI(a, 1)
		extModPtr := argI(a, 2)
		extBasePtr := argI(a, 3)
		globalType := argU(a, 4)
		mutable := argBool(a, 5)
		intName := cgoCString(lm.ReadString(intNamePtr))
		defer cgoFree(unsafe.Pointer(intName))
		extMod := cgoCString(lm.ReadString(extModPtr))
		defer cgoFree(unsafe.Pointer(extMod))
		extBase := cgoCString(lm.ReadString(extBasePtr))
		defer cgoFree(unsafe.Pointer(extBase))
		cgoAddGlobalImport(module, intName, extMod, extBase, globalType, mutable)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenAddMemoryImport", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		intNamePtr := argI(a, 1)
		extModPtr := argI(a, 2)
		extBasePtr := argI(a, 3)
		shared := argBool(a, 4)
		intName := cgoCString(lm.ReadString(intNamePtr))
		defer cgoFree(unsafe.Pointer(intName))
		extMod := cgoCString(lm.ReadString(extModPtr))
		defer cgoFree(unsafe.Pointer(extMod))
		extBase := cgoCString(lm.ReadString(extBasePtr))
		defer cgoFree(unsafe.Pointer(extBase))
		cgoAddMemoryImport(module, intName, extMod, extBase, shared)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenAddTableImport", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		intNamePtr := argI(a, 1)
		extModPtr := argI(a, 2)
		extBasePtr := argI(a, 3)
		intName := cgoCString(lm.ReadString(intNamePtr))
		defer cgoFree(unsafe.Pointer(intName))
		extMod := cgoCString(lm.ReadString(extModPtr))
		defer cgoFree(unsafe.Pointer(extMod))
		extBase := cgoCString(lm.ReadString(extBasePtr))
		defer cgoFree(unsafe.Pointer(extBase))
		cgoAddTableImport(module, intName, extMod, extBase)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenAddTagImport", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		intNamePtr := argI(a, 1)
		extModPtr := argI(a, 2)
		extBasePtr := argI(a, 3)
		params := argU(a, 4)
		results := argU(a, 5)
		intName := cgoCString(lm.ReadString(intNamePtr))
		defer cgoFree(unsafe.Pointer(intName))
		extMod := cgoCString(lm.ReadString(extModPtr))
		defer cgoFree(unsafe.Pointer(extMod))
		extBase := cgoCString(lm.ReadString(extBasePtr))
		defer cgoFree(unsafe.Pointer(extBase))
		cgoAddTagImport(module, intName, extMod, extBase, params, results)
		return retVoid(this.Context())
	})
}

func registerTagImpls(ctx *qjs.Context, lm *LinearMemory) {
	ctx.SetFunc("_BinaryenAddTag", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		params := argU(a, 2)
		results := argU(a, 3)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(this.Context(), cgoAddTag(module, name, params, results))
	})
	ctx.SetFunc("_BinaryenGetTag", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(this.Context(), cgoGetTag(module, name))
	})
	ctx.SetFunc("_BinaryenRemoveTag", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		cgoRemoveTag(module, name)
		return retVoid(this.Context())
	})
	// TagGetName, TagGetParams, TagGetResults — keep as stubs
}

func registerTableImpls(ctx *qjs.Context, lm *LinearMemory) {
	ctx.SetFunc("_BinaryenAddTable", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		initial := argU32(a, 2)
		maximum := argU32(a, 3)
		tableType := argU(a, 4)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(this.Context(), cgoAddTable(module, name, initial, maximum, tableType))
	})
	ctx.SetFunc("_BinaryenRemoveTable", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		cgoRemoveTable(module, name)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenGetNumTables", func(this *qjs.This) (*qjs.Value, error) {
		return retI(this.Context(), cgoGetNumTables(argU(this.Args(), 0)))
	})
	ctx.SetFunc("_BinaryenGetTable", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(this.Context(), cgoGetTable(module, name))
	})
	// TableGetName, TableSetName, TableGetInitial, etc. — keep as stubs
}

func registerMemoryImpls(ctx *qjs.Context, lm *LinearMemory) {
	ctx.SetFunc("_BinaryenSetMemory", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		initial := argU32(a, 1)
		maximum := argU32(a, 2)
		exportNamePtr := argI(a, 3)
		// segments not used in basic compilation, pass 0
		shared := argBool(a, 10)
		memory64 := argBool(a, 11)
		memNamePtr := argI(a, 12)
		var exportName unsafe.Pointer
		if exportNamePtr != 0 {
			exportName = cgoCString(lm.ReadString(exportNamePtr))
			defer cgoFree(unsafe.Pointer(exportName))
		}
		var memName unsafe.Pointer
		if memNamePtr != 0 {
			memName = cgoCString(lm.ReadString(memNamePtr))
			defer cgoFree(unsafe.Pointer(memName))
		}
		cgoSetMemory(module, initial, maximum, exportName, nil, nil, nil, nil, nil, 0, shared, memory64, memName)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenAddDataSegment", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		segNamePtr := argI(a, 1)
		memNamePtr := argI(a, 2)
		passive := argBool(a, 3)
		offset := argU(a, 4)
		dataPtr := argI(a, 5)
		dataLen := argI(a, 6)

		var segName unsafe.Pointer
		if segNamePtr != 0 {
			segName = cgoCString(lm.ReadString(segNamePtr))
			defer cgoFree(unsafe.Pointer(segName))
		}
		var memName unsafe.Pointer
		if memNamePtr != 0 {
			memName = cgoCString(lm.ReadString(memNamePtr))
			defer cgoFree(unsafe.Pointer(memName))
		}
		data := lm.ReadBytes(dataPtr, dataLen)
		cgoAddDataSegment(module, segName, memName, passive, offset, data)
		return retVoid(this.Context())
	})
	ctx.SetFunc("_BinaryenHasMemory", func(this *qjs.This) (*qjs.Value, error) {
		// Use cgoHasMemory if available
		return retBool(this.Context(), cgoHasMemory(argU(this.Args(), 0)))
	})
	ctx.SetFunc("_BinaryenMemoryGetInitial", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		var name unsafe.Pointer
		if namePtr != 0 {
			name = cgoCString(lm.ReadString(namePtr))
			defer cgoFree(unsafe.Pointer(name))
		}
		return retU32(this.Context(), cgoMemoryGetInitial(module, name))
	})
	ctx.SetFunc("_BinaryenMemoryHasMax", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		var name unsafe.Pointer
		if namePtr != 0 {
			name = cgoCString(lm.ReadString(namePtr))
			defer cgoFree(unsafe.Pointer(name))
		}
		return retBool(this.Context(), cgoMemoryHasMax(module, name))
	})
	ctx.SetFunc("_BinaryenMemoryGetMax", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		var name unsafe.Pointer
		if namePtr != 0 {
			name = cgoCString(lm.ReadString(namePtr))
			defer cgoFree(unsafe.Pointer(name))
		}
		return retU32(this.Context(), cgoMemoryGetMax(module, name))
	})
	ctx.SetFunc("_BinaryenMemoryIsShared", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		var name unsafe.Pointer
		if namePtr != 0 {
			name = cgoCString(lm.ReadString(namePtr))
			defer cgoFree(unsafe.Pointer(name))
		}
		return retBool(this.Context(), cgoMemoryIsShared(module, name))
	})
	ctx.SetFunc("_BinaryenMemoryIs64", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		var name unsafe.Pointer
		if namePtr != 0 {
			name = cgoCString(lm.ReadString(namePtr))
			defer cgoFree(unsafe.Pointer(name))
		}
		return retBool(this.Context(), cgoMemoryIs64(module, name))
	})
}

func registerElementSegmentImpls(ctx *qjs.Context, lm *LinearMemory) {
	ctx.SetFunc("_BinaryenAddActiveElementSegment", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		tablePtr := argI(a, 1)
		namePtr := argI(a, 2)
		funcNamesPtr := argI(a, 3)
		numFuncNames := argI(a, 4)
		offset := argU(a, 5)
		table := cgoCString(lm.ReadString(tablePtr))
		defer cgoFree(unsafe.Pointer(table))
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		funcNames := make([]unsafe.Pointer, numFuncNames)
		for i := 0; i < numFuncNames; i++ {
			sp := lm.I32Load(funcNamesPtr + i*4)
			funcNames[i] = cgoCString(lm.ReadString(sp))
		}
		result := cgoAddActiveElementSegment(module, table, name, funcNames, offset)
		for _, fn := range funcNames {
			cgoFree(unsafe.Pointer(fn))
		}
		return retF(this.Context(), result)
	})
	ctx.SetFunc("_BinaryenAddPassiveElementSegment", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		funcNamesPtr := argI(a, 2)
		numFuncNames := argI(a, 3)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		funcNames := make([]unsafe.Pointer, numFuncNames)
		for i := 0; i < numFuncNames; i++ {
			sp := lm.I32Load(funcNamesPtr + i*4)
			funcNames[i] = cgoCString(lm.ReadString(sp))
		}
		result := cgoAddPassiveElementSegment(module, name, funcNames)
		for _, fn := range funcNames {
			cgoFree(unsafe.Pointer(fn))
		}
		return retF(this.Context(), result)
	})
	ctx.SetFunc("_BinaryenAddCustomSection", func(this *qjs.This) (*qjs.Value, error) {
		a := this.Args()
		module := argU(a, 0)
		namePtr := argI(a, 1)
		contentsPtr := argI(a, 2)
		contentsLen := argI(a, 3)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		contents := lm.ReadBytes(contentsPtr, contentsLen)
		cgoAddCustomSection(module, name, contents)
		return retVoid(this.Context())
	})
}

var _ = unsafe.Pointer(nil)
