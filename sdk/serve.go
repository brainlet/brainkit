package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/brainlet/brainkit/internal/messaging"
	"github.com/brainlet/brainkit/sdk/messages"
)

// Run starts the plugin, connects to the Watermill transport, publishes the
// manifest, and runs the message router until shutdown.
func (p *Plugin) Run() error {
	logger := watermill.NopLogger{}
	namespace := os.Getenv("BRAINKIT_NAMESPACE")
	transport, err := messaging.NewTransportSet(messaging.TransportConfig{
		Type:     os.Getenv("BRAINKIT_TRANSPORT"),
		NATSURL:  os.Getenv("BRAINKIT_NATS_URL"),
		NATSName: os.Getenv("BRAINKIT_NATS_NAME"),
	})
	if err != nil {
		return fmt.Errorf("sdk: transport: %w", err)
	}
	defer transport.Close()

	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		return fmt.Errorf("sdk: create router: %w", err)
	}

	rt := &pluginClient{
		remote:    messaging.NewRemoteClientWithTransport(namespace, fmt.Sprintf("plugin/%s/%s@%s", p.owner, p.name, p.version), transport),
		namespace: namespace,
	}

	// resolveTopic applies namespace + transport sanitizer (e.g., dots→dashes for NATS JetStream)
	resolveTopic := func(logicalTopic string) string {
		topic := messaging.NamespacedTopic(namespace, logicalTopic)
		return transport.SanitizeTopic(topic)
	}

	// Register tool call handlers
	for _, t := range p.tools {
		toolHandler := t.handler
		toolTopic := fmt.Sprintf("plugin.tool.%s/%s@%s/%s", p.owner, p.name, p.version, t.name)

		router.AddConsumerHandler(
			toolTopic+"_handler",
			resolveTopic(toolTopic),
			transport.Subscriber,
			func(wmsg *message.Message) error {
				result, toolErr := toolHandler(context.Background(), rt, json.RawMessage(wmsg.Payload))
				resp := messages.ToolCallResp{}
				if toolErr != nil {
					resp.SetError(toolErr.Error())
				} else {
					resp.Result = result
				}
				replyPayload, err := json.Marshal(resp)
				if err != nil {
					return err
				}
				replyMsg := message.NewMessage(watermill.NewUUID(), replyPayload)
				if correlationID := wmsg.Metadata.Get("correlationId"); correlationID != "" {
					replyMsg.Metadata.Set("correlationId", correlationID)
				} else {
					replyMsg.Metadata.Set("correlationId", wmsg.UUID)
				}
				return transport.Publisher.Publish(resolveTopic(toolTopic+".result"), replyMsg)
			},
		)
	}

	// Register event subscription handlers
	for _, sub := range p.subscriptions {
		subHandler := sub.handler
		router.AddConsumerHandler(
			sub.topic+"_sub_handler",
			resolveTopic(sub.topic),
			transport.Subscriber,
			func(wmsg *message.Message) error {
				subHandler(context.Background(), json.RawMessage(wmsg.Payload), rt)
				return nil
			},
		)
	}

	// Start router
	go func() {
		if routerErr := router.Run(context.Background()); routerErr != nil {
			log.Printf("[plugin:%s] router stopped: %v", p.name, routerErr)
		}
	}()
	<-router.Running()

	manifest := p.buildManifest()
	manifestMsg := messages.PluginManifestMsg{
		Owner:       manifest.Owner,
		Name:        manifest.Name,
		Version:     manifest.Version,
		Description: manifest.Description,
		Tools:       toPluginToolDefs(manifest.Tools),
		Subscriptions: func() []string {
			out := make([]string, 0, len(manifest.Subscriptions))
			for _, sub := range manifest.Subscriptions {
				out = append(out, sub.Topic)
			}
			return out
		}(),
		Events: func() []string {
			out := make([]string, 0, len(manifest.Events))
			for _, evt := range manifest.Events {
				out = append(out, evt.Name)
			}
			return out
		}(),
	}
	pubResult, err := Publish(rt, context.Background(), manifestMsg)
	if err != nil {
		return fmt.Errorf("sdk: publish manifest: %w", err)
	}
	regCh := make(chan error, 1)
	cancelReg, err := SubscribeTo[messages.PluginManifestResp](rt, context.Background(), pubResult.ReplyTo, func(resp messages.PluginManifestResp, msg messages.Message) {
		if resp.Error != "" {
			regCh <- fmt.Errorf("sdk: register manifest: %s", resp.Error)
		} else {
			regCh <- nil
		}
	})
	if err != nil {
		return fmt.Errorf("sdk: subscribe manifest result: %w", err)
	}
	defer cancelReg()

	select {
	case regErr := <-regCh:
		if regErr != nil {
			return regErr
		}
	case <-time.After(30 * time.Second):
		return &TimeoutError{Operation: "plugin manifest registration"}
	}

	// Call OnStart
	if p.onStartFn != nil {
		go func() {
			if startErr := p.onStartFn(rt); startErr != nil {
				log.Printf("[plugin:%s] OnStart error: %v", p.name, startErr)
			}
		}()
	}

	// Print ready signal
	fmt.Fprintf(os.Stdout, "READY:%s/%s@%s\n", p.owner, p.name, p.version)

	// Wait for shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh

	if p.onStopFn != nil {
		if err := p.onStopFn(); err != nil {
			log.Printf("[plugin:%s] OnStop error: %v", p.name, err)
		}
	}

	return router.Close()
}

func toPluginToolDefs(defs []ToolDefinition) []messages.PluginToolDef {
	out := make([]messages.PluginToolDef, 0, len(defs))
	for _, def := range defs {
		out = append(out, messages.PluginToolDef{
			Name:        def.Name,
			Description: def.Description,
			InputSchema: def.InputSchema,
		})
	}
	return out
}
