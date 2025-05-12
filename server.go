package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/PIRSON21/dfs/p2p"
)

type FileServerOpts struct {
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
}

type FileServer struct {
	FileServerOpts

	peerLock sync.Mutex
	peers    map[string]p2p.Peer

	store  *Store
	quitCh chan struct{}
}

func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := StoreOpts{
		Root:              opts.StorageRoot,
		PathTransformFunc: opts.PathTransformFunc,
	}

	return &FileServer{
		FileServerOpts: opts,
		peers:          make(map[string]p2p.Peer),
		store:          NewStore(storeOpts),
		quitCh:         make(chan struct{}),
	}
}

type Message struct {
	From    string
	Payload any
}

type MessageStoreFile struct {
	Key string
}

func (fs *FileServer) broadcast(msg *Message) error {
	peers := []io.Writer{}
	for _, peer := range fs.peers {
		peers = append(peers, peer)
	}

	mw := io.MultiWriter(peers...)

	return gob.NewEncoder(mw).Encode(msg)
}

func (fs *FileServer) StoreData(key string, r io.Reader) error {
	// 1. Store this file to disk
	// 2. Broadcast this file to all known peers in the network

	buf := new(bytes.Buffer)
	msg := Message{
		Payload: MessageStoreFile{
			Key: key,
		},
	}
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}

	for _, peer := range fs.peers {
		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}
	}

	// time.Sleep(3 * time.Second)

	// payload := []byte("THIS LARGE FILE")
	// for _, peer := range fs.peers {
	// 	err := peer.Send(payload)
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	return nil

	// buf := new(bytes.Buffer)
	// tee := io.TeeReader(r, buf)

	// if err := fs.store.Write(key, tee); err != nil {
	// 	return err
	// }

	// p := &DataMessage{
	// 	Key:  key,
	// 	Data: buf.Bytes(),
	// }

	// return fs.broadcast(&Message{
	// 	From:    "todo", // TODO:
	// 	Payload: p,
	// })
}

func (fs *FileServer) Stop() {
	close(fs.quitCh)
}

func (fs *FileServer) OnPeer(p p2p.Peer) error {
	fs.peerLock.Lock()
	defer fs.peerLock.Unlock()

	fs.peers[p.RemoteAddr().String()] = p

	log.Printf("connected with remote %s\n", p.RemoteAddr())

	return nil
}

func (fs *FileServer) loop() {
	defer func() {
		log.Println("file server stopped due to user quit action")
	}()

	for {
		select {
		case rpc := <-fs.Transport.Consume():
			var msg Message
			err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg)
			if err != nil {
				log.Println(err)
			}

			fmt.Printf("%+v\n", msg.Payload)

			peer, ok := fs.peers[rpc.From]
			if !ok {
				panic("peer not found in peers map")
			}

			b := make([]byte, 1024)
			_, err = peer.Read(b)
			if err != nil {
				panic(err)
			}

			peer.(*p2p.TCPPeer).Wg.Done()

			// if err := fs..Bytes(handleMessage(&m); err != nil {
			// 	log.Println(err)
			// }
		case <-fs.quitCh:
			return
		}
	}
}

// func (fs *FileServer) handleMessage(msg *Message) error {
// 	switch v := msg.Payload.(type) {
// 	case *DataMessage:
// 		fmt.Printf("received data %+v\n", v)
// 	}
// 	return nil
// }

func (fs *FileServer) bootstrapNetwork() error {
	for _, addr := range fs.BootstrapNodes {
		if len(addr) == 0 {
			continue
		}
		go func(addr string) {
			log.Println("attempting to connect with remote: ", addr)
			if err := fs.Transport.Dial(addr); err != nil {
				log.Printf("dial error: %s\n", err)
			}
		}(addr)
	}

	return nil
}

func (fs *FileServer) Start() error {
	if err := fs.Transport.ListenAndAccept(); err != nil {
		return err
	}

	fs.bootstrapNetwork()

	fs.loop()

	return nil
}

func init() {
	gob.Register(MessageStoreFile{})
}
