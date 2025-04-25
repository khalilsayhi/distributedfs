package p2p

// HandshakeFunc is a function that performs a handshake with the peer
type HandshakeFunc func(Peer) error

func NOPHandshakeFunc(Peer) error {
	return nil
}
