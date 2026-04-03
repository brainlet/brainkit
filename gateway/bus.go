package gateway

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/brainlet/brainkit/sdk/messages"
)

// subscribeBusCommands subscribes to gateway.http.route.* bus topics
// so .ts admin code can manage routes dynamically.
func (gw *Gateway) subscribeBusCommands() {
	ctx := context.Background()

	// gateway.http.route.add
	if unsub, err := gw.rt.SubscribeRaw(ctx, "gateway.http.route.add", func(msg messages.Message) {
		if gw.rbacChecker != nil {
			if err := gw.rbacChecker.CheckCommand(msg.CallerID, "gateway.http.route.add"); err != nil {
				gw.replyError(msg, "permission denied: "+err.Error())
				return
			}
		}
		var req messages.GatewayRouteAddMsg
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			gw.replyError(msg, "invalid payload: "+err.Error())
			return
		}
		rt := routeTypeFromName(req.Type)
		gw.routes.add(&route{
			Method: req.Method, Path: req.Path, Topic: req.Topic,
			Type: rt, Owner: req.Owner,
		})
		log.Printf("[gateway] route added via bus: %s %s → %s (owner: %s)", req.Method, req.Path, req.Topic, req.Owner)
		gw.replyJSON(msg, messages.GatewayRouteAddResp{Added: true})
	}); err == nil {
		gw.busUnsubs = append(gw.busUnsubs, unsub)
	}

	// gateway.http.route.remove
	if unsub, err := gw.rt.SubscribeRaw(ctx, "gateway.http.route.remove", func(msg messages.Message) {
		if gw.rbacChecker != nil {
			if err := gw.rbacChecker.CheckCommand(msg.CallerID, "gateway.http.route.remove"); err != nil {
				gw.replyError(msg, "permission denied: "+err.Error())
				return
			}
		}
		var req messages.GatewayRouteRemoveMsg
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			gw.replyError(msg, "invalid payload: "+err.Error())
			return
		}
		removed := 0
		callerSource := msg.CallerID
		// Ownership isolation: .ts deployments can only remove their own routes.
		// Go callers (no .ts suffix) have full access — they own the infrastructure
		// and the RBAC check above already authorizes the command.
		isDeploymentCaller := strings.HasSuffix(callerSource, ".ts")
		if req.Owner != "" {
			if isDeploymentCaller && req.Owner != callerSource {
				gw.replyError(msg, "cannot remove routes owned by "+req.Owner)
				return
			}
			removed = gw.routes.removeByOwner(req.Owner)
		} else if req.Method != "" && req.Path != "" {
			matched, _ := gw.routes.match(req.Method, req.Path)
			if matched != nil {
				if isDeploymentCaller && matched.Owner != callerSource && matched.Owner != "" {
					gw.replyError(msg, "cannot remove route owned by "+matched.Owner)
					return
				}
				if gw.routes.remove(req.Method, req.Path) {
					removed = 1
				}
			}
		}
		log.Printf("[gateway] routes removed via bus: %d", removed)
		gw.replyJSON(msg, messages.GatewayRouteRemoveResp{Removed: removed})
	}); err == nil {
		gw.busUnsubs = append(gw.busUnsubs, unsub)
	}

	// gateway.http.route.list
	if unsub, err := gw.rt.SubscribeRaw(ctx, "gateway.http.route.list", func(msg messages.Message) {
		if gw.rbacChecker != nil {
			if err := gw.rbacChecker.CheckCommand(msg.CallerID, "gateway.http.route.list"); err != nil {
				gw.replyError(msg, "permission denied: "+err.Error())
				return
			}
		}
		routes := gw.routes.list()
		infos := make([]messages.GatewayRouteInfo, len(routes))
		for i, r := range routes {
			infos[i] = messages.GatewayRouteInfo{
				Method: r.Method, Path: r.Path, Topic: r.Topic,
				Type: r.Type, Owner: r.Owner,
			}
		}
		gw.replyJSON(msg, messages.GatewayRouteListResp{Routes: infos})
	}); err == nil {
		gw.busUnsubs = append(gw.busUnsubs, unsub)
	}

	// gateway.http.status
	if unsub, err := gw.rt.SubscribeRaw(ctx, "gateway.http.status", func(msg messages.Message) {
		if gw.rbacChecker != nil {
			if err := gw.rbacChecker.CheckCommand(msg.CallerID, "gateway.http.status"); err != nil {
				gw.replyError(msg, "permission denied: "+err.Error())
				return
			}
		}
		gw.routes.mu.RLock()
		routeCount := len(gw.routes.routes)
		gw.routes.mu.RUnlock()
		gw.replyJSON(msg, messages.GatewayStatusResp{
			Listening:         gw.ln != nil,
			Address:           gw.Addr(),
			RouteCount:        routeCount,
			ActiveConnections: gw.active.Load(),
		})
	}); err == nil {
		gw.busUnsubs = append(gw.busUnsubs, unsub)
	}
}

func (gw *Gateway) replyJSON(msg messages.Message, resp any) {
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

func (gw *Gateway) replyError(msg messages.Message, errMsg string) {
	gw.replyJSON(msg, map[string]string{"error": errMsg})
}
