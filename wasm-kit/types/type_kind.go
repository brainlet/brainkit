package types

// TypeKind indicates the kind of a type.
type TypeKind int32

const (
	TypeKindBool TypeKind = iota

	// signed integers
	TypeKindI8
	TypeKindI16
	TypeKindI32
	TypeKindI64
	TypeKindIsize

	// unsigned integers
	TypeKindU8
	TypeKindU16
	TypeKindU32
	TypeKindU64
	TypeKindUsize

	// floats
	TypeKindF32
	TypeKindF64

	// vectors
	TypeKindV128

	// references (keep in same order as Binaryen)
	TypeKindExtern
	TypeKindFunc
	TypeKindAny
	TypeKindEq
	TypeKindStruct
	TypeKindArray
	TypeKindI31
	TypeKindString
	TypeKindStringviewWTF8
	TypeKindStringviewWTF16
	TypeKindStringviewIter

	// other
	TypeKindVoid
)

// TypeFlags indicates capabilities of a type.
type TypeFlags uint32

const (
	TypeFlagNone      TypeFlags = 0
	TypeFlagSigned    TypeFlags = 1 << 0
	TypeFlagUnsigned  TypeFlags = 1 << 1
	TypeFlagInteger   TypeFlags = 1 << 2
	TypeFlagFloat     TypeFlags = 1 << 3
	TypeFlagVarying   TypeFlags = 1 << 4
	TypeFlagShort     TypeFlags = 1 << 5
	TypeFlagLong      TypeFlags = 1 << 6
	TypeFlagValue     TypeFlags = 1 << 7
	TypeFlagReference TypeFlags = 1 << 8
	TypeFlagNullable  TypeFlags = 1 << 9
	TypeFlagVector    TypeFlags = 1 << 10
	TypeFlagExternal  TypeFlags = 1 << 11
	TypeFlagClass     TypeFlags = 1 << 12
	TypeFlagFunction  TypeFlags = 1 << 13
)
