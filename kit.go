package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	agentembed "github.com/brainlet/brainkit/agent-embed"
	"github.com/brainlet/brainkit/bus"
	"github.com/brainlet/brainkit/internal/network"
	iplugin "github.com/brainlet/brainkit/internal/plugin"
	"github.com/brainlet/brainkit/internal/transport"
	iwasm "github.com/brainlet/brainkit/internal/wasm"
	"github.com/brainlet/brainkit/jsbridge"
	"github.com/brainlet/brainkit/libsql"
	mcppkg "github.com/brainlet/brainkit/mcp"
	"github.com/brainlet/brainkit/registry"
	"github.com/nats-io/nats.go"
)

// Kit is the brainkit execution engine.
// One Kit = one QuickJS runtime = one isolation boundary.
// All agents, AI calls, workflows share the same JS context.
type Kit struct {
	Bus         *bus.Bus
	Tools       *registry.ToolRegistry
	MCP         *mcppkg.MCPManager
	config      Config
	namespace   string
	callerID    string
	bridge      *jsbridge.Bridge
	agents      *agentembed.Sandbox
	wasm        *WASMService
	plugins     *iplugin.Manager
	storages    map[string]*libsql.Server // named embedded SQLite bridges
	agentReg    *agentRegistry
	deployments map[string]*deploymentInfo
	network     *network.HostServer      // gRPC server for incoming peer connections
	transport   *transport.GRPCTransport // transport with peer routing (nil if no network)
	discovery   transport.Discovery      // optional peer discovery

	mu     sync.Mutex
	closed bool
}

// New creates a Kit with one QuickJS runtime.
func New(cfg Config) (*Kit, error) {
	if cfg.Namespace == "" {
		cfg.Namespace = "user"
	}
	if cfg.CallerID == "" {
		cfg.CallerID = cfg.Namespace
	}

	var grpcTransport *GRPCTransport
	sharedBus := cfg.SharedBus
	if sharedBus == nil {
		switch cfg.Transport {
		case "nats":
			if cfg.NATS.URL == "" {
				return nil, fmt.Errorf("brainkit: NATS transport requires NATS.URL")
			}
			natsName := cfg.NATS.Name
			if natsName == "" {
				natsName = cfg.Name
			}
			nt, err := NewNATSTransport(cfg.NATS.URL, nats.Name(natsName))
			if err != nil {
				return nil, fmt.Errorf("brainkit: %w", err)
			}
			sharedBus = bus.NewBus(nt)
		default:
			if cfg.Network.Listen != "" || len(cfg.Network.Peers) > 0 || cfg.Network.Discovery.Type != "" {
				grpcTransport = NewGRPCTransport()
				sharedBus = bus.NewBus(grpcTransport)
			} else {
				sharedBus = bus.NewBus(bus.NewInProcessTransport())
			}
		}
	}
	if cfg.Name != "" {
		if err := sharedBus.RegisterName(cfg.Name); err != nil {
			return nil, fmt.Errorf("brainkit: %w", err)
		}
	}
	sharedTools := cfg.SharedTools
	if sharedTools == nil {
		sharedTools = registry.New()
	}

	k := &Kit{
		Bus:         sharedBus,
		Tools:       sharedTools,
		config:      cfg,
		namespace:   cfg.Namespace,
		callerID:    cfg.CallerID,
		storages:    make(map[string]*libsql.Server),
		agentReg:    newAgentRegistry(),
		deployments: make(map[string]*deploymentInfo),
		transport:   grpcTransport,
	}

	// Start network listener if configured
	if cfg.Network.Listen != "" {
		k.network = network.NewHostServer(k.Bus, cfg.Name)
		if err := k.network.Start(cfg.Network.Listen); err != nil {
			return nil, fmt.Errorf("brainkit: %w", err)
		}
	}

	// Connect to known peers
	for name, addr := range cfg.Network.Peers {
		if err := k.connectPeer(name, addr); err != nil {
			log.Printf("[kit] failed to connect to peer %q at %s: %v", name, addr, err)
		}
	}

	// Create discovery if configured
	switch cfg.Network.Discovery.Type {
	case "multicast":
		disc, err := NewMulticastDiscovery(cfg.Network.Discovery.ServiceName)
		if err != nil {
			log.Printf("[kit] multicast discovery failed: %v", err)
		} else {
			k.discovery = disc
		}
	case "static":
		if len(cfg.Network.Peers) > 0 {
			k.discovery = NewStaticDiscovery(cfg.Network.Peers)
		}
	}

	if k.discovery != nil && k.transport != nil {
		k.transport.Discovery = k.discovery
		k.transport.ConnectFunc = k.connectPeer
	}

	// Register self with discovery
	if k.discovery != nil && cfg.Network.Listen != "" && cfg.Name != "" {
		k.discovery.Register(Peer{
			Name:    cfg.Name,
			Address: k.network.Addr(),
		})
	}

	// Build provider config for agent-embed
	providers := make(map[string]agentembed.ProviderConfig)
	for name, pc := range cfg.Providers {
		providers[name] = agentembed.ProviderConfig{APIKey: pc.APIKey, BaseURL: pc.BaseURL}
	}

	// Create THE single agent-embed sandbox (one QuickJS runtime)
	agentSandbox, err := agentembed.NewSandbox(agentembed.SandboxConfig{
		Providers:    providers,
		EnvVars:      cfg.EnvVars,
		MaxStackSize: cfg.MaxStackSize,
	})
	if err != nil {
		return nil, fmt.Errorf("brainkit: create runtime: %w", err)
	}
	k.agents = agentSandbox
	k.bridge = agentSandbox.Bridge()

	// Register Go bridges for PLATFORM operations
	k.registerBridges()

	// Start embedded LibSQL bridges for configured storages
	for name, scfg := range cfg.Storages {
		if err := k.addStorageInternal(name, scfg); err != nil {
			// Clean up already-started storages
			for _, srv := range k.storages {
				srv.Close()
			}
			agentSandbox.Close()
			return nil, fmt.Errorf("brainkit: start storage %q: %w", name, err)
		}
	}

	// Inject observability config for brainlet-runtime.js to read
	obsEnabled := cfg.Observability.Enabled == nil || *cfg.Observability.Enabled
	obsStrategy := cfg.Observability.Strategy
	if obsStrategy == "" {
		obsStrategy = "realtime"
	}
	obsServiceName := cfg.Observability.ServiceName
	if obsServiceName == "" {
		obsServiceName = "brainkit"
	}
	k.bridge.Eval("__obs_config.js", fmt.Sprintf(
		`globalThis.__brainkit_obs_config = { enabled: %v, strategy: %q, serviceName: %q }`,
		obsEnabled, obsStrategy, obsServiceName,
	))

	// Load kit_runtime.js + register "kit" ES module
	if err := k.loadRuntime(); err != nil {
		agentSandbox.Close()
		return nil, err
	}

	// Create WASM service (compiler is lazy — only created on first wasm.compile)
	k.wasm = iwasm.NewService(&kitBusBridge{kit: k})

	// Register bus handlers
	k.registerHandlers()

	// Load persisted modules + shards from store (if configured)
	if cfg.Store != nil {
		if err := k.wasm.LoadFromStore(cfg.Store); err != nil {
			log.Printf("[brainkit] warning: failed to load persisted data: %v", err)
		}
	}

	// Connect to MCP servers and auto-register their tools
	if len(cfg.MCPServers) > 0 {
		k.MCP = mcppkg.New()
		for name, serverCfg := range cfg.MCPServers {
			if err := k.MCP.Connect(context.Background(), name, serverCfg); err != nil {
				// Log but don't fail — MCP servers may be unavailable
				continue
			}
			// Register each MCP tool in the ToolRegistry
			for _, tool := range k.MCP.ListToolsForServer(name) {
				toolCopy := tool // capture loop variable
				fullName := registry.ComposeName("mcp", toolCopy.ServerName, "1.0.0", toolCopy.Name)
				k.Tools.Register(registry.RegisteredTool{
					Name:        fullName,
					ShortName:   toolCopy.Name,
					Owner:       "mcp",
					Package:     toolCopy.ServerName,
					Version:     "1.0.0",
					Description: toolCopy.Description,
					InputSchema: toolCopy.InputSchema,
					Executor: &registry.GoFuncExecutor{
						Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
							return k.MCP.CallTool(ctx, toolCopy.ServerName, toolCopy.Name, input)
						},
					},
				})
			}
		}
	}

	// Start plugin manager
	if len(cfg.Plugins) > 0 {
		k.plugins = iplugin.NewManager(&kitPluginBridge{kit: k})
		k.plugins.StartAll(cfg.Plugins)
	}

	return k, nil
}

// Close shuts down the runtime and the bus.
func (k *Kit) Close() {
	k.mu.Lock()
	if k.closed {
		k.mu.Unlock()
		return
	}
	k.closed = true
	k.mu.Unlock()

	// Shutdown order: network/transport FIRST (kills gRPC streams so goroutines can exit),
	// then plugins, then bridge (waits for goroutines — now they can exit cleanly).
	if k.network != nil {
		k.network.Stop()
	}
	if k.discovery != nil {
		k.discovery.Close()
	}
	if k.agentReg != nil && k.agents != nil {
		k.agentReg.unregisterAllForKit(k.agents.ID())
	}
	if k.plugins != nil {
		k.plugins.StopAll()
	}
	if k.MCP != nil {
		k.MCP.Close()
	}
	if k.wasm != nil {
		k.wasm.Close()
	}
	if k.config.Store != nil {
		k.config.Store.Close()
	}
	if k.agents != nil {
		k.agents.Close()
	}
	for _, srv := range k.storages {
		srv.Close()
	}
	if k.config.Name != "" {
		k.Bus.UnregisterName(k.config.Name)
	}
	if k.config.SharedBus == nil {
		k.Bus.Close()
	}
}

// Namespace returns the Kit's namespace.
func (k *Kit) Namespace() string { return k.namespace }

// CallerID returns the Kit's identity for bus messages.
func (k *Kit) CallerID() string { return k.callerID }

// CreateAgent creates a persistent agent in the Kit's runtime.
func (k *Kit) CreateAgent(cfg agentembed.AgentConfig) (*agentembed.Agent, error) {
	return k.agents.CreateAgent(cfg)
}
