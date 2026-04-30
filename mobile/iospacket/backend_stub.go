//go:build !ios
// +build !ios

package iospacket

import "fmt"

type PacketFlow interface {
	ReadPacket() ([]byte, error)
	WritePacket(packet []byte) error
	Close()
}

type Backend struct{}

func NewBackend(packetFlow PacketFlow, remoteAddress string, password string, mtu int, socks5Port int, poolSize int) (*Backend, error) {
	return nil, fmt.Errorf("iospacket backend is only available when building for iOS")
}

func (b *Backend) Start() error {
	return fmt.Errorf("iospacket backend is only available when building for iOS")
}

func (b *Backend) Stop() {}
