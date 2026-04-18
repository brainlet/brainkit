module github.com/brainlet/brainkit/examples/plugin-author

go 1.26.0

require github.com/brainlet/brainkit/sdk v0.0.0-00010101000000-000000000000

require (
	github.com/brainlet/brainkit v0.0.0-00010101000000-000000000000 // indirect
	github.com/coder/websocket v1.8.14 // indirect
	github.com/google/uuid v1.6.0 // indirect
)

replace (
	github.com/brainlet/brainkit => ../..
	github.com/brainlet/brainkit/sdk => ../../sdk
	github.com/brainlet/brainkit/vendor_quickjs => ../../vendor_quickjs
	github.com/brainlet/brainkit/vendor_typescript => ../../vendor_typescript
)
