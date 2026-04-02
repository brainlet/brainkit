package brainkit

import (
	"context"
	"crypto/hmac"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	agentembed "github.com/brainlet/brainkit/internal/embed/agent"
	"github.com/brainlet/brainkit/internal/messaging"
	"github.com/brainlet/brainkit/internal/sdkerrors"
	"github.com/brainlet/brainkit/internal/jsbridge"
	"github.com/brainlet/brainkit/packages"
	"github.com/brainlet/brainkit/rbac"
	"github.com/brainlet/brainkit/secrets"
	"github.com/brainlet/brainkit/tracing"
	"github.com/brainlet/brainkit/workflow"
	"github.com/brainlet/brainkit/internal/libsql"
	mcppkg "github.com/brainlet/brainkit/internal/mcp"
	toolreg "github.com/brainlet/brainkit/internal/registry"
	provreg "github.com/brainlet/brainkit/registry"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"golang.org/x/time/rate"
)

// Kernel is the local brainkit runtime. Implements sdk.Runtime.
// It owns JS/WASM/runtime state and an internal Watermill transport.
type Kernel struct {
	// Domain handlers (internal — accessed via command catalog, not directly)
	// Category B: take interfaces, not *Kernel
	toolsDomain    *ToolsDomain
	agentsDomain   *AgentsDomain
	packagesDomain *PackagesDomain
	secretsDomain  *SecretsDomain
	workflowDomain *WorkflowDomain
	// Category C: stay on *Kernel (touch too many subsystems)
	wasmDomainInst      *WASMDomain
	lifecycle           *LifecycleDomain
	packageDeployDomain *PackageDeployDomain
	automationDomain    *AutomationDomain
	testingDomain       *TestingDomain
	// Eliminated (inlined into catalog): RBACDomain, TracingDomain, MCPDomain, RegistryDomain

	Tools     *toolreg.ToolRegistry
	packages       *packages.Manager
	workflowEngine *workflow.Engine
	hostFunctions  *workflow.HostFunctionRegistry
	mcp            *mcppkg.MCPManager
	providers *provreg.ProviderRegistry
	rbac                *rbac.Manager
	tracer              *tracing.Tracer
	busRateLimiters      map[string]*rate.Limiter // role → limiter
	wasmCommandAllowlist map[string]bool          // live — protected by mu
	replyHMACKey         []byte                   // 32-byte key for reply token HMAC; nil if RBAC not configured

	// Internal Watermill transport — always present
	transport      *messaging.Transport
	router         *message.Router
	remote         *messaging.RemoteClient
	host           *messaging.Host
	ownsTransport  bool // true if Kernel created the transport (false if injected by Node)

	config    KernelConfig
	namespace string
	callerID  string
	bridge    *jsbridge.Bridge
	agents    *agentembed.Sandbox
	wasm      *WASMService
	storages  map[string]*libsql.Server

	secretStore   secrets.SecretStore
	node          *Node   // optional back-reference, set by Node after creation
	currentSource string  // active deployment source for RBAC — set by subscribe callback

	deployments map[string]*deploymentInfo
	bridgeSubs  map[string]func()

	mu     sync.Mutex
	closed bool

	// Graceful shutdown
	activeHandlers atomic.Int64
	draining       atomic.Bool

	// Deployment ordering (for persistence)
	deployOrder atomic.Int32

	// Metrics
	pumpCycles atomic.Int64

	// Schedules
	schedules map[string]*scheduleEntry

	// Health
	startedAt time.Time
}

type scheduleEntry struct {
	PersistedSchedule
	timer *time.Timer
}

// enterHandler marks a bus handler as active.
// Returns false if draining — caller should drop the message.
func (k *Kernel) enterHandler() bool {
	if k.draining.Load() {
		return false
	}
	k.activeHandlers.Add(1)
	return true
}

// exitHandler marks a bus handler as complete.
func (k *Kernel) exitHandler() {
	k.activeHandlers.Add(-1)
}

// IsDraining returns true during the drain phase of Shutdown.
func (k *Kernel) IsDraining() bool {
	return k.draining.Load()
}

// SetDraining sets the draining state. Used for testing.
func (k *Kernel) SetDraining(v bool) {
	k.draining.Store(v)
}

// waitForDrain polls until all active handlers finish or ctx expires.
func (k *Kernel) waitForDrain(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			active := k.activeHandlers.Load()
			if active > 0 {
				log.Printf("[brainkit] drain timeout: %d handlers still active, forcing shutdown", active)
			}
			return
		case <-ticker.C:
			if k.activeHandlers.Load() == 0 {
				log.Printf("[brainkit] drain complete")
				return
			}
		}
	}
}

func (k *Kernel) nextDeployOrder() int {
	return int(k.deployOrder.Add(1))
}

// --- Scheduling ---

func parseScheduleExpression(expr string) (time.Duration, bool, error) {
	if strings.HasPrefix(expr, "every ") {
		d, err := time.ParseDuration(strings.TrimPrefix(expr, "every "))
		return d, false, err
	}
	if strings.HasPrefix(expr, "in ") {
		d, err := time.ParseDuration(strings.TrimPrefix(expr, "in "))
		return d, true, err
	}
	return 0, false, fmt.Errorf("unsupported schedule expression: %q (use 'every <duration>' or 'in <duration>')", expr)
}

func (k *Kernel) addSchedule(ps PersistedSchedule) {
	delay := time.Until(ps.NextFire)
	if delay < 0 {
		delay = 0
	}
	entry := &scheduleEntry{PersistedSchedule: ps}
	entry.timer = time.AfterFunc(delay, func() {
		k.fireSchedule(entry)
	})
	k.mu.Lock()
	k.schedules[ps.ID] = entry
	k.mu.Unlock()
}

func (k *Kernel) fireSchedule(entry *scheduleEntry) {
	if k.IsDraining() {
		return
	}
	k.publish(context.Background(), entry.Topic, entry.Payload)

	if entry.OneTime {
		k.mu.Lock()
		delete(k.schedules, entry.ID)
		k.mu.Unlock()
		if k.config.Store != nil {
			k.config.Store.DeleteSchedule(entry.ID)
		}
		return
	}

	entry.NextFire = time.Now().Add(entry.Duration)
	entry.timer.Reset(entry.Duration)
	if k.config.Store != nil {
		if err := k.config.Store.SaveSchedule(entry.PersistedSchedule); err != nil {
			k.persistenceError(context.Background(), "SaveSchedule", entry.ID, err)
		}
	}
}

func (k *Kernel) removeSchedule(id string) {
	k.mu.Lock()
	entry, ok := k.schedules[id]
	if ok {
		entry.timer.Stop()
		delete(k.schedules, id)
	}
	k.mu.Unlock()
	if ok && k.config.Store != nil {
		k.config.Store.DeleteSchedule(id)
	}
}

// Schedule creates a new scheduled bus message.
func (k *Kernel) Schedule(ctx context.Context, cfg ScheduleConfig) (string, error) {
	// Block scheduling to command topics — scheduled messages fire from Go
	// with no RBAC context, so they'd bypass all permission checks.
	if commandCatalog().HasCommand(cfg.Topic) {
		return "", &sdkerrors.ValidationError{Field: "topic", Message: cfg.Topic + " is a command topic; schedules cannot target commands"}
	}

	duration, oneTime, err := parseScheduleExpression(cfg.Expression)
	if err != nil {
		return "", err
	}
	id := cfg.ID
	if id == "" {
		id = uuid.NewString()
	}
	ps := PersistedSchedule{
		ID:         id,
		Expression: cfg.Expression,
		Duration:   duration,
		Topic:      cfg.Topic,
		Payload:    cfg.Payload,
		Source:     cfg.Source,
		CreatedAt:  time.Now(),
		NextFire:   time.Now().Add(duration),
		OneTime:    oneTime,
	}
	k.addSchedule(ps)
	if k.config.Store != nil {
		if err := k.config.Store.SaveSchedule(ps); err != nil {
			k.persistenceError(ctx, "SaveSchedule", id, err)
		}
	}
	return id, nil
}

// Unschedule cancels and removes a schedule.
func (k *Kernel) Unschedule(ctx context.Context, id string) error {
	k.removeSchedule(id)
	return nil
}

// ListSchedules returns all active schedules.
func (k *Kernel) ListSchedules() []PersistedSchedule {
	k.mu.Lock()
	defer k.mu.Unlock()
	result := make([]PersistedSchedule, 0, len(k.schedules))
	for _, entry := range k.schedules {
		result = append(result, entry.PersistedSchedule)
	}
	return result
}

func (k *Kernel) restoreSchedules() {
	schedules, err := k.config.Store.LoadSchedules()
	if err != nil {
		InvokeErrorHandler(k.config.ErrorHandler, &sdkerrors.PersistenceError{
			Operation: "LoadSchedules", Cause: err,
		}, ErrorContext{Operation: "LoadSchedules", Component: "kernel"})
		return
	}
	now := time.Now()
	restored := 0
	for _, s := range schedules {
		if s.OneTime {
			if s.NextFire.Before(now) {
				k.publish(context.Background(), s.Topic, s.Payload)
				k.config.Store.DeleteSchedule(s.ID)
				continue
			}
		} else {
			if s.NextFire.Before(now) {
				k.publish(context.Background(), s.Topic, s.Payload)
				s.NextFire = now.Add(s.Duration)
				k.config.Store.SaveSchedule(s)
			}
		}
		k.addSchedule(s)
		restored++
	}
	if restored > 0 {
		log.Printf("[brainkit] restored %d persisted schedules", restored)
	}
}

// --- Reply Tokens ---

// generateReplyToken creates an HMAC token for a specific reply context.
// Returns "" if RBAC is not configured (no signing needed).
func (k *Kernel) generateReplyToken(correlationID, replyTo, source string) string {
	if k.replyHMACKey == nil {
		return ""
	}
	mac := hmac.New(sha256.New, k.replyHMACKey)
	mac.Write([]byte(correlationID + "\x00" + replyTo + "\x00" + source))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// validateReplyToken checks if a reply token is valid for the given context.
// Returns nil if valid, error if invalid. Always valid if RBAC is not configured.
func (k *Kernel) validateReplyToken(correlationID, replyTo, source, token string) error {
	if k.replyHMACKey == nil {
		return nil // no RBAC = no token enforcement
	}
	if token == "" {
		return &sdkerrors.ReplyDeniedError{Source: source, ReplyTo: replyTo, CorrelationID: correlationID}
	}
	expected := k.generateReplyToken(correlationID, replyTo, source)
	if !hmac.Equal([]byte(token), []byte(expected)) {
		return &sdkerrors.ReplyDeniedError{Source: source, ReplyTo: replyTo, CorrelationID: correlationID}
	}
	return nil
}

// emitReplyDenied publishes the bus.reply.denied audit event.
func (k *Kernel) emitReplyDenied(replyTo, correlationID, source, reason string) {
	payload, _ := json.Marshal(messages.ReplyDeniedEvent{
		Source: source, Topic: replyTo, CorrelationID: correlationID, Reason: reason,
	})
	k.remote.PublishRaw(context.Background(), "bus.reply.denied", payload)
}

// handleHandlerFailure is called when a bus handler throws a JS exception.
func (k *Kernel) handleHandlerFailure(msg messages.Message, topic string, handlerErr error) {
	policy := k.findRetryPolicy(topic)

	retryCount := 0
	if msg.Metadata != nil {
		if rc, ok := msg.Metadata["retryCount"]; ok {
			retryCount, _ = strconv.Atoi(rc)
		}
	}

	// Emit failure event
	k.emitHandlerFailed(topic, handlerErr, retryCount, policy != nil && retryCount < policy.MaxRetries)

	// No retry policy — send error response immediately
	if policy == nil || policy.MaxRetries == 0 {
		log.Printf("[brainkit] handler error on %s: %v", topic, handlerErr)
		k.sendErrorResponse(msg, handlerErr)
		return
	}

	// Retries exhausted — dead letter + error response
	if retryCount >= policy.MaxRetries {
		log.Printf("[brainkit] handler exhausted on %s after %d retries: %v", topic, retryCount, handlerErr)
		k.deadLetter(msg, topic, handlerErr, retryCount, policy)
		k.sendErrorResponse(msg, fmt.Errorf("handler failed after %d retries: %w", retryCount, handlerErr))
		k.emitHandlerExhausted(topic, handlerErr, retryCount)
		return
	}

	// Retry with backoff
	delay := policy.computeDelay(retryCount)
	nextRetry := retryCount + 1

	log.Printf("[brainkit] handler failed on %s, retry %d/%d in %s: %v",
		topic, nextRetry, policy.MaxRetries, delay, handlerErr)

	k.bridge.Go(func(goCtx context.Context) {
		select {
		case <-time.After(delay):
		case <-goCtx.Done():
			return
		}
		k.remote.PublishRawWithMeta(context.Background(), topic, msg.Payload, map[string]string{
			"retryCount":    strconv.Itoa(nextRetry),
			"replyTo":       msg.Metadata["replyTo"],
			"correlationId": msg.Metadata["correlationId"],
		})
	})
}

func (k *Kernel) sendErrorResponse(msg messages.Message, err error) {
	replyTo := ""
	correlationID := ""
	if msg.Metadata != nil {
		replyTo = msg.Metadata["replyTo"]
		correlationID = msg.Metadata["correlationId"]
	}
	if replyTo == "" {
		return
	}
	errResp, _ := json.Marshal(map[string]any{
		"error": err.Error(),
	})
	k.ReplyRaw(context.Background(), replyTo, correlationID, errResp, true)
}

func (k *Kernel) findRetryPolicy(topic string) *RetryPolicy {
	if len(k.config.RetryPolicies) == 0 {
		return nil
	}
	if p, ok := k.config.RetryPolicies[topic]; ok {
		return &p
	}
	for pattern, p := range k.config.RetryPolicies {
		if strings.HasSuffix(pattern, ".*") {
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(topic, prefix) {
				p := p
				return &p
			}
		}
	}
	return nil
}

func (k *Kernel) deadLetter(msg messages.Message, topic string, err error, retries int, policy *RetryPolicy) {
	if policy.DeadLetterTopic == "" {
		return
	}
	dlPayload, _ := json.Marshal(map[string]any{
		"originalTopic":   topic,
		"originalPayload": json.RawMessage(msg.Payload),
		"error":           err.Error(),
		"retryCount":      retries,
		"source":          msg.CallerID,
		"exhaustedAt":     time.Now().Format(time.RFC3339),
	})
	k.publish(context.Background(), policy.DeadLetterTopic, dlPayload)
}

func (k *Kernel) emitHandlerFailed(topic string, err error, retryCount int, willRetry bool) {
	payload, _ := json.Marshal(messages.HandlerFailedEvent{
		Topic: topic, Source: k.callerID, Error: err.Error(),
		RetryCount: retryCount, WillRetry: willRetry,
	})
	k.publish(context.Background(), messages.HandlerFailedEvent{}.BusTopic(), payload)
}

func (k *Kernel) emitHandlerExhausted(topic string, err error, retryCount int) {
	payload, _ := json.Marshal(messages.HandlerExhaustedEvent{
		Topic: topic, Source: k.callerID, Error: err.Error(),
		RetryCount: retryCount,
	})
	k.publish(context.Background(), messages.HandlerExhaustedEvent{}.BusTopic(), payload)
}

func (p *RetryPolicy) computeDelay(retryCount int) time.Duration {
	delay := p.InitialDelay
	if delay == 0 {
		delay = 1 * time.Second
	}
	factor := p.BackoffFactor
	if factor == 0 {
		factor = 2.0
	}
	for i := 0; i < retryCount; i++ {
		delay = time.Duration(float64(delay) * factor)
	}
	if p.MaxDelay > 0 && delay > p.MaxDelay {
		delay = p.MaxDelay
	}
	return delay
}

// autoDetectProviders scans os.Getenv and cfg.EnvVars for known API key patterns
// and registers AI providers that aren't already explicitly configured.
// Priority: explicit AIProviders > EnvVars > os.Getenv.
func autoDetectProviders(cfg *KernelConfig) {
	if cfg.AIProviders == nil {
		cfg.AIProviders = make(map[string]provreg.AIProviderRegistration)
	}

	type providerMapping struct {
		name string
		typ  provreg.AIProviderType
		make func(apiKey string) any
	}

	mappings := map[string]providerMapping{
		"OPENAI_API_KEY":     {"openai", provreg.AIProviderOpenAI, func(k string) any { return provreg.OpenAIProviderConfig{APIKey: k} }},
		"ANTHROPIC_API_KEY":  {"anthropic", provreg.AIProviderAnthropic, func(k string) any { return provreg.AnthropicProviderConfig{APIKey: k} }},
		"GOOGLE_API_KEY":     {"google", provreg.AIProviderGoogle, func(k string) any { return provreg.GoogleProviderConfig{APIKey: k} }},
		"MISTRAL_API_KEY":    {"mistral", provreg.AIProviderMistral, func(k string) any { return provreg.MistralProviderConfig{APIKey: k} }},
		"GROQ_API_KEY":       {"groq", provreg.AIProviderGroq, func(k string) any { return provreg.GroqProviderConfig{APIKey: k} }},
		"DEEPSEEK_API_KEY":   {"deepseek", provreg.AIProviderDeepSeek, func(k string) any { return provreg.DeepSeekProviderConfig{APIKey: k} }},
		"XAI_API_KEY":        {"xai", provreg.AIProviderXAI, func(k string) any { return provreg.XAIProviderConfig{APIKey: k} }},
		"COHERE_API_KEY":     {"cohere", provreg.AIProviderCohere, func(k string) any { return provreg.CohereProviderConfig{APIKey: k} }},
		"PERPLEXITY_API_KEY": {"perplexity", provreg.AIProviderPerplexity, func(k string) any { return provreg.PerplexityProviderConfig{APIKey: k} }},
		"TOGETHER_API_KEY":   {"togetherai", provreg.AIProviderTogetherAI, func(k string) any { return provreg.TogetherAIProviderConfig{APIKey: k} }},
		"FIREWORKS_API_KEY":  {"fireworks", provreg.AIProviderFireworks, func(k string) any { return provreg.FireworksProviderConfig{APIKey: k} }},
		"CEREBRAS_API_KEY":   {"cerebras", provreg.AIProviderCerebras, func(k string) any { return provreg.CerebrasProviderConfig{APIKey: k} }},
	}

	for envKey, mapping := range mappings {
		if _, explicit := cfg.AIProviders[mapping.name]; explicit {
			continue
		}
		apiKey := ""
		if v, ok := cfg.EnvVars[envKey]; ok && v != "" {
			apiKey = v
		} else {
			apiKey = os.Getenv(envKey)
		}
		if apiKey == "" {
			continue
		}
		cfg.AIProviders[mapping.name] = provreg.AIProviderRegistration{
			Type:   mapping.typ,
			Config: mapping.make(apiKey),
		}
	}
}

// NewKernel creates a local runtime with no attached transport.
func NewKernel(cfg KernelConfig) (*Kernel, error) {
	if cfg.Namespace == "" {
		cfg.Namespace = "user"
	}
	if cfg.CallerID == "" {
		cfg.CallerID = cfg.Namespace
	}

	// Auto-detect AI providers from OS env + EnvVars before sandbox creation
	autoDetectProviders(&cfg)

	sharedTools := cfg.SharedTools
	if sharedTools == nil {
		sharedTools = toolreg.New()
	}

	wasmAllowlist := cfg.WASMCommandAllowlist
	if wasmAllowlist == nil {
		// Copy default so runtime mutations don't modify the package-level var
		wasmAllowlist = make(map[string]bool, len(DefaultWASMCommandAllowlist))
		for k, v := range DefaultWASMCommandAllowlist {
			wasmAllowlist[k] = v
		}
	}

	kernel := &Kernel{
		Tools:                sharedTools,
		config:               cfg,
		namespace:            cfg.Namespace,
		callerID:             cfg.CallerID,
		storages:             make(map[string]*libsql.Server),
		deployments:          make(map[string]*deploymentInfo),
		bridgeSubs:           make(map[string]func()),
		schedules:            make(map[string]*scheduleEntry),
		wasmCommandAllowlist: wasmAllowlist,
	}
	providers := make(map[string]agentembed.ProviderConfig)
	for name, reg := range cfg.AIProviders {
		pc := extractProviderCredentials(reg)
		providers[name] = agentembed.ProviderConfig{APIKey: pc.APIKey, BaseURL: pc.BaseURL}
	}

	agentSandbox, err := agentembed.NewSandbox(agentembed.SandboxConfig{
		Providers:    providers,
		EnvVars:      cfg.EnvVars,
		MaxStackSize: cfg.MaxStackSize,
		CWD:          cfg.FSRoot,
		FetchSpanHook: func(method, url string) func(int, error) {
			// Lazy reference — tracer is initialized after sandbox creation
			if kernel.tracer == nil {
				return nil
			}
			span := kernel.tracer.StartSpan("fetch", context.Background())
			span.SetAttribute("method", method)
			span.SetAttribute("url", url)
			return func(statusCode int, err error) {
				if statusCode > 0 {
					span.SetAttribute("status", strconv.Itoa(statusCode))
				}
				span.End(err)
			}
		},
	})
	if err != nil {
		return nil, fmt.Errorf("brainkit: create runtime: %w", err)
	}
	kernel.agents = agentSandbox
	kernel.bridge = agentSandbox.Bridge()

	kernel.agentsDomain = newAgentsDomain()

	kernel.registerBridges()

	// Start sqlite storage bridges (must happen before loadRuntime)
	bridgeURLs, err := kernel.initStorages(cfg)
	if err != nil {
		for _, srv := range kernel.storages {
			_ = srv.Close()
		}
		agentSandbox.Close()
		return nil, fmt.Errorf("brainkit: start storage: %w", err)
	}

	obsEnabled := cfg.Observability.Enabled == nil || *cfg.Observability.Enabled
	obsStrategy := cfg.Observability.Strategy
	if obsStrategy == "" {
		obsStrategy = "realtime"
	}
	obsServiceName := cfg.Observability.ServiceName
	if obsServiceName == "" {
		obsServiceName = "brainkit"
	}
	kernel.bridge.Eval("__obs_config.js", fmt.Sprintf(
		`globalThis.__brainkit_obs_config = { enabled: %v, strategy: %q, serviceName: %q }`,
		obsEnabled, obsStrategy, obsServiceName,
	))

	if err := kernel.loadRuntime(); err != nil {
		agentSandbox.Close()
		return nil, err
	}

	// Inject provider configs into the JS runtime for ai.generate/embed model resolution.
	// The JS runtime's resolveModel() reads from globalThis.__kit_providers.
	// Convert typed AIProvider registrations to the shape the JS runtime expects.
	if len(cfg.AIProviders) > 0 {
		provMap := make(map[string]map[string]string)
		for name, reg := range cfg.AIProviders {
			creds := extractProviderCredentials(reg)
			entry := map[string]string{"APIKey": creds.APIKey}
			if creds.BaseURL != "" {
				entry["BaseURL"] = creds.BaseURL
			}
			provMap[name] = entry
		}
		provJSON, _ := json.Marshal(provMap)
		kernel.bridge.Eval("__providers.js", fmt.Sprintf(
			`globalThis.__kit_providers = %s;`, string(provJSON),
		))
	}

	// Initialize the provider registry
	kernel.providers = provreg.New(cfg.Probe)
	for name, reg := range cfg.AIProviders {
		kernel.providers.RegisterAIProvider(name, reg)
	}
	// Register all storages and vectors in the provider registry
	kernel.registerStorages(cfg, bridgeURLs)
	if err := kernel.registerVectors(cfg, bridgeURLs); err != nil {
		for _, srv := range kernel.storages {
			_ = srv.Close()
		}
		agentSandbox.Close()
		return nil, fmt.Errorf("brainkit: register vectors: %w", err)
	}

	// Initialize package manager
	registries := cfg.PluginRegistries
	if len(registries) == 0 {
		registries = []RegistryConfig{DefaultRegistry}
	}
	pluginDir := cfg.PluginDir
	if pluginDir == "" && cfg.FSRoot != "" {
		pluginDir = filepath.Join(cfg.FSRoot, "plugins")
	} else if pluginDir == "" {
		pluginDir = filepath.Join(os.TempDir(), "brainkit-plugins")
	}
	var pkgStore packages.PluginStore
	if cfg.Store != nil {
		pkgStore = &kitStoreAdapter{store: cfg.Store}
	}
	regSources := make([]packages.RegistrySource, len(registries))
	for i, r := range registries {
		regSources[i] = packages.RegistrySource{Name: r.Name, URL: r.URL, AuthToken: r.AuthToken}
	}
	kernel.packages = packages.NewManager(
		packages.NewRegistryClient(regSources),
		pluginDir,
		pkgStore,
	)
	kernel.packagesDomain = newPackagesDomain(kernel.packages)
	kernel.packageDeployDomain = newPackageDeployDomain(kernel)

	// Initialize secret store
	if cfg.SecretStore != nil {
		kernel.secretStore = cfg.SecretStore
	} else {
		secretKey := cfg.SecretKey
		if secretKey == "" {
			secretKey = os.Getenv("BRAINKIT_SECRET_KEY")
		}
		if secretKey != "" && cfg.Store != nil {
			store, ok := cfg.Store.(*SQLiteStore)
			if ok {
				encStore, err := secrets.NewEncryptedKVStore(store.db, secretKey)
				if err != nil {
					InvokeErrorHandler(cfg.ErrorHandler, &sdkerrors.PersistenceError{
						Operation: "CreateEncryptedSecretStore", Cause: err,
					}, ErrorContext{Operation: "CreateEncryptedSecretStore", Component: "kernel"})
					kernel.secretStore = secrets.NewEnvStore()
				} else {
					kernel.secretStore = encStore
				}
			} else {
				kernel.secretStore = secrets.NewEnvStore()
			}
		} else {
			if secretKey == "" && cfg.Store != nil {
				log.Printf("[brainkit] WARNING: SecretKey not set — secrets stored without encryption")
				store, ok := cfg.Store.(*SQLiteStore)
				if ok {
					encStore, _ := secrets.NewEncryptedKVStore(store.db, "")
					if encStore != nil {
						kernel.secretStore = encStore
					} else {
						kernel.secretStore = secrets.NewEnvStore()
					}
				} else {
					kernel.secretStore = secrets.NewEnvStore()
				}
			} else {
				kernel.secretStore = secrets.NewEnvStore()
			}
		}
	}
	// SecretsDomain constructed later (needs kernel.remote for bus publishing)

	// Initialize RBAC
	if len(cfg.Roles) > 0 {
		kernel.rbac = rbac.NewManager(cfg.Roles, cfg.DefaultRole)
	}

	// Generate reply token HMAC key when RBAC is active
	if kernel.rbac != nil {
		key := make([]byte, 32)
		if _, err := cryptorand.Read(key); err != nil {
			return nil, fmt.Errorf("brainkit: generate reply HMAC key: %w", err)
		}
		kernel.replyHMACKey = key
	}

	// Initialize bus rate limiters (per-role token buckets)
	if len(cfg.BusRateLimits) > 0 {
		kernel.busRateLimiters = make(map[string]*rate.Limiter, len(cfg.BusRateLimits))
		for role, rps := range cfg.BusRateLimits {
			kernel.busRateLimiters[role] = rate.NewLimiter(rate.Limit(rps), int(rps))
		}
	}

	// Initialize tracer
	sampleRate := cfg.TraceSampleRate
	if sampleRate == 0 {
		sampleRate = 1.0
	}
	kernel.tracer = tracing.NewTracer(cfg.TraceStore, sampleRate)
	// (TracingDomain eliminated — inlined into catalog)

	// ToolsDomain needs tracer — constructed here after tracer init
	kernel.toolsDomain = newToolsDomain(sharedTools, kernel.bridge, kernel.tracer, cfg.CallerID)

	// Initialize workflow engine with persistence if KitStore available
	kernel.hostFunctions = workflow.NewHostFunctionRegistry()
	var wfStore workflow.RunStore
	if sqlStore, ok := cfg.Store.(*SQLiteStore); ok {
		wfStore = &workflowStoreAdapter{store: sqlStore}
	}
	kernel.workflowEngine = workflow.NewEngine(kernel.hostFunctions, wfStore,
		workflow.WithAI(&kernelAIGenerator{kernel: kernel}),
		workflow.WithBusSubscriber(func(topic string, handler func(json.RawMessage)) (func(), error) {
			return kernel.remote.SubscribeRaw(context.Background(), topic, func(msg messages.Message) {
				handler(msg.Payload)
			})
		}),
		workflow.WithBusPublisher(func(ctx context.Context, topic string, payload json.RawMessage) error {
			return kernel.publish(ctx, topic, payload)
		}),
		workflow.WithSpanRecorder(func(workflowID, runID string, entries []workflow.JournalEntry) {
			if kernel.tracer == nil {
				return
			}
			for _, entry := range entries {
				span := kernel.tracer.StartSpan("workflow.step:"+entry.StepName, context.Background())
				span.SetAttribute("workflowId", workflowID)
				span.SetAttribute("runId", runID)
				span.SetAttribute("stepIndex", strconv.Itoa(entry.StepIndex))
				span.SetSource(workflowID)
				// Record host calls within the step as child attributes
				for _, call := range entry.Calls {
					span.SetAttribute("call:"+call.Function, call.Duration.String())
				}
				if entry.Error != "" {
					span.End(fmt.Errorf("%s", entry.Error))
				} else {
					span.End(nil)
				}
			}
		}),
		workflow.WithPluginCaller(func(ctx context.Context, topic string, args json.RawMessage) (json.RawMessage, error) {
			resultTopic := topic + ".result"
			correlationID := uuid.NewString()
			waitCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			resultCh := make(chan json.RawMessage, 1)
			unsub, err := kernel.remote.SubscribeRaw(waitCtx, resultTopic, func(msg messages.Message) {
				if msg.Metadata["correlationId"] == correlationID {
					select {
					case resultCh <- json.RawMessage(msg.Payload):
					default:
					}
					cancel()
				}
			})
			if err != nil {
				return nil, err
			}
			defer unsub()

			if _, err := kernel.remote.PublishRaw(messaging.ContextWithCorrelationID(waitCtx, correlationID), topic, args); err != nil {
				return nil, err
			}
			select {
			case result := <-resultCh:
				return result, nil
			case <-waitCtx.Done():
				return nil, fmt.Errorf("plugin call %s: timeout", topic)
			}
		}),
	)
	kernel.workflowDomain = newWorkflowDomain(kernel.workflowEngine)
	kernel.automationDomain = newAutomationDomain(kernel)
	kernel.testingDomain = newTestingDomain(kernel)

	// Restore active workflow runs from previous session
	if wfStore != nil {
		kernel.workflowEngine.RestoreActiveRuns(context.Background())
	}

	kernel.wasm = newWASMService(kernel)
	kernel.wasmDomainInst = newWASMDomain(kernel, kernel.wasm)
	kernel.lifecycle = newLifecycleDomain(kernel)
	// (RegistryDomain eliminated — inlined into catalog)

	// Start periodic probing if configured
	kernel.startPeriodicProbing()

	// Initial probe — don't wait for first periodic tick
	go kernel.ProbeAll()

	if cfg.Store != nil {
		if err := kernel.wasm.loadFromStore(cfg.Store); err != nil {
			InvokeErrorHandler(cfg.ErrorHandler, &sdkerrors.PersistenceError{
				Operation: "LoadFromStore", Cause: err,
			}, ErrorContext{Operation: "LoadFromStore", Component: "kernel"})
		}
	}

	if len(cfg.MCPServers) > 0 {
		kernel.mcp = mcppkg.New()
		// (MCPDomain eliminated — inlined into catalog)
		for name, serverCfg := range cfg.MCPServers {
			if err := kernel.mcp.Connect(context.Background(), name, serverCfg); err != nil {
				continue
			}
			for _, tool := range kernel.mcp.ListToolsForServer(name) {
				toolCopy := tool
				fullName := toolreg.ComposeName("mcp", toolCopy.ServerName, "1.0.0", toolCopy.Name)
				_ = kernel.Tools.Register(toolreg.RegisteredTool{
					Name:        fullName,
					ShortName:   toolCopy.Name,
					Owner:       "mcp",
					Package:     toolCopy.ServerName,
					Version:     "1.0.0",
					Description: toolCopy.Description,
					InputSchema: toolCopy.InputSchema,
					Executor: &toolreg.GoFuncExecutor{
						Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
							return kernel.mcp.CallTool(ctx, toolCopy.ServerName, toolCopy.Name, input)
						},
					},
				})
			}
		}
	}
	// (MCPDomain eliminated — no else branch needed)

	// Set up internal Watermill transport + router
	if cfg.Transport != nil {
		kernel.transport = cfg.Transport
		kernel.ownsTransport = false
	} else {
		transport, err := messaging.NewTransportSet(messaging.TransportConfig{Type: "memory"})
		if err != nil {
			agentSandbox.Close()
			return nil, fmt.Errorf("brainkit: internal transport: %w", err)
		}
		kernel.transport = transport
		kernel.ownsTransport = true
	}

	kernel.remote = messaging.NewRemoteClientWithTransport(cfg.Namespace, cfg.CallerID, kernel.transport)

	// SecretsDomain — needs kernel.remote for bus event publishing
	kernel.secretsDomain = newSecretsDomain(kernel.secretStore, kernel.remote, cfg.CallerID, nil, kernel.refreshProviderIfSecret)

	logger := watermill.NopLogger{}
	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		if kernel.ownsTransport {
			kernel.transport.Close()
		}
		agentSandbox.Close()
		return nil, fmt.Errorf("brainkit: router: %w", err)
	}

	metrics := messaging.NewMetrics()
	router.AddMiddleware(
		messaging.DepthMiddleware,
		messaging.CallerIDMiddleware(cfg.CallerID),
		messaging.MetricsMiddleware(metrics),
	)
	if cfg.MaxConcurrency > 0 {
		router.AddMiddleware(messaging.MaxConcurrencyMiddleware(cfg.MaxConcurrency))
	}

	kernel.router = router
	kernel.host = messaging.NewHostWithTransport(cfg.Namespace, router, kernel.transport)

	if !cfg.DeferRouterStart {
		// Standalone Kernel: register kernel-only bindings and start router now
		kernel.host.RegisterCommands(commandBindingsForKernel(kernel))
		go func() {
			_ = router.Run(context.Background())
		}()
		<-router.Running()
	}
	// If DeferRouterStart: caller (Node) registers all bindings and starts the router

	// Start background job pump — processes qctx.Schedule'd callbacks
	// even when no EvalTS is active. Enables deployed .ts services to
	// receive bus messages asynchronously.
	kernel.startJobPump()

	// Auto-redeploy persisted .ts deployments
	if cfg.Store != nil {
		kernel.redeployPersistedDeployments()
	}

	// Restore persisted schedules
	if cfg.Store != nil {
		kernel.restoreSchedules()
	}

	// Wire WASM shard bus subscriptions for standalone Kernel.
	// Node.Start() does this separately after transport provisioning.
	// Without this, shard bus.on handlers don't fire on standalone Kernel (bug #9).
	if !cfg.DeferRouterStart && kernel.wasm != nil {
		if err := kernel.wasm.restoreTransportSubscriptions(); err != nil {
			InvokeErrorHandler(cfg.ErrorHandler, &sdkerrors.TransportError{
				Operation: "RestoreShardSubscriptions", Cause: err,
			}, ErrorContext{Operation: "RestoreShardSubscriptions", Component: "kernel"})
		}
	}

	kernel.startedAt = time.Now()

	return kernel, nil
}

// redeployPersistedDeployments loads and re-deploys all persisted .ts deployments.
func (k *Kernel) redeployPersistedDeployments() {
	deployments, err := k.config.Store.LoadDeployments()
	if err != nil {
		InvokeErrorHandler(k.config.ErrorHandler, &sdkerrors.PersistenceError{
			Operation: "LoadDeployments", Cause: err,
		}, ErrorContext{Operation: "LoadDeployments", Component: "kernel"})
		return
	}
	if len(deployments) == 0 {
		return
	}

	sort.Slice(deployments, func(i, j int) bool {
		return deployments[i].Order < deployments[j].Order
	})

	maxOrder := int32(deployments[len(deployments)-1].Order)
	k.deployOrder.Store(maxOrder)

	for _, d := range deployments {
		var opts []DeployOption
		opts = append(opts, WithRestoring()) // don't re-persist what was just loaded
		if d.Role != "" {
			opts = append(opts, WithRole(d.Role))
		}
		if d.PackageName != "" {
			opts = append(opts, WithPackageName(d.PackageName))
		}
		if _, err := k.Deploy(context.Background(), d.Source, d.Code, opts...); err != nil {
			InvokeErrorHandler(k.config.ErrorHandler, &sdkerrors.DeployError{
				Source: d.Source, Phase: "redeploy", Cause: err,
			}, ErrorContext{Operation: "RedeployPersisted", Component: "kernel", Source: d.Source})
		}
	}

	log.Printf("[brainkit] redeployed %d persisted deployments", len(deployments))
}

// --- sdk.Runtime implementation ---

// PublishRaw sends a message to a topic. Returns correlationID.
func (k *Kernel) PublishRaw(ctx context.Context, topic string, payload json.RawMessage) (string, error) {
	return k.remote.PublishRaw(ctx, topic, payload)
}

// SubscribeRaw subscribes to a topic. Subscription is active before this returns.
func (k *Kernel) SubscribeRaw(ctx context.Context, topic string, handler func(messages.Message)) (func(), error) {
	return k.remote.SubscribeRaw(ctx, topic, handler)
}

// --- sdk.CrossNamespaceRuntime implementation ---

// persistenceError routes a persistence failure through the ErrorHandler and emits a bus event.
// The original operation still succeeds in memory — persistence is best-effort.
func (k *Kernel) persistenceError(ctx context.Context, operation, source string, err error) {
	typedErr := &sdkerrors.PersistenceError{Operation: operation, Source: source, Cause: err}
	InvokeErrorHandler(k.config.ErrorHandler, typedErr, ErrorContext{
		Operation: operation, Component: "persistence", Source: source,
	})
	payload, _ := json.Marshal(map[string]any{
		"operation": operation,
		"source":    source,
		"error":     err.Error(),
		"timestamp": time.Now().Format(time.RFC3339),
	})
	k.publish(ctx, "kit.persistence.error", payload)
}

// PublishRawTo publishes to a specific Kit's namespace.
func (k *Kernel) PublishRawTo(ctx context.Context, targetNamespace, topic string, payload json.RawMessage) (string, error) {
	return k.remote.PublishRawToNamespace(ctx, targetNamespace, topic, payload)
}

// SubscribeRawTo subscribes to a topic in a specific Kit's namespace.
func (k *Kernel) SubscribeRawTo(ctx context.Context, targetNamespace, topic string, handler func(messages.Message)) (func(), error) {
	return k.remote.SubscribeRawToNamespace(ctx, targetNamespace, topic, handler)
}

// publish is an internal convenience for fire-and-forget event publishing.
func (k *Kernel) publish(ctx context.Context, topic string, payload json.RawMessage) error {
	_, err := k.remote.PublishRaw(ctx, topic, payload)
	return err
}

// subscribe is an internal convenience for subscribing with full message.
func (k *Kernel) subscribe(topic string, handler func(messages.Message)) (func(), error) {
	return k.remote.SubscribeRaw(context.Background(), topic, handler)
}

// ReplyRaw publishes directly to a resolved replyTo topic without namespace prefixing.
// This is the Go equivalent of __go_brainkit_bus_reply in bridges.go.
// Used by sdk.Reply and sdk.SendChunk.
func (k *Kernel) ReplyRaw(ctx context.Context, replyTo, correlationID string, payload json.RawMessage, done bool) error {
	if replyTo == "" {
		return nil
	}
	wmsg := message.NewMessage(watermill.NewUUID(), []byte(payload))
	wmsg.Metadata.Set("correlationId", correlationID)
	if done {
		wmsg.Metadata.Set("done", "true")
	}
	// replyTo is already namespaced+sanitized — publish directly to transport
	return k.transport.Publisher.Publish(replyTo, wmsg)
}

// Shutdown drains in-flight handlers, then closes everything.
// The context controls the drain timeout — when ctx expires, force-close proceeds.
func (k *Kernel) Shutdown(ctx context.Context) error {
	k.draining.Store(true)
	k.waitForDrain(ctx)
	return k.close()
}

// Close shuts down with a short drain timeout (5s).
func (k *Kernel) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return k.Shutdown(ctx)
}

// close is the internal shutdown logic.
func (k *Kernel) close() error {
	k.mu.Lock()
	if k.closed {
		k.mu.Unlock()
		return nil
	}
	k.closed = true
	subs := make([]func(), 0, len(k.bridgeSubs))
	for _, cancel := range k.bridgeSubs {
		subs = append(subs, cancel)
	}
	k.bridgeSubs = map[string]func(){}
	k.mu.Unlock()

	for _, cancel := range subs {
		cancel()
	}

	// Stop all schedule timers
	k.mu.Lock()
	for _, entry := range k.schedules {
		entry.timer.Stop()
	}
	k.schedules = nil
	k.mu.Unlock()

	var firstErr error
	collect := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	// Shut down router first (stops processing messages)
	if k.router != nil {
		collect(k.router.Close())
	}

	if k.agentsDomain != nil && k.agents != nil {
		k.agentsDomain.UnregisterAllForKit(k.agents.ID())
	}
	if k.mcp != nil {
		collect(k.mcp.Close())
	}
	if k.wasm != nil {
		k.wasm.close()
	}
	if k.config.Store != nil {
		collect(k.config.Store.Close())
	}
	if k.agents != nil {
		k.agents.Close()
	}
	for name, srv := range k.storages {
		if err := srv.Close(); err != nil {
			collect(fmt.Errorf("storage %q: %w", name, err))
		}
	}

	// Shut down transport last (only if we own it — Node owns its own)
	if k.ownsTransport && k.transport != nil {
		collect(k.transport.Close())
	}

	return firstErr
}

// Namespace returns the runtime namespace.
func (k *Kernel) Namespace() string { return k.namespace }

// CallerID returns the runtime identity.
func (k *Kernel) CallerID() string { return k.callerID }

// CreateAgent creates a persistent agent in the runtime.
func (k *Kernel) CreateAgent(cfg agentembed.AgentConfig) (*agentembed.Agent, error) {
	return k.agents.CreateAgent(cfg)
}

// AddStorage registers a new named storage at runtime.
// For sqlite: starts a libsql bridge + registers in provider registry.
// For others: registers in provider registry only.
func (k *Kernel) AddStorage(name string, cfg StorageConfig) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if cfg.Type == "sqlite" {
		if _, exists := k.storages[name]; exists {
			return &sdk.AlreadyExistsError{Resource: "storage", Name: name}
		}
		srv, err := libsql.NewServer(cfg.Path)
		if err != nil {
			return err
		}
		k.storages[name] = srv
		reg := storageToRegistration(cfg, srv.URL())
		k.providers.RegisterStorage(name, reg)
	} else {
		reg := storageToRegistration(cfg, "")
		k.providers.RegisterStorage(name, reg)
	}
	return nil
}

// RemoveStorage stops and removes a named storage.
func (k *Kernel) RemoveStorage(name string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if srv, ok := k.storages[name]; ok {
		_ = srv.Close()
		delete(k.storages, name)
	}
	k.providers.UnregisterStorage(name)
	return nil
}

// StorageURL returns the HTTP URL for a named storage bridge.
func (k *Kernel) StorageURL(name string) string {
	k.mu.Lock()
	defer k.mu.Unlock()
	if srv, ok := k.storages[name]; ok {
		return srv.URL()
	}
	return ""
}

// ListResources returns all tracked resources, optionally filtered by type.
func (k *Kernel) ListResources(resourceType ...string) ([]ResourceInfo, error) {
	filter := ""
	if len(resourceType) > 0 {
		filter = resourceType[0]
	}
	code := fmt.Sprintf(`return JSON.stringify(globalThis.__kit_registry.list(%q))`, filter)
	result, err := k.EvalTS(context.Background(), "__list_resources.ts", code)
	if err != nil {
		return nil, err
	}
	var resources []ResourceInfo
	if err := json.Unmarshal([]byte(result), &resources); err != nil {
		return nil, fmt.Errorf("list resources: %w", err)
	}
	return resources, nil
}

// ResourcesFrom returns all resources created by a specific .ts file.
func (k *Kernel) ResourcesFrom(filename string) ([]ResourceInfo, error) {
	code := fmt.Sprintf(`return JSON.stringify(globalThis.__kit_registry.listBySource(%q))`, filename)
	result, err := k.EvalTS(context.Background(), "__resources_from.ts", code)
	if err != nil {
		return nil, err
	}
	var resources []ResourceInfo
	if err := json.Unmarshal([]byte(result), &resources); err != nil {
		return nil, fmt.Errorf("resources from: %w", err)
	}
	return resources, nil
}

// TeardownFile removes all resources created by a specific .ts file.
func (k *Kernel) TeardownFile(filename string) (int, error) {
	code := fmt.Sprintf(`
		var resources = globalThis.__kit_registry.listBySource(%q);
		var count = 0;
		for (var i = resources.length - 1; i >= 0; i--) {
			globalThis.__kit_registry.unregister(resources[i].type, resources[i].id);
			count++;
		}
		return JSON.stringify(count);
	`, filename)
	result, err := k.EvalTS(context.Background(), "__teardown_file.ts", code)
	if err != nil {
		return 0, err
	}
	var count int
	if err := json.Unmarshal([]byte(result), &count); err != nil {
		return 0, nil
	}
	return count, nil
}

// RemoveResource removes a specific resource by type and ID.
func (k *Kernel) RemoveResource(resourceType, id string) error {
	code := fmt.Sprintf(`
		var entry = globalThis.__kit_registry.unregister(%q, %q);
		return JSON.stringify(entry !== null);
	`, resourceType, id)
	_, err := k.EvalTS(context.Background(), "__remove_resource.ts", code)
	return err
}

// ListWASMModules returns metadata for all compiled WASM modules.
func (k *Kernel) ListWASMModules() ([]WASMModuleInfo, error) {
	k.wasm.mu.Lock()
	defer k.wasm.mu.Unlock()
	infos := make([]WASMModuleInfo, 0, len(k.wasm.modules))
	for _, mod := range k.wasm.modules {
		infos = append(infos, WASMModuleInfo{
			Name:       mod.Name,
			Size:       mod.Size,
			Exports:    mod.Exports,
			CompiledAt: mod.CompiledAt.Format(time.RFC3339),
			SourceHash: mod.SourceHash,
		})
	}
	return infos, nil
}

// GetWASMModule returns metadata for a specific module by name.
func (k *Kernel) GetWASMModule(name string) (*WASMModuleInfo, error) {
	k.wasm.mu.Lock()
	mod, ok := k.wasm.modules[name]
	k.wasm.mu.Unlock()
	if !ok {
		return nil, nil
	}
	return &WASMModuleInfo{
		Name:       mod.Name,
		Size:       mod.Size,
		Exports:    mod.Exports,
		CompiledAt: mod.CompiledAt.Format(time.RFC3339),
		SourceHash: mod.SourceHash,
	}, nil
}

// RemoveWASMModule unloads a compiled module by name via the typed domain.
func (k *Kernel) RemoveWASMModule(name string) error {
	_, err := k.wasmDomainInst.Remove(context.Background(), messages.WasmRemoveMsg{Name: name})
	return err
}

// DeployWASM activates a compiled shard via the typed domain.
func (k *Kernel) DeployWASM(name string) (*ShardDescriptor, error) {
	resp, err := k.wasmDomainInst.Deploy(context.Background(), messages.WasmDeployMsg{Name: name})
	if err != nil {
		return nil, err
	}
	return &ShardDescriptor{
		Module:     resp.Module,
		Mode:       resp.Mode,
		Handlers:   resp.Handlers,
		DeployedAt: time.Now(),
	}, nil
}

// UndeployWASM removes all event subscriptions for a deployed shard.
func (k *Kernel) UndeployWASM(name string) error {
	_, err := k.wasmDomainInst.Undeploy(context.Background(), messages.WasmUndeployMsg{Name: name})
	return err
}

// DescribeWASM returns the shard's registrations via the typed domain.
func (k *Kernel) DescribeWASM(name string) (*ShardDescriptor, error) {
	resp, err := k.wasmDomainInst.Describe(context.Background(), messages.WasmDescribeMsg{Name: name})
	if err != nil {
		return nil, err
	}
	return &ShardDescriptor{
		Module:     resp.Module,
		Mode:       resp.Mode,
		Handlers:   resp.Handlers,
		DeployedAt: time.Now(),
	}, nil
}

// ListDeployedWASM returns all active shard descriptors.
func (k *Kernel) ListDeployedWASM() []ShardDescriptor {
	return k.wasm.listDeployedShards()
}

// InjectWASMEvent manually triggers a shard handler.
func (k *Kernel) InjectWASMEvent(shardName, topic string, payload json.RawMessage) (*WASMEventResult, error) {
	return k.wasm.invokeShardHandler(context.Background(), shardName, topic, payload, nil)
}

// initStorages starts sqlite bridges for all sqlite storage entries.
// Must be called before loadRuntime — libsql servers need to be running.
// Returns a path→URL map for sqlite bridge sharing with vectors.
func (k *Kernel) initStorages(cfg KernelConfig) (map[string]string, error) {
	bridgeURLs := make(map[string]string)
	for name, scfg := range cfg.Storages {
		if scfg.Type == "sqlite" {
			srv, err := libsql.NewServer(scfg.Path)
			if err != nil {
				return nil, fmt.Errorf("storage %q: %w", name, err)
			}
			k.storages[name] = srv
			bridgeURLs[scfg.Path] = srv.URL()
		}
	}
	return bridgeURLs, nil
}

// registerStorages registers all storages in the provider registry.
func (k *Kernel) registerStorages(cfg KernelConfig, bridgeURLs map[string]string) {
	for name, scfg := range cfg.Storages {
		bridgeURL := ""
		if scfg.Type == "sqlite" {
			bridgeURL = bridgeURLs[scfg.Path]
		}
		reg := storageToRegistration(scfg, bridgeURL)
		k.providers.RegisterStorage(name, reg)
	}
}

// registerVectors registers all vector stores in the provider registry.
// For sqlite vectors, reuses the bridge URL from a matching storage path.
func (k *Kernel) registerVectors(cfg KernelConfig, bridgeURLs map[string]string) error {
	for name, vcfg := range cfg.Vectors {
		bridgeURL := ""
		if vcfg.Type == "sqlite" {
			bridgeURL = bridgeURLs[vcfg.Path]
			if bridgeURL == "" {
				srv, err := libsql.NewServer(vcfg.Path)
				if err != nil {
					return fmt.Errorf("vector %q: %w", name, err)
				}
				k.storages["vec_"+name] = srv
				bridgeURL = srv.URL()
				bridgeURLs[vcfg.Path] = bridgeURL
			}
		}
		reg := vectorToRegistration(vcfg, bridgeURL)
		k.providers.RegisterVectorStore(name, reg)
	}
	return nil
}

// evalDomain marshals a request into JS globals and evaluates code atomically.
// Replaces per-domain evalAI/evalMemory/evalVector/evalWorkflow methods.
func (k *Kernel) evalDomain(ctx context.Context, req any, filename, code string) (json.RawMessage, error) {
	reqJSON, _ := json.Marshal(req)
	wrappedCode := fmt.Sprintf(`
		globalThis.__pending_req = %s;
		%s
	`, string(reqJSON), code)
	resultJSON, err := k.EvalTS(ctx, filename, wrappedCode)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(resultJSON), nil
}

// EvalTS runs .ts-style code with brainkit infrastructure imports destructured.
func (k *Kernel) EvalTS(ctx context.Context, filename, code string) (string, error) {
	wrapped := fmt.Sprintf(`(async () => {
		return await globalThis.__kitRunWithSource(%q, async () => {
			const { bus, kit, model, provider, storage, vectorStore, registry, tools, fs, mcp, output, secrets } = globalThis.__kit;
			%s
		});
	})()`, filename, code)

	if k.bridge.IsEvalBusy() {
		return k.bridge.EvalOnJSThread(filename, wrapped)
	}
	return k.agents.Eval(ctx, filename, wrapped)
}

// EvalModule runs code as an ES module with import { ... } from "kit".
func (k *Kernel) EvalModule(ctx context.Context, filename, code string) (string, error) {
	k.bridge.Eval("__clear_result.js", `delete globalThis.__module_result`)
	val, err := k.bridge.EvalAsyncModule(filename, code)
	if err != nil {
		return "", fmt.Errorf("brainkit: eval module: %w", err)
	}
	if val != nil {
		val.Free()
	}
	result, err := k.bridge.Eval("__get_result.js",
		`typeof globalThis.__module_result !== 'undefined' ? String(globalThis.__module_result) : ""`)
	if err != nil {
		return "", err
	}
	defer result.Free()
	return result.String(), nil
}

// RegisterTool is a convenience method for registering typed Go tools.
func RegisterTool[T any](k *Kernel, name string, tool toolreg.TypedTool[T]) error {
	return toolreg.Register(k.Tools, name, tool)
}

// ResumeWorkflow resumes a suspended workflow run from the Go side.
func (k *Kernel) ResumeWorkflow(ctx context.Context, runId, stepId, resumeDataJSON string) (string, error) {
	stepArg := "undefined"
	if stepId != "" {
		stepArg = fmt.Sprintf("%q", stepId)
	}
	code := fmt.Sprintf(`(async () => {
		var result = await globalThis.__kit.resumeWorkflow(%q, %s, %s);
		globalThis.__module_result = JSON.stringify(result);
	})()`, runId, stepArg, resumeDataJSON)

	val, err := k.bridge.EvalAsync("__resume_workflow.js", code)
	if err != nil {
		return "", fmt.Errorf("resume workflow %s: %w", runId, err)
	}
	if val != nil {
		val.Free()
	}
	result, err := k.bridge.Eval("__get_resume_result.js", `typeof globalThis.__module_result !== 'undefined' ? String(globalThis.__module_result) : ""`)
	if err != nil {
		return "", err
	}
	defer result.Free()
	return result.String(), nil
}

// --- Provider Registry delegation ---

// RegisterAIProvider registers a typed AI provider at runtime.
// Injects env vars into the JS runtime's process.env.
func (k *Kernel) RegisterAIProvider(name string, typ provreg.AIProviderType, config any) error {
	reg := provreg.AIProviderRegistration{Type: typ, Config: config}
	return k.providers.RegisterAIProvider(name, reg)
}

// UnregisterAIProvider removes an AI provider.
func (k *Kernel) UnregisterAIProvider(name string) { k.providers.UnregisterAIProvider(name) }

// ListAIProviders returns all registered AI providers.
func (k *Kernel) ListAIProviders() []provreg.ProviderInfo { return k.providers.ListAIProviders() }

// RegisterVectorStore registers a typed vector store at runtime.
func (k *Kernel) RegisterVectorStore(name string, typ provreg.VectorStoreType, config any) error {
	return k.providers.RegisterVectorStore(name, provreg.VectorStoreRegistration{Type: typ, Config: config})
}

// UnregisterVectorStore removes a vector store.
func (k *Kernel) UnregisterVectorStore(name string) { k.providers.UnregisterVectorStore(name) }

// ListVectorStores returns all registered vector stores.
func (k *Kernel) ListVectorStores() []provreg.VectorStoreInfo { return k.providers.ListVectorStores() }

// RegisterStorage registers a typed Mastra storage at runtime.
func (k *Kernel) RegisterStorage(name string, typ provreg.StorageType, config any) error {
	return k.providers.RegisterStorage(name, provreg.StorageRegistration{Type: typ, Config: config})
}

// UnregisterStorage removes a Mastra storage.
func (k *Kernel) UnregisterStorage(name string) { k.providers.UnregisterStorage(name) }

// ListStorages returns all registered Mastra storages.
func (k *Kernel) ListStorages() []provreg.StorageInfo { return k.providers.ListStorages() }


// --- Kernel-level probing (uses JS runtime for vector/storage) ---

// ProbeAIProvider runs a live HTTP probe against a registered AI provider.
func (k *Kernel) ProbeAIProvider(name string) provreg.ProbeResult {
	return k.providers.ProbeAIProvider(name)
}

// ProbeVectorStore probes a vector store by instantiating it in the JS runtime
// and calling listIndexes(). This tests real connectivity, not just config validity.
func (k *Kernel) ProbeVectorStore(name string) provreg.ProbeResult {
	start := time.Now()
	result, err := k.EvalTS(context.Background(), "__probe_vectorstore.ts", fmt.Sprintf(`
		try {
			var vs = vectorStore(%q);
			await vs.listIndexes();
			return JSON.stringify({ available: true });
		} catch(e) {
			return JSON.stringify({ available: false, error: e.message || String(e) });
		}
	`, name))
	latency := time.Since(start)

	if err != nil {
		k.providers.UpdateProbeResult("vectorStore", name, false, latency, err.Error())
		return provreg.ProbeResult{Error: err.Error(), Latency: latency}
	}

	var parsed struct {
		Available bool   `json:"available"`
		Error     string `json:"error"`
	}
	json.Unmarshal([]byte(result), &parsed)

	k.providers.UpdateProbeResult("vectorStore", name, parsed.Available, latency, parsed.Error)
	return provreg.ProbeResult{
		Available:    parsed.Available,
		Capabilities: provreg.DefaultVectorCapabilities(),
		Latency:      latency,
		Error:        parsed.Error,
	}
}

// ProbeStorage probes a storage backend by instantiating it in the JS runtime
// and calling a simple operation. Tests real connectivity.
func (k *Kernel) ProbeStorage(name string) provreg.ProbeResult {
	start := time.Now()
	result, err := k.EvalTS(context.Background(), "__probe_storage.ts", fmt.Sprintf(`
		try {
			var s = storage(%q);
			if (s && typeof s.listThreads === "function") {
				await s.listThreads({});
			}
			return JSON.stringify({ available: true });
		} catch(e) {
			return JSON.stringify({ available: false, error: e.message || String(e) });
		}
	`, name))
	latency := time.Since(start)

	if err != nil {
		k.providers.UpdateProbeResult("storage", name, false, latency, err.Error())
		return provreg.ProbeResult{Error: err.Error(), Latency: latency}
	}

	var parsed struct {
		Available bool   `json:"available"`
		Error     string `json:"error"`
	}
	json.Unmarshal([]byte(result), &parsed)

	k.providers.UpdateProbeResult("storage", name, parsed.Available, latency, parsed.Error)
	return provreg.ProbeResult{
		Available:    parsed.Available,
		Capabilities: provreg.DefaultStorageCapabilities(),
		Latency:      latency,
		Error:        parsed.Error,
	}
}

// ProbeAll runs probes for all registered providers, vector stores, and storages.
func (k *Kernel) ProbeAll() {
	for _, p := range k.providers.ListAIProviders() {
		k.ProbeAIProvider(p.Name)
	}
	for _, v := range k.providers.ListVectorStores() {
		k.ProbeVectorStore(v.Name)
	}
	for _, s := range k.providers.ListStorages() {
		k.ProbeStorage(s.Name)
	}
}

// startPeriodicProbing starts a background goroutine that probes all registered
// resources at the configured interval. Stops when the Kernel is closed.
func (k *Kernel) startPeriodicProbing() {
	interval := k.config.Probe.PeriodicInterval
	if interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			k.mu.Lock()
			closed := k.closed
			k.mu.Unlock()
			if closed {
				ticker.Stop()
				return
			}
			k.ProbeAll()
		}
	}()
}

// startJobPump starts a background goroutine that processes QuickJS scheduled
// callbacks AND JS microtasks. Wakes immediately when Schedule'd callbacks are
// pending (via pumpSignal), with a 100ms fallback for pure-JS microtasks.
//
// Uses bridge.Go() so the goroutine is tracked by bridge.wg — Close() waits
// for it to finish before touching the QuickJS context.
func (k *Kernel) startJobPump() {
	fallback := time.NewTicker(100 * time.Millisecond)
	pumpSignal := k.bridge.PumpSignal()

	k.bridge.Go(func(goCtx context.Context) {
		defer fallback.Stop()
		for {
			select {
			case <-pumpSignal:
				k.processScheduledJobs()
			case <-fallback.C:
				k.processScheduledJobs()
			case <-goCtx.Done():
				return
			}
		}
	})
}

func (k *Kernel) processScheduledJobs() {
	k.mu.Lock()
	closed := k.closed
	k.mu.Unlock()
	if closed {
		return
	}
	k.pumpCycles.Add(1)
	k.bridge.ProcessScheduledJobs()
}

// extractProviderCredentials extracts APIKey and BaseURL from a typed provider registration.
func extractProviderCredentials(reg provreg.AIProviderRegistration) struct{ APIKey, BaseURL string } {
	switch cfg := reg.Config.(type) {
	case provreg.OpenAIProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, cfg.BaseURL}
	case provreg.AnthropicProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, cfg.BaseURL}
	case provreg.GoogleProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, cfg.BaseURL}
	case provreg.MistralProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, cfg.BaseURL}
	case provreg.CohereProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, cfg.BaseURL}
	case provreg.GroqProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, cfg.BaseURL}
	case provreg.PerplexityProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, cfg.BaseURL}
	case provreg.DeepSeekProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, cfg.BaseURL}
	case provreg.FireworksProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, cfg.BaseURL}
	case provreg.TogetherAIProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, cfg.BaseURL}
	case provreg.XAIProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, cfg.BaseURL}
	case provreg.AzureProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, cfg.BaseURL}
	case provreg.HuggingFaceProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, cfg.BaseURL}
	case provreg.CerebrasProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, cfg.BaseURL}
	case provreg.VertexProviderConfig:
		return struct{ APIKey, BaseURL string }{cfg.APIKey, ""}
	case provreg.BedrockProviderConfig:
		return struct{ APIKey, BaseURL string }{"", ""}
	default:
		return struct{ APIKey, BaseURL string }{}
	}
}
