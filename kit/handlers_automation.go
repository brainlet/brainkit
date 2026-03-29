package kit

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/brainlet/brainkit/kit/workflow"
	"github.com/brainlet/brainkit/sdk/messages"
)

// AutomationDomain handles automation.deploy/teardown/list/info bus commands.
type AutomationDomain struct {
	kit *Kernel

	mu        sync.Mutex
	deployed  map[string]*workflow.DeployedAutomation
}

func newAutomationDomain(k *Kernel) *AutomationDomain {
	return &AutomationDomain{
		kit:      k,
		deployed: make(map[string]*workflow.DeployedAutomation),
	}
}

func (d *AutomationDomain) Deploy(ctx context.Context, req messages.AutomationDeployMsg) (*messages.AutomationDeployResp, error) {
	var manifest workflow.AutomationManifest
	if len(req.Manifest) > 0 {
		if err := json.Unmarshal(req.Manifest, &manifest); err != nil {
			return nil, fmt.Errorf("automation.deploy: invalid manifest: %w", err)
		}
	} else {
		return nil, fmt.Errorf("automation.deploy: manifest is required")
	}

	if manifest.Name == "" {
		return nil, fmt.Errorf("automation.deploy: name is required")
	}

	// Compile workflow source to WASM via the AS compiler
	workflowSource := req.WorkflowSource
	if workflowSource == "" {
		return nil, fmt.Errorf("automation.deploy: workflowSource is required")
	}

	compileResp, err := d.kit.wasmDomainInst.Compile(ctx, messages.WasmCompileMsg{
		Source:  workflowSource,
		Options: &messages.WasmCompileOpts{Name: manifest.Name},
	})
	if err != nil {
		return nil, fmt.Errorf("automation.deploy: compile workflow: %w", err)
	}

	// Register the workflow in the engine
	timeout := time.Duration(manifest.Workflow.Timeout) * time.Second
	if timeout == 0 {
		timeout = 24 * time.Hour
	}

	// Get the compiled binary from WASM service
	modInfo, _ := d.kit.GetWASMModule(manifest.Name)
	var binary []byte
	if modInfo != nil {
		d.kit.wasm.mu.Lock()
		if mod, ok := d.kit.wasm.modules[manifest.Name]; ok {
			binary = make([]byte, len(mod.Binary))
			copy(binary, mod.Binary)
		}
		d.kit.wasm.mu.Unlock()
	}

	workflowID := manifest.Name
	d.kit.workflowEngine.RegisterWorkflow(workflow.WorkflowDef{
		ID:        workflowID,
		Name:      manifest.Name,
		Binary:    binary,
		EntryFunc: "processLead", // default, could be configurable
		Triggers:  manifest.Workflow.Triggers,
		Timeout:   timeout,
		MaxRetries: manifest.Workflow.Retries,
	})

	// Deploy admin .ts if provided
	adminSource := ""
	if req.AdminSource != "" {
		adminSource = manifest.Name + "/admin.ts"
		if _, err := d.kit.Deploy(ctx, adminSource, req.AdminSource); err != nil {
			return nil, fmt.Errorf("automation.deploy: deploy admin: %w", err)
		}
	}

	deployed := &workflow.DeployedAutomation{
		Manifest:    manifest,
		WorkflowID:  workflowID,
		AdminSource: adminSource,
		DeployedAt:  time.Now(),
		Status:      "active",
	}

	d.mu.Lock()
	d.deployed[manifest.Name] = deployed
	d.mu.Unlock()

	_ = compileResp // used for side effect (compilation)

	return &messages.AutomationDeployResp{
		Deployed:   true,
		WorkflowID: workflowID,
	}, nil
}

func (d *AutomationDomain) Teardown(ctx context.Context, req messages.AutomationTeardownMsg) (*messages.AutomationTeardownResp, error) {
	d.mu.Lock()
	deployed, ok := d.deployed[req.Name]
	if !ok {
		d.mu.Unlock()
		return nil, fmt.Errorf("automation %q not deployed", req.Name)
	}
	delete(d.deployed, req.Name)
	d.mu.Unlock()

	// Unregister workflow
	d.kit.workflowEngine.UnregisterWorkflow(deployed.WorkflowID)

	// Teardown admin .ts
	if deployed.AdminSource != "" {
		d.kit.Teardown(ctx, deployed.AdminSource)
	}

	// Remove WASM module
	d.kit.RemoveWASMModule(req.Name)

	return &messages.AutomationTeardownResp{Removed: true}, nil
}

func (d *AutomationDomain) List(_ context.Context, _ messages.AutomationListMsg) (*messages.AutomationListResp, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	infos := make([]messages.AutomationInfo, 0, len(d.deployed))
	for _, dep := range d.deployed {
		runs := d.kit.workflowEngine.ListRuns()
		activeCount := 0
		for _, r := range runs {
			if r.WorkflowID == dep.WorkflowID {
				activeCount++
			}
		}
		infos = append(infos, messages.AutomationInfo{
			Name:       dep.Manifest.Name,
			Status:     dep.Status,
			ActiveRuns: activeCount,
		})
	}
	return &messages.AutomationListResp{Automations: infos}, nil
}

func (d *AutomationDomain) Info(_ context.Context, req messages.AutomationInfoMsg) (*messages.AutomationInfoResp, error) {
	d.mu.Lock()
	dep, ok := d.deployed[req.Name]
	d.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("automation %q not deployed", req.Name)
	}
	manifestJSON, _ := json.Marshal(dep.Manifest)
	return &messages.AutomationInfoResp{
		Manifest: manifestJSON,
		Status:   dep.Status,
	}, nil
}
