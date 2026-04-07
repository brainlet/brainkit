package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	quickjs "github.com/buke/quickjs-go"
	"github.com/google/uuid"
	js "github.com/brainlet/brainkit/internal/contract"
	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/sdkerrors"
	"github.com/brainlet/brainkit/sdk/messages"
)

// registerApprovalBridges adds __go_brainkit_await_approval for bus-based HITL tool approval.
func (k *Kernel) registerApprovalBridges(qctx *quickjs.Context) {
	qctx.Globals().Set(js.JSBridgeAwaitApproval,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 3 {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "args", Message: "await_approval: expected 3 args"})
			}
			approvalTopic := args[0].String()
			payload := json.RawMessage(args[1].String())
			timeoutMs := args[2].ToInt64()
			if timeoutMs <= 0 {
				timeoutMs = 30000
			}

			return qctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
				k.bridge.Go(func(goCtx context.Context) {
					timeout := time.Duration(timeoutMs) * time.Millisecond
					waitCtx, waitCancel := context.WithTimeout(goCtx, timeout)
					defer waitCancel()

					correlationID := uuid.NewString()
					replyTo := approvalTopic + ".reply." + correlationID

					// Subscribe BEFORE publishing (avoid race)
					replyCh := make(chan messages.Message, 1)
					unsub, subErr := k.remote.SubscribeRaw(waitCtx, replyTo, func(msg messages.Message) {
						select {
						case replyCh <- msg:
						default:
						}
					})
					if subErr != nil {
						qctx.Schedule(func(qctx *quickjs.Context) {
							errVal := qctx.NewError(fmt.Errorf("await_approval: subscribe: %w", subErr))
							defer errVal.Free()
							reject(errVal)
						})
						return
					}
					defer unsub()

					// Publish approval request with replyTo
					pubCtx := transport.WithPublishMeta(waitCtx, correlationID, replyTo)
					if _, pubErr := k.remote.PublishRaw(pubCtx, approvalTopic, payload); pubErr != nil {
						qctx.Schedule(func(qctx *quickjs.Context) {
							errVal := qctx.NewError(fmt.Errorf("await_approval: publish: %w", pubErr))
							defer errVal.Free()
							reject(errVal)
						})
						return
					}

					// Wait for response or timeout
					select {
					case msg := <-replyCh:
						responseJSON := string(msg.Payload)
						qctx.Schedule(func(qctx *quickjs.Context) {
							resolve(qctx.NewString(responseJSON))
						})
					case <-waitCtx.Done():
						timeoutJSON := `{"approved":false,"reason":"timeout"}`
						qctx.Schedule(func(qctx *quickjs.Context) {
							resolve(qctx.NewString(timeoutJSON))
						})
					}
				})
			})
		}))
}
