package p2p

import "net"

// Peer is an interface that represents the remote node
type Peer interface {
	Send([]byte) error
	net.Conn
}

// Transport is anything that handles communication
// between the nodes in the network. This can be of the
// form (TCP, UDP, SOCKET, ...)
type Transport interface {
	ListenAndAccept() error
	Consume() <-chan RPC
	Close() error
	Dial(string) error
}
