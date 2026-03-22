package kit

import (
	"context"
	"encoding/json"

	"github.com/brainlet/brainkit/sdk/messages"
)

// WASMDomain wraps WASMService with typed domain methods.
type WASMDomain struct {
	kit     *Kernel
	service *WASMService
}

func newWASMDomain(k *Kernel, svc *WASMService) *WASMDomain {
	return &WASMDomain{kit: k, service: svc}
}

func (d *WASMDomain) Compile(ctx context.Context, req messages.WasmCompileMsg) (*messages.WasmCompileResp, error) {
	return decodeWASMResponse[messages.WasmCompileResp](d.service.handleCompile(ctx, mustMarshalJSON(req)))
}

func (d *WASMDomain) Run(ctx context.Context, req messages.WasmRunMsg) (*messages.WasmRunResp, error) {
	return decodeWASMResponse[messages.WasmRunResp](d.service.handleRun(ctx, mustMarshalJSON(req)))
}

func (d *WASMDomain) Deploy(ctx context.Context, req messages.WasmDeployMsg) (*messages.WasmDeployResp, error) {
	return decodeWASMResponse[messages.WasmDeployResp](d.service.handleDeploy(ctx, mustMarshalJSON(req)))
}

func (d *WASMDomain) Undeploy(ctx context.Context, req messages.WasmUndeployMsg) (*messages.WasmUndeployResp, error) {
	return decodeWASMResponse[messages.WasmUndeployResp](d.service.handleUndeploy(ctx, mustMarshalJSON(req)))
}

func (d *WASMDomain) List(ctx context.Context, req messages.WasmListMsg) (*messages.WasmListResp, error) {
	raw, err := d.service.handleList(ctx, mustMarshalJSON(req))
	if err != nil {
		return nil, err
	}
	var modules []messages.WasmModuleInfo
	if err := json.Unmarshal(raw, &modules); err != nil {
		return nil, err
	}
	return &messages.WasmListResp{Modules: modules}, nil
}

func (d *WASMDomain) Get(ctx context.Context, req messages.WasmGetMsg) (*messages.WasmGetResp, error) {
	return decodeWASMResponse[messages.WasmGetResp](d.service.handleGet(ctx, mustMarshalJSON(req)))
}

func (d *WASMDomain) Remove(ctx context.Context, req messages.WasmRemoveMsg) (*messages.WasmRemoveResp, error) {
	return decodeWASMResponse[messages.WasmRemoveResp](d.service.handleRemove(ctx, mustMarshalJSON(req)))
}

func (d *WASMDomain) Describe(ctx context.Context, req messages.WasmDescribeMsg) (*messages.WasmDescribeResp, error) {
	return decodeWASMResponse[messages.WasmDescribeResp](d.service.handleDescribe(ctx, mustMarshalJSON(req)))
}

func decodeWASMResponse[T any](raw json.RawMessage, err error) (*T, error) {
	if err != nil {
		return nil, err
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func mustMarshalJSON(v any) json.RawMessage {
	payload, _ := json.Marshal(v)
	return payload
}
