package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/brainlet/brainkit/bus"
	pluginv1 "github.com/brainlet/brainkit/proto/plugin/v1"
	"github.com/brainlet/brainkit/registry"
	"github.com/google/uuid"
)

// processManifest registers ALL 6 plugin capability types on the Kit.
func (pm *Manager) processManifest(name string, pc *conn) {
	m := pc.manifest

	owner := m.Owner
	if owner == "" {
		log.Printf("[plugin:%s] ERROR: manifest missing owner field", name)
		return
	}
	version := m.Version
	if version == "" {
		log.Printf("[plugin:%s] ERROR: manifest missing version field", name)
		return
	}

	pkgName := m.Name

	// 1. Register tools with new naming: owner/package@version/tool
	for _, t := range m.Tools {
		fullName := registry.ComposeName(owner, pkgName, version, t.Name)
		toolDef := t
		pm.bridge.Tools().Register(registry.RegisteredTool{
			Name:        fullName,
			ShortName:   toolDef.Name,
			Owner:       owner,
			Package:     pkgName,
			Version:     version,
			Description: toolDef.Description,
			InputSchema: json.RawMessage(toolDef.InputSchema),
			Executor: &registry.GoFuncExecutor{
				Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
					return pm.callPluginTool(name, toolDef.Name, input)
				},
			},
		})
	}

	// 2. Register interceptors
	for _, i := range m.Interceptors {
		iDef := i
		pm.bridge.Bus().AddInterceptor(&pluginInterceptor{
			name:     owner + "/" + pkgName + "@" + version + "/" + iDef.Name,
			priority: int(iDef.Priority),
			filter:   iDef.TopicFilter,
			pc:       pc,
		})
	}

	// 3. Record event schemas (declarations only)

	// 4. Register subscriptions — forward matching bus events to plugin stream
	for _, sub := range m.Subscriptions {
		topic := sub.Topic
		subID := pm.bridge.Bus().On(topic, func(msg bus.Message, _ bus.ReplyFunc) {
			pc.safeSend(&pluginv1.PluginMessage{
				Id:       uuid.NewString(),
				Type:     "event",
				Topic:    msg.Topic,
				CallerId: msg.CallerID,
				TraceId:  msg.TraceID,
				Payload:  msg.Payload,
				Metadata: msg.Metadata,
			})
		})
		pc.subs = append(pc.subs, subID)
	}

	// 5. Load agents via Deploy (each in its own SES Compartment)
	for _, a := range m.Agents {
		code := fmt.Sprintf(`agent({ name: %q, instructions: %q, model: %q });`,
			a.Name, a.Instructions, a.Model)
		source := fmt.Sprintf("__plugin_%s_agent_%s.ts", name, a.Name)
		if err := pm.bridge.Deploy(context.Background(), source, code); err != nil {
			log.Printf("[plugin:%s] failed to deploy agent %q: %v", name, a.Name, err)
		}
	}

	// 6. Load files via Deploy (each in its own SES Compartment)
	for _, f := range m.Files {
		source := fmt.Sprintf("__plugin_%s_%s", name, f.Path)
		if err := pm.bridge.Deploy(context.Background(), source, f.Content); err != nil {
			log.Printf("[plugin:%s] failed to deploy file %q: %v", name, f.Path, err)
		}
	}
}

// pluginInterceptor sends messages to the plugin for interception.
type pluginInterceptor struct {
	name     string
	priority int
	filter   string
	pc       *conn
}

func (i *pluginInterceptor) Name() string            { return i.name }
func (i *pluginInterceptor) Priority() int           { return i.priority }
func (i *pluginInterceptor) Match(topic string) bool { return bus.TopicMatches(i.filter, topic) }

func (i *pluginInterceptor) Process(msg *bus.Message) error {
	replyTo := "_intercept." + uuid.NewString()
	ch := make(chan interceptResult, 1)

	i.pc.interceptMu.Lock()
	i.pc.interceptReplies[replyTo] = ch
	i.pc.interceptMu.Unlock()

	defer func() {
		i.pc.interceptMu.Lock()
		delete(i.pc.interceptReplies, replyTo)
		i.pc.interceptMu.Unlock()
	}()

	payload := msg.Payload
	if payload == nil {
		payload = json.RawMessage(`{}`)
	}

	err := i.pc.safeSend(&pluginv1.PluginMessage{
		Id:       uuid.NewString(),
		Type:     "intercept",
		Topic:    msg.Topic,
		CallerId: msg.CallerID,
		TraceId:  msg.TraceID,
		ReplyTo:  replyTo,
		Payload:  payload,
		Metadata: msg.Metadata,
	})
	if err != nil {
		return fmt.Errorf("plugin interceptor %s: send failed: %w", i.name, err)
	}

	select {
	case result := <-ch:
		var errCheck struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(result.Payload, &errCheck) == nil && errCheck.Error != "" {
			return fmt.Errorf("interceptor rejected: %s", errCheck.Error)
		}
		if result.Payload != nil {
			msg.Payload = result.Payload
		}
		if result.Metadata != nil {
			if msg.Metadata == nil {
				msg.Metadata = make(map[string]string)
			}
			for k, v := range result.Metadata {
				msg.Metadata[k] = v
			}
		}
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("plugin interceptor %s: timeout (5s)", i.name)
	case <-i.pc.done:
		return fmt.Errorf("plugin interceptor %s: plugin died", i.name)
	}
}
