package p2p

import (
	"fmt"
	"net"
	"sync"
)

// TCPPeer is an interface that represents the remote node over TCP established connection
type TCPPeer struct {
	// The underlying connection of the peer
	conn net.Conn
	// If we dial out this peer ==> outbound is true
	// If we accept a connection from this peer ==> outbound is false
	outbound bool
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

type TCPTransport struct {
	listenAddress string
	listener      net.Listener
	shakeHands    HandshakeFunc
	decoder       Decoder

	mu    sync.RWMutex
	peers map[net.Addr]Peer
}

func NewTCPTransport(listenerAddress string) *TCPTransport {
	return &TCPTransport{
		listenAddress: listenerAddress,
		shakeHands:    NOPHandshakeFunc,
	}

}

func (t *TCPTransport) ListenAndAccept() error {
	var err error
	t.listener, err = net.Listen("tcp", t.listenAddress)
	if err != nil {
		return err
	}
	go t.startAcceptLoop()

	return nil
}

func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			fmt.Printf("TCP Accept error: %s\n", err)
		}
		fmt.Printf("New incoming connection from %+v\n", conn)

		go t.handleConn(conn)
	}
}

type Temp struct {
}

func (t *TCPTransport) handleConn(conn net.Conn) {
	peer := NewTCPPeer(conn, true)
	if err := t.shakeHands(peer); err != nil {
		conn.Close()
		fmt.Printf("Handshake error: %s\n", err)
		return
	}

	// read loop
	msg := &Temp{}
	for {
		if err := t.decoder.Decode(conn, msg); err != nil {
			fmt.Printf("TCP error: %s\n", err)
		}
	}
}
