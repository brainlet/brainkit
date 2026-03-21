package plugin

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brainlet/brainkit/bus"
	pluginv1 "github.com/brainlet/brainkit/proto/plugin/v1"
	"github.com/google/uuid"
)

// readStream processes messages from plugin -> Kit.
func (pm *Manager) readStream(name string, pc *conn) {
	defer close(pc.done)

	for {
		msg, err := pc.stream.Recv()
		if err != nil {
			log.Printf("[plugin:%s] stream error: %v", name, err)

			select {
			case <-pc.done:
				return
			case <-pc.stopping:
				return
			default:
				if pm.recoverStream(name, pc) {
					continue
				}
			}

			return
		}

		select {
		case <-pc.eventSem:
		default:
		}

		switch msg.Type {
		case "tool.result", "reply":
			if msg.ReplyTo != "" {
				pm.bridge.Bus().Send(bus.Message{
					Topic:    msg.ReplyTo,
					CallerID: "plugin/" + name,
					Payload:  msg.Payload,
					TraceID:  msg.TraceId,
				})
			}

		case "bus.send", "send":
			pm.bridge.Bus().Send(bus.Message{
				Topic:    msg.Topic,
				CallerID: "plugin/" + name,
				Payload:  msg.Payload,
				TraceID:  msg.TraceId,
				Metadata: msg.Metadata,
			})

		case "bus.ask", "ask":
			go func(m *pluginv1.PluginMessage) {
				pm.bridge.Bus().Ask(bus.Message{
					Topic:    m.Topic,
					CallerID: "plugin/" + name,
					Payload:  m.Payload,
					TraceID:  m.TraceId,
				}, func(reply bus.Message) {
					pc.safeSend(&pluginv1.PluginMessage{
						Id:      uuid.NewString(),
						Type:    "ask.reply",
						ReplyTo: m.ReplyTo,
						TraceId: reply.TraceID,
						Payload: reply.Payload,
					})
				})
			}(msg)

		case "intercept.result":
			replyTo := msg.ReplyTo
			if replyTo == "" {
				log.Printf("[plugin:%s] intercept.result missing reply_to", name)
				continue
			}
			result := interceptResult{
				Payload:  msg.Payload,
				Metadata: msg.Metadata,
			}
			pc.interceptMu.Lock()
			ch, ok := pc.interceptReplies[replyTo]
			if ok {
				delete(pc.interceptReplies, replyTo)
			}
			pc.interceptMu.Unlock()
			if ok {
				ch <- result
			} else {
				log.Printf("[plugin:%s] intercept.result for unknown reply %q", name, replyTo)
			}
		}
	}
}

func (pm *Manager) recoverStream(name string, pc *conn) bool {
	log.Printf("[plugin:%s] attempting stream recovery", name)

	stream, err := pc.client.MessageStream(context.Background())
	if err != nil {
		log.Printf("[plugin:%s] stream recovery failed: %v", name, err)
		return false
	}

	pc.sendMu.Lock()
	pc.stream = stream
	pc.sendMu.Unlock()

	pc.safeSend(&pluginv1.PluginMessage{
		Id:   uuid.NewString(),
		Type: "lifecycle.start",
	})

	pm.reprocessSubscriptions(name, pc)

	log.Printf("[plugin:%s] stream recovered", name)
	return true
}

func (pm *Manager) reprocessSubscriptions(name string, pc *conn) {
	for _, subID := range pc.subs {
		pm.bridge.Bus().Off(subID)
	}
	pc.subs = pc.subs[:0]

	m := pc.manifest
	for _, sub := range m.Subscriptions {
		topic := sub.Topic
		subID := pm.bridge.Bus().On(topic, func(msg bus.Message, _ bus.ReplyFunc) {
			pc.safeSendEvent(&pluginv1.PluginMessage{
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
}

func (pm *Manager) callPluginTool(pluginName, toolName string, input []byte) ([]byte, error) {
	pm.mu.Lock()
	pc, ok := pm.plugins[pluginName]
	pm.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("plugin %q not connected", pluginName)
	}

	replyTo := "_plugin_tool." + uuid.NewString()

	ch := make(chan []byte, 1)
	subID := pm.bridge.Bus().On(replyTo, func(msg bus.Message, _ bus.ReplyFunc) {
		ch <- msg.Payload
	})
	defer pm.bridge.Bus().Off(subID)

	pc.safeSend(&pluginv1.PluginMessage{
		Id:      uuid.NewString(),
		Type:    "tool.call",
		Topic:   toolName,
		ReplyTo: replyTo,
		Payload: input,
	})

	select {
	case result := <-ch:
		return result, nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("plugin %q tool %q: timeout", pluginName, toolName)
	}
}
