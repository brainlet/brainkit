package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"github.com/brainlet/brainkit/internal/syncx"
	"time"

	"github.com/brainlet/brainkit/sdk"
)

const (
	defaultMulticastService = "_brainkit._tcp"
	multicastAddr           = "224.0.0.251:5353"
)

// Multicast uses UDP multicast for zero-config peer discovery on a LAN.
// Custom protocol: "BRAINKIT|{service}|{name}|{address}" — NOT actual mDNS/DNS-SD.
type Multicast struct {
	service string
	self    *Peer
	mu      syncx.RWMutex
	peers   map[string]Peer
	conn    *net.UDPConn
	cancel  context.CancelFunc
	closed  chan struct{}
}

func NewMulticast(service string) (*Multicast, error) {
	if service == "" {
		service = defaultMulticastService
	}

	addr, err := net.ResolveUDPAddr("udp4", multicastAddr)
	if err != nil {
		return nil, fmt.Errorf("multicast: resolve addr: %w", err)
	}

	conn, err := net.ListenMulticastUDP("udp4", nil, addr)
	if err != nil {
		return nil, fmt.Errorf("multicast: listen: %w", err)
	}
	conn.SetReadBuffer(8192)

	ctx, cancel := context.WithCancel(context.Background())

	d := &Multicast{
		service: service,
		peers:   make(map[string]Peer),
		conn:    conn,
		cancel:  cancel,
		closed:  make(chan struct{}),
	}

	go d.listenLoop(ctx)
	return d, nil
}

func (d *Multicast) Resolve(name string) (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	peer, ok := d.peers[name]
	if !ok {
		return "", &sdk.NotFoundError{Resource: "peer", Name: name}
	}
	return peer.Address, nil
}

func (d *Multicast) Browse() ([]Peer, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	result := make([]Peer, 0, len(d.peers))
	for _, peer := range d.peers {
		result = append(result, peer)
	}
	return result, nil
}

func (d *Multicast) Register(self Peer) error {
	d.mu.Lock()
	d.self = &self
	d.mu.Unlock()
	return d.announce(self)
}

func (d *Multicast) Close() error {
	d.cancel()
	d.conn.Close()
	<-d.closed
	return nil
}

func (d *Multicast) announce(self Peer) error {
	addr, err := net.ResolveUDPAddr("udp4", multicastAddr)
	if err != nil {
		return err
	}
	sendConn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		return err
	}
	defer sendConn.Close()

	msg := fmt.Sprintf("BRAINKIT|%s|%s|%s", d.service, self.Name, self.Address)
	_, err = sendConn.Write([]byte(msg))
	return err
}

func (d *Multicast) listenLoop(ctx context.Context) {
	defer close(d.closed)

	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		d.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, _, err := d.conn.ReadFromUDP(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				d.mu.RLock()
				self := d.self
				d.mu.RUnlock()
				if self != nil {
					d.announce(*self)
				}
				continue
			}
			if ctx.Err() != nil {
				return
			}
			continue
		}

		d.parseAnnouncement(string(buf[:n]))
	}
}

func (d *Multicast) parseAnnouncement(msg string) {
	parts := strings.SplitN(msg, "|", 4)
	if len(parts) != 4 || parts[0] != "BRAINKIT" {
		return
	}
	service, name, address := parts[1], parts[2], parts[3]
	if service != d.service {
		return
	}

	d.mu.RLock()
	if d.self != nil && name == d.self.Name {
		d.mu.RUnlock()
		return
	}
	d.mu.RUnlock()

	d.mu.Lock()
	d.peers[name] = Peer{Name: name, Address: address}
	d.mu.Unlock()

	slog.Info("discovered peer", slog.String("peer", name), slog.String("address", address))
}
