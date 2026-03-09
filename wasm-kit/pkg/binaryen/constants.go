package binaryen

/*
#include "binaryen-c.h"
*/
import "C"

// ---------------------------------------------------------------------------
// Unary operators — Int32
// ---------------------------------------------------------------------------

func ClzInt32() Op     { return Op(C.BinaryenClzInt32()) }
func CtzInt32() Op     { return Op(C.BinaryenCtzInt32()) }
func PopcntInt32() Op  { return Op(C.BinaryenPopcntInt32()) }
func EqZInt32() Op     { return Op(C.BinaryenEqZInt32()) }

// ---------------------------------------------------------------------------
// Unary operators — Float32
// ---------------------------------------------------------------------------

func NegFloat32() Op     { return Op(C.BinaryenNegFloat32()) }
func AbsFloat32() Op     { return Op(C.BinaryenAbsFloat32()) }
func CeilFloat32() Op    { return Op(C.BinaryenCeilFloat32()) }
func FloorFloat32() Op   { return Op(C.BinaryenFloorFloat32()) }
func TruncFloat32() Op   { return Op(C.BinaryenTruncFloat32()) }
func NearestFloat32() Op { return Op(C.BinaryenNearestFloat32()) }
func SqrtFloat32() Op    { return Op(C.BinaryenSqrtFloat32()) }

// ---------------------------------------------------------------------------
// Unary operators — Int64
// ---------------------------------------------------------------------------

func ClzInt64() Op     { return Op(C.BinaryenClzInt64()) }
func CtzInt64() Op     { return Op(C.BinaryenCtzInt64()) }
func PopcntInt64() Op  { return Op(C.BinaryenPopcntInt64()) }
func EqZInt64() Op     { return Op(C.BinaryenEqZInt64()) }

// ---------------------------------------------------------------------------
// Unary operators — Float64
// ---------------------------------------------------------------------------

func NegFloat64() Op     { return Op(C.BinaryenNegFloat64()) }
func AbsFloat64() Op     { return Op(C.BinaryenAbsFloat64()) }
func CeilFloat64() Op    { return Op(C.BinaryenCeilFloat64()) }
func FloorFloat64() Op   { return Op(C.BinaryenFloorFloat64()) }
func TruncFloat64() Op   { return Op(C.BinaryenTruncFloat64()) }
func NearestFloat64() Op { return Op(C.BinaryenNearestFloat64()) }
func SqrtFloat64() Op    { return Op(C.BinaryenSqrtFloat64()) }

// ---------------------------------------------------------------------------
// Unary operators — conversions and casts
// ---------------------------------------------------------------------------

func ExtendSInt32() Op  { return Op(C.BinaryenExtendSInt32()) }
func ExtendUInt32() Op  { return Op(C.BinaryenExtendUInt32()) }
func WrapInt64() Op     { return Op(C.BinaryenWrapInt64()) }

func TruncSFloat32ToInt32() Op { return Op(C.BinaryenTruncSFloat32ToInt32()) }
func TruncSFloat32ToInt64() Op { return Op(C.BinaryenTruncSFloat32ToInt64()) }
func TruncUFloat32ToInt32() Op { return Op(C.BinaryenTruncUFloat32ToInt32()) }
func TruncUFloat32ToInt64() Op { return Op(C.BinaryenTruncUFloat32ToInt64()) }
func TruncSFloat64ToInt32() Op { return Op(C.BinaryenTruncSFloat64ToInt32()) }
func TruncSFloat64ToInt64() Op { return Op(C.BinaryenTruncSFloat64ToInt64()) }
func TruncUFloat64ToInt32() Op { return Op(C.BinaryenTruncUFloat64ToInt32()) }
func TruncUFloat64ToInt64() Op { return Op(C.BinaryenTruncUFloat64ToInt64()) }

func ReinterpretFloat32() Op { return Op(C.BinaryenReinterpretFloat32()) }
func ReinterpretFloat64() Op { return Op(C.BinaryenReinterpretFloat64()) }

func ConvertSInt32ToFloat32() Op { return Op(C.BinaryenConvertSInt32ToFloat32()) }
func ConvertSInt32ToFloat64() Op { return Op(C.BinaryenConvertSInt32ToFloat64()) }
func ConvertUInt32ToFloat32() Op { return Op(C.BinaryenConvertUInt32ToFloat32()) }
func ConvertUInt32ToFloat64() Op { return Op(C.BinaryenConvertUInt32ToFloat64()) }
func ConvertSInt64ToFloat32() Op { return Op(C.BinaryenConvertSInt64ToFloat32()) }
func ConvertSInt64ToFloat64() Op { return Op(C.BinaryenConvertSInt64ToFloat64()) }
func ConvertUInt64ToFloat32() Op { return Op(C.BinaryenConvertUInt64ToFloat32()) }
func ConvertUInt64ToFloat64() Op { return Op(C.BinaryenConvertUInt64ToFloat64()) }

func PromoteFloat32() Op   { return Op(C.BinaryenPromoteFloat32()) }
func DemoteFloat64() Op    { return Op(C.BinaryenDemoteFloat64()) }
func ReinterpretInt32() Op { return Op(C.BinaryenReinterpretInt32()) }
func ReinterpretInt64() Op { return Op(C.BinaryenReinterpretInt64()) }

// ---------------------------------------------------------------------------
// Unary operators — sign extension
// ---------------------------------------------------------------------------

func ExtendS8Int32() Op  { return Op(C.BinaryenExtendS8Int32()) }
func ExtendS16Int32() Op { return Op(C.BinaryenExtendS16Int32()) }
func ExtendS8Int64() Op  { return Op(C.BinaryenExtendS8Int64()) }
func ExtendS16Int64() Op { return Op(C.BinaryenExtendS16Int64()) }
func ExtendS32Int64() Op { return Op(C.BinaryenExtendS32Int64()) }

// ---------------------------------------------------------------------------
// Binary operators — Int32
// ---------------------------------------------------------------------------

func AddInt32() Op  { return Op(C.BinaryenAddInt32()) }
func SubInt32() Op  { return Op(C.BinaryenSubInt32()) }
func MulInt32() Op  { return Op(C.BinaryenMulInt32()) }
func DivSInt32() Op { return Op(C.BinaryenDivSInt32()) }
func DivUInt32() Op { return Op(C.BinaryenDivUInt32()) }
func RemSInt32() Op { return Op(C.BinaryenRemSInt32()) }
func RemUInt32() Op { return Op(C.BinaryenRemUInt32()) }
func AndInt32() Op  { return Op(C.BinaryenAndInt32()) }
func OrInt32() Op   { return Op(C.BinaryenOrInt32()) }
func XorInt32() Op  { return Op(C.BinaryenXorInt32()) }
func ShlInt32() Op  { return Op(C.BinaryenShlInt32()) }
func ShrUInt32() Op { return Op(C.BinaryenShrUInt32()) }
func ShrSInt32() Op { return Op(C.BinaryenShrSInt32()) }
func RotLInt32() Op { return Op(C.BinaryenRotLInt32()) }
func RotRInt32() Op { return Op(C.BinaryenRotRInt32()) }
func EqInt32() Op   { return Op(C.BinaryenEqInt32()) }
func NeInt32() Op   { return Op(C.BinaryenNeInt32()) }
func LtSInt32() Op  { return Op(C.BinaryenLtSInt32()) }
func LtUInt32() Op  { return Op(C.BinaryenLtUInt32()) }
func LeSInt32() Op  { return Op(C.BinaryenLeSInt32()) }
func LeUInt32() Op  { return Op(C.BinaryenLeUInt32()) }
func GtSInt32() Op  { return Op(C.BinaryenGtSInt32()) }
func GtUInt32() Op  { return Op(C.BinaryenGtUInt32()) }
func GeSInt32() Op  { return Op(C.BinaryenGeSInt32()) }
func GeUInt32() Op  { return Op(C.BinaryenGeUInt32()) }

// ---------------------------------------------------------------------------
// Binary operators — Int64
// ---------------------------------------------------------------------------

func AddInt64() Op  { return Op(C.BinaryenAddInt64()) }
func SubInt64() Op  { return Op(C.BinaryenSubInt64()) }
func MulInt64() Op  { return Op(C.BinaryenMulInt64()) }
func DivSInt64() Op { return Op(C.BinaryenDivSInt64()) }
func DivUInt64() Op { return Op(C.BinaryenDivUInt64()) }
func RemSInt64() Op { return Op(C.BinaryenRemSInt64()) }
func RemUInt64() Op { return Op(C.BinaryenRemUInt64()) }
func AndInt64() Op  { return Op(C.BinaryenAndInt64()) }
func OrInt64() Op   { return Op(C.BinaryenOrInt64()) }
func XorInt64() Op  { return Op(C.BinaryenXorInt64()) }
func ShlInt64() Op  { return Op(C.BinaryenShlInt64()) }
func ShrUInt64() Op { return Op(C.BinaryenShrUInt64()) }
func ShrSInt64() Op { return Op(C.BinaryenShrSInt64()) }
func RotLInt64() Op { return Op(C.BinaryenRotLInt64()) }
func RotRInt64() Op { return Op(C.BinaryenRotRInt64()) }
func EqInt64() Op   { return Op(C.BinaryenEqInt64()) }
func NeInt64() Op   { return Op(C.BinaryenNeInt64()) }
func LtSInt64() Op  { return Op(C.BinaryenLtSInt64()) }
func LtUInt64() Op  { return Op(C.BinaryenLtUInt64()) }
func LeSInt64() Op  { return Op(C.BinaryenLeSInt64()) }
func LeUInt64() Op  { return Op(C.BinaryenLeUInt64()) }
func GtSInt64() Op  { return Op(C.BinaryenGtSInt64()) }
func GtUInt64() Op  { return Op(C.BinaryenGtUInt64()) }
func GeSInt64() Op  { return Op(C.BinaryenGeSInt64()) }
func GeUInt64() Op  { return Op(C.BinaryenGeUInt64()) }

// ---------------------------------------------------------------------------
// Binary operators — Float32
// ---------------------------------------------------------------------------

func AddFloat32() Op      { return Op(C.BinaryenAddFloat32()) }
func SubFloat32() Op      { return Op(C.BinaryenSubFloat32()) }
func MulFloat32() Op      { return Op(C.BinaryenMulFloat32()) }
func DivFloat32() Op      { return Op(C.BinaryenDivFloat32()) }
func CopySignFloat32() Op { return Op(C.BinaryenCopySignFloat32()) }
func MinFloat32() Op      { return Op(C.BinaryenMinFloat32()) }
func MaxFloat32() Op      { return Op(C.BinaryenMaxFloat32()) }
func EqFloat32() Op       { return Op(C.BinaryenEqFloat32()) }
func NeFloat32() Op       { return Op(C.BinaryenNeFloat32()) }
func LtFloat32() Op       { return Op(C.BinaryenLtFloat32()) }
func LeFloat32() Op       { return Op(C.BinaryenLeFloat32()) }
func GtFloat32() Op       { return Op(C.BinaryenGtFloat32()) }
func GeFloat32() Op       { return Op(C.BinaryenGeFloat32()) }

// ---------------------------------------------------------------------------
// Binary operators — Float64
// ---------------------------------------------------------------------------

func AddFloat64() Op      { return Op(C.BinaryenAddFloat64()) }
func SubFloat64() Op      { return Op(C.BinaryenSubFloat64()) }
func MulFloat64() Op      { return Op(C.BinaryenMulFloat64()) }
func DivFloat64() Op      { return Op(C.BinaryenDivFloat64()) }
func CopySignFloat64() Op { return Op(C.BinaryenCopySignFloat64()) }
func MinFloat64() Op      { return Op(C.BinaryenMinFloat64()) }
func MaxFloat64() Op      { return Op(C.BinaryenMaxFloat64()) }
func EqFloat64() Op       { return Op(C.BinaryenEqFloat64()) }
func NeFloat64() Op       { return Op(C.BinaryenNeFloat64()) }
func LtFloat64() Op       { return Op(C.BinaryenLtFloat64()) }
func LeFloat64() Op       { return Op(C.BinaryenLeFloat64()) }
func GtFloat64() Op       { return Op(C.BinaryenGtFloat64()) }
func GeFloat64() Op       { return Op(C.BinaryenGeFloat64()) }

// ---------------------------------------------------------------------------
// Atomic RMW operators
// ---------------------------------------------------------------------------

func AtomicRMWAdd() Op  { return Op(C.BinaryenAtomicRMWAdd()) }
func AtomicRMWSub() Op  { return Op(C.BinaryenAtomicRMWSub()) }
func AtomicRMWAnd() Op  { return Op(C.BinaryenAtomicRMWAnd()) }
func AtomicRMWOr() Op   { return Op(C.BinaryenAtomicRMWOr()) }
func AtomicRMWXor() Op  { return Op(C.BinaryenAtomicRMWXor()) }
func AtomicRMWXchg() Op { return Op(C.BinaryenAtomicRMWXchg()) }

// ---------------------------------------------------------------------------
// Saturating truncation operators
// ---------------------------------------------------------------------------

func TruncSatSFloat32ToInt32() Op { return Op(C.BinaryenTruncSatSFloat32ToInt32()) }
func TruncSatSFloat32ToInt64() Op { return Op(C.BinaryenTruncSatSFloat32ToInt64()) }
func TruncSatUFloat32ToInt32() Op { return Op(C.BinaryenTruncSatUFloat32ToInt32()) }
func TruncSatUFloat32ToInt64() Op { return Op(C.BinaryenTruncSatUFloat32ToInt64()) }
func TruncSatSFloat64ToInt32() Op { return Op(C.BinaryenTruncSatSFloat64ToInt32()) }
func TruncSatSFloat64ToInt64() Op { return Op(C.BinaryenTruncSatSFloat64ToInt64()) }
func TruncSatUFloat64ToInt32() Op { return Op(C.BinaryenTruncSatUFloat64ToInt32()) }
func TruncSatUFloat64ToInt64() Op { return Op(C.BinaryenTruncSatUFloat64ToInt64()) }

// ---------------------------------------------------------------------------
// SIMD — splat, extract, replace lane
// ---------------------------------------------------------------------------

func SplatVecI8x16() Op       { return Op(C.BinaryenSplatVecI8x16()) }
func ExtractLaneSVecI8x16() Op { return Op(C.BinaryenExtractLaneSVecI8x16()) }
func ExtractLaneUVecI8x16() Op { return Op(C.BinaryenExtractLaneUVecI8x16()) }
func ReplaceLaneVecI8x16() Op { return Op(C.BinaryenReplaceLaneVecI8x16()) }

func SplatVecI16x8() Op       { return Op(C.BinaryenSplatVecI16x8()) }
func ExtractLaneSVecI16x8() Op { return Op(C.BinaryenExtractLaneSVecI16x8()) }
func ExtractLaneUVecI16x8() Op { return Op(C.BinaryenExtractLaneUVecI16x8()) }
func ReplaceLaneVecI16x8() Op { return Op(C.BinaryenReplaceLaneVecI16x8()) }

func SplatVecI32x4() Op      { return Op(C.BinaryenSplatVecI32x4()) }
func ExtractLaneVecI32x4() Op { return Op(C.BinaryenExtractLaneVecI32x4()) }
func ReplaceLaneVecI32x4() Op { return Op(C.BinaryenReplaceLaneVecI32x4()) }

func SplatVecI64x2() Op      { return Op(C.BinaryenSplatVecI64x2()) }
func ExtractLaneVecI64x2() Op { return Op(C.BinaryenExtractLaneVecI64x2()) }
func ReplaceLaneVecI64x2() Op { return Op(C.BinaryenReplaceLaneVecI64x2()) }

func SplatVecF32x4() Op      { return Op(C.BinaryenSplatVecF32x4()) }
func ExtractLaneVecF32x4() Op { return Op(C.BinaryenExtractLaneVecF32x4()) }
func ReplaceLaneVecF32x4() Op { return Op(C.BinaryenReplaceLaneVecF32x4()) }

func SplatVecF64x2() Op      { return Op(C.BinaryenSplatVecF64x2()) }
func ExtractLaneVecF64x2() Op { return Op(C.BinaryenExtractLaneVecF64x2()) }
func ReplaceLaneVecF64x2() Op { return Op(C.BinaryenReplaceLaneVecF64x2()) }

// ---------------------------------------------------------------------------
// SIMD — comparison operators
// ---------------------------------------------------------------------------

func EqVecI8x16() Op  { return Op(C.BinaryenEqVecI8x16()) }
func NeVecI8x16() Op  { return Op(C.BinaryenNeVecI8x16()) }
func LtSVecI8x16() Op { return Op(C.BinaryenLtSVecI8x16()) }
func LtUVecI8x16() Op { return Op(C.BinaryenLtUVecI8x16()) }
func GtSVecI8x16() Op { return Op(C.BinaryenGtSVecI8x16()) }
func GtUVecI8x16() Op { return Op(C.BinaryenGtUVecI8x16()) }
func LeSVecI8x16() Op { return Op(C.BinaryenLeSVecI8x16()) }
func LeUVecI8x16() Op { return Op(C.BinaryenLeUVecI8x16()) }
func GeSVecI8x16() Op { return Op(C.BinaryenGeSVecI8x16()) }
func GeUVecI8x16() Op { return Op(C.BinaryenGeUVecI8x16()) }

func EqVecI16x8() Op  { return Op(C.BinaryenEqVecI16x8()) }
func NeVecI16x8() Op  { return Op(C.BinaryenNeVecI16x8()) }
func LtSVecI16x8() Op { return Op(C.BinaryenLtSVecI16x8()) }
func LtUVecI16x8() Op { return Op(C.BinaryenLtUVecI16x8()) }
func GtSVecI16x8() Op { return Op(C.BinaryenGtSVecI16x8()) }
func GtUVecI16x8() Op { return Op(C.BinaryenGtUVecI16x8()) }
func LeSVecI16x8() Op { return Op(C.BinaryenLeSVecI16x8()) }
func LeUVecI16x8() Op { return Op(C.BinaryenLeUVecI16x8()) }
func GeSVecI16x8() Op { return Op(C.BinaryenGeSVecI16x8()) }
func GeUVecI16x8() Op { return Op(C.BinaryenGeUVecI16x8()) }

func EqVecI32x4() Op  { return Op(C.BinaryenEqVecI32x4()) }
func NeVecI32x4() Op  { return Op(C.BinaryenNeVecI32x4()) }
func LtSVecI32x4() Op { return Op(C.BinaryenLtSVecI32x4()) }
func LtUVecI32x4() Op { return Op(C.BinaryenLtUVecI32x4()) }
func GtSVecI32x4() Op { return Op(C.BinaryenGtSVecI32x4()) }
func GtUVecI32x4() Op { return Op(C.BinaryenGtUVecI32x4()) }
func LeSVecI32x4() Op { return Op(C.BinaryenLeSVecI32x4()) }
func LeUVecI32x4() Op { return Op(C.BinaryenLeUVecI32x4()) }
func GeSVecI32x4() Op { return Op(C.BinaryenGeSVecI32x4()) }
func GeUVecI32x4() Op { return Op(C.BinaryenGeUVecI32x4()) }

func EqVecI64x2() Op  { return Op(C.BinaryenEqVecI64x2()) }
func NeVecI64x2() Op  { return Op(C.BinaryenNeVecI64x2()) }
func LtSVecI64x2() Op { return Op(C.BinaryenLtSVecI64x2()) }
func GtSVecI64x2() Op { return Op(C.BinaryenGtSVecI64x2()) }
func LeSVecI64x2() Op { return Op(C.BinaryenLeSVecI64x2()) }
func GeSVecI64x2() Op { return Op(C.BinaryenGeSVecI64x2()) }

func EqVecF32x4() Op { return Op(C.BinaryenEqVecF32x4()) }
func NeVecF32x4() Op { return Op(C.BinaryenNeVecF32x4()) }
func LtVecF32x4() Op { return Op(C.BinaryenLtVecF32x4()) }
func GtVecF32x4() Op { return Op(C.BinaryenGtVecF32x4()) }
func LeVecF32x4() Op { return Op(C.BinaryenLeVecF32x4()) }
func GeVecF32x4() Op { return Op(C.BinaryenGeVecF32x4()) }

func EqVecF64x2() Op { return Op(C.BinaryenEqVecF64x2()) }
func NeVecF64x2() Op { return Op(C.BinaryenNeVecF64x2()) }
func LtVecF64x2() Op { return Op(C.BinaryenLtVecF64x2()) }
func GtVecF64x2() Op { return Op(C.BinaryenGtVecF64x2()) }
func LeVecF64x2() Op { return Op(C.BinaryenLeVecF64x2()) }
func GeVecF64x2() Op { return Op(C.BinaryenGeVecF64x2()) }

// ---------------------------------------------------------------------------
// SIMD — bitwise vec128
// ---------------------------------------------------------------------------

func NotVec128() Op      { return Op(C.BinaryenNotVec128()) }
func AndVec128() Op      { return Op(C.BinaryenAndVec128()) }
func OrVec128() Op       { return Op(C.BinaryenOrVec128()) }
func XorVec128() Op      { return Op(C.BinaryenXorVec128()) }
func AndNotVec128() Op   { return Op(C.BinaryenAndNotVec128()) }
func BitselectVec128() Op { return Op(C.BinaryenBitselectVec128()) }

// ---------------------------------------------------------------------------
// SIMD — relaxed ternary operators
// ---------------------------------------------------------------------------

func RelaxedMaddVecF32x4() Op  { return Op(C.BinaryenRelaxedMaddVecF32x4()) }
func RelaxedNmaddVecF32x4() Op { return Op(C.BinaryenRelaxedNmaddVecF32x4()) }
func RelaxedMaddVecF64x2() Op  { return Op(C.BinaryenRelaxedMaddVecF64x2()) }
func RelaxedNmaddVecF64x2() Op { return Op(C.BinaryenRelaxedNmaddVecF64x2()) }

func LaneselectI8x16() Op { return Op(C.BinaryenLaneselectI8x16()) }
func LaneselectI16x8() Op { return Op(C.BinaryenLaneselectI16x8()) }
func LaneselectI32x4() Op { return Op(C.BinaryenLaneselectI32x4()) }
func LaneselectI64x2() Op { return Op(C.BinaryenLaneselectI64x2()) }

func DotI8x16I7x16AddSToVecI32x4() Op { return Op(C.BinaryenDotI8x16I7x16AddSToVecI32x4()) }

// ---------------------------------------------------------------------------
// SIMD — vec128 any/all true
// ---------------------------------------------------------------------------

func AnyTrueVec128() Op { return Op(C.BinaryenAnyTrueVec128()) }

// ---------------------------------------------------------------------------
// SIMD — I8x16 unary, shift, arithmetic
// ---------------------------------------------------------------------------

func PopcntVecI8x16() Op  { return Op(C.BinaryenPopcntVecI8x16()) }
func AbsVecI8x16() Op     { return Op(C.BinaryenAbsVecI8x16()) }
func NegVecI8x16() Op     { return Op(C.BinaryenNegVecI8x16()) }
func AllTrueVecI8x16() Op { return Op(C.BinaryenAllTrueVecI8x16()) }
func BitmaskVecI8x16() Op { return Op(C.BinaryenBitmaskVecI8x16()) }
func ShlVecI8x16() Op     { return Op(C.BinaryenShlVecI8x16()) }
func ShrSVecI8x16() Op    { return Op(C.BinaryenShrSVecI8x16()) }
func ShrUVecI8x16() Op    { return Op(C.BinaryenShrUVecI8x16()) }
func AddVecI8x16() Op     { return Op(C.BinaryenAddVecI8x16()) }
func AddSatSVecI8x16() Op { return Op(C.BinaryenAddSatSVecI8x16()) }
func AddSatUVecI8x16() Op { return Op(C.BinaryenAddSatUVecI8x16()) }
func SubVecI8x16() Op     { return Op(C.BinaryenSubVecI8x16()) }
func SubSatSVecI8x16() Op { return Op(C.BinaryenSubSatSVecI8x16()) }
func SubSatUVecI8x16() Op { return Op(C.BinaryenSubSatUVecI8x16()) }
func MinSVecI8x16() Op    { return Op(C.BinaryenMinSVecI8x16()) }
func MinUVecI8x16() Op    { return Op(C.BinaryenMinUVecI8x16()) }
func MaxSVecI8x16() Op    { return Op(C.BinaryenMaxSVecI8x16()) }
func MaxUVecI8x16() Op    { return Op(C.BinaryenMaxUVecI8x16()) }
func AvgrUVecI8x16() Op   { return Op(C.BinaryenAvgrUVecI8x16()) }

// ---------------------------------------------------------------------------
// SIMD — I16x8 unary, shift, arithmetic
// ---------------------------------------------------------------------------

func AbsVecI16x8() Op     { return Op(C.BinaryenAbsVecI16x8()) }
func NegVecI16x8() Op     { return Op(C.BinaryenNegVecI16x8()) }
func AllTrueVecI16x8() Op { return Op(C.BinaryenAllTrueVecI16x8()) }
func BitmaskVecI16x8() Op { return Op(C.BinaryenBitmaskVecI16x8()) }
func ShlVecI16x8() Op     { return Op(C.BinaryenShlVecI16x8()) }
func ShrSVecI16x8() Op    { return Op(C.BinaryenShrSVecI16x8()) }
func ShrUVecI16x8() Op    { return Op(C.BinaryenShrUVecI16x8()) }
func AddVecI16x8() Op     { return Op(C.BinaryenAddVecI16x8()) }
func AddSatSVecI16x8() Op { return Op(C.BinaryenAddSatSVecI16x8()) }
func AddSatUVecI16x8() Op { return Op(C.BinaryenAddSatUVecI16x8()) }
func SubVecI16x8() Op     { return Op(C.BinaryenSubVecI16x8()) }
func SubSatSVecI16x8() Op { return Op(C.BinaryenSubSatSVecI16x8()) }
func SubSatUVecI16x8() Op { return Op(C.BinaryenSubSatUVecI16x8()) }
func MulVecI16x8() Op     { return Op(C.BinaryenMulVecI16x8()) }
func MinSVecI16x8() Op    { return Op(C.BinaryenMinSVecI16x8()) }
func MinUVecI16x8() Op    { return Op(C.BinaryenMinUVecI16x8()) }
func MaxSVecI16x8() Op    { return Op(C.BinaryenMaxSVecI16x8()) }
func MaxUVecI16x8() Op    { return Op(C.BinaryenMaxUVecI16x8()) }
func AvgrUVecI16x8() Op   { return Op(C.BinaryenAvgrUVecI16x8()) }

func Q15MulrSatSVecI16x8() Op  { return Op(C.BinaryenQ15MulrSatSVecI16x8()) }
func ExtMulLowSVecI16x8() Op   { return Op(C.BinaryenExtMulLowSVecI16x8()) }
func ExtMulHighSVecI16x8() Op  { return Op(C.BinaryenExtMulHighSVecI16x8()) }
func ExtMulLowUVecI16x8() Op   { return Op(C.BinaryenExtMulLowUVecI16x8()) }
func ExtMulHighUVecI16x8() Op  { return Op(C.BinaryenExtMulHighUVecI16x8()) }

// ---------------------------------------------------------------------------
// SIMD — I32x4 unary, shift, arithmetic
// ---------------------------------------------------------------------------

func AbsVecI32x4() Op     { return Op(C.BinaryenAbsVecI32x4()) }
func NegVecI32x4() Op     { return Op(C.BinaryenNegVecI32x4()) }
func AllTrueVecI32x4() Op { return Op(C.BinaryenAllTrueVecI32x4()) }
func BitmaskVecI32x4() Op { return Op(C.BinaryenBitmaskVecI32x4()) }
func ShlVecI32x4() Op     { return Op(C.BinaryenShlVecI32x4()) }
func ShrSVecI32x4() Op    { return Op(C.BinaryenShrSVecI32x4()) }
func ShrUVecI32x4() Op    { return Op(C.BinaryenShrUVecI32x4()) }
func AddVecI32x4() Op     { return Op(C.BinaryenAddVecI32x4()) }
func SubVecI32x4() Op     { return Op(C.BinaryenSubVecI32x4()) }
func MulVecI32x4() Op     { return Op(C.BinaryenMulVecI32x4()) }
func MinSVecI32x4() Op    { return Op(C.BinaryenMinSVecI32x4()) }
func MinUVecI32x4() Op    { return Op(C.BinaryenMinUVecI32x4()) }
func MaxSVecI32x4() Op    { return Op(C.BinaryenMaxSVecI32x4()) }
func MaxUVecI32x4() Op    { return Op(C.BinaryenMaxUVecI32x4()) }

func DotSVecI16x8ToVecI32x4() Op { return Op(C.BinaryenDotSVecI16x8ToVecI32x4()) }
func ExtMulLowSVecI32x4() Op     { return Op(C.BinaryenExtMulLowSVecI32x4()) }
func ExtMulHighSVecI32x4() Op    { return Op(C.BinaryenExtMulHighSVecI32x4()) }
func ExtMulLowUVecI32x4() Op     { return Op(C.BinaryenExtMulLowUVecI32x4()) }
func ExtMulHighUVecI32x4() Op    { return Op(C.BinaryenExtMulHighUVecI32x4()) }

// ---------------------------------------------------------------------------
// SIMD — I64x2 unary, shift, arithmetic
// ---------------------------------------------------------------------------

func AbsVecI64x2() Op     { return Op(C.BinaryenAbsVecI64x2()) }
func NegVecI64x2() Op     { return Op(C.BinaryenNegVecI64x2()) }
func AllTrueVecI64x2() Op { return Op(C.BinaryenAllTrueVecI64x2()) }
func BitmaskVecI64x2() Op { return Op(C.BinaryenBitmaskVecI64x2()) }
func ShlVecI64x2() Op     { return Op(C.BinaryenShlVecI64x2()) }
func ShrSVecI64x2() Op    { return Op(C.BinaryenShrSVecI64x2()) }
func ShrUVecI64x2() Op    { return Op(C.BinaryenShrUVecI64x2()) }
func AddVecI64x2() Op     { return Op(C.BinaryenAddVecI64x2()) }
func SubVecI64x2() Op     { return Op(C.BinaryenSubVecI64x2()) }
func MulVecI64x2() Op     { return Op(C.BinaryenMulVecI64x2()) }

func ExtMulLowSVecI64x2() Op  { return Op(C.BinaryenExtMulLowSVecI64x2()) }
func ExtMulHighSVecI64x2() Op { return Op(C.BinaryenExtMulHighSVecI64x2()) }
func ExtMulLowUVecI64x2() Op  { return Op(C.BinaryenExtMulLowUVecI64x2()) }
func ExtMulHighUVecI64x2() Op { return Op(C.BinaryenExtMulHighUVecI64x2()) }

// ---------------------------------------------------------------------------
// SIMD — F32x4 unary, arithmetic, rounding
// ---------------------------------------------------------------------------

func AbsVecF32x4() Op     { return Op(C.BinaryenAbsVecF32x4()) }
func NegVecF32x4() Op     { return Op(C.BinaryenNegVecF32x4()) }
func SqrtVecF32x4() Op    { return Op(C.BinaryenSqrtVecF32x4()) }
func AddVecF32x4() Op     { return Op(C.BinaryenAddVecF32x4()) }
func SubVecF32x4() Op     { return Op(C.BinaryenSubVecF32x4()) }
func MulVecF32x4() Op     { return Op(C.BinaryenMulVecF32x4()) }
func DivVecF32x4() Op     { return Op(C.BinaryenDivVecF32x4()) }
func MinVecF32x4() Op     { return Op(C.BinaryenMinVecF32x4()) }
func MaxVecF32x4() Op     { return Op(C.BinaryenMaxVecF32x4()) }
func PMinVecF32x4() Op    { return Op(C.BinaryenPMinVecF32x4()) }
func PMaxVecF32x4() Op    { return Op(C.BinaryenPMaxVecF32x4()) }
func CeilVecF32x4() Op    { return Op(C.BinaryenCeilVecF32x4()) }
func FloorVecF32x4() Op   { return Op(C.BinaryenFloorVecF32x4()) }
func TruncVecF32x4() Op   { return Op(C.BinaryenTruncVecF32x4()) }
func NearestVecF32x4() Op { return Op(C.BinaryenNearestVecF32x4()) }

// ---------------------------------------------------------------------------
// SIMD — F64x2 unary, arithmetic, rounding
// ---------------------------------------------------------------------------

func AbsVecF64x2() Op     { return Op(C.BinaryenAbsVecF64x2()) }
func NegVecF64x2() Op     { return Op(C.BinaryenNegVecF64x2()) }
func SqrtVecF64x2() Op    { return Op(C.BinaryenSqrtVecF64x2()) }
func AddVecF64x2() Op     { return Op(C.BinaryenAddVecF64x2()) }
func SubVecF64x2() Op     { return Op(C.BinaryenSubVecF64x2()) }
func MulVecF64x2() Op     { return Op(C.BinaryenMulVecF64x2()) }
func DivVecF64x2() Op     { return Op(C.BinaryenDivVecF64x2()) }
func MinVecF64x2() Op     { return Op(C.BinaryenMinVecF64x2()) }
func MaxVecF64x2() Op     { return Op(C.BinaryenMaxVecF64x2()) }
func PMinVecF64x2() Op    { return Op(C.BinaryenPMinVecF64x2()) }
func PMaxVecF64x2() Op    { return Op(C.BinaryenPMaxVecF64x2()) }
func CeilVecF64x2() Op    { return Op(C.BinaryenCeilVecF64x2()) }
func FloorVecF64x2() Op   { return Op(C.BinaryenFloorVecF64x2()) }
func TruncVecF64x2() Op   { return Op(C.BinaryenTruncVecF64x2()) }
func NearestVecF64x2() Op { return Op(C.BinaryenNearestVecF64x2()) }

// ---------------------------------------------------------------------------
// SIMD — extended pairwise addition
// ---------------------------------------------------------------------------

func ExtAddPairwiseSVecI8x16ToI16x8() Op { return Op(C.BinaryenExtAddPairwiseSVecI8x16ToI16x8()) }
func ExtAddPairwiseUVecI8x16ToI16x8() Op { return Op(C.BinaryenExtAddPairwiseUVecI8x16ToI16x8()) }
func ExtAddPairwiseSVecI16x8ToI32x4() Op { return Op(C.BinaryenExtAddPairwiseSVecI16x8ToI32x4()) }
func ExtAddPairwiseUVecI16x8ToI32x4() Op { return Op(C.BinaryenExtAddPairwiseUVecI16x8ToI32x4()) }

// ---------------------------------------------------------------------------
// SIMD — truncation / conversion between vec types
// ---------------------------------------------------------------------------

func TruncSatSVecF32x4ToVecI32x4() Op { return Op(C.BinaryenTruncSatSVecF32x4ToVecI32x4()) }
func TruncSatUVecF32x4ToVecI32x4() Op { return Op(C.BinaryenTruncSatUVecF32x4ToVecI32x4()) }
func ConvertSVecI32x4ToVecF32x4() Op  { return Op(C.BinaryenConvertSVecI32x4ToVecF32x4()) }
func ConvertUVecI32x4ToVecF32x4() Op  { return Op(C.BinaryenConvertUVecI32x4ToVecF32x4()) }

// ---------------------------------------------------------------------------
// SIMD — load/store operations
// ---------------------------------------------------------------------------

func Load8SplatVec128() Op  { return Op(C.BinaryenLoad8SplatVec128()) }
func Load16SplatVec128() Op { return Op(C.BinaryenLoad16SplatVec128()) }
func Load32SplatVec128() Op { return Op(C.BinaryenLoad32SplatVec128()) }
func Load64SplatVec128() Op { return Op(C.BinaryenLoad64SplatVec128()) }

func Load8x8SVec128() Op  { return Op(C.BinaryenLoad8x8SVec128()) }
func Load8x8UVec128() Op  { return Op(C.BinaryenLoad8x8UVec128()) }
func Load16x4SVec128() Op { return Op(C.BinaryenLoad16x4SVec128()) }
func Load16x4UVec128() Op { return Op(C.BinaryenLoad16x4UVec128()) }
func Load32x2SVec128() Op { return Op(C.BinaryenLoad32x2SVec128()) }
func Load32x2UVec128() Op { return Op(C.BinaryenLoad32x2UVec128()) }

func Load32ZeroVec128() Op { return Op(C.BinaryenLoad32ZeroVec128()) }
func Load64ZeroVec128() Op { return Op(C.BinaryenLoad64ZeroVec128()) }

func Load8LaneVec128() Op  { return Op(C.BinaryenLoad8LaneVec128()) }
func Load16LaneVec128() Op { return Op(C.BinaryenLoad16LaneVec128()) }
func Load32LaneVec128() Op { return Op(C.BinaryenLoad32LaneVec128()) }
func Load64LaneVec128() Op { return Op(C.BinaryenLoad64LaneVec128()) }

func Store8LaneVec128() Op  { return Op(C.BinaryenStore8LaneVec128()) }
func Store16LaneVec128() Op { return Op(C.BinaryenStore16LaneVec128()) }
func Store32LaneVec128() Op { return Op(C.BinaryenStore32LaneVec128()) }
func Store64LaneVec128() Op { return Op(C.BinaryenStore64LaneVec128()) }

// ---------------------------------------------------------------------------
// SIMD — narrowing operators
// ---------------------------------------------------------------------------

func NarrowSVecI16x8ToVecI8x16() Op { return Op(C.BinaryenNarrowSVecI16x8ToVecI8x16()) }
func NarrowUVecI16x8ToVecI8x16() Op { return Op(C.BinaryenNarrowUVecI16x8ToVecI8x16()) }
func NarrowSVecI32x4ToVecI16x8() Op { return Op(C.BinaryenNarrowSVecI32x4ToVecI16x8()) }
func NarrowUVecI32x4ToVecI16x8() Op { return Op(C.BinaryenNarrowUVecI32x4ToVecI16x8()) }

// ---------------------------------------------------------------------------
// SIMD — extend (widen) operators
// ---------------------------------------------------------------------------

func ExtendLowSVecI8x16ToVecI16x8() Op  { return Op(C.BinaryenExtendLowSVecI8x16ToVecI16x8()) }
func ExtendHighSVecI8x16ToVecI16x8() Op { return Op(C.BinaryenExtendHighSVecI8x16ToVecI16x8()) }
func ExtendLowUVecI8x16ToVecI16x8() Op  { return Op(C.BinaryenExtendLowUVecI8x16ToVecI16x8()) }
func ExtendHighUVecI8x16ToVecI16x8() Op { return Op(C.BinaryenExtendHighUVecI8x16ToVecI16x8()) }

func ExtendLowSVecI16x8ToVecI32x4() Op  { return Op(C.BinaryenExtendLowSVecI16x8ToVecI32x4()) }
func ExtendHighSVecI16x8ToVecI32x4() Op { return Op(C.BinaryenExtendHighSVecI16x8ToVecI32x4()) }
func ExtendLowUVecI16x8ToVecI32x4() Op  { return Op(C.BinaryenExtendLowUVecI16x8ToVecI32x4()) }
func ExtendHighUVecI16x8ToVecI32x4() Op { return Op(C.BinaryenExtendHighUVecI16x8ToVecI32x4()) }

func ExtendLowSVecI32x4ToVecI64x2() Op  { return Op(C.BinaryenExtendLowSVecI32x4ToVecI64x2()) }
func ExtendHighSVecI32x4ToVecI64x2() Op { return Op(C.BinaryenExtendHighSVecI32x4ToVecI64x2()) }
func ExtendLowUVecI32x4ToVecI64x2() Op  { return Op(C.BinaryenExtendLowUVecI32x4ToVecI64x2()) }
func ExtendHighUVecI32x4ToVecI64x2() Op { return Op(C.BinaryenExtendHighUVecI32x4ToVecI64x2()) }

// ---------------------------------------------------------------------------
// SIMD — F64x2 <-> I32x4/F32x4 conversions
// ---------------------------------------------------------------------------

func ConvertLowSVecI32x4ToVecF64x2() Op      { return Op(C.BinaryenConvertLowSVecI32x4ToVecF64x2()) }
func ConvertLowUVecI32x4ToVecF64x2() Op      { return Op(C.BinaryenConvertLowUVecI32x4ToVecF64x2()) }
func TruncSatZeroSVecF64x2ToVecI32x4() Op    { return Op(C.BinaryenTruncSatZeroSVecF64x2ToVecI32x4()) }
func TruncSatZeroUVecF64x2ToVecI32x4() Op    { return Op(C.BinaryenTruncSatZeroUVecF64x2ToVecI32x4()) }
func DemoteZeroVecF64x2ToVecF32x4() Op       { return Op(C.BinaryenDemoteZeroVecF64x2ToVecF32x4()) }
func PromoteLowVecF32x4ToVecF64x2() Op       { return Op(C.BinaryenPromoteLowVecF32x4ToVecF64x2()) }

// ---------------------------------------------------------------------------
// SIMD — relaxed truncation operators
// ---------------------------------------------------------------------------

func RelaxedTruncSVecF32x4ToVecI32x4() Op     { return Op(C.BinaryenRelaxedTruncSVecF32x4ToVecI32x4()) }
func RelaxedTruncUVecF32x4ToVecI32x4() Op     { return Op(C.BinaryenRelaxedTruncUVecF32x4ToVecI32x4()) }
func RelaxedTruncZeroSVecF64x2ToVecI32x4() Op { return Op(C.BinaryenRelaxedTruncZeroSVecF64x2ToVecI32x4()) }
func RelaxedTruncZeroUVecF64x2ToVecI32x4() Op { return Op(C.BinaryenRelaxedTruncZeroUVecF64x2ToVecI32x4()) }

// ---------------------------------------------------------------------------
// SIMD — swizzle operators
// ---------------------------------------------------------------------------

func SwizzleVecI8x16() Op        { return Op(C.BinaryenSwizzleVecI8x16()) }
func RelaxedSwizzleVecI8x16() Op { return Op(C.BinaryenRelaxedSwizzleVecI8x16()) }

// ---------------------------------------------------------------------------
// SIMD — relaxed min/max operators
// ---------------------------------------------------------------------------

func RelaxedMinVecF32x4() Op { return Op(C.BinaryenRelaxedMinVecF32x4()) }
func RelaxedMaxVecF32x4() Op { return Op(C.BinaryenRelaxedMaxVecF32x4()) }
func RelaxedMinVecF64x2() Op { return Op(C.BinaryenRelaxedMinVecF64x2()) }
func RelaxedMaxVecF64x2() Op { return Op(C.BinaryenRelaxedMaxVecF64x2()) }

// ---------------------------------------------------------------------------
// SIMD — relaxed Q15 multiply and dot product
// ---------------------------------------------------------------------------

func RelaxedQ15MulrSVecI16x8() Op    { return Op(C.BinaryenRelaxedQ15MulrSVecI16x8()) }
func DotI8x16I7x16SToVecI16x8() Op  { return Op(C.BinaryenDotI8x16I7x16SToVecI16x8()) }

// ---------------------------------------------------------------------------
// Reference type operators
// ---------------------------------------------------------------------------

func RefAsNonNull() Op          { return Op(C.BinaryenRefAsNonNull()) }
func RefAsExternInternalize() Op { return Op(C.BinaryenRefAsExternInternalize()) }
func RefAsExternExternalize() Op { return Op(C.BinaryenRefAsExternExternalize()) }
func RefAsAnyConvertExtern() Op { return Op(C.BinaryenRefAsAnyConvertExtern()) }
func RefAsExternConvertAny() Op { return Op(C.BinaryenRefAsExternConvertAny()) }

// ---------------------------------------------------------------------------
// Branch operators
// ---------------------------------------------------------------------------

func BrOnNull() Op     { return Op(C.BinaryenBrOnNull()) }
func BrOnNonNull() Op  { return Op(C.BinaryenBrOnNonNull()) }
func BrOnCast() Op     { return Op(C.BinaryenBrOnCast()) }
func BrOnCastFail() Op { return Op(C.BinaryenBrOnCastFail()) }

// ---------------------------------------------------------------------------
// String operators
// ---------------------------------------------------------------------------

func StringNewLossyUTF8Array() Op    { return Op(C.BinaryenStringNewLossyUTF8Array()) }
func StringNewWTF16Array() Op        { return Op(C.BinaryenStringNewWTF16Array()) }
func StringNewFromCodePoint() Op     { return Op(C.BinaryenStringNewFromCodePoint()) }
func StringMeasureUTF8() Op          { return Op(C.BinaryenStringMeasureUTF8()) }
func StringMeasureWTF16() Op         { return Op(C.BinaryenStringMeasureWTF16()) }
func StringEncodeLossyUTF8Array() Op { return Op(C.BinaryenStringEncodeLossyUTF8Array()) }
func StringEncodeWTF16Array() Op     { return Op(C.BinaryenStringEncodeWTF16Array()) }
func StringEqEqual() Op             { return Op(C.BinaryenStringEqEqual()) }
func StringEqCompare() Op           { return Op(C.BinaryenStringEqCompare()) }
