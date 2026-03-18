package asembed

import (
	"unsafe"

	quickjs "github.com/buke/quickjs-go"
)

func registerFunctionImpls(ctx *quickjs.Context, lm *LinearMemory) {
	setFunc(ctx, "_BinaryenAddFunction", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
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
		return retF(c, cgoAddFunction(module, name, params, results, varTypes, body))
	})
	setFunc(ctx, "_BinaryenGetFunction", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		namePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(c, cgoGetFunction(module, name))
	})
	setFunc(ctx, "_BinaryenRemoveFunction", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		namePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		cgoRemoveFunction(module, name)
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenGetNumFunctions", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retI(c, cgoGetNumFunctions(argU(args, 0)))
	})
	setFunc(ctx, "_BinaryenGetFunctionByIndex", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoGetFunctionByIndex(argU(a, 0), argI(a, 1)))
	})
	setFunc(ctx, "_BinaryenFunctionGetName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		cName := cgoFunctionGetName(argU(args, 0))
		if cName == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cName)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})
	setFunc(ctx, "_BinaryenFunctionGetParams", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoFunctionGetParams(argU(args, 0)))
	})
	setFunc(ctx, "_BinaryenFunctionGetResults", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoFunctionGetResults(argU(args, 0)))
	})
	setFunc(ctx, "_BinaryenFunctionGetNumVars", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retI(c, cgoFunctionGetNumVars(argU(args, 0)))
	})
	setFunc(ctx, "_BinaryenFunctionGetVar", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retF(c, cgoFunctionGetVar(argU(a, 0), argI(a, 1)))
	})
	setFunc(ctx, "_BinaryenFunctionGetBody", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retF(c, cgoFunctionGetBody(argU(args, 0)))
	})
	setFunc(ctx, "_BinaryenFunctionSetBody", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cgoFunctionSetBody(argU(a, 0), argU(a, 1))
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenFunctionGetNumLocals", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retI(c, cgoFunctionGetNumLocals(argU(args, 0)))
	})
	setFunc(ctx, "_BinaryenFunctionHasLocalName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		return retBool(c, cgoFunctionHasLocalName(argU(a, 0), argI(a, 1)))
	})
	setFunc(ctx, "_BinaryenFunctionGetLocalName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		cName := cgoFunctionGetLocalName(argU(a, 0), argI(a, 1))
		if cName == nil {
			return retI(c, 0)
		}
		s := cgoGoString(cName)
		ptr := lm.Malloc(len(s) + 1)
		lm.WriteString(ptr, s)
		return retI(c, ptr)
	})
	setFunc(ctx, "_BinaryenFunctionSetLocalName", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		fn := argU(a, 0)
		index := argI(a, 1)
		namePtr := argI(a, 2)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		cgoFunctionSetLocalName(fn, index, name)
		return retVoid(c)
	})
	// FunctionAddVar, FunctionGetType, FunctionSetType, FunctionOptimize,
	// FunctionRunPasses, FunctionSetDebugLocation — keep as stubs for now
}

func registerGlobalImpls(ctx *quickjs.Context, lm *LinearMemory) {
	setFunc(ctx, "_BinaryenAddGlobal", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		namePtr := argI(a, 1)
		typ := argU(a, 2)
		mutable := argBool(a, 3)
		init := argU(a, 4)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		result := cgoAddGlobal(module, name, typ, mutable, init)
		return retF(c, result)
	})
	setFunc(ctx, "_BinaryenGetGlobal", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		namePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(c, cgoGetGlobal(module, name))
	})
	setFunc(ctx, "_BinaryenRemoveGlobal", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		namePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		cgoRemoveGlobal(module, name)
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenGetNumGlobals", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retI(c, cgoGetNumGlobals(argU(args, 0)))
	})
	// GlobalGetName, GlobalGetType, GlobalIsMutable, GlobalGetInitExpr,
	// GetGlobalByIndex — keep as stubs
}

func registerExportImpls(ctx *quickjs.Context, lm *LinearMemory) {
	setFunc(ctx, "_BinaryenAddFunctionExport", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		intNamePtr := argI(a, 1)
		extNamePtr := argI(a, 2)
		intName := cgoCString(lm.ReadString(intNamePtr))
		defer cgoFree(unsafe.Pointer(intName))
		extName := cgoCString(lm.ReadString(extNamePtr))
		defer cgoFree(unsafe.Pointer(extName))
		return retF(c, cgoAddFunctionExport(module, intName, extName))
	})
	setFunc(ctx, "_BinaryenAddTableExport", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		intNamePtr := argI(a, 1)
		extNamePtr := argI(a, 2)
		intName := cgoCString(lm.ReadString(intNamePtr))
		defer cgoFree(unsafe.Pointer(intName))
		extName := cgoCString(lm.ReadString(extNamePtr))
		defer cgoFree(unsafe.Pointer(extName))
		return retF(c, cgoAddTableExport(module, intName, extName))
	})
	setFunc(ctx, "_BinaryenAddMemoryExport", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		intNamePtr := argI(a, 1)
		extNamePtr := argI(a, 2)
		intName := cgoCString(lm.ReadString(intNamePtr))
		defer cgoFree(unsafe.Pointer(intName))
		extName := cgoCString(lm.ReadString(extNamePtr))
		defer cgoFree(unsafe.Pointer(extName))
		return retF(c, cgoAddMemoryExport(module, intName, extName))
	})
	setFunc(ctx, "_BinaryenAddGlobalExport", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		intNamePtr := argI(a, 1)
		extNamePtr := argI(a, 2)
		intName := cgoCString(lm.ReadString(intNamePtr))
		defer cgoFree(unsafe.Pointer(intName))
		extName := cgoCString(lm.ReadString(extNamePtr))
		defer cgoFree(unsafe.Pointer(extName))
		return retF(c, cgoAddGlobalExport(module, intName, extName))
	})
	setFunc(ctx, "_BinaryenAddTagExport", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		intNamePtr := argI(a, 1)
		extNamePtr := argI(a, 2)
		intName := cgoCString(lm.ReadString(intNamePtr))
		defer cgoFree(unsafe.Pointer(intName))
		extName := cgoCString(lm.ReadString(extNamePtr))
		defer cgoFree(unsafe.Pointer(extName))
		return retF(c, cgoAddTagExport(module, intName, extName))
	})
	setFunc(ctx, "_BinaryenGetExport", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		namePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(c, cgoGetExport(module, name))
	})
	setFunc(ctx, "_BinaryenRemoveExport", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		namePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		cgoRemoveExport(module, name)
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenGetNumExports", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retI(c, cgoGetNumExports(argU(args, 0)))
	})
	// ExportGetKind, ExportGetName, ExportGetValue, GetExportByIndex — keep as stubs
}

func registerImportImpls(ctx *quickjs.Context, lm *LinearMemory) {
	setFunc(ctx, "_BinaryenAddFunctionImport", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
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
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenAddGlobalImport", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
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
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenAddMemoryImport", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
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
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenAddTableImport", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
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
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenAddTagImport", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
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
		return retVoid(c)
	})
}

func registerTagImpls(ctx *quickjs.Context, lm *LinearMemory) {
	setFunc(ctx, "_BinaryenAddTag", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		namePtr := argI(a, 1)
		params := argU(a, 2)
		results := argU(a, 3)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(c, cgoAddTag(module, name, params, results))
	})
	setFunc(ctx, "_BinaryenGetTag", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		namePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(c, cgoGetTag(module, name))
	})
	setFunc(ctx, "_BinaryenRemoveTag", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		namePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		cgoRemoveTag(module, name)
		return retVoid(c)
	})
	// TagGetName, TagGetParams, TagGetResults — keep as stubs
}

func registerTableImpls(ctx *quickjs.Context, lm *LinearMemory) {
	setFunc(ctx, "_BinaryenAddTable", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		namePtr := argI(a, 1)
		initial := argU32(a, 2)
		maximum := argU32(a, 3)
		tableType := argU(a, 4)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(c, cgoAddTable(module, name, initial, maximum, tableType))
	})
	setFunc(ctx, "_BinaryenRemoveTable", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		namePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		cgoRemoveTable(module, name)
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenGetNumTables", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		return retI(c, cgoGetNumTables(argU(args, 0)))
	})
	setFunc(ctx, "_BinaryenGetTable", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		namePtr := argI(a, 1)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		return retF(c, cgoGetTable(module, name))
	})
	// TableGetName, TableSetName, TableGetInitial, etc. — keep as stubs
}

func registerMemoryImpls(ctx *quickjs.Context, lm *LinearMemory) {
	setFunc(ctx, "_BinaryenSetMemory", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		initial := argU32(a, 1)
		maximum := argU32(a, 2)
		exportNamePtr := argI(a, 3)
		// arg 4 = segment names array (always 0/NULL from AS compiler)
		cSegsPtr := argI(a, 5)     // pointer to array of segment data pointers
		cPassivePtr := argI(a, 6)  // pointer to array of passive flags (u8)
		cOffsetsPtr := argI(a, 7)  // pointer to array of offset ExpressionRefs
		cSizesPtr := argI(a, 8)    // pointer to array of segment sizes (u32)
		numSegments := argI(a, 9)
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
		// Set memory properties (initial/max/shared/name) without segments.
		cgoSetMemory(module, initial, maximum, exportName, nil, nil, nil, nil, nil, 0, shared, memory64, memName)
		// Add data segments individually. The segment data was allocated by the AS
		// compiler via _malloc + __i32_store8 into LinearMemory. We read it back
		// and pass each segment to BinaryenAddDataSegment via CGo.
		if numSegments > 0 && cSegsPtr != 0 && cOffsetsPtr != 0 && cSizesPtr != 0 {
			for i := 0; i < numSegments; i++ {
				segDataAddr := int(lm.I32LoadPtr(cSegsPtr + i*4))
				size := int(lm.I32LoadPtr(cSizesPtr + i*4))
				offsetRef := lm.I32LoadPtr(cOffsetsPtr + i*4)
				passive := false
				if cPassivePtr != 0 {
					passive = lm.I32Load8U(cPassivePtr+i) != 0
				}
				if segDataAddr != 0 && size > 0 {
					data := lm.ReadBytes(segDataAddr, size)
					cgoAddDataSegment(module, nil, memName, passive, uintptr(offsetRef), data)
				}
			}
		}
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenAddDataSegment", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
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
		return retVoid(c)
	})
	setFunc(ctx, "_BinaryenHasMemory", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		// Use cgoHasMemory if available
		return retBool(c, cgoHasMemory(argU(args, 0)))
	})
	setFunc(ctx, "_BinaryenMemoryGetInitial", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		namePtr := argI(a, 1)
		var name unsafe.Pointer
		if namePtr != 0 {
			name = cgoCString(lm.ReadString(namePtr))
			defer cgoFree(unsafe.Pointer(name))
		}
		return retU32(c, cgoMemoryGetInitial(module, name))
	})
	setFunc(ctx, "_BinaryenMemoryHasMax", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		namePtr := argI(a, 1)
		var name unsafe.Pointer
		if namePtr != 0 {
			name = cgoCString(lm.ReadString(namePtr))
			defer cgoFree(unsafe.Pointer(name))
		}
		return retBool(c, cgoMemoryHasMax(module, name))
	})
	setFunc(ctx, "_BinaryenMemoryGetMax", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		namePtr := argI(a, 1)
		var name unsafe.Pointer
		if namePtr != 0 {
			name = cgoCString(lm.ReadString(namePtr))
			defer cgoFree(unsafe.Pointer(name))
		}
		return retU32(c, cgoMemoryGetMax(module, name))
	})
	setFunc(ctx, "_BinaryenMemoryIsShared", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		namePtr := argI(a, 1)
		var name unsafe.Pointer
		if namePtr != 0 {
			name = cgoCString(lm.ReadString(namePtr))
			defer cgoFree(unsafe.Pointer(name))
		}
		return retBool(c, cgoMemoryIsShared(module, name))
	})
	setFunc(ctx, "_BinaryenMemoryIs64", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		namePtr := argI(a, 1)
		var name unsafe.Pointer
		if namePtr != 0 {
			name = cgoCString(lm.ReadString(namePtr))
			defer cgoFree(unsafe.Pointer(name))
		}
		return retBool(c, cgoMemoryIs64(module, name))
	})
}

func registerElementSegmentImpls(ctx *quickjs.Context, lm *LinearMemory) {
	setFunc(ctx, "_BinaryenAddActiveElementSegment", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
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
		return retF(c, result)
	})
	setFunc(ctx, "_BinaryenAddPassiveElementSegment", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
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
		return retF(c, result)
	})
	setFunc(ctx, "_BinaryenAddCustomSection", func(c *quickjs.Context, args []*quickjs.Value) *quickjs.Value {
		a := args
		module := argU(a, 0)
		namePtr := argI(a, 1)
		contentsPtr := argI(a, 2)
		contentsLen := argI(a, 3)
		name := cgoCString(lm.ReadString(namePtr))
		defer cgoFree(unsafe.Pointer(name))
		contents := lm.ReadBytes(contentsPtr, contentsLen)
		cgoAddCustomSection(module, name, contents)
		return retVoid(c)
	})
}

var _ = unsafe.Pointer(nil)
