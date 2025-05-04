package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/khalilsayhi/myfsstore/p2p"
)

type FileServerOpts struct {
	StorageRoot      string
	KeyTransformFunc KeyTransformFunc
	Transport        p2p.Transport
	BootstrapNodes   []string
}
type FileServer struct {
	FileServerOpts
	store    *Store
	peerLock sync.Mutex
	peers    map[string]p2p.Peer
	quitch   chan struct{}
}

func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := StoreOpts{
		Root:              opts.StorageRoot,
		KeyTransformeFunc: opts.KeyTransformFunc,
	}
	return &FileServer{
		FileServerOpts: opts,
		store: NewStore(
			storeOpts,
		),
		peers:  make(map[string]p2p.Peer),
		quitch: make(chan struct{}),
	}
}

type Message struct {
	Payload any
}
type MessageStoreFile struct {
	Key  string
	Size int64
}
type MessageGetFile struct {
	Key string
}

func (fs *FileServer) stream(msg *Message) error {
	var peers []io.Writer
	for _, peer := range fs.peers {
		peers = append(peers, peer)
	}
	mw := io.MultiWriter(peers...)
	return gob.NewEncoder(mw).Encode(msg)
}

func (fs *FileServer) broadcast(msg *Message) error {
	buf := new(bytes.Buffer)

	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}

	for _, peer := range fs.peers {
		_ = peer.Send([]byte{p2p.IncomingMessage})
		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}
	}

	return nil
}

func (fs *FileServer) Get(key string) (io.Reader, error) {
	if fs.store.HasKey(key) {
		fmt.Printf("[%s] file (%s) found on local disk\n", fs.Transport.Addr(), key)
		_, r, err := fs.store.Read(key)

		return r, err
	}
	fmt.Printf("[%s] file (%s) not found on local disk, fetching from network...\n", fs.Transport.Addr(), key)

	if len(fs.peers) == 0 {
		return nil, fmt.Errorf("no peers available")
	}
	msg := Message{
		Payload: MessageGetFile{
			Key: key,
		},
	}
	if err := fs.broadcast(&msg); err != nil {
		return nil, err
	}

	time.Sleep(5 * time.Millisecond)
	for _, peer := range fs.peers {
		/*		fileBuffer := new(bytes.Buffer)
				n, err := io.Copy(peer, fileBuffer)
				if err != nil {
					return nil, err
				}*/
		// first read the file size so we can limit the reader
		var fileSize int64
		if err := binary.Read(peer, binary.LittleEndian, &fileSize); err != nil {
			return nil, err
		}
		n, err := fs.store.Write(key, io.LimitReader(peer, fileSize))
		if err != nil {
			return nil, err
		}

		fmt.Printf("[%s] received (%d) bytes from network from %s\n", fs.Transport.Addr(), n, peer.RemoteAddr())

		peer.CloseStream()
	}

	_, r, err := fs.store.Read(key)

	return r, err
}

func (fs *FileServer) Delete(key string) error {
	/*	if err := fs.store.Delete(key); err != nil {
			return err
		}

		msg := Message{
			Payload: MessageStoreFile{
				Key: key,
			},
		}
		if err := fs.broadcast(&msg); err != nil {
			return err
		}
	*/
	return nil
}

func (fs *FileServer) Store(key string, r io.Reader) error {
	// store file on disk
	// broadcast to all known peers
	var (
		fileBuffer = new(bytes.Buffer)
		tee        = io.TeeReader(r, fileBuffer)
	)

	size, err := fs.store.Write(key, tee)
	if err != nil {
		return err
	}

	msg := Message{
		Payload: MessageStoreFile{
			Key:  key,
			Size: size,
		},
	}
	if err := fs.broadcast(&msg); err != nil {
		return err
	}

	time.Sleep(5 * time.Millisecond)
	for _, peer := range fs.peers {
		_ = peer.Send([]byte{p2p.IncomingStream})
		n, err := io.Copy(peer, fileBuffer)
		if err != nil {
			return err
		}
		fmt.Printf("[%s] received and wrote (%d) bytes to disk: \n", fs.Transport.Addr(), n)
	}
	return nil
}

func (fs *FileServer) Stop() {
	close(fs.quitch)

}

func (fs *FileServer) OnPeer(peer p2p.Peer) error {
	fs.peerLock.Lock()
	defer fs.peerLock.Unlock()
	if _, ok := fs.peers[peer.RemoteAddr().String()]; ok {
		return nil
	}
	fs.peers[peer.RemoteAddr().String()] = peer

	log.Printf("peer %s added", peer.RemoteAddr().String())

	return nil
}

func (fs *FileServer) loop() {
	defer func() {
		fmt.Println("FileServer stopped due to error or user interruption")
		_ = fs.Transport.Close()
	}()

	for {
		select {
		case rpc := <-fs.Transport.Consume():
			var msg Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				log.Println("decoding error: ", err)
			}
			if err := fs.handleMessage(rpc.From, &msg); err != nil {
				log.Println("handle message error: ", err)
			}

		case <-fs.quitch:
			return
		}
	}
}

func (fs *FileServer) handleMessage(from string, msg *Message) error {
	switch v := msg.Payload.(type) {
	case MessageStoreFile:
		return fs.handleMessageStoreFile(from, v)
	case MessageGetFile:
		return fs.handleMessageGetFile(from, v)
	}
	return nil
}

func (fs *FileServer) handleMessageStoreFile(from string, msg MessageStoreFile) error {
	peer, ok := fs.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) not found in the peer map", from)
	}

	n, err := fs.store.Write(msg.Key, io.LimitReader(peer, msg.Size))
	if err != nil {
		return err
	}

	fmt.Printf("wrote %d bytes to disk\n", n)

	peer.CloseStream()

	return nil
}

func (fs *FileServer) handleMessageGetFile(from string, msg MessageGetFile) error {
	if !fs.store.HasKey(msg.Key) {
		fmt.Printf("[%s] need to serve file (%s) but it doese not exist on dis\n", fs.Transport.Addr(), msg.Key)

		return nil
	}
	fmt.Printf("[%s] serving file (%s) over the network\n", fs.Transport.Addr(), msg.Key)
	peer, ok := fs.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) not found in the peer map", from)
	}

	fileSize, r, err := fs.store.Read(msg.Key)
	if err != nil {
		return err
	}
	if rc, ok := r.(io.ReadCloser); ok {
		defer func(rc io.ReadCloser) {
			err := rc.Close()
			if err != nil {

			}
		}(rc)
	}
	// send incoming stream message
	// then send the file size
	_ = peer.Send([]byte{p2p.IncomingStream})
	_ = binary.Write(peer, binary.LittleEndian, fileSize)

	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}

	fmt.Printf("[%s] sent (%d) bytes over the network to peer: %s\n", fs.Transport.Addr(), n, from)

	return nil
}

func (fs *FileServer) bootstrapNetwork() error {
	for _, addr := range fs.BootstrapNodes {
		if addr == "" {
			continue
		}
		go func(addr string) {
			if err := fs.Transport.Dial(addr); err != nil {
				log.Printf("Dial error: %s\n", err)
			}
		}(addr)
	}

	return nil
}

func (fs *FileServer) Start() error {
	if err := fs.Transport.ListenAndAccept(); err != nil {
		return err
	}

	if len(fs.BootstrapNodes) != 0 {
		_ = fs.bootstrapNetwork()
	}

	fs.loop()

	return nil
}

func init() {
	gob.Register(Message{})
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
}
