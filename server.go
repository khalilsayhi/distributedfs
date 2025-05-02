package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/khalilsayhi/myfsstore/p2p"
)

6:51:29

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
		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}
	}

	return nil
}

func (fs *FileServer) Get(key string) (io.Reader, error) {
	if fs.store.HasKey(key) {
		return fs.store.Read(key)
	}

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
	for _, peer := range fs.peers {
		fileBuffer := new(bytes.Buffer)
		n, err := io.Copy(peer, fileBuffer)
		if err != nil {
			return nil, err
		}
		fmt.Printf("received (%d) file from peer: ", n)
	}

	select {}

	return nil, fmt.Errorf("file not found")
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

	time.Sleep(time.Second * 3)
	for _, peer := range fs.peers {
		_, err := io.Copy(peer, fileBuffer)
		if err != nil {
			return err
		}

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

	peer.(*p2p.TCPPeer).Wg.Done()

	return nil
}

func (fs *FileServer) handleMessageGetFile(from string, msg MessageGetFile) error {
	if !fs.store.HasKey(msg.Key) {
		fmt.Printf("file (%s) not found on disk", msg.Key)

		return nil
	}
	fmt.Printf("serving file (%s) over the network\n", msg.Key)
	peer, ok := fs.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) not found in the peer map", from)
	}

	r, err := fs.store.Read(msg.Key)
	if err != nil {
		return err
	}

	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}
	log.Printf("sent %d bytes over the network to peer: %s\n", n, from)

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
