package brainkit

import (
	"context"
	"fmt"
	"log"

	"github.com/brainlet/brainkit/bus"
	"github.com/brainlet/brainkit/internal/transport"
	pluginv1 "github.com/brainlet/brainkit/proto/plugin/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// connectPeer establishes a gRPC connection to a remote Kit.
func (k *Kit) connectPeer(name, addr string) error {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("connect to peer %s at %s: %w", name, addr, err)
	}

	client := pluginv1.NewBrainkitHostServiceClient(conn)

	resp, err := client.Handshake(context.Background(), &pluginv1.HandshakeRequest{
		Name:    k.config.Name,
		Version: "v1",
		Type:    "kit",
	})
	if err != nil {
		conn.Close()
		return fmt.Errorf("handshake with %s: %w", name, err)
	}
	if !resp.Accepted {
		conn.Close()
		return fmt.Errorf("handshake with %s rejected: %s", name, resp.RejectionReason)
	}

	stream, err := client.MessageStream(context.Background())
	if err != nil {
		conn.Close()
		return fmt.Errorf("open stream to %s: %w", name, err)
	}

	// Send identity message so the remote peer can route callbacks.
	stream.Send(&pluginv1.PluginMessage{
		Id:       uuid.NewString(),
		Type:     "identity",
		CallerId: k.config.Name,
	})

	pc := &transport.PeerConn{
		Name:   name,
		Addr:   addr,
		Conn:   conn,
		Stream: stream,
		Done:   make(chan struct{}),
	}

	if k.transport != nil {
		k.transport.AddPeer(pc)
	}

	go k.readPeerStream(name, pc)

	log.Printf("[kit] connected to peer %q at %s", name, addr)
	return nil
}

// readPeerStream forwards messages received from a remote Kit back onto the local bus.
func (k *Kit) readPeerStream(name string, pc *transport.PeerConn) {
	defer func() {
		if k.transport != nil {
			k.transport.RemovePeer(name)
		}
		pc.CloseDone()
		log.Printf("[kit] peer %q stream closed", name)
	}()

	for {
		msg, err := pc.Stream.Recv()
		if err != nil {
			log.Printf("[kit] peer %s recv: %v", name, err)
			return
		}

		switch msg.Type {
		case "ask.reply":
			if msg.ReplyTo != "" {
				k.Bus.Send(bus.Message{
					Topic:    msg.ReplyTo,
					CallerID: "peer/" + name,
					Payload:  msg.Payload,
					Metadata: msg.Metadata,
				})
			}
		case "event":
			k.Bus.Send(bus.Message{
				Topic:    msg.Topic,
				CallerID: "peer/" + name,
				TraceID:  msg.TraceId,
				Payload:  msg.Payload,
				Metadata: msg.Metadata,
			})
		default:
			log.Printf("[kit] peer %s: unknown message type %q", name, msg.Type)
		}
	}
}
