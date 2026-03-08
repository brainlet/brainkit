// Ported from: packages/ai/src/telemetry/assemble-operation-name.ts
package telemetry

// AssembleOperationNameResult holds the assembled operation name attributes.
type AssembleOperationNameResult struct {
	// OperationName is the standardized operation name.
	OperationName string
	// ResourceName is the resource name (function ID).
	ResourceName *string
	// AIOperationID is the AI SDK specific operation ID.
	AIOperationID string
	// AITelemetryFunctionID is the AI SDK specific function ID.
	AITelemetryFunctionID *string
}

// AssembleOperationName creates standardized operation and resource name attributes.
func AssembleOperationName(operationID string, telemetry *TelemetrySettings) AssembleOperationNameResult {
	var functionID *string
	if telemetry != nil {
		functionID = telemetry.FunctionID
	}

	operationName := operationID
	if functionID != nil {
		operationName = operationID + " " + *functionID
	}

	return AssembleOperationNameResult{
		OperationName:         operationName,
		ResourceName:          functionID,
		AIOperationID:         operationID,
		AITelemetryFunctionID: functionID,
	}
}

// ToAttributes converts the result to an Attributes map matching the TypeScript keys.
func (r AssembleOperationNameResult) ToAttributes() Attributes {
	attrs := Attributes{
		"operation.name":            r.OperationName,
		"ai.operationId":           r.AIOperationID,
	}
	if r.ResourceName != nil {
		attrs["resource.name"] = *r.ResourceName
	}
	if r.AITelemetryFunctionID != nil {
		attrs["ai.telemetry.functionId"] = *r.AITelemetryFunctionID
	}
	return attrs
}
