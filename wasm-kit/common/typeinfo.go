package common

type TypeinfoFlags uint32

const (
	TypeinfoFlagsNONE            TypeinfoFlags = 0
	TypeinfoFlagsARRAYBUFFERVIEW TypeinfoFlags = 1 << 0
	TypeinfoFlagsARRAY           TypeinfoFlags = 1 << 1
	TypeinfoFlagsSTATICARRAY     TypeinfoFlags = 1 << 2
	TypeinfoFlagsSET             TypeinfoFlags = 1 << 3
	TypeinfoFlagsMAP             TypeinfoFlags = 1 << 4
	TypeinfoFlagsPOINTERFREE     TypeinfoFlags = 1 << 5
	TypeinfoFlagsVALUE_ALIGN_0   TypeinfoFlags = 1 << 6
	TypeinfoFlagsVALUE_ALIGN_1   TypeinfoFlags = 1 << 7
	TypeinfoFlagsVALUE_ALIGN_2   TypeinfoFlags = 1 << 8
	TypeinfoFlagsVALUE_ALIGN_3   TypeinfoFlags = 1 << 9
	TypeinfoFlagsVALUE_ALIGN_4   TypeinfoFlags = 1 << 10
	TypeinfoFlagsVALUE_SIGNED    TypeinfoFlags = 1 << 11
	TypeinfoFlagsVALUE_FLOAT     TypeinfoFlags = 1 << 12
	TypeinfoFlagsVALUE_NULLABLE  TypeinfoFlags = 1 << 13
	TypeinfoFlagsVALUE_MANAGED   TypeinfoFlags = 1 << 14
	TypeinfoFlagsKEY_ALIGN_0     TypeinfoFlags = 1 << 15
	TypeinfoFlagsKEY_ALIGN_1     TypeinfoFlags = 1 << 16
	TypeinfoFlagsKEY_ALIGN_2     TypeinfoFlags = 1 << 17
	TypeinfoFlagsKEY_ALIGN_3     TypeinfoFlags = 1 << 18
	TypeinfoFlagsKEY_ALIGN_4     TypeinfoFlags = 1 << 19
	TypeinfoFlagsKEY_SIGNED      TypeinfoFlags = 1 << 20
	TypeinfoFlagsKEY_FLOAT       TypeinfoFlags = 1 << 21
	TypeinfoFlagsKEY_NULLABLE    TypeinfoFlags = 1 << 22
	TypeinfoFlagsKEY_MANAGED     TypeinfoFlags = 1 << 23
)

type Typeinfo struct {
	Flags TypeinfoFlags
}
