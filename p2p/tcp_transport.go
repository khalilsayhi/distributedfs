package p2p

import (
	"errors"
	"fmt"
	"net"
	"sync"
)

// TCPPeer is an interface that represents the remote node over TCP-established connection
type TCPPeer struct {
	// The underlying connection of the peer
	net.Conn
	// If we dial out this peer ==> outbound is true
	// If we accept a connection from this peer ==> outbound is false
	outbound bool

	wg *sync.WaitGroup
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		Conn:     conn,
		outbound: outbound,
		wg:       &sync.WaitGroup{},
	}
}

// CloseStream closes the stream
func (p *TCPPeer) CloseStream() {
	p.wg.Done()
}

// Send sends a message to the peer
func (p *TCPPeer) Send(msg []byte) error {
	_, err := p.Write(msg)

	return err
}

type TCPTransportOpt struct {
	ListenAddress string
	HandshakeFunc HandshakeFunc
	Decoder       Decoder
	OnPeer        func(Peer) error
}
type TCPTransport struct {
	listener net.Listener
	TCPTransportOpt
	rpcch chan RPC
}

func NewTCPTransport(opts TCPTransportOpt) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpt: opts,
		rpcch:           make(chan RPC, 1024),
	}
}

// Addr implements the Transport interface
func (t *TCPTransport) Addr() string {
	return t.ListenAddress
}

// ListenAndAccept implements the Transport interface
func (t *TCPTransport) ListenAndAccept() error {
	var err error
	t.listener, err = net.Listen("tcp", t.ListenAddress)
	if err != nil {
		return err
	}

	fmt.Printf("TCP transport Listening on %s\n", t.ListenAddress)

	go t.startAcceptLoop()

	return nil
}

// Close implements the Transport interface
func (t *TCPTransport) Close() error {
	return t.listener.Close()
}

// Consume implements the Transport interface
// It returns a channel that will receive RPC messages from another peer
func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcch
}

// Dial implements the Transport interface
func (t *TCPTransport) Dial(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	go t.handleConn(conn, true)

	return nil
}

func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			fmt.Printf("TCP Accept error: %s\n", err)
		}

		go t.handleConn(conn, false)
	}
}

func (t *TCPTransport) handleConn(conn net.Conn, isOutbound bool) {
	var err error
	defer func() {
		fmt.Printf("Dropping peer connection: %s\n", err)
		_ = conn.Close()
	}()
	peer := NewTCPPeer(conn, isOutbound)
	if err = t.HandshakeFunc(peer); err != nil {
		return
	}
	if t.OnPeer != nil {
		if err = t.OnPeer(peer); err != nil {
			fmt.Printf("Error calling OnPeer: %s\n", err)
			return
		}
	}
	// read loop
	for {
		rpc := RPC{}
		err = t.Decoder.Decode(conn, &rpc)
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			fmt.Printf("TCP read error: %s\n", err)
			continue
		}

		rpc.From = conn.RemoteAddr().String()
		if rpc.IsStreaming {
			fmt.Printf("[%s] incoming stream, Waiting...\n", conn.RemoteAddr())
			peer.wg.Add(1)
			peer.wg.Wait()
			fmt.Printf("[%s] stream closed, resuming read loop\n", conn.RemoteAddr())
			continue
		}
		t.rpcch <- rpc
	}
}
