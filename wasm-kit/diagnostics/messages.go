package diagnostics

// DiagnosticCode represents a diagnostic message code.
// Generated from diagnosticMessages.json. DO NOT EDIT.
type DiagnosticCode int32

const (
	DiagnosticCodeNotImplemented0 DiagnosticCode = 100
	DiagnosticCodeOperationIsUnsafe DiagnosticCode = 101
	DiagnosticCodeUserDefined0 DiagnosticCode = 102
	DiagnosticCodeFeature0IsNotEnabled DiagnosticCode = 103
	DiagnosticCodeLowMemoryLimitExceededByStaticData01 DiagnosticCode = 104
	DiagnosticCodeModuleRequiresAtLeast0PagesOfInitialMemory DiagnosticCode = 105
	DiagnosticCodeModuleRequiresAtLeast0PagesOfMaximumMemory DiagnosticCode = 106
	DiagnosticCodeSharedMemoryRequiresMaximumMemoryToBeDefined DiagnosticCode = 107
	DiagnosticCodeSharedMemoryRequiresFeatureThreadsToBeEnabled DiagnosticCode = 108
	DiagnosticCodeTransform01 DiagnosticCode = 109
	DiagnosticCodeStartFunctionName0IsInvalidOrConflictsWithAnotherExport DiagnosticCode = 110
	DiagnosticCodeElement0NotFound DiagnosticCode = 111
	DiagnosticCodeExchangeOf0ValuesIsNotSupportedByAllEmbeddings DiagnosticCode = 112
	DiagnosticCodeConversionFromType0To1RequiresAnExplicitCast DiagnosticCode = 200
	DiagnosticCodeConversionFromType0To1WillRequireAnExplicitCastWhenSwitchingBetween3264Bit DiagnosticCode = 201
	DiagnosticCodeType0CannotBeChangedToType1 DiagnosticCode = 202
	DiagnosticCodeOperation0CannotBeAppliedToType1 DiagnosticCode = 203
	DiagnosticCodeType0CannotBeNullable DiagnosticCode = 204
	DiagnosticCodeMutableValueCannotBeInlined DiagnosticCode = 206
	DiagnosticCodeUnmanagedClassesCannotExtendManagedClassesAndViceVersa DiagnosticCode = 207
	DiagnosticCodeUnmanagedClassesCannotImplementInterfaces DiagnosticCode = 208
	DiagnosticCodeInvalidRegularExpressionFlags DiagnosticCode = 209
	DiagnosticCodeExpressionIsNeverNull DiagnosticCode = 210
	DiagnosticCodeClass0IsFinalAndCannotBeExtended DiagnosticCode = 211
	DiagnosticCodeDecorator0IsNotValidHere DiagnosticCode = 212
	DiagnosticCodeDuplicateDecorator DiagnosticCode = 213
	DiagnosticCodeType0IsIllegalInThisContext DiagnosticCode = 214
	DiagnosticCodeOptionalParameterMustHaveAnInitializer DiagnosticCode = 215
	DiagnosticCodeClass0CannotDeclareAConstructorWhenInstantiatedFromAnObjectLiteral DiagnosticCode = 216
	DiagnosticCodeFunction0CannotBeInlinedIntoItself DiagnosticCode = 217
	DiagnosticCodeCannotAccessMethod0WithoutCallingItAsItRequiresThisToBeSet DiagnosticCode = 218
	DiagnosticCodeOptionalPropertiesAreNotSupported DiagnosticCode = 219
	DiagnosticCodeExpressionMustBeACompileTimeConstant DiagnosticCode = 220
	DiagnosticCodeType0IsNotAFunctionIndexOrFunctionReference DiagnosticCode = 221
	DiagnosticCode0MustBeAValueBetween1And2Inclusive DiagnosticCode = 222
	DiagnosticCode0MustBeAPowerOfTwo DiagnosticCode = 223
	DiagnosticCode0IsNotAValidOperator DiagnosticCode = 224
	DiagnosticCodeExpressionCannotBeRepresentedByAType DiagnosticCode = 225
	DiagnosticCodeExpressionResolvesToUnusualType0 DiagnosticCode = 226
	DiagnosticCodeArrayLiteralExpected DiagnosticCode = 227
	DiagnosticCodeFunction0IsVirtualAndWillNotBeInlined DiagnosticCode = 228
	DiagnosticCodeProperty0OnlyHasASetterAndIsMissingAGetter DiagnosticCode = 229
	DiagnosticCode0KeywordCannotBeUsedHere DiagnosticCode = 230
	DiagnosticCodeAClassWithAConstructorExplicitlyReturningSomethingElseThanThisMustBeFinal DiagnosticCode = 231
	DiagnosticCodeProperty0IsAlwaysAssignedBeforeBeingUsed DiagnosticCode = 233
	DiagnosticCodeExpressionDoesNotCompileToAValueAtRuntime DiagnosticCode = 234
	DiagnosticCodeOnlyVariablesFunctionsAndEnumsBecomeWebassemblyModuleExports DiagnosticCode = 235
	DiagnosticCodeLiteral0DoesNotFitIntoI64OrU64Types DiagnosticCode = 236
	DiagnosticCodeIndexSignatureAccessorsInType0DifferInTypes DiagnosticCode = 237
	DiagnosticCodeInitializerDefinitiveAssignmentOrNullableTypeExpected DiagnosticCode = 238
	DiagnosticCodeDefinitiveAssignmentHasNoEffectOnLocalVariables DiagnosticCode = 239
	DiagnosticCodeAmbiguousOperatorOverload0ConflictingOverloads1And2 DiagnosticCode = 240
	DiagnosticCodeImportingTheTableDisablesSomeIndirectCallOptimizations DiagnosticCode = 901
	DiagnosticCodeExportingTheTableDisablesSomeIndirectCallOptimizations DiagnosticCode = 902
	DiagnosticCodeExpressionCompilesToADynamicCheckAtRuntime DiagnosticCode = 903
	DiagnosticCodeIndexedAccessMayInvolveBoundsChecking DiagnosticCode = 904
	DiagnosticCodeExplicitlyReturningConstructorDropsThisAllocation DiagnosticCode = 905
	DiagnosticCodeUnnecessaryDefiniteAssignment DiagnosticCode = 906
	DiagnosticCodeNanDoesNotCompareEqualToAnyOtherValueIncludingItselfUseIsnanXInstead DiagnosticCode = 907
	DiagnosticCodeComparisonWith00IsSignInsensitiveUseObjectIsX00IfTheSignMatters DiagnosticCode = 908
	DiagnosticCodeUnterminatedStringLiteral DiagnosticCode = 1002
	DiagnosticCodeIdentifierExpected DiagnosticCode = 1003
	DiagnosticCode0Expected DiagnosticCode = 1005
	DiagnosticCodeAFileCannotHaveAReferenceToItself DiagnosticCode = 1006
	DiagnosticCodeTrailingCommaNotAllowed DiagnosticCode = 1009
	DiagnosticCodeUnexpectedToken DiagnosticCode = 1012
	DiagnosticCodeARestParameterMustBeLastInAParameterList DiagnosticCode = 1014
	DiagnosticCodeParameterCannotHaveQuestionMarkAndInitializer DiagnosticCode = 1015
	DiagnosticCodeARequiredParameterCannotFollowAnOptionalParameter DiagnosticCode = 1016
	DiagnosticCode0ModifierCannotAppearOnClassElementsOfThisKind DiagnosticCode = 1031
	DiagnosticCodeStatementsAreNotAllowedInAmbientContexts DiagnosticCode = 1036
	DiagnosticCodeInitializersAreNotAllowedInAmbientContexts DiagnosticCode = 1039
	DiagnosticCode0ModifierCannotBeUsedHere DiagnosticCode = 1042
	DiagnosticCodeARestParameterCannotBeOptional DiagnosticCode = 1047
	DiagnosticCodeARestParameterCannotHaveAnInitializer DiagnosticCode = 1048
	DiagnosticCodeASetAccessorMustHaveExactlyOneParameter DiagnosticCode = 1049
	DiagnosticCodeASetAccessorParameterCannotHaveAnInitializer DiagnosticCode = 1052
	DiagnosticCodeAGetAccessorCannotHaveParameters DiagnosticCode = 1054
	DiagnosticCodeEnumMemberMustHaveInitializer DiagnosticCode = 1061
	DiagnosticCodeTypeParametersCannotAppearOnAConstructorDeclaration DiagnosticCode = 1092
	DiagnosticCodeTypeAnnotationCannotAppearOnAConstructorDeclaration DiagnosticCode = 1093
	DiagnosticCodeAnAccessorCannotHaveTypeParameters DiagnosticCode = 1094
	DiagnosticCodeASetAccessorCannotHaveAReturnTypeAnnotation DiagnosticCode = 1095
	DiagnosticCodeTypeParameterListCannotBeEmpty DiagnosticCode = 1098
	DiagnosticCodeTypeArgumentListCannotBeEmpty DiagnosticCode = 1099
	DiagnosticCodeAContinueStatementCanOnlyBeUsedWithinAnEnclosingIterationStatement DiagnosticCode = 1104
	DiagnosticCodeABreakStatementCanOnlyBeUsedWithinAnEnclosingIterationOrSwitchStatement DiagnosticCode = 1105
	DiagnosticCodeAReturnStatementCanOnlyBeUsedWithinAFunctionBody DiagnosticCode = 1108
	DiagnosticCodeExpressionExpected DiagnosticCode = 1109
	DiagnosticCodeTypeExpected DiagnosticCode = 1110
	DiagnosticCodeADefaultClauseCannotAppearMoreThanOnceInASwitchStatement DiagnosticCode = 1113
	DiagnosticCodeDuplicateLabel0 DiagnosticCode = 1114
	DiagnosticCodeAnExportAssignmentCannotHaveModifiers DiagnosticCode = 1120
	DiagnosticCodeOctalLiteralsAreNotAllowedInStrictMode DiagnosticCode = 1121
	DiagnosticCodeDigitExpected DiagnosticCode = 1124
	DiagnosticCodeHexadecimalDigitExpected DiagnosticCode = 1125
	DiagnosticCodeUnexpectedEndOfText DiagnosticCode = 1126
	DiagnosticCodeInvalidCharacter DiagnosticCode = 1127
	DiagnosticCodeCaseOrDefaultExpected DiagnosticCode = 1130
	DiagnosticCodeSuperMustBeFollowedByAnArgumentListOrMemberAccess DiagnosticCode = 1034
	DiagnosticCodeADeclareModifierCannotBeUsedInAnAlreadyAmbientContext DiagnosticCode = 1038
	DiagnosticCodeTypeArgumentExpected DiagnosticCode = 1140
	DiagnosticCodeStringLiteralExpected DiagnosticCode = 1141
	DiagnosticCodeLineBreakNotPermittedHere DiagnosticCode = 1142
	DiagnosticCodeDeclarationExpected DiagnosticCode = 1146
	DiagnosticCodeConstDeclarationsMustBeInitialized DiagnosticCode = 1155
	DiagnosticCodeUnterminatedRegularExpressionLiteral DiagnosticCode = 1161
	DiagnosticCodeDeclarationsWithInitializersCannotAlsoHaveDefiniteAssignmentAssertions DiagnosticCode = 1263
	DiagnosticCodeInterfaceDeclarationCannotHaveImplementsClause DiagnosticCode = 1176
	DiagnosticCodeBinaryDigitExpected DiagnosticCode = 1177
	DiagnosticCodeOctalDigitExpected DiagnosticCode = 1178
	DiagnosticCodeAnImplementationCannotBeDeclaredInAmbientContexts DiagnosticCode = 1183
	DiagnosticCodeTheVariableDeclarationOfAForOfStatementCannotHaveAnInitializer DiagnosticCode = 1190
	DiagnosticCodeAnExtendedUnicodeEscapeValueMustBeBetween0x0And0x10ffffInclusive DiagnosticCode = 1198
	DiagnosticCodeUnterminatedUnicodeEscapeSequence DiagnosticCode = 1199
	DiagnosticCodeDecoratorsAreNotValidHere DiagnosticCode = 1206
	DiagnosticCodeAbstractModifierCanOnlyAppearOnAClassMethodOrPropertyDeclaration DiagnosticCode = 1242
	DiagnosticCodeMethod0CannotHaveAnImplementationBecauseItIsMarkedAbstract DiagnosticCode = 1245
	DiagnosticCodeAnInterfacePropertyCannotHaveAnInitializer DiagnosticCode = 1246
	DiagnosticCodeADefiniteAssignmentAssertionIsNotPermittedInThisContext DiagnosticCode = 1255
	DiagnosticCodeAClassMayOnlyExtendAnotherClass DiagnosticCode = 1311
	DiagnosticCodeAParameterPropertyCannotBeDeclaredUsingARestParameter DiagnosticCode = 1317
	DiagnosticCodeADefaultExportCanOnlyBeUsedInAModule DiagnosticCode = 1319
	DiagnosticCodeAnExpressionOfType0CannotBeTestedForTruthiness DiagnosticCode = 1345
	DiagnosticCodeAnIdentifierOrKeywordCannotImmediatelyFollowANumericLiteral DiagnosticCode = 1351
	DiagnosticCodeDuplicateIdentifier0 DiagnosticCode = 2300
	DiagnosticCodeCannotFindName0 DiagnosticCode = 2304
	DiagnosticCodeModule0HasNoExportedMember1 DiagnosticCode = 2305
	DiagnosticCodeAnInterfaceCanOnlyExtendAnInterface DiagnosticCode = 2312
	DiagnosticCodeGenericType0Requires1TypeArgumentS DiagnosticCode = 2314
	DiagnosticCodeType0IsNotGeneric DiagnosticCode = 2315
	DiagnosticCodeType0IsNotAssignableToType1 DiagnosticCode = 2322
	DiagnosticCodeProperty0IsPrivateInType1ButNotInType2 DiagnosticCode = 2325
	DiagnosticCodeIndexSignatureIsMissingInType0 DiagnosticCode = 2329
	DiagnosticCodeThisCannotBeReferencedInCurrentLocation DiagnosticCode = 2332
	DiagnosticCodeThisCannotBeReferencedInConstructorArguments DiagnosticCode = 2333
	DiagnosticCodeSuperCanOnlyBeReferencedInADerivedClass DiagnosticCode = 2335
	DiagnosticCodeSuperCannotBeReferencedInConstructorArguments DiagnosticCode = 2336
	DiagnosticCodeSuperCallsAreNotPermittedOutsideConstructorsOrInNestedFunctionsInsideConstructors DiagnosticCode = 2337
	DiagnosticCodeProperty0DoesNotExistOnType1 DiagnosticCode = 2339
	DiagnosticCodeProperty0IsPrivateAndOnlyAccessibleWithinClass1 DiagnosticCode = 2341
	DiagnosticCodeCannotInvokeAnExpressionWhoseTypeLacksACallSignatureType0HasNoCompatibleCallSignatures DiagnosticCode = 2349
	DiagnosticCodeThisExpressionIsNotConstructable DiagnosticCode = 2351
	DiagnosticCodeAFunctionWhoseDeclaredTypeIsNotVoidMustReturnAValue DiagnosticCode = 2355
	DiagnosticCodeTheOperandOfAnIncrementOrDecrementOperatorMustBeAVariableOrAPropertyAccess DiagnosticCode = 2357
	DiagnosticCodeTheLeftHandSideOfAnAssignmentExpressionMustBeAVariableOrAPropertyAccess DiagnosticCode = 2364
	DiagnosticCodeOperator0CannotBeAppliedToTypes1And2 DiagnosticCode = 2365
	DiagnosticCodeASuperCallMustBeTheFirstStatementInTheConstructor DiagnosticCode = 2376
	DiagnosticCodeConstructorsForDerivedClassesMustContainASuperCall DiagnosticCode = 2377
	DiagnosticCodeGetAndSetAccessorMustHaveTheSameType DiagnosticCode = 2380
	DiagnosticCodeOverloadSignaturesMustAllBePublicPrivateOrProtected DiagnosticCode = 2385
	DiagnosticCodeConstructorImplementationIsMissing DiagnosticCode = 2390
	DiagnosticCodeFunctionImplementationIsMissingOrNotImmediatelyFollowingTheDeclaration DiagnosticCode = 2391
	DiagnosticCodeMultipleConstructorImplementationsAreNotAllowed DiagnosticCode = 2392
	DiagnosticCodeDuplicateFunctionImplementation DiagnosticCode = 2393
	DiagnosticCodeThisOverloadSignatureIsNotCompatibleWithItsImplementationSignature DiagnosticCode = 2394
	DiagnosticCodeIndividualDeclarationsInMergedDeclaration0MustBeAllExportedOrAllLocal DiagnosticCode = 2395
	DiagnosticCodeProperty0InType1IsNotAssignableToTheSamePropertyInBaseType2 DiagnosticCode = 2416
	DiagnosticCodeAClassCanOnlyImplementAnInterface DiagnosticCode = 2422
	DiagnosticCodeANamespaceDeclarationCannotBeLocatedPriorToAClassOrFunctionWithWhichItIsMerged DiagnosticCode = 2434
	DiagnosticCodeTypesHaveSeparateDeclarationsOfAPrivateProperty0 DiagnosticCode = 2442
	DiagnosticCodeProperty0IsProtectedInType1ButPublicInType2 DiagnosticCode = 2444
	DiagnosticCodeProperty0IsProtectedAndOnlyAccessibleWithinClass1AndItsSubclasses DiagnosticCode = 2445
	DiagnosticCodeVariable0UsedBeforeItsDeclaration DiagnosticCode = 2448
	DiagnosticCodeCannotRedeclareBlockScopedVariable0 DiagnosticCode = 2451
	DiagnosticCodeTheTypeArgumentForTypeParameter0CannotBeInferredFromTheUsageConsiderSpecifyingTheTypeArgumentsExplicitly DiagnosticCode = 2453
	DiagnosticCodeVariable0IsUsedBeforeBeingAssigned DiagnosticCode = 2454
	DiagnosticCodeTypeAlias0CircularlyReferencesItself DiagnosticCode = 2456
	DiagnosticCodeType0HasNoProperty1 DiagnosticCode = 2460
	DiagnosticCodeThe0OperatorCannotBeAppliedToType1 DiagnosticCode = 2469
	DiagnosticCodeInConstEnumDeclarationsMemberInitializerMustBeConstantExpression DiagnosticCode = 2474
	DiagnosticCodeAConstEnumMemberCanOnlyBeAccessedUsingAStringLiteral DiagnosticCode = 2476
	DiagnosticCodeExportDeclarationConflictsWithExportedDeclarationOf0 DiagnosticCode = 2484
	DiagnosticCode0IsReferencedDirectlyOrIndirectlyInItsOwnBaseExpression DiagnosticCode = 2506
	DiagnosticCodeCannotCreateAnInstanceOfAnAbstractClass DiagnosticCode = 2511
	DiagnosticCodeNonAbstractClass0DoesNotImplementInheritedAbstractMember1From2 DiagnosticCode = 2515
	DiagnosticCodeObjectIsPossiblyNull DiagnosticCode = 2531
	DiagnosticCodeCannotAssignTo0BecauseItIsAConstantOrAReadOnlyProperty DiagnosticCode = 2540
	DiagnosticCodeTheTargetOfAnAssignmentMustBeAVariableOrAPropertyAccess DiagnosticCode = 2541
	DiagnosticCodeIndexSignatureInType0OnlyPermitsReading DiagnosticCode = 2542
	DiagnosticCodeExpected0ArgumentsButGot1 DiagnosticCode = 2554
	DiagnosticCodeExpectedAtLeast0ArgumentsButGot1 DiagnosticCode = 2555
	DiagnosticCodeExpected0TypeArgumentsButGot1 DiagnosticCode = 2558
	DiagnosticCodeProperty0HasNoInitializerAndIsNotAssignedInTheConstructorBeforeThisIsUsedOrReturned DiagnosticCode = 2564
	DiagnosticCodeProperty0IsUsedBeforeBeingAssigned DiagnosticCode = 2565
	DiagnosticCode0IsDefinedAsAnAccessorInClass1ButIsOverriddenHereIn2AsAnInstanceProperty DiagnosticCode = 2610
	DiagnosticCode0IsDefinedAsAPropertyInClass1ButIsOverriddenHereIn2AsAnAccessor DiagnosticCode = 2611
	DiagnosticCodeAMemberInitializerInAEnumDeclarationCannotReferenceMembersDeclaredAfterItIncludingMembersDefinedInOtherEnums DiagnosticCode = 2651
	DiagnosticCodeConstructorOfClass0IsPrivateAndOnlyAccessibleWithinTheClassDeclaration DiagnosticCode = 2673
	DiagnosticCodeConstructorOfClass0IsProtectedAndOnlyAccessibleWithinTheClassDeclaration DiagnosticCode = 2674
	DiagnosticCodeCannotExtendAClass0ClassConstructorIsMarkedAsPrivate DiagnosticCode = 2675
	DiagnosticCodeTheThisTypesOfEachSignatureAreIncompatible DiagnosticCode = 2685
	DiagnosticCodeNamespace0HasNoExportedMember1 DiagnosticCode = 2694
	DiagnosticCodeNamespaceCanOnlyHaveDeclarations DiagnosticCode = 2695
	DiagnosticCodeRequiredTypeParametersMayNotFollowOptionalTypeParameters DiagnosticCode = 2706
	DiagnosticCodeDuplicateProperty0 DiagnosticCode = 2718
	DiagnosticCodeProperty0IsMissingInType1ButRequiredInType2 DiagnosticCode = 2741
	DiagnosticCodeType0HasNoCallSignatures DiagnosticCode = 2757
	DiagnosticCodeGetAccessor0MustBeAtLeastAsAccessibleAsTheSetter DiagnosticCode = 2808
	DiagnosticCodeThisMemberCannotHaveAnOverrideModifierBecauseItIsNotDeclaredInTheBaseClass0 DiagnosticCode = 4117
	DiagnosticCodeFile0NotFound DiagnosticCode = 6054
	DiagnosticCodeNumericSeparatorsAreNotAllowedHere DiagnosticCode = 6188
	DiagnosticCodeMultipleConsecutiveNumericSeparatorsAreNotPermitted DiagnosticCode = 6189
	DiagnosticCodeThisExpressionIsNotCallableBecauseItIsAGetAccessorDidYouMeanToUseItWithout DiagnosticCode = 6234
	DiagnosticCodeSuperMustBeCalledBeforeAccessingThisInTheConstructorOfADerivedClass DiagnosticCode = 17009
	DiagnosticCodeSuperMustBeCalledBeforeAccessingAPropertyOfSuperInTheConstructorOfADerivedClass DiagnosticCode = 17011
)

// DiagnosticCodeToString returns the message template for the given code.
func DiagnosticCodeToString(code DiagnosticCode) string {
	switch code {
	case 100:
		return "Not implemented: {0}"
	case 101:
		return "Operation is unsafe."
	case 102:
		return "User-defined: {0}"
	case 103:
		return "Feature '{0}' is not enabled."
	case 104:
		return "Low memory limit exceeded by static data: {0} > {1}"
	case 105:
		return "Module requires at least '{0}' pages of initial memory."
	case 106:
		return "Module requires at least '{0}' pages of maximum memory."
	case 107:
		return "Shared memory requires maximum memory to be defined."
	case 108:
		return "Shared memory requires feature 'threads' to be enabled."
	case 109:
		return "Transform '{0}': {1}"
	case 110:
		return "Start function name '{0}' is invalid or conflicts with another export."
	case 111:
		return "Element '{0}' not found."
	case 112:
		return "Exchange of '{0}' values is not supported by all embeddings"
	case 200:
		return "Conversion from type '{0}' to '{1}' requires an explicit cast."
	case 201:
		return "Conversion from type '{0}' to '{1}' will require an explicit cast when switching between 32/64-bit."
	case 202:
		return "Type '{0}' cannot be changed to type '{1}'."
	case 203:
		return "Operation '{0}' cannot be applied to type '{1}'."
	case 204:
		return "Type '{0}' cannot be nullable."
	case 206:
		return "Mutable value cannot be inlined."
	case 207:
		return "Unmanaged classes cannot extend managed classes and vice-versa."
	case 208:
		return "Unmanaged classes cannot implement interfaces."
	case 209:
		return "Invalid regular expression flags."
	case 210:
		return "Expression is never 'null'."
	case 211:
		return "Class '{0}' is final and cannot be extended."
	case 212:
		return "Decorator '{0}' is not valid here."
	case 213:
		return "Duplicate decorator."
	case 214:
		return "Type '{0}' is illegal in this context."
	case 215:
		return "Optional parameter must have an initializer."
	case 216:
		return "Class '{0}' cannot declare a constructor when instantiated from an object literal."
	case 217:
		return "Function '{0}' cannot be inlined into itself."
	case 218:
		return "Cannot access method '{0}' without calling it as it requires 'this' to be set."
	case 219:
		return "Optional properties are not supported."
	case 220:
		return "Expression must be a compile-time constant."
	case 221:
		return "Type '{0}' is not a function index or function reference."
	case 222:
		return "'{0}' must be a value between '{1}' and '{2}' inclusive."
	case 223:
		return "'{0}' must be a power of two."
	case 224:
		return "'{0}' is not a valid operator."
	case 225:
		return "Expression cannot be represented by a type."
	case 226:
		return "Expression resolves to unusual type '{0}'."
	case 227:
		return "Array literal expected."
	case 228:
		return "Function '{0}' is virtual and will not be inlined."
	case 229:
		return "Property '{0}' only has a setter and is missing a getter."
	case 230:
		return "'{0}' keyword cannot be used here."
	case 231:
		return "A class with a constructor explicitly returning something else than 'this' must be '@final'."
	case 233:
		return "Property '{0}' is always assigned before being used."
	case 234:
		return "Expression does not compile to a value at runtime."
	case 235:
		return "Only variables, functions and enums become WebAssembly module exports."
	case 236:
		return "Literal '{0}' does not fit into 'i64' or 'u64' types."
	case 237:
		return "Index signature accessors in type '{0}' differ in types."
	case 238:
		return "Initializer, definitive assignment or nullable type expected."
	case 239:
		return "Definitive assignment has no effect on local variables."
	case 240:
		return "Ambiguous operator overload '{0}' (conflicting overloads '{1}' and '{2}')."
	case 901:
		return "Importing the table disables some indirect call optimizations."
	case 902:
		return "Exporting the table disables some indirect call optimizations."
	case 903:
		return "Expression compiles to a dynamic check at runtime."
	case 904:
		return "Indexed access may involve bounds checking."
	case 905:
		return "Explicitly returning constructor drops 'this' allocation."
	case 906:
		return "Unnecessary definite assignment."
	case 907:
		return "'NaN' does not compare equal to any other value including itself. Use isNaN(x) instead."
	case 908:
		return "Comparison with -0.0 is sign insensitive. Use Object.is(x, -0.0) if the sign matters."
	case 1002:
		return "Unterminated string literal."
	case 1003:
		return "Identifier expected."
	case 1005:
		return "'{0}' expected."
	case 1006:
		return "A file cannot have a reference to itself."
	case 1009:
		return "Trailing comma not allowed."
	case 1012:
		return "Unexpected token."
	case 1014:
		return "A rest parameter must be last in a parameter list."
	case 1015:
		return "Parameter cannot have question mark and initializer."
	case 1016:
		return "A required parameter cannot follow an optional parameter."
	case 1031:
		return "'{0}' modifier cannot appear on class elements of this kind."
	case 1036:
		return "Statements are not allowed in ambient contexts."
	case 1039:
		return "Initializers are not allowed in ambient contexts."
	case 1042:
		return "'{0}' modifier cannot be used here."
	case 1047:
		return "A rest parameter cannot be optional."
	case 1048:
		return "A rest parameter cannot have an initializer."
	case 1049:
		return "A 'set' accessor must have exactly one parameter."
	case 1052:
		return "A 'set' accessor parameter cannot have an initializer."
	case 1054:
		return "A 'get' accessor cannot have parameters."
	case 1061:
		return "Enum member must have initializer."
	case 1092:
		return "Type parameters cannot appear on a constructor declaration."
	case 1093:
		return "Type annotation cannot appear on a constructor declaration."
	case 1094:
		return "An accessor cannot have type parameters."
	case 1095:
		return "A 'set' accessor cannot have a return type annotation."
	case 1098:
		return "Type parameter list cannot be empty."
	case 1099:
		return "Type argument list cannot be empty."
	case 1104:
		return "A 'continue' statement can only be used within an enclosing iteration statement."
	case 1105:
		return "A 'break' statement can only be used within an enclosing iteration or switch statement."
	case 1108:
		return "A 'return' statement can only be used within a function body."
	case 1109:
		return "Expression expected."
	case 1110:
		return "Type expected."
	case 1113:
		return "A 'default' clause cannot appear more than once in a 'switch' statement."
	case 1114:
		return "Duplicate label '{0}'."
	case 1120:
		return "An export assignment cannot have modifiers."
	case 1121:
		return "Octal literals are not allowed in strict mode."
	case 1124:
		return "Digit expected."
	case 1125:
		return "Hexadecimal digit expected."
	case 1126:
		return "Unexpected end of text."
	case 1127:
		return "Invalid character."
	case 1130:
		return "'case' or 'default' expected."
	case 1034:
		return "'super' must be followed by an argument list or member access."
	case 1038:
		return "A 'declare' modifier cannot be used in an already ambient context."
	case 1140:
		return "Type argument expected."
	case 1141:
		return "String literal expected."
	case 1142:
		return "Line break not permitted here."
	case 1146:
		return "Declaration expected."
	case 1155:
		return "'const' declarations must be initialized."
	case 1161:
		return "Unterminated regular expression literal."
	case 1263:
		return "Declarations with initializers cannot also have definite assignment assertions."
	case 1176:
		return "Interface declaration cannot have 'implements' clause."
	case 1177:
		return "Binary digit expected."
	case 1178:
		return "Octal digit expected."
	case 1183:
		return "An implementation cannot be declared in ambient contexts."
	case 1190:
		return "The variable declaration of a 'for...of' statement cannot have an initializer."
	case 1198:
		return "An extended Unicode escape value must be between 0x0 and 0x10FFFF inclusive."
	case 1199:
		return "Unterminated Unicode escape sequence."
	case 1206:
		return "Decorators are not valid here."
	case 1242:
		return "'abstract' modifier can only appear on a class, method, or property declaration."
	case 1245:
		return "Method '{0}' cannot have an implementation because it is marked abstract."
	case 1246:
		return "An interface property cannot have an initializer."
	case 1255:
		return "A definite assignment assertion '!' is not permitted in this context."
	case 1311:
		return "A class may only extend another class."
	case 1317:
		return "A parameter property cannot be declared using a rest parameter."
	case 1319:
		return "A default export can only be used in a module."
	case 1345:
		return "An expression of type '{0}' cannot be tested for truthiness."
	case 1351:
		return "An identifier or keyword cannot immediately follow a numeric literal."
	case 2300:
		return "Duplicate identifier '{0}'."
	case 2304:
		return "Cannot find name '{0}'."
	case 2305:
		return "Module '{0}' has no exported member '{1}'."
	case 2312:
		return "An interface can only extend an interface."
	case 2314:
		return "Generic type '{0}' requires {1} type argument(s)."
	case 2315:
		return "Type '{0}' is not generic."
	case 2322:
		return "Type '{0}' is not assignable to type '{1}'."
	case 2325:
		return "Property '{0}' is private in type '{1}' but not in type '{2}'."
	case 2329:
		return "Index signature is missing in type '{0}'."
	case 2332:
		return "'this' cannot be referenced in current location."
	case 2333:
		return "'this' cannot be referenced in constructor arguments."
	case 2335:
		return "'super' can only be referenced in a derived class."
	case 2336:
		return "'super' cannot be referenced in constructor arguments."
	case 2337:
		return "Super calls are not permitted outside constructors or in nested functions inside constructors."
	case 2339:
		return "Property '{0}' does not exist on type '{1}'."
	case 2341:
		return "Property '{0}' is private and only accessible within class '{1}'."
	case 2349:
		return "Cannot invoke an expression whose type lacks a call signature. Type '{0}' has no compatible call signatures."
	case 2351:
		return "This expression is not constructable."
	case 2355:
		return "A function whose declared type is not 'void' must return a value."
	case 2357:
		return "The operand of an increment or decrement operator must be a variable or a property access."
	case 2364:
		return "The left-hand side of an assignment expression must be a variable or a property access."
	case 2365:
		return "Operator '{0}' cannot be applied to types '{1}' and '{2}'."
	case 2376:
		return "A 'super' call must be the first statement in the constructor."
	case 2377:
		return "Constructors for derived classes must contain a 'super' call."
	case 2380:
		return "'get' and 'set' accessor must have the same type."
	case 2385:
		return "Overload signatures must all be public, private or protected."
	case 2390:
		return "Constructor implementation is missing."
	case 2391:
		return "Function implementation is missing or not immediately following the declaration."
	case 2392:
		return "Multiple constructor implementations are not allowed."
	case 2393:
		return "Duplicate function implementation."
	case 2394:
		return "This overload signature is not compatible with its implementation signature."
	case 2395:
		return "Individual declarations in merged declaration '{0}' must be all exported or all local."
	case 2416:
		return "Property '{0}' in type '{1}' is not assignable to the same property in base type '{2}'."
	case 2422:
		return "A class can only implement an interface."
	case 2434:
		return "A namespace declaration cannot be located prior to a class or function with which it is merged."
	case 2442:
		return "Types have separate declarations of a private property '{0}'."
	case 2444:
		return "Property '{0}' is protected in type '{1}' but public in type '{2}'."
	case 2445:
		return "Property '{0}' is protected and only accessible within class '{1}' and its subclasses."
	case 2448:
		return "Variable '{0}' used before its declaration."
	case 2451:
		return "Cannot redeclare block-scoped variable '{0}'"
	case 2453:
		return "The type argument for type parameter '{0}' cannot be inferred from the usage. Consider specifying the type arguments explicitly."
	case 2454:
		return "Variable '{0}' is used before being assigned."
	case 2456:
		return "Type alias '{0}' circularly references itself."
	case 2460:
		return "Type '{0}' has no property '{1}'."
	case 2469:
		return "The '{0}' operator cannot be applied to type '{1}'."
	case 2474:
		return "In 'const' enum declarations member initializer must be constant expression."
	case 2476:
		return "A const enum member can only be accessed using a string literal."
	case 2484:
		return "Export declaration conflicts with exported declaration of '{0}'."
	case 2506:
		return "'{0}' is referenced directly or indirectly in its own base expression."
	case 2511:
		return "Cannot create an instance of an abstract class."
	case 2515:
		return "Non-abstract class '{0}' does not implement inherited abstract member '{1}' from '{2}'."
	case 2531:
		return "Object is possibly 'null'."
	case 2540:
		return "Cannot assign to '{0}' because it is a constant or a read-only property."
	case 2541:
		return "The target of an assignment must be a variable or a property access."
	case 2542:
		return "Index signature in type '{0}' only permits reading."
	case 2554:
		return "Expected {0} arguments, but got {1}."
	case 2555:
		return "Expected at least {0} arguments, but got {1}."
	case 2558:
		return "Expected {0} type arguments, but got {1}."
	case 2564:
		return "Property '{0}' has no initializer and is not assigned in the constructor before 'this' is used or returned."
	case 2565:
		return "Property '{0}' is used before being assigned."
	case 2610:
		return "'{0}' is defined as an accessor in class '{1}', but is overridden here in '{2}' as an instance property."
	case 2611:
		return "'{0}' is defined as a property in class '{1}', but is overridden here in '{2}' as an accessor."
	case 2651:
		return "A member initializer in a enum declaration cannot reference members declared after it, including members defined in other enums."
	case 2673:
		return "Constructor of class '{0}' is private and only accessible within the class declaration."
	case 2674:
		return "Constructor of class '{0}' is protected and only accessible within the class declaration."
	case 2675:
		return "Cannot extend a class '{0}'. Class constructor is marked as private."
	case 2685:
		return "The 'this' types of each signature are incompatible."
	case 2694:
		return "Namespace '{0}' has no exported member '{1}'."
	case 2695:
		return "Namespace can only have declarations."
	case 2706:
		return "Required type parameters may not follow optional type parameters."
	case 2718:
		return "Duplicate property '{0}'."
	case 2741:
		return "Property '{0}' is missing in type '{1}' but required in type '{2}'."
	case 2757:
		return "Type '{0}' has no call signatures."
	case 2808:
		return "Get accessor '{0}' must be at least as accessible as the setter."
	case 4117:
		return "This member cannot have an 'override' modifier because it is not declared in the base class '{0}'."
	case 6054:
		return "File '{0}' not found."
	case 6188:
		return "Numeric separators are not allowed here."
	case 6189:
		return "Multiple consecutive numeric separators are not permitted."
	case 6234:
		return "This expression is not callable because it is a 'get' accessor. Did you mean to use it without '()'?"
	case 17009:
		return "'super' must be called before accessing 'this' in the constructor of a derived class."
	case 17011:
		return "'super' must be called before accessing a property of 'super' in the constructor of a derived class."
	default:
		return ""
	}
}
