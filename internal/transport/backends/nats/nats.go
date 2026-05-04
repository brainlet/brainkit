package nats

import (
	"fmt"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	wmnats "github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	core "github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/transport/backends/backutil"
	"github.com/brainlet/brainkit/internal/transport/backends/wm"
	natsgo "github.com/nats-io/nats.go"
)

// NewTransport creates an external NATS JetStream transport.
func NewTransport(cfg core.TransportConfig) (*core.Transport, error) {
	url := cfg.NATSURL
	if url == "" {
		url = "nats://127.0.0.1:4222"
	}

	natsSubjectCalc := func(queueGroupPrefix, topic string) *wmnats.SubjectDetail {
		safeTopic := strings.ReplaceAll(topic, ".", "-")
		qg := ""
		if queueGroupPrefix != "" {
			qg = queueGroupPrefix
		}
		return &wmnats.SubjectDetail{Primary: safeTopic, QueueGroup: qg}
	}

	consumerGroup := cfg.Namespace
	if consumerGroup == "" {
		consumerGroup = "brainkit"
	}
	consumerGroup = backutil.SanitizeDurable(consumerGroup)

	logger := watermill.NopLogger{}
	publisher, err := wmnats.NewPublisher(wmnats.PublisherConfig{
		URL:               url,
		Marshaler:         wmnats.JSONMarshaler{},
		SubjectCalculator: natsSubjectCalc,
		JetStream:         wmnats.JetStreamConfig{AutoProvision: true, TrackMsgId: true},
	}, logger)
	if err != nil {
		return nil, fmt.Errorf("nats publisher: %w", err)
	}

	// DeliverNew avoids replaying persisted stream messages after restart when
	// Watermill recreates the JetStream consumer.
	deliverNew := []natsgo.SubOpt{natsgo.DeliverNew()}

	subscriber, err := wmnats.NewSubscriber(wmnats.SubscriberConfig{
		URL:               url,
		QueueGroupPrefix:  consumerGroup,
		SubscribersCount:  1,
		CloseTimeout:      15 * time.Second,
		AckWaitTimeout:    30 * time.Second,
		SubscribeTimeout:  30 * time.Second,
		Unmarshaler:       wmnats.JSONMarshaler{},
		SubjectCalculator: natsSubjectCalc,
		JetStream: wmnats.JetStreamConfig{
			AutoProvision:    true,
			DurablePrefix:    consumerGroup,
			TrackMsgId:       true,
			SubscribeOptions: deliverNew,
		},
	}, logger)
	if err != nil {
		_ = publisher.Close()
		return nil, fmt.Errorf("nats subscriber: %w", err)
	}

	fanOutID := consumerGroup + "-fo-" + watermill.NewShortUUID()
	fanOutSub, err := wmnats.NewSubscriber(wmnats.SubscriberConfig{
		URL:               url,
		QueueGroupPrefix:  "",
		SubscribersCount:  1,
		CloseTimeout:      15 * time.Second,
		AckWaitTimeout:    30 * time.Second,
		SubscribeTimeout:  30 * time.Second,
		Unmarshaler:       wmnats.JSONMarshaler{},
		SubjectCalculator: natsSubjectCalc,
		JetStream: wmnats.JetStreamConfig{
			AutoProvision:    true,
			DurablePrefix:    fanOutID,
			TrackMsgId:       true,
			SubscribeOptions: deliverNew,
		},
	}, logger)
	if err != nil {
		_ = publisher.Close()
		_ = subscriber.Close()
		return nil, fmt.Errorf("nats fan-out subscriber: %w", err)
	}

	return core.NewManagedTransport(
		"nats",
		wm.Publisher{Inner: publisher},
		wm.Subscriber{Inner: subscriber},
		wm.Subscriber{Inner: fanOutSub},
		func(topic string) string {
			r := strings.NewReplacer(".", "-", "/", "-", "@", "-", " ", "-")
			return r.Replace(topic)
		},
		publisher.Close,
		core.OnceCloser(subscriber.Close),
		core.OnceCloser(fanOutSub.Close),
	), nil
}
