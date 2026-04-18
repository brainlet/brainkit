package harness

import (
	"encoding/json"

	tools "github.com/brainlet/brainkit/internal/tools"
)

// StateSchemaOf generates a JSON Schema map from a Go struct type.
// The struct's fields define the Harness state shape. Uses struct tags:
//
//	`json:"fieldName"` — property name in the schema
//	`desc:"..."` — field description
//	`default:"value"` — default value (coerced to the field's type)
//	`optional:"true"` — marks the field as optional (not required)
//
// Example:
//
//	type MyState struct {
//	    ProjectName string  `json:"projectName" default:""`
//	    Yolo        bool    `json:"yolo" default:"true"`
//	    Counter     float64 `json:"counter" default:"0"`
//	    Tasks       []Task  `json:"tasks" default:"[]"`
//	}
//
//	cfg := brainkit.HarnessConfig{
//	    StateSchema:  brainkit.StateSchemaOf[MyState](),
//	    InitialState: MyState{ProjectName: "my-project", Yolo: true},
//	}
func StateSchemaOf[T any]() map[string]any {
	var zero T
	raw := tools.StructToJSONSchema(zero)
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		return map[string]any{"type": "object"}
	}
	return schema
}
