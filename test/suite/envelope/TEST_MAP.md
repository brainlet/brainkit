# envelope test map

Suite for `sdk.Envelope` — the structured error reply format every
Call-path error traverses. Covers both the shape on the wire and
the typed error decoding on the Go side.

## Files

- `run.go` — exports `Run(t, env)`; registers all tests.
- `envelope_test.go` — standalone entry: `TestEnvelope`.
- `shape.go` — wire shape round trips (`EnvelopeOK`, `EnvelopeError`).
- `typed_errors.go` — typed error replies (`NotFound`, `Validation`).
- `call.go` — `brainkit.Call` return value decoding on envelope errors.

## Tests

| Name | What it checks |
|------|----------------|
| `not_found_round_trip` | NotFound error payload encodes + decodes through the envelope |
| `validation_error_round_trip` | Validation error carries `details` through the envelope |
| `unknown_code_becomes_bus_error` | Unknown error code decodes into the generic bus error type |
| `success_reply_is_envelope` | Successful replies are wrapped in an envelope (not raw payload) |
| `error_reply_is_envelope` | Error replies are wrapped in an envelope with a code |
| `envelope_metadata_flag_present` | `_envelope: true` flag is emitted so decoders can branch safely |
| `call_returns_typed_not_found` | `brainkit.Call` returns `*sdkerrors.NotFound` for NotFound replies |
| `call_returns_typed_validation` | `brainkit.Call` returns `*sdkerrors.Validation` with details |
| `call_returns_bus_error_on_unknown_code` | Unknown codes collapse to `sdkerrors.BusError` |

## Adding a test

1. Add function to the right .go file by topic (`shape.go`,
   `typed_errors.go`, `call.go`).
2. Register in `run.go` under the matching subtest group.
3. Update this file.
