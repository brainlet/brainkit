package types

import "strings"

// Signature represents a fully resolved function signature.
type Signature struct {
	// Program is the program that created this signature.
	Program ProgramReference
	// ParameterTypes are the parameter types, excluding `this`.
	ParameterTypes []*Type
	// ReturnType is the return type.
	ReturnType *Type
	// ThisType is the `this` type, if an instance signature.
	ThisType *Type
	// RequiredParameters is the number of required parameters excluding `this`.
	RequiredParameters int32
	// HasRest indicates whether the last parameter is a rest parameter.
	HasRest bool
	// ID is the unique id representing this signature.
	ID uint32
	// Type is the respective function type.
	Type *Type
}

// CreateSignature constructs a new signature with deduplication.
func CreateSignature(
	program ProgramReference,
	parameterTypes []*Type,
	returnType *Type,
	thisType *Type,
	requiredParameters int32,
	hasRest bool,
) *Signature {
	if parameterTypes == nil {
		parameterTypes = []*Type{}
	}
	if returnType == nil {
		returnType = TypeVoid
	}
	if requiredParameters < 0 {
		requiredParameters = int32(len(parameterTypes))
	}

	// get the usize type, and the type of the signature
	usizeType := program.GetUsizeType()
	sigType := NewType(
		usizeType.Kind,
		usizeType.Flags&^TypeFlagValue|TypeFlagReference,
		usizeType.Size,
	)

	// calculate the properties
	signatureTypes := program.GetUniqueSignatures()
	nextID := program.GetNextSignatureId()

	// construct the signature and calculate its unique key
	sig := &Signature{
		Program:            program,
		ParameterTypes:     parameterTypes,
		ReturnType:         returnType,
		ThisType:           thisType,
		RequiredParameters: requiredParameters,
		HasRest:            hasRest,
		ID:                 nextID,
		Type:               sigType,
	}
	uniqueKey := sig.ToString(false)

	// check if it exists, and return it
	if existing, ok := signatureTypes[uniqueKey]; ok {
		return existing
	}

	// otherwise increment the program's signature id, set the signature reference, and memoize
	program.SetNextSignatureId(nextID + 1)
	sigType.SignatureReference = sig
	signatureTypes[uniqueKey] = sig
	return sig
}

// ParamRefs returns the Binaryen TypeRef for this signature's parameter types,
// including `this` if present. Creates a tuple type from all parameter refs.
// Ported from: assemblyscript/src/types.ts Signature.paramRefs.
func (s *Signature) ParamRefs() uintptr {
	if CreateTypeFunc == nil {
		panic("types: CreateTypeFunc not wired")
	}
	params := s.ParameterTypes
	offset := 0
	if s.ThisType != nil {
		offset = 1
	}
	refs := make([]uintptr, len(params)+offset)
	if s.ThisType != nil {
		refs[0] = s.ThisType.ToRef()
	}
	for i, p := range params {
		refs[offset+i] = p.ToRef()
	}
	return CreateTypeFunc(refs)
}

// ResultRefs returns the Binaryen TypeRef for this signature's return type.
// Ported from: assemblyscript/src/types.ts Signature.resultRefs.
func (s *Signature) ResultRefs() uintptr {
	return s.ReturnType.ToRef()
}

// Equals tests if this signature equals the specified.
func (s *Signature) Equals(other *Signature) bool {
	// check `this` type
	if s.ThisType != nil {
		if other.ThisType == nil || !s.ThisType.Equals(other.ThisType) {
			return false
		}
	} else if other.ThisType != nil {
		return false
	}

	// check rest parameter
	if s.HasRest != other.HasRest {
		return false
	}

	// check return type
	if !s.ReturnType.Equals(other.ReturnType) {
		return false
	}

	// check parameter types
	selfParams := s.ParameterTypes
	otherParams := other.ParameterTypes
	numParams := len(selfParams)
	if numParams != len(otherParams) {
		return false
	}
	for i := 0; i < numParams; i++ {
		if !selfParams[i].Equals(otherParams[i]) {
			return false
		}
	}
	return true
}

// IsAssignableTo tests if a value of this function type is assignable to a target.
func (s *Signature) IsAssignableTo(target *Signature, checkCompatibleOverride bool) bool {
	thisThisType := s.ThisType
	targetThisType := target.ThisType

	if thisThisType != nil && targetThisType != nil {
		var compatibleThisType bool
		if checkCompatibleOverride {
			compatibleThisType = thisThisType.CanExtendOrImplement(targetThisType)
		} else {
			compatibleThisType = targetThisType.IsAssignableTo(thisThisType, false)
		}
		if !compatibleThisType {
			return false
		}
	} else if thisThisType != nil || targetThisType != nil {
		return false
	}

	// check rest parameter
	if s.HasRest != target.HasRest {
		return false
	}

	// check return type (covariant)
	thisReturnType := s.ReturnType
	targetReturnType := target.ReturnType
	if thisReturnType != targetReturnType && !thisReturnType.IsAssignableTo(targetReturnType, false) {
		return false
	}

	// check parameter types (invariant)
	thisParams := s.ParameterTypes
	targetParams := target.ParameterTypes
	numParams := len(thisParams)
	if numParams != len(targetParams) {
		return false
	}
	for i := 0; i < numParams; i++ {
		if thisParams[i] != targetParams[i] {
			return false
		}
	}
	return true
}

// HasManagedOperands tests if this signature has at least one managed operand.
func (s *Signature) HasManagedOperands() bool {
	if s.ThisType != nil && s.ThisType.IsManaged() {
		return true
	}
	for _, pt := range s.ParameterTypes {
		if pt.IsManaged() {
			return true
		}
	}
	return false
}

// GetManagedOperandIndices gets the indices of all managed operands.
func (s *Signature) GetManagedOperandIndices() []int32 {
	var indices []int32
	index := int32(0)
	if s.ThisType != nil {
		if s.ThisType.IsManaged() {
			indices = append(indices, index)
		}
		index++
	}
	for _, pt := range s.ParameterTypes {
		if pt.IsManaged() {
			indices = append(indices, index)
		}
		index++
	}
	return indices
}

// HasVectorValueOperands tests if this signature has at least one v128 operand.
func (s *Signature) HasVectorValueOperands() bool {
	if s.ThisType != nil && s.ThisType.IsVectorValue() {
		return true
	}
	for _, pt := range s.ParameterTypes {
		if pt.IsVectorValue() {
			return true
		}
	}
	return false
}

// GetVectorValueOperandIndices gets the indices of all v128 operands.
func (s *Signature) GetVectorValueOperandIndices() []int32 {
	var indices []int32
	index := int32(0)
	if s.ThisType != nil {
		if s.ThisType.IsVectorValue() {
			indices = append(indices, index)
		}
		index++
	}
	for _, pt := range s.ParameterTypes {
		if pt.IsVectorValue() {
			indices = append(indices, index)
		}
		index++
	}
	return indices
}

// ToString converts this signature to a string.
func (s *Signature) ToString(validWat bool) string {
	var sb strings.Builder
	if validWat {
		sb.WriteString("%28")
	} else {
		sb.WriteString("(")
	}
	index := 0
	if s.ThisType != nil {
		if validWat {
			sb.WriteString("this:")
		} else {
			sb.WriteString("this: ")
		}
		sb.WriteString(s.ThisType.ToString(validWat))
		index = 1
	}
	params := s.ParameterTypes
	numParams := len(params)
	if numParams > 0 {
		optionalStart := s.RequiredParameters
		restIndex := int32(-1)
		if s.HasRest {
			restIndex = int32(numParams) - 1
		}
		for i := int32(0); i < int32(numParams); i++ {
			if index > 0 {
				if validWat {
					sb.WriteString("%2C")
				} else {
					sb.WriteString(", ")
				}
			}
			if i == restIndex {
				sb.WriteString("...")
			}
			sb.WriteString(params[i].ToString(validWat))
			if i >= optionalStart && i != restIndex {
				sb.WriteString("?")
			}
			index++
		}
	}
	if validWat {
		sb.WriteString("%29=>")
	} else {
		sb.WriteString(") => ")
	}
	sb.WriteString(s.ReturnType.ToString(validWat))
	return sb.String()
}

// String returns a string representation of the signature.
func (s *Signature) String() string {
	return s.ToString(false)
}

// Clone creates a clone of this signature that is safe to modify.
func (s *Signature) Clone(requiredParameters int32, hasRest bool) *Signature {
	cloneParams := make([]*Type, len(s.ParameterTypes))
	copy(cloneParams, s.ParameterTypes)
	return CreateSignature(
		s.Program,
		cloneParams,
		s.ReturnType,
		s.ThisType,
		requiredParameters,
		hasRest,
	)
}

// CloneDefault creates a clone preserving the original's requiredParameters and hasRest.
func (s *Signature) CloneDefault() *Signature {
	return s.Clone(s.RequiredParameters, s.HasRest)
}
