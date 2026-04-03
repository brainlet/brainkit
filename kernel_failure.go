package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/brainlet/brainkit/sdk/messages"
)

// --- Failure Handling (retry, dead letter, error events) ---

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
