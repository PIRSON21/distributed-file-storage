package main

import (
	"bytes"
	"log"
	"time"

	"github.com/PIRSON21/dfs/p2p"
)

func mustMakeServer(listenAddr string, nodes ...string) *FileServer {
	tcpTransportOpts := p2p.TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
	}

	tcpTransport := p2p.NewTCPTransport(tcpTransportOpts)

	fileServerOpts := FileServerOpts{
		StorageRoot:       listenAddr + "_network",
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tcpTransport,
		BootstrapNodes:    nodes,
	}

	fs := NewFileServer(fileServerOpts)

	tcpTransport.OnPeer = fs.OnPeer

	return fs
}

func main() {
	fs1 := mustMakeServer(":3000", "")

	go func() {
		log.Fatal(fs1.Start())
	}()
	fs2 := mustMakeServer(":4000", ":3000")

	go func() {
		log.Fatal(fs2.Start())
	}()
	time.Sleep(time.Second * 1)

	data := bytes.NewReader([]byte("my big data file here!"))

	if err := fs2.StoreData("myprivatedata", data); err != nil {
		log.Fatal(err)
	}

	select {}
}
