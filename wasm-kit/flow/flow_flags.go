package flow

// FlowFlags indicates specific control flow conditions.
type FlowFlags uint32

const (
	FlowFlagNone FlowFlags = 0

	// categorical

	FlowFlagReturns        FlowFlags = 1 << 0
	FlowFlagReturnsWrapped FlowFlags = 1 << 1
	FlowFlagReturnsNonNull FlowFlags = 1 << 2
	FlowFlagThrows         FlowFlags = 1 << 3
	FlowFlagBreaks         FlowFlags = 1 << 4
	FlowFlagContinues      FlowFlags = 1 << 5
	FlowFlagAccessesThis   FlowFlags = 1 << 6
	FlowFlagCallsSuper     FlowFlags = 1 << 7
	FlowFlagTerminates     FlowFlags = 1 << 8 // does NOT cover Breaks

	// conditional

	FlowFlagConditionallyReturns       FlowFlags = 1 << 9
	FlowFlagConditionallyThrows        FlowFlags = 1 << 10
	FlowFlagConditionallyBreaks        FlowFlags = 1 << 11
	FlowFlagConditionallyContinues     FlowFlags = 1 << 12
	FlowFlagConditionallyAccessesThis  FlowFlags = 1 << 13
	FlowFlagMayReturnNonThis           FlowFlags = 1 << 14

	// other

	FlowFlagUncheckedContext FlowFlags = 1 << 15
	FlowFlagCtorParamContext FlowFlags = 1 << 16
	FlowFlagInlineContext    FlowFlags = 1 << 17

	// masks

	FlowFlagAnyCategorical FlowFlags = FlowFlagReturns |
		FlowFlagReturnsWrapped |
		FlowFlagReturnsNonNull |
		FlowFlagThrows |
		FlowFlagBreaks |
		FlowFlagContinues |
		FlowFlagAccessesThis |
		FlowFlagCallsSuper |
		FlowFlagTerminates

	FlowFlagAnyConditional FlowFlags = FlowFlagConditionallyReturns |
		FlowFlagConditionallyThrows |
		FlowFlagConditionallyBreaks |
		FlowFlagConditionallyContinues |
		FlowFlagConditionallyAccessesThis
)

// LocalFlags indicates the state of a local variable.
type LocalFlags uint8

const (
	LocalFlagNone        LocalFlags = 0
	LocalFlagConstant    LocalFlags = 1 << 0
	LocalFlagWrapped     LocalFlags = 1 << 1
	LocalFlagNonNull     LocalFlags = 1 << 2
	LocalFlagInitialized LocalFlags = 1 << 3
)

// AllLocalFlags is the mask of all local flags.
const AllLocalFlags = LocalFlagConstant | LocalFlagWrapped | LocalFlagNonNull | LocalFlagInitialized

// FieldFlags indicates the state of a class field.
type FieldFlags uint8

const (
	FieldFlagNone        FieldFlags = 0
	FieldFlagInitialized FieldFlags = 1 << 0
)

// ConditionKind indicates the known outcome of a condition.
type ConditionKind int32

const (
	ConditionKindUnknown ConditionKind = iota
	ConditionKindTrue
	ConditionKindFalse
)
