package sock5

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"time"
)

const (
	// Time allowed to read from or write a message to the peer.
	writeWait = 30 * time.Second
)

type Socks5Server struct {
	listener net.Listener
	clients  map[string]net.Conn

	peers       map[string]*TcpConnection
	inPeerChan  chan *TcpConnection
	outPeerChan chan *TcpConnection
}

func NewSocks5Server(address string) (*Socks5Server, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}

	fmt.Printf("server start at [ %s ]", address)
	socksServer := &Socks5Server{
		listener:    listener,
		peers:       make(map[string]*TcpConnection),
		clients:     make(map[string]net.Conn),
		inPeerChan:  make(chan *TcpConnection),
		outPeerChan: make(chan *TcpConnection),
	}
	go socksServer.Run()
	return socksServer, nil
}

func (server *Socks5Server) HandleConnection(conn net.Conn) {
	tcpConnection := &TcpConnection{
		id:          randSeq(10),
		conn:        conn,
		reader:      bufio.NewReader(conn),
		server:      server,
		message:     make(chan []byte),
		processor:   &HandshakeProcessor{},
		targetProxy: make(map[string]*ProxyConnection),
	}

	go startTransform(tcpConnection)

	server.inPeerChan <- tcpConnection
}

func startTransform(tcpConnection *TcpConnection) {
	defer func() {
		tcpConnection.Close()
	}()

	go tcpConnection.Run()
	for {
		success := tcpConnection.processor.execute(tcpConnection)
		if !success {
			return
		}

		processor := tcpConnection.processor.nextProcessor()
		if processor == nil {
			return
		}
		tcpConnection.processor = processor

	}
}

func (server *Socks5Server) Run() {
	for {
		select {
		case inPeer := <-server.inPeerChan:
			server.peers[inPeer.id] = inPeer
			server.clients[inPeer.id] = inPeer.conn
		case outPeer := <-server.outPeerChan:
			delete(server.peers, outPeer.id)
			delete(server.clients, outPeer.id)
		}
	}
}

func (server *Socks5Server) Start() error {
	for {
		conn, err := server.listener.Accept()

		if err != nil {
			return err
		}

		// 异步处理，防止阻塞处理routine
		server.HandleConnection(conn)
	}
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
