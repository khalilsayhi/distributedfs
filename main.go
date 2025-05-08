package main

import (
	"bytes"
	"log"
	"time"

	"github.com/khalilsayhi/myfsstore/p2p"
)

func makeServer(listenAddr string, nodes ...string) *FileServer {
	tcpTransportOpts := p2p.TCPTransportOpt{
		ListenAddress: listenAddr,
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
	}
	tcpTransport := p2p.NewTCPTransport(tcpTransportOpts)

	fileServerOpts := FileServerOpts{
		StorageRoot:      listenAddr + "_network",
		KeyTransformFunc: CASKeyTransformFunc,
		Transport:        tcpTransport,
		BootstrapNodes:   nodes,
		EncKey:           newEncryptorKey(),
	}

	s := NewFileServer(fileServerOpts)
	tcpTransport.OnPeer = s.OnPeer

	return s
}

func main() {
	/*	tcpOpts := p2p.TCPTransportOpt{
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
	*/

	s1 := makeServer(":3000", "")
	s2 := makeServer(":4000", ":3000")

	go func() {
		log.Fatal(s1.Start())
	}()

	go s2.Start()
	time.Sleep(1 * time.Second)

	data := bytes.NewReader([]byte("my big data is here"))
	s2.Store("myprivatedata", data)

	/*	r, err := s2.Get("myprivatedata")
		if err != nil {
			log.Printf("Error getting file: %v", err)
		}
		b, err := io.ReadAll(r)
		if err != nil {
			log.Printf("Error reading file: %v", err)
		}
		log.Printf("File content: %s", string(b))
		log.Printf("File retrieved successfully")*/
}
