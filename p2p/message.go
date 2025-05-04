package p2p

const (
	IncomingMessage = 0x1
	IncomingStream  = 0x2
)

type RPC struct {
	Payload     []byte
	From        string
	IsStreaming bool
}
