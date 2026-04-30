//go:build ios
// +build ios

package iospacket

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/observerss/detour2/common"
	"github.com/observerss/detour2/local"
	"github.com/xjasonlyu/tun2socks/v2/core"
	"github.com/xjasonlyu/tun2socks/v2/core/device/iobased"
	tunproxy "github.com/xjasonlyu/tun2socks/v2/proxy"
	"github.com/xjasonlyu/tun2socks/v2/tunnel"
	"github.com/xjasonlyu/tun2socks/v2/tunnel/statistic"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

const (
	defaultMTU          = 1500
	defaultSocks5Port   = 7776
	defaultPoolSize     = 64
	localStartupTimeout = 3 * time.Second
)

type PacketFlow interface {
	ReadPacket() ([]byte, error)
	WritePacket(packet []byte) error
	Close()
}

type Backend struct {
	mu          sync.Mutex
	packetFlow  PacketFlow
	rw          *packetFlowReadWriter
	localServer *local.Local
	endpoint    *iobased.Endpoint
	stack       *stack.Stack
	tunnel      *tunnel.Tunnel
	mtu         int
	started     bool
}

func NewBackend(packetFlow PacketFlow, remoteAddress string, password string, mtu int, socks5Port int, poolSize int) (*Backend, error) {
	if packetFlow == nil {
		return nil, fmt.Errorf("packetFlow is nil")
	}
	remoteAddress, err := normalizeRemoteAddress(remoteAddress)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(password) == "" {
		return nil, fmt.Errorf("password is empty")
	}
	if mtu <= 0 {
		mtu = defaultMTU
	}
	if socks5Port <= 0 {
		socks5Port = defaultSocks5Port
	}
	if poolSize <= 0 {
		poolSize = defaultPoolSize
	}

	server := local.NewLocal(&common.LocalConfig{
		Listen:   fmt.Sprintf("tcp://127.0.0.1:%d", socks5Port),
		Remotes:  remoteAddress,
		Password: password,
		Proto:    "socks5",
		PoolSize: poolSize,
	})

	return &Backend{
		packetFlow:  packetFlow,
		localServer: server,
		mtu:         mtu,
	}, nil
}

func (b *Backend) Start() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.started {
		return nil
	}

	socksAddress := b.localServer.Address
	localErr := make(chan error, 1)
	go func() {
		if err := b.localServer.RunLocal(); err != nil && !b.localServer.IsStopped() {
			localErr <- err
		}
	}()

	if err := waitForLocalSocks(socksAddress, localErr); err != nil {
		b.localServer.StopLocal()
		return err
	}

	proxyClient, err := tunproxy.NewSocks5(socksAddress, "", "")
	if err != nil {
		b.localServer.StopLocal()
		return err
	}

	rw := &packetFlowReadWriter{packetFlow: b.packetFlow}
	endpoint, err := iobased.New(rw, uint32(b.mtu), 0)
	if err != nil {
		b.localServer.StopLocal()
		return err
	}
	tunTunnel := tunnel.New(proxyClient, statistic.DefaultManager)
	tunTunnel.ProcessAsync()

	tunStack, err := core.CreateStack(&core.Config{
		LinkEndpoint:     endpoint,
		TransportHandler: tunTunnel,
	})
	if err != nil {
		tunTunnel.Close()
		endpoint.Close()
		b.localServer.StopLocal()
		return err
	}

	b.rw = rw
	b.endpoint = endpoint
	b.tunnel = tunTunnel
	b.stack = tunStack
	b.started = true
	return nil
}

func (b *Backend) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.rw != nil {
		_ = b.rw.Close()
		b.rw = nil
	}
	if b.endpoint != nil {
		b.endpoint.Close()
		b.endpoint = nil
	}
	if b.stack != nil {
		b.stack.Close()
		b.stack.Wait()
		b.stack = nil
	}
	if b.tunnel != nil {
		b.tunnel.Close()
		b.tunnel = nil
	}
	if b.localServer != nil {
		b.localServer.StopLocal()
	}
	b.started = false
}

func waitForLocalSocks(address string, localErr <-chan error) error {
	deadline := time.Now().Add(localStartupTimeout)
	var lastDialErr error
	for {
		select {
		case err := <-localErr:
			return fmt.Errorf("start detour2 local socks5: %w", err)
		default:
		}

		conn, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		lastDialErr = err
		if time.Now().After(deadline) {
			return fmt.Errorf("detour2 local socks5 did not listen on %s: %w", address, lastDialErr)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

type packetFlowReadWriter struct {
	mu         sync.Mutex
	packetFlow PacketFlow
	closed     bool
}

func (rw *packetFlowReadWriter) Read(p []byte) (int, error) {
	if rw.isClosed() {
		return 0, io.ErrClosedPipe
	}
	packet, err := rw.packetFlow.ReadPacket()
	if err != nil {
		return 0, err
	}
	if len(packet) > len(p) {
		return 0, fmt.Errorf("packet size %d exceeds buffer size %d", len(packet), len(p))
	}
	copy(p, packet)
	return len(packet), nil
}

func (rw *packetFlowReadWriter) Write(p []byte) (int, error) {
	if rw.isClosed() {
		return 0, io.ErrClosedPipe
	}
	packet := append([]byte(nil), p...)
	if err := rw.packetFlow.WritePacket(packet); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (rw *packetFlowReadWriter) Close() error {
	rw.mu.Lock()
	if rw.closed {
		rw.mu.Unlock()
		return nil
	}
	rw.closed = true
	rw.mu.Unlock()
	rw.packetFlow.Close()
	return nil
}

func (rw *packetFlowReadWriter) isClosed() bool {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	return rw.closed
}

func normalizeRemoteAddress(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("remoteAddress is empty")
	}
	if !strings.Contains(value, "://") {
		value = "tcp://" + value
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("remoteAddress is invalid: %w", err)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("remoteAddress is missing host: %s", value)
	}

	switch parsed.Scheme {
	case "tcp", "http":
		parsed.Scheme = "ws"
	case "https":
		parsed.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", fmt.Errorf("remoteAddress scheme is unsupported: %s", parsed.Scheme)
	}
	if parsed.Path == "" || parsed.Path == "/" {
		parsed.Path = "/ws"
	}
	return parsed.String(), nil
}
