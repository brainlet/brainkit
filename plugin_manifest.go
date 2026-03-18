package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/brainlet/brainkit/bus"
	pluginv1 "github.com/brainlet/brainkit/proto/plugin/v1"
	"github.com/brainlet/brainkit/registry"
	"github.com/google/uuid"
)

// processManifest registers ALL 6 plugin capability types on the Kit.
func (pm *pluginManager) processManifest(name string, pc *pluginConn) {
	m := pc.manifest
	ns := "plugin." + name

	// 1. Register tools
	for _, t := range m.Tools {
		fullName := ns + "." + t.Name
		toolDef := t // capture
		pm.kit.Tools.Register(registry.RegisteredTool{
			Name:        fullName,
			ShortName:   toolDef.Name,
			Namespace:   ns,
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
		iDef := i // capture
		pm.kit.Bus.AddInterceptor(&pluginInterceptor{
			name:     ns + "." + iDef.Name,
			priority: int(iDef.Priority),
			filter:   iDef.TopicFilter,
		})
	}

	// 3. Record event schemas (declarations of what plugin CAN emit — no registration needed)

	// 4. Register subscriptions — forward matching bus events to plugin stream
	for _, sub := range m.Subscriptions {
		topic := sub.Topic
		subID := pm.kit.Bus.On(topic, func(msg bus.Message, _ bus.ReplyFunc) {
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

	// 5. Load agents via EvalTS
	for _, a := range m.Agents {
		code := fmt.Sprintf(`const _a = agent({ name: %q, instructions: %q, model: %q }); return "ok";`,
			a.Name, a.Instructions, a.Model)
		filename := fmt.Sprintf("__plugin_%s_agent_%s.ts", name, a.Name)
		if _, err := pm.kit.EvalTS(context.Background(), filename, code); err != nil {
			log.Printf("[plugin:%s] failed to load agent %q: %v", name, a.Name, err)
		}
	}

	// 6. Load files via EvalTS
	for _, f := range m.Files {
		filename := fmt.Sprintf("__plugin_%s_%s", name, f.Path)
		if _, err := pm.kit.EvalTS(context.Background(), filename, f.Content); err != nil {
			log.Printf("[plugin:%s] failed to load file %q: %v", name, f.Path, err)
		}
	}
}

// pluginInterceptor wraps a plugin's interceptor declaration for the bus pipeline.
// Note: Plugin interceptors are registered but currently pass-through.
// Full round-trip intercept (send to plugin, wait for result) requires async interceptor
// support which is deferred — interceptors run synchronously in the bus pipeline.
type pluginInterceptor struct {
	name     string
	priority int
	filter   string
}

func (i *pluginInterceptor) Name() string        { return i.name }
func (i *pluginInterceptor) Priority() int        { return i.priority }
func (i *pluginInterceptor) Match(topic string) bool {
	return bus.TopicMatches(i.filter, topic)
}
func (i *pluginInterceptor) Process(msg *bus.Message) error {
	// TODO: Plugin interceptor round-trip requires async interceptor support.
	// For v1, interceptors are registered (visible in metrics/debug) but pass-through.
	// The spec's intercept message type is defined in the proto for when this is implemented.
	return nil
}
