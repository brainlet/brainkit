package gateway

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/brainlet/brainkit/sdk"
)

// subscribeBusCommands subscribes to gateway.http.route.* bus topics
// so .ts admin code can manage routes dynamically.
func (gw *Gateway) subscribeBusCommands() {
	ctx := context.Background()

	// gateway.http.route.add
	if unsub, err := gw.rt.SubscribeRaw(ctx, "gateway.http.route.add", func(msg sdk.Message) {
		var req sdk.GatewayRouteAddMsg
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			gw.replyError(msg, "invalid payload: "+err.Error())
			return
		}
		rt := routeTypeFromName(req.Type)
		gw.routes.add(&route{
			Method: req.Method, Path: req.Path, Topic: req.Topic,
			Type: rt, Owner: req.Owner,
		})
		gw.logger.Info("route added via bus", slog.String("method", req.Method), slog.String("path", req.Path), slog.String("topic", req.Topic), slog.String("owner", req.Owner))
		gw.replyJSON(msg, sdk.GatewayRouteAddResp{Added: true})
	}); err == nil {
		gw.busUnsubs = append(gw.busUnsubs, unsub)
	}

	// gateway.http.route.remove
	if unsub, err := gw.rt.SubscribeRaw(ctx, "gateway.http.route.remove", func(msg sdk.Message) {
		var req sdk.GatewayRouteRemoveMsg
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			gw.replyError(msg, "invalid payload: "+err.Error())
			return
		}
		removed := 0
		callerSource := msg.CallerID
		// Ownership isolation: .ts deployments can only remove their own routes.
		// Go callers (no .ts suffix) have full access — they own the infrastructure.
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
		gw.logger.Info("routes removed via bus", slog.Int("removed", removed))
		gw.replyJSON(msg, sdk.GatewayRouteRemoveResp{Removed: removed})
	}); err == nil {
		gw.busUnsubs = append(gw.busUnsubs, unsub)
	}

	// gateway.http.route.list
	if unsub, err := gw.rt.SubscribeRaw(ctx, "gateway.http.route.list", func(msg sdk.Message) {
		routes := gw.routes.list()
		infos := make([]sdk.GatewayRouteInfo, len(routes))
		for i, r := range routes {
			infos[i] = sdk.GatewayRouteInfo{
				Method: r.Method, Path: r.Path, Topic: r.Topic,
				Type: r.Type, Owner: r.Owner,
			}
		}
		gw.replyJSON(msg, sdk.GatewayRouteListResp{Routes: infos})
	}); err == nil {
		gw.busUnsubs = append(gw.busUnsubs, unsub)
	}

	// gateway.http.status
	if unsub, err := gw.rt.SubscribeRaw(ctx, "gateway.http.status", func(msg sdk.Message) {
		gw.routes.mu.RLock()
		routeCount := len(gw.routes.routes)
		gw.routes.mu.RUnlock()
		gw.replyJSON(msg, sdk.GatewayStatusResp{
			Listening:         gw.ln != nil,
			Address:           gw.Addr(),
			RouteCount:        routeCount,
			ActiveConnections: gw.active.Load(),
		})
	}); err == nil {
		gw.busUnsubs = append(gw.busUnsubs, unsub)
	}
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
