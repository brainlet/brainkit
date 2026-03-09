package types

import (
	"math/bits"
	"strings"

	"github.com/brainlet/brainkit/wasm-kit/common"
)

// ClassReference is an interface representing a class type from the program package.
// This avoids a circular dependency between types and program.
type ClassReference interface {
	HasDecorator(flags uint32) bool
	IsAssignableTo(target ClassReference) bool
	HasSubclassAssignableTo(target ClassReference) bool
	InternalName() string
	LookupOverload(kind int32) FunctionReference
}

// FunctionReference is an opaque interface for a Function from the program package.
type FunctionReference interface{}

// ProgramReference is an interface representing the Program from the program package.
type ProgramReference interface {
	GetUsizeType() *Type
	GetFunctionPrototype() interface{}
	GetWrapperClasses() map[*Type]ClassReference
	GetUniqueSignatures() map[string]*Signature
	GetNextSignatureId() uint32
	SetNextSignatureId(id uint32)
	ResolveClass(prototype interface{}, typeArguments []*Type) ClassReference
}

// DecoratorFlagUnmanaged is the decorator flag for @unmanaged classes.
// Matches program.DecoratorFlags.Unmanaged. Set by program package at init.
var DecoratorFlagUnmanaged uint32

// LeastUpperBoundFunc is a function to compute the LUB of two classes.
// Set by the program package at init to avoid circular dependency.
var LeastUpperBoundFunc func(a, b ClassReference) ClassReference

// Type represents a resolved type.
type Type struct {
	Kind               TypeKind
	Flags              TypeFlags
	Size               int32
	ClassRef           ClassReference
	SignatureReference *Signature
	nonNullableType    *Type
	nullableType       *Type
	Ref                uint32 // cached Binaryen type reference
}

// NewType creates a new resolved type.
func NewType(kind TypeKind, flags TypeFlags, size int32) *Type {
	t := &Type{
		Kind:  kind,
		Flags: flags,
		Size:  size,
	}
	if flags&TypeFlagNullable == 0 {
		t.nonNullableType = t
	} else {
		t.nullableType = t
	}
	return t
}

// Is tests if this type has ALL of the specified flags.
func (t *Type) Is(flags TypeFlags) bool {
	return t.Flags&flags == flags
}

// IsAny tests if this type has ANY of the specified flags.
func (t *Type) IsAny(flags TypeFlags) bool {
	return t.Flags&flags != 0
}

// IntType returns the closest int type representing this type.
func (t *Type) IntType() *Type {
	if t == TypeAuto {
		return t
	}
	switch t.Kind {
	case TypeKindBool, TypeKindI32, TypeKindF32:
		return TypeI32
	case TypeKindI8:
		return TypeI8
	case TypeKindI16:
		return TypeI16
	case TypeKindF64, TypeKindI64:
		return TypeI64
	case TypeKindIsize:
		if t.Size == 64 {
			return TypeIsize64
		}
		return TypeIsize32
	case TypeKindU8:
		return TypeU8
	case TypeKindU16:
		return TypeU16
	case TypeKindU32:
		return TypeU32
	case TypeKindU64:
		return TypeU64
	case TypeKindUsize:
		if t.Size == 64 {
			return TypeUsize64
		}
		return TypeUsize32
	default:
		return TypeI32
	}
}

// ExceptVoid substitutes this type with the auto type if void.
func (t *Type) ExceptVoid() *Type {
	if t.Kind == TypeKindVoid {
		return TypeAuto
	}
	return t
}

// ByteSize returns the size in bytes (ceiled div by 8).
func (t *Type) ByteSize() int32 {
	return (t.Size + 7) >> 3
}

// AlignLog2 returns this type's logarithmic alignment in memory.
func (t *Type) AlignLog2() int32 {
	return 31 - int32(bits.LeadingZeros32(uint32(t.ByteSize())))
}

// IsValue tests if this type represents a basic value.
func (t *Type) IsValue() bool {
	return t.Is(TypeFlagValue)
}

// IsIntegerValue tests if this type represents an integer value.
func (t *Type) IsIntegerValue() bool {
	return t.Is(TypeFlagInteger | TypeFlagValue)
}

// IsShortIntegerValue tests if this type is a small (< 32 bits) integer value.
func (t *Type) IsShortIntegerValue() bool {
	return t.Is(TypeFlagShort | TypeFlagInteger | TypeFlagValue)
}

// IsLongIntegerValue tests if this type is a long (> 32 bits) integer value.
func (t *Type) IsLongIntegerValue() bool {
	return t.Is(TypeFlagLong | TypeFlagInteger | TypeFlagValue)
}

// IsSignedIntegerValue tests if this type is a signed integer value.
func (t *Type) IsSignedIntegerValue() bool {
	return t.Is(TypeFlagSigned | TypeFlagInteger | TypeFlagValue)
}

// IsUnsignedIntegerValue tests if this type is an unsigned integer value.
func (t *Type) IsUnsignedIntegerValue() bool {
	return t.Is(TypeFlagUnsigned | TypeFlagInteger | TypeFlagValue)
}

// IsVaryingIntegerValue tests if this type is a varying (in size) integer value.
func (t *Type) IsVaryingIntegerValue() bool {
	return t.Is(TypeFlagVarying | TypeFlagInteger | TypeFlagValue)
}

// IsIntegerInclReference tests if this type is an integer, including references.
func (t *Type) IsIntegerInclReference() bool {
	return t.Is(TypeFlagInteger)
}

// IsFloatValue tests if this type represents a floating point value.
func (t *Type) IsFloatValue() bool {
	return t.Is(TypeFlagFloat | TypeFlagValue)
}

// IsNumericValue tests if this type is numeric (integer or floating point).
func (t *Type) IsNumericValue() bool {
	return t.IsIntegerValue() || t.IsFloatValue()
}

// IsBooleanValue tests if this type represents a boolean value.
func (t *Type) IsBooleanValue() bool {
	return t == TypeBool
}

// IsVectorValue tests if this type represents a vector value.
func (t *Type) IsVectorValue() bool {
	return t.Is(TypeFlagVector | TypeFlagValue)
}

// IsReference tests if this type represents an internal or external reference.
func (t *Type) IsReference() bool {
	return t.Is(TypeFlagReference)
}

// IsNullableReference tests if this type is a nullable reference.
func (t *Type) IsNullableReference() bool {
	return t.Is(TypeFlagNullable | TypeFlagReference)
}

// IsInternalReference tests if this type is an internal reference.
func (t *Type) IsInternalReference() bool {
	return t.Is(TypeFlagInteger | TypeFlagReference)
}

// IsExternalReference tests if this type is an external reference.
func (t *Type) IsExternalReference() bool {
	return t.Is(TypeFlagExternal | TypeFlagReference)
}

// IsNullableExternalReference tests if this type is a nullable external reference.
func (t *Type) IsNullableExternalReference() bool {
	return t.Is(TypeFlagNullable | TypeFlagExternal | TypeFlagReference)
}

// GetClass returns the underlying class of this type, if any.
func (t *Type) GetClass() ClassReference {
	if t.IsInternalReference() {
		return t.ClassRef
	}
	return nil
}

// IsClass tests if this type represents a class.
func (t *Type) IsClass() bool {
	return t.GetClass() != nil
}

// GetClassOrWrapper returns the class, function wrapper, or value wrapper.
func (t *Type) GetClassOrWrapper(program ProgramReference) ClassReference {
	classRef := t.GetClass()
	if classRef != nil {
		return classRef
	}
	sigRef := t.GetSignature()
	if sigRef != nil {
		sigType := sigRef.Type
		wrapper := program.ResolveClass(program.GetFunctionPrototype(), []*Type{sigType})
		return wrapper
	}
	wrapperClasses := program.GetWrapperClasses()
	if wrapper, ok := wrapperClasses[t]; ok {
		return wrapper
	}
	return nil
}

// LookupOverload looks up an operator overload for this type.
func (t *Type) LookupOverload(kind int32, program ProgramReference) FunctionReference {
	classRef := t.GetClassOrWrapper(program)
	if classRef != nil {
		return classRef.LookupOverload(kind)
	}
	return nil
}

// GetSignature returns the function signature of this type, if any.
func (t *Type) GetSignature() *Signature {
	if t.IsInternalReference() {
		return t.SignatureReference
	}
	return nil
}

// IsFunction tests if this type represents a function.
func (t *Type) IsFunction() bool {
	return t.GetSignature() != nil
}

// IsManaged tests if this is a managed type that needs GC hooks.
func (t *Type) IsManaged() bool {
	if t.IsInternalReference() {
		classRef := t.ClassRef
		if classRef != nil {
			return !classRef.HasDecorator(DecoratorFlagUnmanaged)
		}
		return t.SignatureReference != nil
	}
	return false
}

// IsUnmanaged tests if this is an explicitly @unmanaged class type.
func (t *Type) IsUnmanaged() bool {
	classRef := t.ClassRef
	return classRef != nil && classRef.HasDecorator(DecoratorFlagUnmanaged)
}

// IsMemory tests if this type is memory-storable.
func (t *Type) IsMemory() bool {
	switch t.Kind {
	case TypeKindBool, TypeKindI8, TypeKindI16, TypeKindI32, TypeKindI64, TypeKindIsize,
		TypeKindU8, TypeKindU16, TypeKindU32, TypeKindU64, TypeKindUsize,
		TypeKindF32, TypeKindF64, TypeKindV128:
		return true
	}
	return false
}

// NonNullableType returns the corresponding non-nullable type.
func (t *Type) NonNullableType() *Type {
	if t.nonNullableType == nil {
		panic("types: non-nullable type not set")
	}
	return t.nonNullableType
}

// NullableType returns the corresponding nullable type, or nil if not a reference.
func (t *Type) NullableType() *Type {
	if t.IsReference() {
		return t.AsNullable()
	}
	return nil
}

// ComputeSmallIntegerShift computes the sign-extending shift in the target type.
func (t *Type) ComputeSmallIntegerShift(targetType *Type) int32 {
	return targetType.Size - t.Size
}

// ComputeSmallIntegerMask computes the truncating mask in the target type.
func (t *Type) ComputeSmallIntegerMask(targetType *Type) int32 {
	size := t.Size
	if !t.Is(TypeFlagUnsigned) {
		size--
	}
	return int32(^uint32(0) >> uint32(targetType.Size-size))
}

// AsNullable composes the respective nullable type of this type.
func (t *Type) AsNullable() *Type {
	if !t.IsReference() {
		panic("types: AsNullable called on non-reference type")
	}
	nullableType := t.nullableType
	if nullableType == nil {
		if t.IsNullableReference() {
			panic("types: already nullable")
		}
		nullableType = NewType(t.Kind, t.Flags|TypeFlagNullable, t.Size)
		nullableType.ClassRef = t.ClassRef
		nullableType.SignatureReference = t.SignatureReference
		nullableType.nonNullableType = t
		t.nullableType = nullableType
	}
	return nullableType
}

// ToUnsigned returns the unsigned counterpart for signed integer types.
func (t *Type) ToUnsigned() *Type {
	switch t.Kind {
	case TypeKindI8:
		return TypeU8
	case TypeKindI16:
		return TypeU16
	case TypeKindI32:
		return TypeU32
	case TypeKindI64:
		return TypeU64
	case TypeKindIsize:
		if t.Size == 64 {
			return TypeUsize64
		}
		return TypeUsize32
	}
	return t
}

// Equals tests if this type structurally equals the specified type.
func (t *Type) Equals(other *Type) bool {
	if t.Kind != other.Kind {
		return false
	}
	if t.IsReference() {
		return t.ClassRef == other.ClassRef &&
			t.SignatureReference == other.SignatureReference &&
			t.IsNullableReference() == other.IsNullableReference()
	}
	return true
}

// IsAssignableTo tests if a value of this type can be implicitly converted to the target type.
func (t *Type) IsAssignableTo(target *Type, signednessIsRelevant bool) bool {
	if t.IsReference() {
		if target.IsReference() {
			if !t.IsNullableReference() || target.IsNullableReference() {
				if currentClass := t.GetClass(); currentClass != nil {
					if targetClass := target.GetClass(); targetClass != nil {
						return currentClass.IsAssignableTo(targetClass)
					}
				} else if currentFunction := t.GetSignature(); currentFunction != nil {
					if targetFunction := target.GetSignature(); targetFunction != nil {
						return currentFunction.IsAssignableTo(targetFunction, false)
					}
				} else if t.IsExternalReference() {
					if t.Kind == target.Kind ||
						(target.Kind == TypeKindAny && t.Kind != TypeKindExtern) {
						return true
					}
				}
			}
		}
	} else if !target.IsReference() {
		if t.IsIntegerValue() {
			if target.IsIntegerValue() {
				if !signednessIsRelevant ||
					t.IsBooleanValue() ||
					t.IsSignedIntegerValue() == target.IsSignedIntegerValue() {
					return t.Size <= target.Size
				}
			} else if target.Kind == TypeKindF32 {
				return t.Size <= 23
			} else if target.Kind == TypeKindF64 {
				return t.Size <= 52
			}
		} else if t.IsFloatValue() {
			if target.IsFloatValue() {
				return t.Size <= target.Size
			}
		} else if t.IsVectorValue() {
			if target.IsVectorValue() {
				return t.Size == target.Size
			}
		}
	}
	return false
}

// IsStrictlyAssignableTo tests assignability without implicit widening.
func (t *Type) IsStrictlyAssignableTo(target *Type, signednessIsRelevant bool) bool {
	if t.IsReference() {
		return t.IsAssignableTo(target, false)
	}
	if target.IsReference() {
		return false
	}
	if t.IsIntegerValue() {
		return target.IsIntegerValue() && target.Size == t.Size && (!signednessIsRelevant ||
			t.IsSignedIntegerValue() == target.IsSignedIntegerValue())
	}
	return t.Kind == target.Kind
}

// HasSubtypeAssignableTo tests if this type has a subtype assignable to the target.
func (t *Type) HasSubtypeAssignableTo(target *Type) bool {
	thisClass := t.GetClass()
	targetClass := target.GetClass()
	if thisClass == nil || targetClass == nil {
		return false
	}
	return thisClass.HasSubclassAssignableTo(targetClass)
}

// IsChangeableTo tests if this type can be changed to the target using changetype.
func (t *Type) IsChangeableTo(target *Type) bool {
	if t.Is(TypeFlagInteger) && target.Is(TypeFlagInteger) {
		size := t.Size
		return size == target.Size && (size >= 32 ||
			t.Is(TypeFlagSigned) == target.Is(TypeFlagSigned))
	}
	return t.Kind == target.Kind
}

// CanExtendOrImplement tests if this type can extend or implement the given base type.
func (t *Type) CanExtendOrImplement(base *Type) bool {
	thisClass := t.GetClass()
	baseClass := base.GetClass()
	if thisClass == nil || baseClass == nil {
		return false
	}
	if t.IsManaged() != base.IsManaged() {
		return false
	}
	if t.IsInternalReference() {
		if !base.IsInternalReference() {
			return false
		}
	} else if t.IsExternalReference() {
		if !base.IsExternalReference() {
			return false
		}
	} else {
		return false
	}
	return true
}

// CommonType computes the common type of a binary-like expression, if any.
func CommonType(left, right, contextualType *Type, signednessIsRelevant bool) *Type {
	if contextualType == nil {
		contextualType = TypeAuto
	}
	if left.IsInternalReference() {
		if !right.IsInternalReference() {
			return nil
		}
		if contextualType != TypeVoid && left.IsAssignableTo(contextualType, false) && right.IsAssignableTo(contextualType, false) {
			return contextualType
		}
		leftClass := left.GetClass()
		rightClass := right.GetClass()
		if leftClass != nil && rightClass != nil && LeastUpperBoundFunc != nil {
			lubClass := LeastUpperBoundFunc(leftClass, rightClass)
			if lubClass != nil {
				// Get the type from the LUB class
				// Note: This requires the class to expose its Type.
				// For now we return contextualType as a reasonable approximation.
				// The full implementation requires Class.type which is in the program package.
				return contextualType
			}
		}
	} else if right.IsInternalReference() {
		return nil
	}
	if right.IsAssignableTo(left, signednessIsRelevant) {
		return left
	}
	if left.IsAssignableTo(right, signednessIsRelevant) {
		return right
	}
	return nil
}

// KindToString converts this type's kind to a string.
func (t *Type) KindToString() string {
	switch t.Kind {
	case TypeKindBool:
		return common.CommonNameBool
	case TypeKindI8:
		return common.CommonNameI8
	case TypeKindI16:
		return common.CommonNameI16
	case TypeKindI32:
		return common.CommonNameI32
	case TypeKindI64:
		return common.CommonNameI64
	case TypeKindIsize:
		return common.CommonNameIsize
	case TypeKindU8:
		return common.CommonNameU8
	case TypeKindU16:
		return common.CommonNameU16
	case TypeKindU32:
		return common.CommonNameU32
	case TypeKindU64:
		return common.CommonNameU64
	case TypeKindUsize:
		return common.CommonNameUsize
	case TypeKindF32:
		return common.CommonNameF32
	case TypeKindF64:
		return common.CommonNameF64
	case TypeKindV128:
		return common.CommonNameV128
	case TypeKindFunc:
		return common.CommonNameRefFunc
	case TypeKindExtern:
		return common.CommonNameRefExtern
	case TypeKindAny:
		return common.CommonNameRefAny
	case TypeKindEq:
		return common.CommonNameRefEq
	case TypeKindStruct:
		return common.CommonNameRefStruct
	case TypeKindArray:
		return common.CommonNameRefArray
	case TypeKindI31:
		return common.CommonNameRefI31
	case TypeKindString:
		return common.CommonNameRefString
	case TypeKindStringviewWTF8:
		return common.CommonNameRefStringviewWtf8
	case TypeKindStringviewWTF16:
		return common.CommonNameRefStringviewWtf16
	case TypeKindStringviewIter:
		return common.CommonNameRefStringviewIter
	case TypeKindVoid:
		return common.CommonNameVoid
	default:
		panic("types: unknown type kind")
	}
}

// String converts this type to a string.
func (t *Type) String() string {
	return t.ToString(false)
}

// ToString converts this type to a string representation.
func (t *Type) ToString(validWat bool) string {
	nullablePostfix := " | null"
	if validWat {
		nullablePostfix = "|null"
	}
	if t.IsReference() {
		classRef := t.GetClass()
		if classRef != nil {
			if t.IsNullableReference() {
				return classRef.InternalName() + nullablePostfix
			}
			return classRef.InternalName()
		}
		sigRef := t.GetSignature()
		if sigRef != nil {
			if t.IsNullableReference() {
				return "(" + sigRef.ToString(validWat) + ")" + nullablePostfix
			}
			return sigRef.ToString(validWat)
		}
		if t.IsNullableReference() {
			return t.KindToString() + nullablePostfix
		}
		return t.KindToString()
	}
	if t == TypeAuto {
		return "auto"
	}
	return t.KindToString()
}

// --- Built-in type instances ---

var (
	TypeI8  = NewType(TypeKindI8, TypeFlagSigned|TypeFlagShort|TypeFlagInteger|TypeFlagValue, 8)
	TypeI16 = NewType(TypeKindI16, TypeFlagSigned|TypeFlagShort|TypeFlagInteger|TypeFlagValue, 16)
	TypeI32 = NewType(TypeKindI32, TypeFlagSigned|TypeFlagInteger|TypeFlagValue, 32)
	TypeI64 = NewType(TypeKindI64, TypeFlagSigned|TypeFlagLong|TypeFlagInteger|TypeFlagValue, 64)

	TypeIsize32 = NewType(TypeKindIsize, TypeFlagSigned|TypeFlagInteger|TypeFlagVarying|TypeFlagValue, 32)
	TypeIsize64 = NewType(TypeKindIsize, TypeFlagSigned|TypeFlagLong|TypeFlagInteger|TypeFlagVarying|TypeFlagValue, 64)

	TypeU8  = NewType(TypeKindU8, TypeFlagUnsigned|TypeFlagShort|TypeFlagInteger|TypeFlagValue, 8)
	TypeU16 = NewType(TypeKindU16, TypeFlagUnsigned|TypeFlagShort|TypeFlagInteger|TypeFlagValue, 16)
	TypeU32 = NewType(TypeKindU32, TypeFlagUnsigned|TypeFlagInteger|TypeFlagValue, 32)
	TypeU64 = NewType(TypeKindU64, TypeFlagUnsigned|TypeFlagLong|TypeFlagInteger|TypeFlagValue, 64)

	TypeUsize32 = NewType(TypeKindUsize, TypeFlagUnsigned|TypeFlagInteger|TypeFlagVarying|TypeFlagValue, 32)
	TypeUsize64 = NewType(TypeKindUsize, TypeFlagUnsigned|TypeFlagLong|TypeFlagInteger|TypeFlagVarying|TypeFlagValue, 64)

	TypeBool = NewType(TypeKindBool, TypeFlagUnsigned|TypeFlagShort|TypeFlagInteger|TypeFlagValue, 1)

	TypeF32 = NewType(TypeKindF32, TypeFlagSigned|TypeFlagFloat|TypeFlagValue, 32)
	TypeF64 = NewType(TypeKindF64, TypeFlagSigned|TypeFlagLong|TypeFlagFloat|TypeFlagValue, 64)

	TypeV128 = NewType(TypeKindV128, TypeFlagVector|TypeFlagValue, 128)

	TypeFunc             = NewType(TypeKindFunc, TypeFlagExternal|TypeFlagReference, 0)
	TypeExtern           = NewType(TypeKindExtern, TypeFlagExternal|TypeFlagReference, 0)
	TypeAnyRef           = NewType(TypeKindAny, TypeFlagExternal|TypeFlagReference, 0)
	TypeEq               = NewType(TypeKindEq, TypeFlagExternal|TypeFlagReference, 0)
	TypeStructRef        = NewType(TypeKindStruct, TypeFlagExternal|TypeFlagReference, 0)
	TypeArrayRef         = NewType(TypeKindArray, TypeFlagExternal|TypeFlagReference, 0)
	TypeI31              = NewType(TypeKindI31, TypeFlagExternal|TypeFlagReference, 0)
	TypeStringRef        = NewType(TypeKindString, TypeFlagExternal|TypeFlagReference, 0)
	TypeStringviewWTF8   = NewType(TypeKindStringviewWTF8, TypeFlagExternal|TypeFlagReference, 0)
	TypeStringviewWTF16  = NewType(TypeKindStringviewWTF16, TypeFlagExternal|TypeFlagReference, 0)
	TypeStringviewIter   = NewType(TypeKindStringviewIter, TypeFlagExternal|TypeFlagReference, 0)

	TypeVoid = NewType(TypeKindVoid, TypeFlagNone, 0)

	// TypeAuto is an alias of i32 used as a sentinel for type inference.
	// It is a distinct instance (TypeAuto != TypeI32).
	TypeAuto = NewType(TypeKindI32, TypeFlagSigned|TypeFlagInteger|TypeFlagValue, 32)
)

// TypesToString converts an array of types to a combined string representation.
func TypesToString(types []*Type) string {
	if len(types) == 0 {
		return ""
	}
	parts := make([]string, len(types))
	for i, t := range types {
		parts[i] = t.ToString(true)
	}
	return strings.Join(parts, ",")
}
