package embeddednats

import (
	"fmt"
	"os"
	"time"

	core "github.com/brainlet/brainkit/internal/transport"
	natsbackend "github.com/brainlet/brainkit/internal/transport/backends/nats"
	"github.com/nats-io/nats-server/v2/server"
)

// EmbeddedNATS manages an in-process NATS server with JetStream.
type EmbeddedNATS struct {
	server   *server.Server
	storeDir string
	tempDir  bool
}

// EmbeddedNATSConfig configures the embedded NATS server.
type EmbeddedNATSConfig struct {
	// StoreDir is the JetStream persistence directory.
	// Empty = ephemeral temp directory (cleaned up on Shutdown).
	StoreDir string
}

// NewEmbeddedNATS starts an in-process NATS server with JetStream enabled.
// The server listens on a random port on 127.0.0.1.
func NewEmbeddedNATS(cfg EmbeddedNATSConfig) (*EmbeddedNATS, error) {
	storeDir := cfg.StoreDir
	tempDir := false
	if storeDir == "" {
		dir, err := os.MkdirTemp("", "brainkit-nats-*")
		if err != nil {
			return nil, fmt.Errorf("embedded nats: create temp dir: %w", err)
		}
		storeDir = dir
		tempDir = true
	}

	opts := &server.Options{
		Host:      "127.0.0.1",
		Port:      -1,
		NoLog:     true,
		NoSigs:    true,
		JetStream: true,
		StoreDir:  storeDir,
	}

	ns, err := server.NewServer(opts)
	if err != nil {
		if tempDir {
			os.RemoveAll(storeDir)
		}
		return nil, fmt.Errorf("embedded nats: create server: %w", err)
	}

	ns.Start()

	if !ns.ReadyForConnections(10 * time.Second) {
		ns.Shutdown()
		if tempDir {
			os.RemoveAll(storeDir)
		}
		return nil, fmt.Errorf("embedded nats: server not ready after 10s")
	}

	return &EmbeddedNATS{
		server:   ns,
		storeDir: storeDir,
		tempDir:  tempDir,
	}, nil
}

// ClientURL returns the NATS connection URL for this embedded server.
func (e *EmbeddedNATS) ClientURL() string {
	return e.server.ClientURL()
}

// Shutdown stops the embedded server and cleans up temp directories.
func (e *EmbeddedNATS) Shutdown() {
	e.server.Shutdown()
	e.server.WaitForShutdown()
	if e.tempDir {
		os.RemoveAll(e.storeDir)
	}
}

// NewTransport creates an embedded NATS server and connects a NATS transport to it.
func NewTransport(cfg core.TransportConfig) (*core.Transport, error) {
	embedded, err := NewEmbeddedNATS(EmbeddedNATSConfig{
		StoreDir: cfg.NATSStoreDir,
	})
	if err != nil {
		return nil, fmt.Errorf("embedded nats: %w", err)
	}

	cfg.NATSURL = embedded.ClientURL()
	cfg.Type = "nats"

	transport, err := natsbackend.NewTransport(cfg)
	if err != nil {
		embedded.Shutdown()
		return nil, err
	}

	transport.Kind = "embedded"
	transport.AppendClose(func() error {
		embedded.Shutdown()
		return nil
	})

	return transport, nil
}
