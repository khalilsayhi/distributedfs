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
	buf := make([]byte, 2048)
	n, err := r.Read(buf)
	if err != nil {
		return err
	}
	msg.Payload = buf[:n]

	return nil
}
