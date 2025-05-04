package p2p

import (
	"encoding/gob"
	"io"
)

type Decoder interface {
	// Decode decodes the data from the connection
	Decode(io.Reader, *RPC) error
}
type GOBDecoder struct {
}

func (dec GOBDecoder) Decode(r io.Reader, rpc *RPC) error {
	return gob.NewDecoder(r).Decode(rpc)
}

type DefaultDecoder struct {
}

func (dec DefaultDecoder) Decode(r io.Reader, msg *RPC) error {
	peekBuf := make([]byte, 1)
	if _, err := r.Read(peekBuf); err != nil {
		return err
	}

	// IN case of streaming, we don't need to decode what is being sent over the network
	// we just set the flag and return
	isStreaming := peekBuf[0] == IncomingStream
	if isStreaming {
		msg.IsStreaming = true
		return nil
	}

	buf := make([]byte, 2048)
	n, err := r.Read(buf)
	if err != nil {
		return err
	}
	msg.Payload = buf[:n]

	return nil
}
