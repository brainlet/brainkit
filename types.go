package brainkit

import "encoding/json"

// Result is the generic return from sandbox Eval.
type Result struct {
	Value json.RawMessage
	Text  string
}
