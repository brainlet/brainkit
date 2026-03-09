package common

type Feature uint32

const (
	FeatureNone              Feature = 0
	FeatureSignExtension     Feature = 1 << 0
	FeatureMutableGlobals    Feature = 1 << 1
	FeatureNontrappingF2I    Feature = 1 << 2
	FeatureBulkMemory        Feature = 1 << 3
	FeatureSimd              Feature = 1 << 4
	FeatureThreads           Feature = 1 << 5
	FeatureExceptionHandling Feature = 1 << 6
	FeatureTailCalls         Feature = 1 << 7
	FeatureReferenceTypes    Feature = 1 << 8
	FeatureMultiValue        Feature = 1 << 9
	FeatureGC                Feature = 1 << 10
	FeatureMemory64          Feature = 1 << 11
	FeatureRelaxedSimd       Feature = 1 << 12
	FeatureExtendedConst     Feature = 1 << 13
	FeatureStringref         Feature = 1 << 14
	FeatureAll               Feature = (1 << 15) - 1
)

func FeatureToString(f Feature) string {
	switch f {
	case FeatureSignExtension:
		return "sign-extension"
	case FeatureMutableGlobals:
		return "mutable-globals"
	case FeatureNontrappingF2I:
		return "nontrapping-f2i"
	case FeatureBulkMemory:
		return "bulk-memory"
	case FeatureSimd:
		return "simd"
	case FeatureThreads:
		return "threads"
	case FeatureExceptionHandling:
		return "exception-handling"
	case FeatureTailCalls:
		return "tail-calls"
	case FeatureReferenceTypes:
		return "reference-types"
	case FeatureMultiValue:
		return "multi-value"
	case FeatureGC:
		return "gc"
	case FeatureMemory64:
		return "memory64"
	case FeatureRelaxedSimd:
		return "relaxed-simd"
	case FeatureExtendedConst:
		return "extended-const"
	case FeatureStringref:
		return "stringref"
	default:
		return ""
	}
}
