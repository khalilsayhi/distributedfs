package main

import (
	"fmt"
	"log"

	"github.com/khalilsayhi/myfsstore/p2p"
)

func OnPeer(p2p.Peer) error {
	fmt.Println("New peer connected")

	return nil
}
func main() {
	tcpOpts := p2p.TCPTransportOpt{
		ListenAddress: ":4000",
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
		OnPeer:        OnPeer,
	}
	tr := p2p.NewTCPTransport(tcpOpts)

	go func() {
		for {
			msg := <-tr.Consume()
			fmt.Printf("Received message: %+v\n", msg)
		}
	}()
	if err := tr.ListenAndAccept(); err != nil {
		log.Fatalf("Error starting TCP transport: %v", err)
	}

	select {}
}
