package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk"
)

// --- Failure Handling (retry, dead letter, error events) ---

// handleHandlerFailure is called when a bus handler throws a JS exception.
func (k *Kernel) handleHandlerFailure(msg sdk.Message, topic string, handlerErr error) {
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
		k.logger.Error("handler error", slog.String("topic", topic), slog.String("error", handlerErr.Error()))
		k.sendErrorResponse(msg, handlerErr)
		return
	}

	// Retries exhausted — dead letter + error response
	if retryCount >= policy.MaxRetries {
		k.logger.Error("handler exhausted", slog.String("topic", topic), slog.Int("retries", retryCount), slog.String("error", handlerErr.Error()))
		k.deadLetter(msg, topic, handlerErr, retryCount, policy)
		k.sendErrorResponse(msg, fmt.Errorf("handler failed after %d retries: %w", retryCount, handlerErr))
		k.emitHandlerExhausted(topic, handlerErr, retryCount)
		return
	}

	// Retry with backoff
	delay := computeDelay(policy, retryCount)
	nextRetry := retryCount + 1

	k.logger.Warn("handler failed, retrying",
		slog.String("topic", topic),
		slog.Int("retry", nextRetry),
		slog.Int("max_retries", policy.MaxRetries),
		slog.Duration("delay", delay),
		slog.String("error", handlerErr.Error()),
	)

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

func (k *Kernel) sendErrorResponse(msg sdk.Message, err error) {
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

func (k *Kernel) findRetryPolicy(topic string) *types.RetryPolicy {
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

func (k *Kernel) deadLetter(msg sdk.Message, topic string, err error, retries int, policy *types.RetryPolicy) {
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
	payload, _ := json.Marshal(sdk.HandlerFailedEvent{
		Topic: topic, Source: k.callerID, Error: err.Error(),
		RetryCount: retryCount, WillRetry: willRetry,
	})
	k.publish(context.Background(), sdk.HandlerFailedEvent{}.BusTopic(), payload)
}

func (k *Kernel) emitHandlerExhausted(topic string, err error, retryCount int) {
	payload, _ := json.Marshal(sdk.HandlerExhaustedEvent{
		Topic: topic, Source: k.callerID, Error: err.Error(),
		RetryCount: retryCount,
	})
	k.publish(context.Background(), sdk.HandlerExhaustedEvent{}.BusTopic(), payload)
}

func computeDelay(p *types.RetryPolicy, retryCount int) time.Duration {
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
