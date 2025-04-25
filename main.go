package main

import (
	"log"

	"github.com/khalilsayhi/myfsstore/p2p"
)

func main() {
	tr := p2p.NewTCPTransport(":4000")
	if err := tr.ListenAndAccept(); err != nil {
		log.Fatalf("Error starting TCP transport: %v", err)
	}

	select {}
}
