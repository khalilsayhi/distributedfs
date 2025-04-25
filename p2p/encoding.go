package p2p

import "io"

type Decoder interface {
	// Decode decodes the data from the connection
	Decode(io.Reader, any) error
}
