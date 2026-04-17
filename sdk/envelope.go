package sdk

import (
	"encoding/json"
	"errors"

	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

// Envelope is the single wire shape for all bus command responses.
// Exactly one of Data (ok=true) or Error (ok=false) is populated.
// See designs/08-errors.md.
type Envelope struct {
	Ok    bool            `json:"ok"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error *EnvelopeError  `json:"error,omitempty"`
}

// EnvelopeError is the error half of the wire envelope.
type EnvelopeError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// EnvelopeOK wraps a payload as a success envelope.
// If data is already a JSON-serializable value it is marshaled; if nil, Data
// is set to JSON null.
func EnvelopeOK(data any) Envelope {
	var raw json.RawMessage
	if data == nil {
		raw = json.RawMessage("null")
	} else if rm, ok := data.(json.RawMessage); ok {
		raw = rm
	} else if b, ok := data.([]byte); ok {
		raw = json.RawMessage(b)
	} else {
		b, err := json.Marshal(data)
		if err != nil {
			raw = json.RawMessage("null")
		} else {
			raw = b
		}
	}
	return Envelope{Ok: true, Data: raw}
}

// EnvelopeErr builds a failure envelope from code/message/details.
func EnvelopeErr(code, message string, details map[string]any) Envelope {
	return Envelope{Ok: false, Error: &EnvelopeError{
		Code: code, Message: message, Details: details,
	}}
}

// EncodeEnvelope marshals an Envelope to JSON bytes.
func EncodeEnvelope(e Envelope) ([]byte, error) {
	return json.Marshal(e)
}

// DecodeEnvelope parses JSON bytes into an Envelope. If the payload is not
// a valid envelope shape, returns an error.
func DecodeEnvelope(payload []byte) (Envelope, error) {
	var e Envelope
	if err := json.Unmarshal(payload, &e); err != nil {
		return Envelope{}, err
	}
	return e, nil
}

// IsEnvelope reports whether the payload looks like a wire envelope
// (has a top-level `ok` field). Used for decode paths that accept raw or
// enveloped JSON.
func IsEnvelope(payload []byte) bool {
	var probe struct {
		Ok *bool `json:"ok"`
	}
	if err := json.Unmarshal(payload, &probe); err != nil {
		return false
	}
	return probe.Ok != nil
}

// FromEnvelope reconstructs a typed Go error from a wire envelope. Returns
// nil when e.Ok is true. Unknown codes become *sdkerrors.BusError so callers
// can still inspect Code/Message/Details via errors.As.
func FromEnvelope(e Envelope) error {
	if e.Ok {
		return nil
	}
	if e.Error == nil {
		return &sdkerrors.BusError{Code_: "INTERNAL_ERROR", Message: "unknown error"}
	}
	code := e.Error.Code
	msg := e.Error.Message
	d := e.Error.Details
	switch code {
	case "NOT_FOUND":
		return &sdkerrors.NotFoundError{Resource: str(d, "resource"), Name: str(d, "name")}
	case "ALREADY_EXISTS":
		return &sdkerrors.AlreadyExistsError{Resource: str(d, "resource"), Name: str(d, "name"), Hint: str(d, "hint")}
	case "VALIDATION_ERROR":
		return &sdkerrors.ValidationError{Field: str(d, "field"), Message: str(d, "message")}
	case "TIMEOUT":
		return &sdkerrors.TimeoutError{Operation: str(d, "operation")}
	case "WORKSPACE_ESCAPE":
		return &sdkerrors.WorkspaceEscapeError{Path: str(d, "path")}
	case "NOT_CONFIGURED":
		return &sdkerrors.NotConfiguredError{Feature: str(d, "feature")}
	case "TRANSPORT_ERROR":
		return &sdkerrors.TransportError{Operation: str(d, "operation"), Cause: errors.New(msg)}
	case "PERSISTENCE_ERROR":
		return &sdkerrors.PersistenceError{Operation: str(d, "operation"), Source: str(d, "source"), Cause: errors.New(msg)}
	case "DEPLOY_ERROR":
		return &sdkerrors.DeployError{Source: str(d, "source"), Phase: str(d, "phase"), Cause: errors.New(msg)}
	case "BRIDGE_ERROR":
		return &sdkerrors.BridgeError{Function: str(d, "function"), Cause: errors.New(msg)}
	case "COMPILER_ERROR":
		return &sdkerrors.CompilerError{Cause: errors.New(msg)}
	case "CYCLE_DETECTED":
		depth := 0
		if v, ok := d["depth"].(float64); ok {
			depth = int(v)
		}
		return &sdkerrors.CycleDetectedError{Depth: depth}
	case "DECODE_ERROR":
		return &sdkerrors.DecodeError{Topic: str(d, "topic"), Cause: errors.New(msg)}
	default:
		return &sdkerrors.BusError{Code_: code, Message: msg, Details_: d}
	}
}

// ToEnvelope builds an outbound Envelope from (data, err). If err is nil,
// data is wrapped in an ok=true envelope. If err implements BrainkitError,
// its Code/Details are preserved; plain errors collapse to INTERNAL_ERROR.
func ToEnvelope(data any, err error) Envelope {
	if err == nil {
		return EnvelopeOK(data)
	}
	var bk sdkerrors.BrainkitError
	if errors.As(err, &bk) {
		return EnvelopeErr(bk.Code(), err.Error(), bk.Details())
	}
	return EnvelopeErr("INTERNAL_ERROR", err.Error(), nil)
}

func str(m map[string]any, k string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[k].(string); ok {
		return v
	}
	return ""
}
