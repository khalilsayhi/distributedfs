package main

import (
	"bytes"
	"fmt"
	"io"
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
	s3 := makeServer(":5000", ":3000", ":4000")

	go func() {
		log.Fatal(s1.Start())
	}()
	time.Sleep(1 * time.Second)
	go func() {
		log.Fatal(s2.Start())
	}()

	time.Sleep(2 * time.Second)
	go s3.Start()
	time.Sleep(2 * time.Second)
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("myprivatedata_%d", i)
		data := bytes.NewReader([]byte("my big data is here"))
		s3.Store(key, data)

		if err := s3.store.Delete(key); err != nil {
			log.Fatal(err)
		}

		r, err := s3.Get(key)
		if err != nil {
			log.Fatal(err)
		}

		b, err := io.ReadAll(r)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("File content: %s", string(b))
	}

}
