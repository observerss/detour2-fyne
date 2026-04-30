//go:build android
// +build android

package proxy

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xjasonlyu/tun2socks/v2/core"
	"github.com/xjasonlyu/tun2socks/v2/core/device"
	"github.com/xjasonlyu/tun2socks/v2/core/device/fdbased"
	M "github.com/xjasonlyu/tun2socks/v2/metadata"
	tunproxy "github.com/xjasonlyu/tun2socks/v2/proxy"
	"github.com/xjasonlyu/tun2socks/v2/tunnel"
	"github.com/xjasonlyu/tun2socks/v2/tunnel/statistic"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

const defaultVPNMTU = 1500

var forwarder = &vpnForwarder{}

type vpnForwarder struct {
	mu          sync.Mutex
	device      device.Device
	stack       *stack.Stack
	tunnel      *tunnel.Tunnel
	proxyServer string
}

func StartTunForwarder(fd int) error {
	forwarder.mu.Lock()
	defer forwarder.mu.Unlock()

	if fd < 0 {
		return errors.New("Android VPN TUN fd 无效")
	}

	proxyServer := strings.TrimSpace(forwarder.proxyServer)
	if proxyServer == "" {
		return errors.New("Android VPN 缺少本地 SOCKS5 代理地址")
	}

	forwarder.stopLocked()

	proxyClient, err := tunproxy.NewSocks5(proxyServer, "", "")
	if err != nil {
		setStatus("Android VPN 创建 SOCKS5 客户端失败: " + err.Error())
		return err
	}
	proxyDialer := &dnsOverTCPDialer{tcp: proxyClient}

	fdDevice, err := fdbased.Open(strconv.Itoa(fd), defaultVPNMTU, 0)
	if err != nil {
		setStatus("Android VPN 打开 TUN fd 失败: " + err.Error())
		return err
	}

	tunTunnel := tunnel.New(proxyDialer, statistic.DefaultManager)
	tunTunnel.ProcessAsync()

	tunStack, err := core.CreateStack(&core.Config{
		LinkEndpoint:     fdDevice,
		TransportHandler: tunTunnel,
	})
	if err != nil {
		tunTunnel.Close()
		fdDevice.Close()
		setStatus("Android VPN 创建网络栈失败: " + err.Error())
		return err
	}

	forwarder.device = fdDevice
	forwarder.stack = tunStack
	forwarder.tunnel = tunTunnel
	setStatus(fmt.Sprintf("Android VPN 已连接到 SOCKS5 %s", proxyServer))
	return nil
}

type dnsOverTCPDialer struct {
	tcp tunproxy.Dialer
}

func (d *dnsOverTCPDialer) DialContext(ctx context.Context, metadata *M.Metadata) (net.Conn, error) {
	return d.tcp.DialContext(ctx, metadata)
}

func (d *dnsOverTCPDialer) DialUDP(metadata *M.Metadata) (net.PacketConn, error) {
	if metadata != nil && metadata.DstPort == 53 {
		return newDNSOverTCPPacketConn(d.tcp, metadata), nil
	}
	return newUDPDropPacketConn(), nil
}

type dnsOverTCPPacketConn struct {
	dialer       tunproxy.Dialer
	metadata     M.Metadata
	responses    chan dnsPacket
	closed       chan struct{}
	closeOnce    sync.Once
	deadlineMu   sync.Mutex
	readDeadline time.Time
}

type dnsPacket struct {
	payload []byte
	addr    net.Addr
}

type dnsTimeoutError struct{}

func (dnsTimeoutError) Error() string   { return "dns over tcp read timeout" }
func (dnsTimeoutError) Timeout() bool   { return true }
func (dnsTimeoutError) Temporary() bool { return true }

func newDNSOverTCPPacketConn(dialer tunproxy.Dialer, metadata *M.Metadata) *dnsOverTCPPacketConn {
	return &dnsOverTCPPacketConn{
		dialer:    dialer,
		metadata:  *metadata,
		responses: make(chan dnsPacket, 16),
		closed:    make(chan struct{}),
	}
}

func (pc *dnsOverTCPPacketConn) ReadFrom(buffer []byte) (int, net.Addr, error) {
	readDeadline := pc.currentReadDeadline()
	var timer <-chan time.Time
	if !readDeadline.IsZero() {
		duration := time.Until(readDeadline)
		if duration <= 0 {
			return 0, nil, dnsTimeoutError{}
		}
		deadlineTimer := time.NewTimer(duration)
		defer deadlineTimer.Stop()
		timer = deadlineTimer.C
	}

	select {
	case response := <-pc.responses:
		copied := copy(buffer, response.payload)
		return copied, response.addr, nil
	case <-pc.closed:
		return 0, nil, io.EOF
	case <-timer:
		return 0, nil, dnsTimeoutError{}
	}
}

func (pc *dnsOverTCPPacketConn) WriteTo(payload []byte, addr net.Addr) (int, error) {
	response, err := pc.query(payload)
	if err != nil {
		return 0, err
	}
	select {
	case pc.responses <- dnsPacket{payload: response, addr: addr}:
		return len(payload), nil
	case <-pc.closed:
		return 0, io.ErrClosedPipe
	}
}

func (pc *dnsOverTCPPacketConn) Close() error {
	pc.closeOnce.Do(func() {
		close(pc.closed)
	})
	return nil
}

func (pc *dnsOverTCPPacketConn) LocalAddr() net.Addr {
	return &net.UDPAddr{IP: net.IPv4zero, Port: 0}
}

func (pc *dnsOverTCPPacketConn) SetDeadline(deadline time.Time) error {
	return pc.SetReadDeadline(deadline)
}

func (pc *dnsOverTCPPacketConn) SetReadDeadline(deadline time.Time) error {
	pc.deadlineMu.Lock()
	pc.readDeadline = deadline
	pc.deadlineMu.Unlock()
	return nil
}

func (pc *dnsOverTCPPacketConn) SetWriteDeadline(time.Time) error {
	return nil
}

func (pc *dnsOverTCPPacketConn) currentReadDeadline() time.Time {
	pc.deadlineMu.Lock()
	defer pc.deadlineMu.Unlock()
	return pc.readDeadline
}

func (pc *dnsOverTCPPacketConn) query(payload []byte) ([]byte, error) {
	metadata := pc.metadata
	metadata.Network = M.TCP
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	conn, err := pc.dialer.DialContext(ctx, &metadata)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(8 * time.Second)); err != nil {
		return nil, err
	}

	request := make([]byte, len(payload)+2)
	binary.BigEndian.PutUint16(request[:2], uint16(len(payload)))
	copy(request[2:], payload)
	if _, err := conn.Write(request); err != nil {
		return nil, err
	}

	var responseLength [2]byte
	if _, err := io.ReadFull(conn, responseLength[:]); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint16(responseLength[:])
	response := make([]byte, length)
	if _, err := io.ReadFull(conn, response); err != nil {
		return nil, err
	}
	return response, nil
}

type udpDropPacketConn struct {
	closed       chan struct{}
	closeOnce    sync.Once
	deadlineMu   sync.Mutex
	readDeadline time.Time
}

func newUDPDropPacketConn() *udpDropPacketConn {
	return &udpDropPacketConn{closed: make(chan struct{})}
}

func (pc *udpDropPacketConn) ReadFrom(buffer []byte) (int, net.Addr, error) {
	readDeadline := pc.currentReadDeadline()
	if readDeadline.IsZero() {
		<-pc.closed
		return 0, nil, io.EOF
	}

	duration := time.Until(readDeadline)
	if duration <= 0 {
		return 0, nil, dnsTimeoutError{}
	}
	deadlineTimer := time.NewTimer(duration)
	defer deadlineTimer.Stop()

	select {
	case <-pc.closed:
		return 0, nil, io.EOF
	case <-deadlineTimer.C:
		return 0, nil, dnsTimeoutError{}
	}
}

func (pc *udpDropPacketConn) WriteTo(payload []byte, addr net.Addr) (int, error) {
	select {
	case <-pc.closed:
		return 0, io.ErrClosedPipe
	default:
		return len(payload), nil
	}
}

func (pc *udpDropPacketConn) Close() error {
	pc.closeOnce.Do(func() {
		close(pc.closed)
	})
	return nil
}

func (pc *udpDropPacketConn) LocalAddr() net.Addr {
	return &net.UDPAddr{IP: net.IPv4zero, Port: 0}
}

func (pc *udpDropPacketConn) SetDeadline(deadline time.Time) error {
	return pc.SetReadDeadline(deadline)
}

func (pc *udpDropPacketConn) SetReadDeadline(deadline time.Time) error {
	pc.deadlineMu.Lock()
	pc.readDeadline = deadline
	pc.deadlineMu.Unlock()
	return nil
}

func (pc *udpDropPacketConn) SetWriteDeadline(time.Time) error {
	return nil
}

func (pc *udpDropPacketConn) currentReadDeadline() time.Time {
	pc.deadlineMu.Lock()
	defer pc.deadlineMu.Unlock()
	return pc.readDeadline
}

func StopTunForwarder() {
	forwarder.mu.Lock()
	defer forwarder.mu.Unlock()

	forwarder.stopLocked()
	setStatus("Android VPN 已停止")
}

func setTunForwarderProxy(proxyServer string) {
	forwarder.mu.Lock()
	forwarder.proxyServer = strings.TrimSpace(proxyServer)
	forwarder.mu.Unlock()
}

func (f *vpnForwarder) stopLocked() {
	if f.device != nil {
		f.device.Close()
		f.device = nil
	}
	if f.stack != nil {
		f.stack.Close()
		f.stack.Wait()
		f.stack = nil
	}
	if f.tunnel != nil {
		f.tunnel.Close()
		f.tunnel = nil
	}
}
