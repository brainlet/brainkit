package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/gateway/gatewaymsg"
	"github.com/brainlet/brainkit/sdk"
)

// subscribeBusCommands subscribes to gateway.http.route.* bus topics
// so .ts admin code can manage routes dynamically.
func (gw *Gateway) subscribeBusCommands() error {
	ctx := context.Background()
	subscribe := func(topic string, handler func(sdk.Message)) error {
		sub, err := gw.subscribeRaw(ctx, topic, handler)
		if err != nil {
			return errors.Join(fmt.Errorf("gateway: subscribe %s: %w", topic, err), gw.unsubscribeBusCommands(ctx))
		}
		gw.addBusSubscription(sub)
		return nil
	}

	// gateway.http.route.add
	if err := subscribe("gateway.http.route.add", func(msg sdk.Message) {
		var req gatewaymsg.GatewayRouteAddMsg
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			gw.replyError(msg, "invalid payload: "+err.Error())
			return
		}
		callerSource, deploymentCaller := routeMutationCaller(msg)
		owner := strings.TrimSpace(req.Owner)
		if deploymentCaller {
			if callerSource == "" {
				gw.replyError(msg, "deployment caller source is required")
				return
			}
			if owner == "" {
				owner = callerSource
			} else if owner != callerSource {
				gw.replyError(msg, "cannot add route owned by "+owner)
				return
			}
			if existingOwner, exists := gw.routes.ownerFor(req.Method, req.Path); exists && existingOwner != callerSource {
				gw.replyError(msg, "cannot replace route owned by "+existingOwner)
				return
			}
			if !topicOwnedBySource(req.Topic, callerSource) {
				gw.replyError(msg, "cannot add route for topic outside "+callerSource)
				return
			}
		}
		rt := routeTypeFromName(req.Type)
		gw.routes.add(&route{
			Method: req.Method, Path: req.Path, Topic: req.Topic,
			Type: rt, Owner: owner,
		})
		gw.logger.Info("route added via bus", slog.String("method", req.Method), slog.String("path", req.Path), slog.String("topic", req.Topic), slog.String("owner", owner))
		gw.replyJSON(msg, gatewaymsg.GatewayRouteAddResp{Added: true})
	}); err != nil {
		return err
	}

	// gateway.http.route.remove
	if err := subscribe("gateway.http.route.remove", func(msg sdk.Message) {
		var req gatewaymsg.GatewayRouteRemoveMsg
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			gw.replyError(msg, "invalid payload: "+err.Error())
			return
		}
		removed := 0
		callerSource, deploymentCaller := routeMutationCaller(msg)
		if req.Owner != "" {
			if deploymentCaller && req.Owner != callerSource {
				gw.replyError(msg, "cannot remove routes owned by "+req.Owner)
				return
			}
			removed = gw.routes.removeByOwner(req.Owner)
		} else if req.Method != "" && req.Path != "" {
			if owner, exists := gw.routes.ownerFor(req.Method, req.Path); exists {
				if deploymentCaller && owner != callerSource {
					gw.replyError(msg, "cannot remove route owned by "+owner)
					return
				}
				if gw.routes.remove(req.Method, req.Path) {
					removed = 1
				}
			}
		}
		gw.logger.Info("routes removed via bus", slog.Int("removed", removed))
		gw.replyJSON(msg, gatewaymsg.GatewayRouteRemoveResp{Removed: removed})
	}); err != nil {
		return err
	}

	// gateway.http.route.list
	if err := subscribe("gateway.http.route.list", func(msg sdk.Message) {
		routes := gw.routes.list()
		callerSource, deploymentCaller := routeMutationCaller(msg)
		infos := make([]gatewaymsg.GatewayRouteInfo, 0, len(routes))
		for _, r := range routes {
			if deploymentCaller && r.Owner != callerSource {
				continue
			}
			infos = append(infos, gatewaymsg.GatewayRouteInfo{
				Method: r.Method, Path: r.Path, Topic: r.Topic,
				Type: r.Type, Owner: r.Owner,
			})
		}
		gw.replyJSON(msg, gatewaymsg.GatewayRouteListResp{Routes: infos})
	}); err != nil {
		return err
	}

	// gateway.http.status
	if err := subscribe("gateway.http.status", func(msg sdk.Message) {
		snapshot := gw.DebugSnapshot()
		gw.replyJSON(msg, gatewaymsg.GatewayStatusResp{
			Listening:           snapshot.Listening,
			ServerAttached:      snapshot.ServerAttached,
			Address:             snapshot.Address,
			RouteCount:          snapshot.RouteCount,
			RouteSubscriptions:  snapshot.RouteSubscriptions,
			ActiveConnections:   snapshot.ActiveConnections,
			StreamSessions:      snapshot.StreamSessions,
			TerminalSessions:    snapshot.TerminalSessions,
			StreamSubscriptions: snapshot.StreamSubscriptions,
			SessionSweepRunning: snapshot.SessionSweepRunning,
		})
	}); err != nil {
		return err
	}
	return nil
}

func routeMutationCaller(msg sdk.Message) (string, bool) {
	callerSource := strings.TrimSpace(msg.CallerID)
	if msg.Metadata != nil {
		if source := strings.TrimSpace(msg.Metadata["brainkit.caller.source"]); source != "" {
			callerSource = source
		}
		if msg.Metadata["brainkit.caller.kind"] == "deployment" {
			return callerSource, true
		}
	}
	return callerSource, strings.HasSuffix(callerSource, ".ts")
}

func topicOwnedBySource(topic, source string) bool {
	if topic == "" || source == "" {
		return false
	}
	name := strings.TrimSuffix(source, ".ts")
	name = strings.TrimSuffix(name, ".js")
	name = strings.ReplaceAll(name, "/", ".")
	return strings.HasPrefix(topic, "ts."+name+".")
}

type rawHandleSubscriber interface {
	SubscribeRawHandle(context.Context, string, func(sdk.Message)) (bkmodule.Handle, error)
}

func (gw *Gateway) subscribeRaw(ctx context.Context, topic string, handler func(sdk.Message)) (bkmodule.Handle, error) {
	if rt, ok := gw.rt.(rawHandleSubscriber); ok {
		return rt.SubscribeRawHandle(ctx, topic, handler)
	}
	unsub, err := gw.rt.SubscribeRaw(ctx, topic, handler)
	if err != nil {
		return nil, err
	}
	return bkmodule.HandleFunc(func(context.Context) error {
		unsub()
		return nil
	}), nil
}

func (gw *Gateway) addBusSubscription(sub bkmodule.Handle) {
	gw.busMu.Lock()
	defer gw.busMu.Unlock()
	gw.busSubs = append(gw.busSubs, sub)
}

func (gw *Gateway) unsubscribeBusCommands(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	gw.busMu.Lock()
	subs := append([]bkmodule.Handle{}, gw.busSubs...)
	gw.busSubs = nil
	gw.busMu.Unlock()
	var firstErr error
	var failed []bkmodule.Handle
	for _, sub := range subs {
		if sub == nil {
			continue
		}
		if err := sub.Close(ctx); err != nil {
			firstErr = errors.Join(firstErr, err)
			failed = append(failed, sub)
		}
	}
	if len(failed) > 0 {
		gw.busMu.Lock()
		gw.busSubs = append(failed, gw.busSubs...)
		gw.busMu.Unlock()
	}
	return firstErr
}

func (gw *Gateway) routeSubscriptionCount() int {
	gw.busMu.Lock()
	defer gw.busMu.Unlock()
	return len(gw.busSubs)
}

func (gw *Gateway) replyJSON(msg sdk.Message, resp any) {
	replyTo := msg.Metadata["replyTo"]
	correlationID := msg.Metadata["correlationId"]
	if replyTo == "" {
		return
	}
	payload, _ := json.Marshal(resp)
	if replier, ok := gw.rt.(interface {
		ReplyRaw(ctx context.Context, replyTo, correlationID string, payload json.RawMessage, done bool) error
	}); ok {
		replier.ReplyRaw(context.Background(), replyTo, correlationID, payload, true)
	}
}

func (gw *Gateway) replyError(msg sdk.Message, errMsg string) {
	gw.replyJSON(msg, map[string]string{"error": errMsg})
}
